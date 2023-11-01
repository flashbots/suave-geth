package main

import (
	//"context"
	"crypto/ecdsa"
	//"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	//"strings"
	"path/filepath"
	"github.com/ethereum/go-ethereum/accounts/abi"	
	"runtime"	

	_ "embed"

	"github.com/ethereum/go-ethereum/common"
	//"github.com/ethereum/go-ethereum/common/hexutil"
	//"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	//"github.com/ethereum/go-ethereum/suave/e2e"
	"github.com/ethereum/go-ethereum/suave/sdk"
)

var (
	exNodeEthAddr  = common.HexToAddress("03493869959c866713c33669ca118e774a30a0e5")
	exNodeNetAddr  = "https://rpc.rigil.suave.flashbots.net"
	// 0x9d8A62f656a8d1615C1294fd71e9CFb3E4855A4F
	fundedAccount      = newPrivKeyFromHex("4646464646464646464646464646464646464646464646464646464646464646")
)

func newArtifact(name string) *Artifact {
	// Get the caller's file path.
	_, filename, _, _ := runtime.Caller(1)

	// Resolve the directory of the caller's file.
	callerDir := filepath.Dir(filename)

	// Construct the absolute path to the target file.
	targetFilePath := filepath.Join(callerDir, "../../artifacts", name)

	data, err := os.ReadFile(targetFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to read artifact %s: %v", name, err))
	}

	var artifactObj struct {
		Abi              *abi.ABI `json:"abi"`
		DeployedBytecode struct {
			Object string
		} `json:"deployedBytecode"`
		Bytecode struct {
			Object string
		} `json:"bytecode"`
	}
	if err := json.Unmarshal(data, &artifactObj); err != nil {
		panic(fmt.Sprintf("failed to unmarshal artifact %s: %v", name, err))
	}

	return &Artifact{
		Abi:          artifactObj.Abi,
		Code:         []byte{},
		DeployedCode: []byte{},
	}
}

type Artifact struct {
	Abi          *abi.ABI
	DeployedCode []byte
	Code         []byte
}

var (
	batchAuctionArtifact = newArtifact("batchauction.sol/BatchAuction.json")
)

func main() {
	rpcClient, _ := rpc.Dial(exNodeNetAddr)
	mevmClt := sdk.NewClient(rpcClient, fundedAccount.priv, exNodeEthAddr)

	// Already deployed BatchAuction contract
	var addr = common.HexToAddress("04134c6A7ff9D5F7FA2900E4e66939637710269f");

	var batchAuctionContract *sdk.Contract
	batchAuctionContract = sdk.GetContract(addr, batchAuctionArtifact.Abi, mevmClt)

	var confidentialDataBytes = []byte{}
	
	txnResult, err := batchAuctionContract.SendTransaction("completeBatch", []interface{}{big.NewInt(0), big.NewInt(int64(20*math.Pow(10,9))), big.NewInt(int64(21*math.Pow(10,6)))}, confidentialDataBytes)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	receipt, err := txnResult.Wait()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	if receipt.Status == 0 {
		fmt.Printf("failed to send bid")
		os.Exit(1)
	}
	fmt.Printf("- completeBatch sent at txn: %s\n", receipt.TxHash.Hex())
}


// Helpers, not unique to SUAVE

type step struct {
	name   string
	action func() error
}


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
