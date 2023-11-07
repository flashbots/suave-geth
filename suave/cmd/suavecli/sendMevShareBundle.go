package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"flag"
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

func cmdSendMevShareBundle() {
	flagset := flag.NewFlagSet("sendBundle", flag.ExitOnError)

	var (
		suaveRpc           = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		kettleAddressHex   = flagset.String("kettleAddress", "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B", "wallet address of execution node")
		mevshareAddressHex = flagset.String("mev_share_addr", "0x42042042028AE1CDE26d5BcF17Ba83f447068E5B", "address of mev share contract")
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

	suaveClient, err := rpc.DialContext(context.TODO(), *suaveRpc)
	RequireNoErrorf(err, "could not connect to suave rpc: %v", err)

	goerliClient, err := rpc.DialContext(context.TODO(), *goerliRpc)
	RequireNoErrorf(err, "could not connect to goerli rpc: %v", err)

	genesis := core.DefaultSuaveGenesisBlock()

	goerliSigner := types.LatestSigner(core.DefaultGoerliGenesisBlock().Config)
	suaveSigner := types.NewSuaveSigner(genesis.Config.ChainID)

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	_, err = sendMevShareBidTxs(suaveClient, goerliClient, suaveSigner, goerliSigner, 1, mevshareAddresss, mevshareAddresss, kettleAddress, privKey)
	if err != nil {
		log.Info("err", "error", err.Error())
		panic(err.Error())
	}
}

type mevShareBidData struct {
	blockNumber uint64
	txHash      common.Hash
}

func sendMevShareBidTxs(
	// clients
	suaveClient *rpc.Client,
	goerliClient *rpc.Client,
	// signers
	suaveSigner types.Signer,
	goerliSigner types.Signer,
	// tx specific
	nBlocks uint64,
	mevShareAddr common.Address,
	blockBuilderAddr common.Address,
	kettleAddress common.Address,
	privKey *ecdsa.PrivateKey,
) ([]mevShareBidData, error) {
	log.Info("sendMevShareBidTx", "kettleAddress", kettleAddress)

	var startingGoerliBlockNum uint64
	err = goerliClient.Call((*hexutil.Uint64)(&startingGoerliBlockNum), "eth_blockNumber")
	if err != nil {
		utils.Fatalf("could not get goerli block: %v", err)
	}

	_, ethBundleBytes, err := prepareEthBundle(goerliClient, goerliSigner, privKey)
	RequireNoErrorf(err, "could not prepare eth bundle: %v", err)

	// Prepare a bundle bid

	var suaveAccNonce hexutil.Uint64
	err = suaveClient.Call(&suaveAccNonce, "eth_getTransactionCount", crypto.PubkeyToAddress(privKey.PublicKey), "pending")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)

	// NOTE: reusing this function selector from bid contract to avoid creating another ABI
	confidentialDataBytes, err := mevShareABI.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(ethBundleBytes)
	RequireNoErrorf(err, "could not encode mev share bid: %v", err)

	allowedPeekers := []common.Address{newBlockBidAddress, extractHintAddress, buildEthBlockAddress, mevShareAddr, blockBuilderAddr}

	mevShareTxHashes := []mevShareBidData{}
	for blockNum := startingGoerliBlockNum + 1; blockNum < startingGoerliBlockNum+nBlocks; blockNum++ {
		calldata, err := mevShareABI.Pack("newBid", blockNum, allowedPeekers)
		if err != nil {
			return mevShareTxHashes, err
		}

		mevShareTx, mevShareTxBytes, err := prepareMevShareBidTx(suaveSigner, privKey, kettleAddress, uint64(suaveAccNonce), calldata, mevShareAddr)
		if err != nil {
			return mevShareTxHashes, err
		}
		suaveAccNonce++

		txJson, _ := mevShareTx.MarshalJSON()
		log.Info("sendMevShareBidTx", "mevShareTx", string(txJson))

		var confidentialRequestTxHash common.Hash
		err = suaveClient.Call(&confidentialRequestTxHash, "eth_sendRawTransaction", hexutil.Encode(mevShareTxBytes), hexutil.Encode(confidentialDataBytes))
		if err != nil {
			return mevShareTxHashes, err
		}

		mevShareTxHashes = append(mevShareTxHashes, mevShareBidData{blockNumber: blockNum, txHash: confidentialRequestTxHash})
	}

	return mevShareTxHashes, nil
}

func prepareEthBundle(
	goerliClient *rpc.Client,
	goerliSigner types.Signer,
	privKey *ecdsa.PrivateKey,
) (types.SBundle, []byte, error) {
	var goerliAccNonce hexutil.Uint64
	err := goerliClient.Call(&goerliAccNonce, "eth_getTransactionCount", crypto.PubkeyToAddress(privKey.PublicKey), "latest")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)

	ethTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    uint64(goerliAccNonce),
		To:       &common.Address{},
		Value:    big.NewInt(1000),
		Gas:      21000,
		Data:     []byte{},
		GasPrice: big.NewInt(15048452934800),
	}), goerliSigner, privKey)

	if err != nil {
		return types.SBundle{}, nil, err
	}

	refundPercent := 10
	bundle := &types.SBundle{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
		RefundPercent:   &refundPercent,
	}
	bundleBytes, err := json.Marshal(bundle)
	if err != nil {
		return types.SBundle{}, nil, err
	}

	return *bundle, bundleBytes, nil
}

func prepareMevShareBidTx(suaveSigner types.Signer, privKey *ecdsa.PrivateKey, kettleAddress common.Address, suaveAccNonce uint64, calldata []byte, mevShareAddr common.Address) (*types.Transaction, hexutil.Bytes, error) {
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
