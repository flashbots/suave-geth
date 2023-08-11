package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type payloadArgs struct {
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
}

func cmdSendBuildShareBlock() {
	flagset := flag.NewFlagSet("sendBuildShareBlock", flag.ExitOnError)

	var (
		suaveRpc                = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		goerliRpc               = flagset.String("goerli_rpc", "http://127.0.0.1:8545", "address of goerli rpc")
		goerliBeaconRpc         = flagset.String("goerli_beacon_rpc", "http://127.0.0.1:5052", "address of goerli beacon rpc")
		boostRelayUrl           = flagset.String("relay_url", "http://127.0.0.1:8091", "address of boost relay that the contract will send blocks to")
		blockSenderAddressHex   = flagset.String("block_sender_addr", "0x42042042028AE1CDE26d5BcF17Ba83f447068E5B", "address of block sender contract")
		executionNodeAddressHex = flagset.String("ex_node_addr", "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B", "wallet address of execution node")
		privKeyHex              = flagset.String("privkey", "", "private key as hex (for testing)")
		verbosity               = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		privKey                 *ecdsa.PrivateKey
	)

	flagset.Parse(os.Args[2:])

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	privKey, err := crypto.HexToECDSA(*privKeyHex)
	RequireNoErrorf(err, "-nodekeyhex: %v", err)
	/* shush linter */ privKey.Public()

	if executionNodeAddressHex == nil || *executionNodeAddressHex == "" {
		utils.Fatalf("please provide ex_node_addr")
	}
	executionNodeAddress := common.HexToAddress(*executionNodeAddressHex)
	blockSenderAddr := common.HexToAddress(*blockSenderAddressHex)

	suaveClient, err := rpc.DialContext(context.TODO(), *suaveRpc)
	RequireNoErrorf(err, "could not connect to suave rpc: %v", err)

	goerliClient, err := rpc.DialContext(context.TODO(), *goerliRpc)
	RequireNoErrorf(err, "could not connect to goerli rpc: %v", err)
	genesis := core.DefaultSuaveGenesisBlock()

	suaveSigner := types.NewOffchainSigner(genesis.Config.ChainID)

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
			_, err = sendBuildShareBlockTx(suaveClient, suaveSigner, privKey, executionNodeAddress, blockSenderAddr, payloadArgsTuple, uint64(goerliBlockNum)+1)
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

func sendBuildShareBlockTx(
	suaveClient *rpc.Client,
	suaveSigner types.Signer,
	privKey *ecdsa.PrivateKey,
	executionNodeAddress common.Address,
	blockSenderAddr common.Address,
	payloadArgsTuple payloadArgs,
	goerliBlockNum uint64,
) (*common.Hash, error) {
	var suaveAccNonceBytes hexutil.Uint64
	err := suaveClient.Call(&suaveAccNonceBytes, "eth_getTransactionCount", crypto.PubkeyToAddress(privKey.PublicKey), "pending")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)
	suaveAccNonce := uint64(suaveAccNonceBytes)

	calldata, err := ethBlockBidSenderAbi.Pack("buildMevShare", payloadArgsTuple, goerliBlockNum)
	RequireNoErrorf(err, "could not pack buildMevShare args: %v", err)

	wrappedTxData := &types.DynamicFeeTx{
		Nonce:     suaveAccNonce,
		To:        &blockSenderAddr,
		Value:     nil,
		Gas:       1000000,
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(33000000000),
		Data:      calldata,
	}

	offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
		ExecutionNode: executionNodeAddress,
		Wrapped:       *types.NewTx(wrappedTxData),
	}), suaveSigner, privKey)
	RequireNoErrorf(err, "could not sign offchainTx: %v", err)

	offchainTxBytes, err := offchainTx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	var offchainTxHash common.Hash
	err = suaveClient.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes))

	return &offchainTxHash, err
}
