package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

func cmdTestDeployAndShare() {
	flagset := flag.NewFlagSet("deployBlockSenderContract", flag.ExitOnError)

	var (
		suaveRpc         = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		goerliRpc        = flagset.String("goerli_rpc", "http://127.0.0.1:8545", "address of goerli rpc")
		goerliBeaconRpc  = flagset.String("goerli_beacon_rpc", "http://127.0.0.1:5052", "address of goerli beacon rpc")
		kettleAddressHex = flagset.String("kettleAddress", "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B", "wallet address of execution node")
		privKeyHex       = flagset.String("privkey", "", "private key as hex (for testing)")
		boostRelayUrl    = flagset.String("relay_url", "http://127.0.0.1:8091", "address of boost relay that the contract will send blocks to")
		verbosity        = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		privKey          *ecdsa.PrivateKey
	)

	flagset.Parse(os.Args[2:])

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	privKey, kettleAddress, suaveClient, goerliClient, suaveSigner, goerliSigner := setUpSuaveAndGoerli(privKeyHex, kettleAddressHex, suaveRpc, goerliRpc)

	// ********** Deploy Builder Contract **********

	blockSenderAddrPtr, txHash, err := sendBlockSenderCreationTx(suaveClient, suaveSigner, privKey, boostRelayUrl)
	if err != nil {
		panic(err.Error())
	}

	waitForTransactionToBeConfirmed(suaveClient, txHash)
	blockSenderAddr := *blockSenderAddrPtr

	/*
		// To avoid redeploying the contract, use this instead of the above
		blockSenderAddr := common.HexToAddress("0xFcF6C8bBa8507E494D2aDf4F5C3CE11D8B749E4C")
	*/

	// ********** 2. Deploy Mevshare Contract **********

	mevShareAddrPtr, txHash, err := sendMevShareCreationTx(suaveClient, suaveSigner, privKey)
	if err != nil {
		panic(err.Error())
	}

	waitForTransactionToBeConfirmed(suaveClient, txHash)
	mevShareAddr := *mevShareAddrPtr

	/*
		// To avoid redeploying the contract, use this instead of the above
		mevShareAddr := common.HexToAddress("0x2847FA9D1d5e8992d7281AF1B205c147cEc83817")
	*/

	// ********** 3. Send Mevshare Bundles for the next 26 blocks **********

	mevShareTxs, err := sendMevShareBidTxs(suaveClient, goerliClient, suaveSigner, goerliSigner, 5, mevShareAddr, blockSenderAddr, kettleAddress, privKey)
	if err != nil {
		err = errors.Wrap(err, unwrapPeekerError(err).Error())
		panic(err.Error())
	}

	// Wait only for the last one, all previous txs must have been included by that time
	// We have to wait as otherwise we can't get the bid id via receipts
	waitForTransactionToBeConfirmed(suaveClient, &mevShareTxs[len(mevShareTxs)-1].txHash)

	// ********** 4. Send Mevshare Matches **********

	for _, mevShareTx := range mevShareTxs {
		bidIdBytes, err := extractBidId(suaveClient, mevShareTx.txHash)
		if err != nil {
			panic(err.Error())
		}

		_, err = sendMevShareMatchTx(
			suaveClient,
			goerliClient,
			suaveSigner,
			goerliSigner,
			mevShareTx.blockNumber,
			mevShareAddr,
			blockSenderAddr,
			kettleAddress,
			bidIdBytes,
			privKey,
		)
		if err != nil {
			err = errors.Wrap(err, unwrapPeekerError(err).Error())
			panic(err.Error())
		}
	}

	// No need to wait for the backruns!

	// ********** 5. Send Build Block **********

	payloadAttrC := make(chan PayloadAttributesEvent)
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go SubscribeToPayloadAttributesEvents(ctx, *goerliBeaconRpc, payloadAttrC)

	// subscribe to payload attribute events from beacon chain to build blocks
	for paEvent := range payloadAttrC {
		var goerliBlockNum hexutil.Uint64
		err = goerliClient.Call(&goerliBlockNum, "eth_blockNumber")
		if err != nil {
			log.Error("could not get goerli block", "err", err)
			continue
		}

		if uint64(goerliBlockNum) >= mevShareTxs[len(mevShareTxs)-1].blockNumber {
			cancel()
			continue
		}

		validatorData, err := getValidatorForSlot(*boostRelayUrl, paEvent.Data.ProposalSlot)
		if err != nil || len(validatorData.Pubkey) == 0 {
			log.Error("could not get validator", "slot", paEvent.Data.ProposalSlot, "err", err)
			continue
		}

		log.Info("got validator", "vd", validatorData)

		payloadArgsTuple := struct {
			Slot           uint64
			ProposerPubkey []byte
			Parent         common.Hash
			Timestamp      uint64
			FeeRecipient   common.Address
			GasLimit       uint64
			Random         common.Hash
			Withdrawals    []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			Slot:           paEvent.Data.ProposalSlot,
			Parent:         paEvent.Data.ParentBlockHash,
			Timestamp:      paEvent.Data.PayloadAttributes.Timestamp,
			Random:         paEvent.Data.PayloadAttributes.PrevRandao,
			ProposerPubkey: hexutil.MustDecode(validatorData.Pubkey),
			FeeRecipient:   validatorData.FeeRecipient,
			GasLimit:       validatorData.GasLimit,
		}

		for _, w := range paEvent.Data.PayloadAttributes.Withdrawals {
			payloadArgsTuple.Withdrawals = append(payloadArgsTuple.Withdrawals, struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}{
				Index:     uint64(w.Index),
				Validator: uint64(w.ValidatorIndex),
				Address:   common.Address(w.Address),
				Amount:    uint64(w.Amount),
			})
		}

		for i := 0; i < 3; i++ {
			_, err = sendBuildShareBlockTx(suaveClient, suaveSigner, privKey, kettleAddress, blockSenderAddr, payloadArgsTuple, uint64(goerliBlockNum)+1)
			if err != nil {
				err = errors.Wrap(err, unwrapPeekerError(err).Error())
				if strings.Contains(err.Error(), "no bids") {
					log.Error("Failed to build a block, no bids")
					time.Sleep(2 * time.Second)
					continue
				}
				log.Error("Failed to send BuildShareBlockTx", "err", err)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Info("Sent block to relay", "payload args", payloadArgsTuple, "blockNum", uint64(goerliBlockNum)+1)
			break
		}
	}
}

var (
	err error // shush linter
)
