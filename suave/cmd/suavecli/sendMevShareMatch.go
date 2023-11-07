package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func cmdSendMevShareMatch() {
	flagset := flag.NewFlagSet("sendBundle", flag.ExitOnError)

	var (
		suaveRpc           = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		kettleAddressHex   = flagset.String("kettleAddress", "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B", "wallet address of execution node")
		mevshareAddressHex = flagset.String("mev_share_addr", "0x42042042028AE1CDE26d5BcF17Ba83f447068E5B", "address of mev share contract")
		blockSenderHex     = flagset.String("block_sender_addr", "0x42042042028AE1CDE26d5BcF17Ba83f447068E5B", "address of mev share contract")
		matchBidId         = flagset.String("match_bid_id", "123-123-123", "ID of mev share bundle bid to back run")
		goerliRpc          = flagset.String("goerli_rpc", "http://127.0.0.1:8545", "address of goerli rpc")
		privKeyHex         = flagset.String("privkey", "", "private key as hex (for testing)")
		verbosity          = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		privKey            *ecdsa.PrivateKey
	)

	flagset.Parse(os.Args[2:])

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	privKey, err := crypto.HexToECDSA(*privKeyHex)
	RequireNoErrorf(err, "-nodekeyhex: %v", err)
	/* shush linter */ privKey.Public()

	if kettleAddressHex == nil || *kettleAddressHex == "" {
		utils.Fatalf("please provide kettleAddress")
	}
	kettleAddress := common.HexToAddress(*kettleAddressHex)
	mevshareAddresss := common.HexToAddress(*mevshareAddressHex)
	blockSenderAddress := common.HexToAddress(*blockSenderHex)

	matchBidIdBytes := [16]byte{}
	copy(matchBidIdBytes[:], []byte(*matchBidId)[:16])
	log.Debug("converted matchBidId to bytes", "matchBidIdBytes", matchBidIdBytes)

	suaveClient, err := rpc.DialContext(context.TODO(), *suaveRpc)
	RequireNoErrorf(err, "could not connect to suave rpc: %v", err)

	goerliClient, err := rpc.DialContext(context.TODO(), *goerliRpc)
	RequireNoErrorf(err, "could not connect to goerli rpc: %v", err)

	genesis := core.DefaultSuaveGenesisBlock()

	goerliSigner := types.LatestSigner(core.DefaultGoerliGenesisBlock().Config)
	suaveSigner := types.NewSuaveSigner(genesis.Config.ChainID)

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	_, err = sendMevShareMatchTx(
		suaveClient,
		goerliClient,
		suaveSigner,
		goerliSigner,
		26,
		mevshareAddresss,
		blockSenderAddress,
		kettleAddress,
		matchBidIdBytes,
		privKey,
	)
	if err != nil {
		log.Info("err", "error", err.Error())
		panic(err.Error())
	}
}

func sendMevShareMatchTx(
	// clients
	suaveClient *rpc.Client,
	goerliClient *rpc.Client,
	// signers
	suaveSigner types.Signer,
	goerliSigner types.Signer,
	// tx specific
	targetBlock uint64,
	mevShareAddr common.Address,
	blockSenderAddr common.Address,
	kettleAddress common.Address,
	matchBidId types.BidId,
	// account specific
	privKey *ecdsa.PrivateKey,
) (*common.Hash, error) {
	_, backrunBundleBytes, err := prepareEthBackrunBundle(goerliClient, goerliSigner, privKey)
	RequireNoErrorf(err, "could not prepare backrun bundle: %v", err)

	// Send a bundle bid
	allowedPeekers := []common.Address{newBlockBidAddress, extractHintAddress, buildEthBlockAddress, mevShareAddr, blockSenderAddr}

	matchCalldata, err := mevShareABI.Pack("newMatch", targetBlock, allowedPeekers, matchBidId)
	if err != nil {
		return &common.Hash{}, fmt.Errorf("could not pack newMatch inputs: %w", err)
	}

	var suaveAccNonce hexutil.Uint64
	err = suaveClient.Call(&suaveAccNonce, "eth_getTransactionCount", crypto.PubkeyToAddress(privKey.PublicKey), "pending")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)

	_, backrunBidTxBytes, err := prepareMevBackrunBidTx(suaveSigner, privKey, kettleAddress, uint64(suaveAccNonce), matchCalldata, mevShareAddr)
	RequireNoErrorf(err, "could not prepare backrun bid: %v", err)

	// TODO : reusing this function selector from bid contract to avoid creating another ABI
	confidentialDataBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(backrunBundleBytes)
	if err != nil {
		return &common.Hash{}, fmt.Errorf("could not pack bundle data: %w", err)
	}

	var confidentialRequestTxHash common.Hash
	err = suaveClient.Call(&confidentialRequestTxHash, "eth_sendRawTransaction", hexutil.Encode(backrunBidTxBytes), hexutil.Encode(confidentialDataBytes))
	if err != nil {
		return &common.Hash{}, fmt.Errorf("confidential request tx failed: %w", err)
	}

	return &confidentialRequestTxHash, nil
}

func prepareEthBackrunBundle(
	goerliClient *rpc.Client,
	goerliSigner types.Signer,
	privKey *ecdsa.PrivateKey,
) (types.SBundle, []byte, error) {
	var goerliAccNonce hexutil.Uint64
	err := goerliClient.Call(&goerliAccNonce, "eth_getTransactionCount", crypto.PubkeyToAddress(privKey.PublicKey), "latest")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)

	ethTx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		Nonce:     uint64(goerliAccNonce) + 1, // wont work with same sender as original tx
		To:        &common.Address{},
		Value:     big.NewInt(1000),
		Gas:       21000,
		GasTipCap: big.NewInt(30821813599),
		GasFeeCap: big.NewInt(100821813599),
		Data:      []byte{},
	}), goerliSigner, privKey)

	if err != nil {
		return types.SBundle{}, nil, err
	}

	bundle := &types.SBundle{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	if err != nil {
		return types.SBundle{}, nil, err
	}

	return *bundle, bundleBytes, nil
}

func prepareMevBackrunBidTx(suaveSigner types.Signer, privKey *ecdsa.PrivateKey, kettleAddress common.Address, suaveAccNonce uint64, calldata []byte, mevShareAddr common.Address) (*types.Transaction, hexutil.Bytes, error) {
	wrappedTxData := &types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: kettleAddress,
			Nonce:         suaveAccNonce,
			To:            &mevShareAddr,
			Value:         nil,
			Gas:           10000000,
			GasPrice:      big.NewInt(33000000000),
			Data:          calldata,
		},
	}

	confidentialRequestTx, err := types.SignTx(types.NewTx(wrappedTxData), suaveSigner, privKey)
	if err != nil {
		return nil, nil, err
	}

	confidentialRequestTxBytes, err := confidentialRequestTx.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	return confidentialRequestTx, confidentialRequestTxBytes, nil
}
