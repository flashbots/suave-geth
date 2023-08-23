package artifacts

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed SuaveLib.json
var suaveAbisol []byte

func loadSuaveLib() *abi.ABI {
	suaveABI, err := abi.JSON(bytes.NewReader(suaveAbisol))
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal suave lib artifact: %v", err))
	}
	return &suaveABI
}

var SuaveAbi = loadSuaveLib()
