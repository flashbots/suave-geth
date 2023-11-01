package main

import (
	//	"context"
	"crypto/ecdsa"
	//"encoding/hex"
	//"encoding/json"
	"fmt"
	//"math/big"
	//"os"
	//"strings"

	_ "embed"

	"github.com/ethereum/go-ethereum/common"
	//"github.com/ethereum/go-ethereum/common/hexutil"
	//"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/e2e"
	"github.com/ethereum/go-ethereum/suave/sdk"
)

var (
	exNodeEthAddr  = common.HexToAddress("03493869959c866713c33669ca118e774a30a0e5")
	exNodeNetAddr  = "https://rpc.rigil.suave.flashbots.net"
	//l1NodeNetAdder = "x"
	// 0x9d8A62f656a8d1615C1294fd71e9CFb3E4855A4F
	fundedAccount      = newPrivKeyFromHex("4646464646464646464646464646464646464646464646464646464646464646")
)

var (
	batchAuctionArtifact = e2e.BatchAuctionContract
)

func main() {
	rpcClient, _ := rpc.Dial(exNodeNetAddr)
	mevmClt := sdk.NewClient(rpcClient, fundedAccount.priv, exNodeEthAddr)

	var batchAuctionContract *sdk.Contract
	_ = batchAuctionContract

	txnResult, err := sdk.DeployContract(batchAuctionArtifact.Code, mevmClt)
	if err != nil {
		fmt.Errorf("Failed to deploy contract: %v", err)
	}
	receipt, err := txnResult.Wait()
	if err != nil {
		fmt.Errorf("Failed to wait for transaction result: %v", err)
	}
	if receipt.Status == 0 {
		fmt.Errorf("Failed to deploy contract: %v", err)
	}

	fmt.Printf("- Example contract deployed: %s\n", receipt.ContractAddress)
	batchAuctionContract = sdk.GetContract(receipt.ContractAddress, batchAuctionArtifact.Abi, mevmClt)
}

// Helpers, not unique to SUAVE

type privKey struct {
	priv *ecdsa.PrivateKey
}

func newPrivKeyFromHex(hex string) *privKey {
	key, err := crypto.HexToECDSA(hex)
	if err != nil {
		panic(fmt.Sprintf("failed to parse private key: %v", err))
	}
	return &privKey{priv: key}
}
