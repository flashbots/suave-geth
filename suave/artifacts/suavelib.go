package artifacts

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed SuaveAbi.sol/SuaveAbi.json
var suaveAbisol []byte

func loadSuaveLib() *abi.ABI {
	var artifactObj struct {
		Abi *abi.ABI `json:"abi"`
	}
	if err := json.Unmarshal(suaveAbisol, &artifactObj); err != nil {
		panic(fmt.Sprintf("failed to unmarshal suave lib artifact: %v", err))
	}
	return artifactObj.Abi
}

var SuaveAbi = loadSuaveLib()
