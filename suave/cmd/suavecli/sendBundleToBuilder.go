package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/e2e"
	"github.com/ethereum/go-ethereum/suave/sdk"
)

func cmdSendBundleToBuilder() {
	flagset := flag.NewFlagSet("sendBundleToBuilder", flag.ExitOnError)

	goerliBuilderUrl := "https://relay-goerli.flashbots.net/"

	var (
		suaveRpc            = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		kettleAddressHex    = flagset.String("kettleAddress", "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B", "wallet address of execution node")
		goerliRpc           = flagset.String("goerli_rpc", "http://127.0.0.1:8555", "address of goerli rpc")
		privKeyHex          = flagset.String("privkey", "", "private key as hex (for testing)")
		contractAddressFlag = flagset.String("contract", "", "contract address to use (default: deploy new one)")
		verbosity           = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		privKey             *ecdsa.PrivateKey
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

	suaveSdkClient := sdk.NewClient(suaveClient, privKey, kettleAddress)

	goerliClient, err := rpc.DialContext(context.TODO(), *goerliRpc)
	RequireNoErrorf(err, "could not connect to goerli rpc: %v", err)

	goerliSigner := types.LatestSigner(core.DefaultGoerliGenesisBlock().Config)

	// Simply forwards to coinbase
	addr := crypto.PubkeyToAddress(privKey.PublicKey)

	var contractAddress *common.Address
	if *contractAddressFlag != "" {
		suaveContractAddress := common.HexToAddress(*contractAddressFlag)
		contractAddress = &suaveContractAddress
	} else {
		constructorArgs, err := e2e.EthBundleSenderContract.Abi.Constructor.Inputs.Pack([]string{goerliBuilderUrl})
		RequireNoErrorf(err, "could not pack inputs: %v", err)

		deploymentTxRes, err := sdk.DeployContract(append(e2e.EthBundleSenderContract.Code, constructorArgs...), suaveSdkClient)
		RequireNoErrorf(err, "could not send deployment tx: %v", err)
		deploymentReceipt, err := deploymentTxRes.Wait()

		RequireNoErrorf(err, "error waiting for deployment tx inclusion: %v", err)
		if deploymentReceipt.Status != 1 {
			jsonEncodedReceipt, _ := deploymentReceipt.MarshalJSON()
			utils.Fatalf("deployment not successful: %s", string(jsonEncodedReceipt))
		}

		contractAddress = &deploymentReceipt.ContractAddress
		fmt.Println("contract address: ", deploymentReceipt.ContractAddress.Hex())
	}

	bundleSenderContract := sdk.GetContract(*contractAddress, e2e.EthBundleSenderContract.Abi, suaveSdkClient)
	allowedPeekers := []common.Address{bundleSenderContract.Address()}
	var goerliAccNonceBytes hexutil.Uint64
	err = goerliClient.Call(&goerliAccNonceBytes, "eth_getTransactionCount", addr, "latest")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on goerli: %v", err)
	goerliAccNonce := uint64(goerliAccNonceBytes)

	// Prepare the bundle to land
	// contractAddr := common.HexToAddress("0xAA5C331DF478c26e6909181fc306Ea535F0e4CCe")
	ethTx1, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		To:        &addr,
		Nonce:     goerliAccNonce,
		GasTipCap: big.NewInt(74285714285),
		GasFeeCap: big.NewInt(74285714285),
		Gas:       21000,
		Value:     big.NewInt(1), // in wei
		Data:      []byte{},
	}), goerliSigner, privKey)
	RequireNoErrorf(err, "could not sign eth tx: %v", err)

	ethTx2, err := types.SignTx(types.NewTx(&types.DynamicFeeTx{
		To:        &addr,
		Nonce:     goerliAccNonce + 1,
		GasTipCap: big.NewInt(714285714285),
		GasFeeCap: big.NewInt(714285714285),
		Gas:       21000,
		Value:     big.NewInt(1), // in wei
		Data:      []byte{},
	}), goerliSigner, privKey)
	RequireNoErrorf(err, "could not sign eth tx: %v", err)

	ethBundle := &types.SBundle{
		Txs:             types.Transactions{ethTx1, ethTx2},
		RevertingHashes: []common.Hash{},
	}
	RequireNoErrorf(err, "could not marshal bundle: %v", err)

	{
		txs := []string{}
		for _, tx := range ethBundle.Txs {
			txJson, _ := tx.MarshalJSON()
			txs = append(txs, string(txJson))
		}
		log.Info("Prepared eth bundle", "txs", txs)
	}

	for {
		var currentGoerliBlockNumber hexutil.Uint64
		err = goerliClient.Call(&currentGoerliBlockNumber, "eth_blockNumber")
		RequireNoErrorf(err, "could not call eth_blockNumber on goerli: %v", err)

		var suaveTxRess []*sdk.TransactionResult

		minTargetBlock := uint64(currentGoerliBlockNumber) + uint64(1)
		maxTargetBlock := minTargetBlock + uint64(10) // TODO: 25
		for cTargetBlock := minTargetBlock; cTargetBlock <= maxTargetBlock; cTargetBlock++ {
			// Send a bundle bid
			ethBundle.BlockNumber = big.NewInt(int64(cTargetBlock))
			ethBundleBytes, err := json.Marshal(ethBundle)
			RequireNoErrorf(err, "could not marshal bundle: %v", err)
			confidentialDataBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(ethBundleBytes)
			RequireNoErrorf(err, "could not pack bundle confidential data: %v", err)

			confidentialRequestTxRes, err := bundleSenderContract.SendTransaction("newBid", []interface{}{cTargetBlock, allowedPeekers, []common.Address{}}, confidentialDataBytes)
			RequireNoErrorf(err, "could not send bundle request: %v", err)
			suaveTxRess = append(suaveTxRess, confidentialRequestTxRes)
		}

		for _, txRes := range suaveTxRess {
			receipt, err := txRes.Wait()
			if err != nil {
				fmt.Println("Could not get the receipt", err)
			}

			if receipt.Status != 1 {
				jsonEncodedReceipt, _ := receipt.MarshalJSON()
				fmt.Println("Sending bundle request failed", string(jsonEncodedReceipt))
			}
		}

		log.Info("All is good!")
		time.Sleep(time.Minute)
	}

	// TODO: mby wait for confidentialRequestTxRes
	// TODO: confirm (maybe in background) that the transactions all landed in blocks
}
