package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	matchBidContract          = newArtifact("bids.sol/MevShareBidContract.json")
	bundleBidContract         = newArtifact("bids.sol/BundleBidContract.json")
	buildEthBlockContract     = newArtifact("bids.sol/EthBlockBidContract.json")
	ethBlockBidSenderContract = newArtifact("bids.sol/EthBlockBidSenderContract.json")
	suaveLibContract          = newArtifact("SuaveAbi.sol/SuaveAbi.json")
)

func newArtifact(name string) *artifact {
	data, err := os.ReadFile(filepath.Join("../artifacts", name))
	if err != nil {
		panic(fmt.Sprintf("failed to read artifact %s: %v", name, err))
	}

	var artifactObj struct {
		Abi              *abi.ABI `json:"abi"`
		DeployedBytecode string   `json:"deployedBytecode"`
		Bytecode         string   `json:"bytecode"`
	}
	if err := json.Unmarshal(data, &artifactObj); err != nil {
		panic(fmt.Sprintf("failed to unmarshal artifact %s: %v", name, err))
	}

	return &artifact{
		Abi:          artifactObj.Abi,
		Code:         hexutil.MustDecode(artifactObj.Bytecode),
		DeployedCode: hexutil.MustDecode(artifactObj.DeployedBytecode),
	}
}

type artifact struct {
	Abi          *abi.ABI
	DeployedCode []byte
	Code         []byte
}
