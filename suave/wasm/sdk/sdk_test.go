package sdk

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestSDK_RegisterAndCall(t *testing.T) {
	r := &router{}
	r.Register(addFunc)

	typ, _ := abi.NewTypeFromString("tuple(a uint64,b uint64)")

	var args = struct {
		A uint64
		B uint64
	}{
		A: 1,
		B: 2,
	}

	input, _ := typ.Pack(args)
	fmt.Println(r.Run(input))
}

func addFunc(a uint64, b uint64) (uint64, error) {
	return a + b, nil
}
