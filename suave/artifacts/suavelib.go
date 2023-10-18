package artifacts

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed Suave.sol/Suave.json
var suaveAbisol []byte

func loadSuaveLib() *abi.ABI {
	fmt.Println(string(suaveAbisol))

	var suaveAbi struct {
		Abi *abi.ABI `json:"abi"`
	}
	if err := json.Unmarshal(suaveAbisol, &suaveAbi); err != nil {
		panic(fmt.Sprintf("failed to unmarshal suave lib artifact: %v", err))
	}
	return suaveAbi.Abi
}

var SuaveAbi = loadSuaveLib()
