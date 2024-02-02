package consolelog

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/umbracle/ethgo/abi"
)

// embed the consolelog2 artifact with the method indentifiers
//
//go:embed console2.json
var console2Artifact string

// console2Methods is a map of method signatures to their
// types. It is populated by loadConsole2Methods
var console2Methods map[string]*abi.Type

// Console2ContractAddr is the address of the console2 contract
var Console2ContractAddr = common.HexToAddress("0x000000000000000000636F6e736F6c652e6c6f67")

func decode(b []byte) (interface{}, error) {
	if len(b) < 4 {
		return nil, fmt.Errorf("invalid console log: %v", b)
	}

	var sig []byte
	sig, b = b[:4], b[4:]

	typ, ok := console2Methods[hex.EncodeToString(sig)]
	if !ok {
		return nil, fmt.Errorf("unknown console log method: %v", hex.EncodeToString(sig))
	}

	val, err := typ.Decode(b)
	if err != nil {
		return nil, err
	}

	return val, nil
}

// Print prints the given bytes to the console
func Print(b []byte) error {
	val, err := decode(b)
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", val)
	return nil
}

func loadConsole2Methods() {
	console2Methods = make(map[string]*abi.Type)

	var console2MethodIdentifiers map[string]string
	if err := json.Unmarshal([]byte(console2Artifact), &console2MethodIdentifiers); err != nil {
		panic(err)
	}

	for sig, sigID := range console2MethodIdentifiers {
		// convert the signature of the method into the form
		// tuple(...)
		indx := strings.Index(sig, "(")
		if indx == -1 {
			panic(fmt.Errorf("invalid signature for %s", sig))
		}

		typ, err := abi.NewType("tuple" + sig[indx:])
		if err != nil {
			panic(fmt.Errorf("invalid signature for %s: %v", "tuple"+sig[indx:], err))
		}

		console2Methods[sigID] = typ
	}
}

func init() {
	loadConsole2Methods()
}
