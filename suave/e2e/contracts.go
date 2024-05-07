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
	MevShareContract             = newArtifact("bundles.sol/MevShareContract.json")
	BundleContract               = newArtifact("bundles.sol/BundleContract.json")
	EthBundleSenderContract      = newArtifact("bundles.sol/EthBundleSenderContract.json")
	MevShareBundleSenderContract = newArtifact("bundles.sol/MevShareBundleSenderContract.json")
	buildEthBlockContract        = newArtifact("bundles.sol/EthBlockContract.json")
	ethBlockBidSenderContract    = newArtifact("bundles.sol/EthBlockBidSenderContract.json")
	exampleCallSourceContract    = newArtifact("example.sol/ExampleEthCallSource.json")
	exampleCallTargetContract    = newArtifact("example.sol/ExampleEthCallTarget.json")

	mossBundle1 = newArtifact("moss.sol/Bundle1.json")
	mossBundle2 = newArtifact("moss.sol/Bundle2.json")
)

func newArtifact(name string) *Artifact {
	// Get the caller's file path.
	_, filename, _, _ := runtime.Caller(1)

	// Resolve the directory of the caller's file.
	callerDir := filepath.Dir(filename)

	// Construct the absolute path to the target file.
	targetFilePath := filepath.Join(callerDir, "../artifacts", name)

	data, err := os.ReadFile(targetFilePath)
	if err != nil {
		panic(fmt.Sprintf("failed to read artifact %s: %v. Maybe you forgot to generate the artifacts? `cd suave && forge build`", name, err))
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
