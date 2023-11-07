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
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func cmdSendBundle() {
	flagset := flag.NewFlagSet("sendBundle", flag.ExitOnError)

	var (
		suaveRpc         = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		kettleAddressHex = flagset.String("kettleAddress", "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B", "wallet address of execution node")
		goerliRpc        = flagset.String("goerli_rpc", "http://127.0.0.1:8545", "address of goerli rpc")
		privKeyHex       = flagset.String("privkey", "", "private key as hex (for testing)")
		verbosity        = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		privKey          *ecdsa.PrivateKey
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

	suaveClient, err := rpc.DialContext(context.TODO(), *suaveRpc)
	RequireNoErrorf(err, "could not connect to suave rpc: %v", err)

	goerliClient, err := rpc.DialContext(context.TODO(), *goerliRpc)
	RequireNoErrorf(err, "could not connect to goerli rpc: %v", err)

	genesis := core.DefaultSuaveGenesisBlock()
	chainId := hexutil.Big(*genesis.Config.ChainID)

	goerliSigner := types.LatestSigner(core.DefaultGoerliGenesisBlock().Config)
	suaveSigner := types.NewSuaveSigner(genesis.Config.ChainID)

	gas := hexutil.Uint64(1000000)

	{ // Sanity check
		var result string
		rpcErr := suaveClient.Call(&result, "eth_call", ethapi.TransactionArgs{
			To:             &isConfidentialAddress,
			Gas:            &gas,
			IsConfidential: true,
			ChainID:        &chainId,
		}, "latest")
		RequireNoErrorf(rpcErr, "could not call IsConfidential precompile: %v", unwrapPeekerError(rpcErr))
		if result != "0x01" {
			utils.Fatalf("unexpected result from confidential call %s, expected 0x01", result)
		}

		log.Info("Suave node seems sane, continuing")
	}

	addr := crypto.PubkeyToAddress(privKey.PublicKey)

	var goerliAccNonceBytes hexutil.Uint64
	err = goerliClient.Call(&goerliAccNonceBytes, "eth_getTransactionCount", addr, "latest")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on goerli: %v", err)
	goerliAccNonce := uint64(goerliAccNonceBytes)

	// Prepare the bundle to land
	ethTx, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		To:        &addr,
		Nonce:     goerliAccNonce,
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(500),
		Gas:       21000,
		Value:     big.NewInt(10000), // in wei
		Data:      []byte{},
	}), goerliSigner, privKey)
	RequireNoErrorf(err, "could not sign eth tx: %v", err)

	ethBundle := &types.SBundle{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	ethBundleBytes, err := json.Marshal(ethBundle)
	RequireNoErrorf(err, "could not marshal bundle: %v", err)

	{
		txs := []string{}
		for _, tx := range ethBundle.Txs {
			txJson, _ := tx.MarshalJSON()
			txs = append(txs, string(txJson))
		}
		log.Info("Prepared eth bundle", "txs", txs)
	}

	var suaveAccNonceBytes hexutil.Uint64
	err = suaveClient.Call(&suaveAccNonceBytes, "eth_getTransactionCount", addr, "latest")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)
	suaveAccNonce := uint64(suaveAccNonceBytes)

	var suaveGp hexutil.Big
	err = suaveClient.Call(&suaveGp, "eth_gasPrice")
	RequireNoErrorf(err, "could not call eth_gasPrice on suave: %v", err)

	confidentialInnerTxTemplate := &types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			Nonce:    suaveAccNonce, // will be incremented later on
			To:       &newBundleBidAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: (*big.Int)(&suaveGp),
			Data:     nil, // FillMe!
		},
	}

	suaveTxHashes := []common.Hash{}

	var currentGoerliBlockNumber hexutil.Uint64
	err = goerliClient.Call(&currentGoerliBlockNumber, "eth_blockNumber")
	RequireNoErrorf(err, "could not call eth_blockNumber on goerli: %v", err)

	minTargetBlock := uint64(currentGoerliBlockNumber) + uint64(1)
	maxTargetBlock := minTargetBlock + uint64(1) // TODO: 25
	for cTargetBlock := minTargetBlock; cTargetBlock <= maxTargetBlock; cTargetBlock++ {
		// Send a bundle bid
		allowedPeekers := []common.Address{newBlockBidAddress, newBundleBidAddress, buildEthBlockAddress}
		calldata, err := bundleBidAbi.Pack("newBid", cTargetBlock, allowedPeekers)
		RequireNoErrorf(err, "could not pack newBid args: %v", err)

		confidentialRequestInner := *confidentialInnerTxTemplate
		confidentialInnerTxTemplate.Nonce += 1
		confidentialRequestInner.Data = calldata
		confidentialRequestInner.KettleAddress = kettleAddress

		confidentialRequestTx, err := types.SignTx(types.NewTx(&confidentialRequestInner), suaveSigner, privKey)
		RequireNoErrorf(err, "could not sign confidentialRequestTx: %v", err)

		confidentialRequestTxBytes, err := confidentialRequestTx.MarshalBinary()
		RequireNoErrorf(err, "could not marshal confidentialRequestTx: %v", err)

		confidentialDataBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(ethBundleBytes)
		RequireNoErrorf(err, "could not pack bundle confidential data: %v", err)

		var confidentialRequestTxHash common.Hash
		rpcErr := suaveClient.Call(&confidentialRequestTxHash, "eth_sendRawTransaction", hexutil.Encode(confidentialRequestTxBytes), hexutil.Encode(confidentialDataBytes))
		RequireNoErrorf(rpcErr, "could not send bundle bid: %v", unwrapPeekerError(rpcErr))
		suaveTxHashes = append(suaveTxHashes, confidentialRequestTxHash)
	}

	log.Info("Submitted bundle to suave", "suaveTxHashes", suaveTxHashes)

	{ // Bid mempool sanity check
		packedCondBytes, err := suaveLibAbi.Methods["fetchBids"].Inputs.Pack(minTargetBlock)
		RequireNoErrorf(err, "could not pack fetchBids: %v", err)

		var result hexutil.Bytes
		rpcErr := suaveClient.Call(&result, "eth_call", ethapi.TransactionArgs{
			To:             &fetchBidsAddress,
			Gas:            &gas,
			IsConfidential: true,
			ChainID:        &chainId,
			Data:           (*hexutil.Bytes)(&packedCondBytes),
		}, "latest")
		RequireNoErrorf(rpcErr, "could not pack fetchBids: %v", unwrapPeekerError(rpcErr))
		unpackedBids, err := suaveLibAbi.Methods["fetchBids"].Outputs.Unpack(result)
		RequireNoErrorf(err, "could not unpack fetchBids response %v: %v", result, err)

		bids := unpackedBids[0].([]struct {
			Id                  [16]uint8        "json:\"id\""
			DecryptionCondition uint64           "json:\"decryptionCondition\""
			AllowedPeekers      []common.Address "json:\"allowedPeekers\""
		})

		if len(bids) == 0 {
			utils.Fatalf("no bids fetched, expected at least one. result: %v", result)
		}

		log.Info("Mempool seems sane, fetched", "bids", bids, "targetBlock", minTargetBlock)
	}

	// TODO: confirm (maybe in background) that the transactions all landed in blocks

	// TODO: do this for every goerli block until success
	{ // Request a goerli block
		var currentGoerliHeader map[string]interface{}
		err = goerliClient.Call(&currentGoerliHeader, "eth_getHeaderByNumber", "latest")
		RequireNoErrorf(err, "could not call eth_getHeaderByNumber on goerli: %v", err)

		timestamp, err := hexutil.DecodeUint64(currentGoerliHeader["timestamp"].(string))
		RequireNoErrorf(err, "could not decode timestamp from %v: %v", currentGoerliHeader["timestamp"], err)

		payloadArgsTuple := struct {
			Parent       common.Hash
			Timestamp    uint64
			FeeRecipient common.Address
			GasLimit     uint64
			Random       common.Hash
			Withdrawals  []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			Parent:       common.HexToHash(currentGoerliHeader["hash"].(string)),
			Timestamp:    timestamp + uint64(12),
			FeeRecipient: common.Address{0x42},
			GasLimit:     30000000,
			Random:       common.Hash{0x44}, // TODO! Should be taken from goerli beacon chain
		}

		log.Info("Prepared payload request", "args", payloadArgsTuple)

		calldata, err := buildEthBlockAbi.Pack("buildFromPool", payloadArgsTuple, minTargetBlock+1)
		RequireNoErrorf(err, "could not pack arguments for block building request: %v", err)

		err = suaveClient.Call(&suaveAccNonceBytes, "eth_getTransactionCount", addr, "pending")
		RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)
		suaveAccNonce = uint64(suaveAccNonceBytes)

		wrappedTxData := &types.ConfidentialComputeRequest{
			ConfidentialComputeRecord: types.ConfidentialComputeRecord{
				KettleAddress: kettleAddress,
				Nonce:         suaveAccNonce,
				To:            &newBlockBidAddress,
				Value:         nil,
				Gas:           1000000,
				GasPrice:      (*big.Int)(&suaveGp),
				Data:          calldata,
			},
		}

		confidentialRequestTx, err := types.SignTx(types.NewTx(wrappedTxData), suaveSigner, privKey)
		RequireNoErrorf(err, "could not sign block build request: %v", err)

		confidentialRequestTxBytes, err := confidentialRequestTx.MarshalBinary()
		RequireNoErrorf(err, "could not marshal block build request: %v", err)

		var confidentialRequestTxHash common.Hash
		rpcErr := suaveClient.Call(&confidentialRequestTxHash, "eth_sendRawTransaction", hexutil.Encode(confidentialRequestTxBytes))
		RequireNoErrorf(rpcErr, "block building peeker failed: %v", unwrapPeekerError(rpcErr))
	}

	log.Info("All is good!")
}
