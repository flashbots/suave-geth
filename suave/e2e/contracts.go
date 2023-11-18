package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	MevShareBidContract          = NewArtifact("bids.sol/MevShareBidContract.json")
	BundleBidContract            = NewArtifact("bids.sol/BundleBidContract.json")
	EthBundleSenderContract      = NewArtifact("bids.sol/EthBundleSenderContract.json")
	MevShareBundleSenderContract = NewArtifact("bids.sol/MevShareBundleSenderContract.json")
	buildEthBlockContract        = NewArtifact("bids.sol/EthBlockBidContract.json")
	ethBlockBidSenderContract    = NewArtifact("bids.sol/EthBlockBidSenderContract.json")
	suaveLibContract             = NewArtifact("SuaveAbi.sol/SuaveAbi.json")
	exampleCallSourceContract    = NewArtifact("example.sol/ExampleEthCallSource.json")
	exampleCallTargetContract    = NewArtifact("example.sol/ExampleEthCallTarget.json")
)

func NewArtifact(name string) *Artifact {
	// Get the caller's file path.
	_, filename, _, _ := runtime.Caller(1)

	// Resolve the directory of the caller's file.
	callerDir := filepath.Dir(filename)

	// Construct the absolute path to the target file.
	targetFilePath := filepath.Join(callerDir, "../artifacts", name)

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
		Code:         hexutil.MustDecode(artifactObj.Bytecode.Object),
		DeployedCode: hexutil.MustDecode(artifactObj.DeployedBytecode.Object),
	}
}

type Artifact struct {
	Abi          *abi.ABI
	DeployedCode []byte
	Code         []byte
}
