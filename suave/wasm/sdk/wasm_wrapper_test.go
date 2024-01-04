package sdk

import (
	_ "embed"
	"fmt"
	"testing"

	abi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/suave/wasm"
	"github.com/stretchr/testify/require"
)

//go:embed fixtures/add/add.wasm
var addLib []byte

func TestWasmWrapper(t *testing.T) {
	typ, _ := abi.NewTypeFromString("tuple(a uint64,b uint64)")

	var args = struct {
		A uint64
		B uint64
	}{
		A: 1,
		B: 2,
	}

	input, _ := typ.Pack(args)

	r, _ := wasm.NewRuntime()
	require.NoError(t, r.Register(common.Address{}, addLib))

	fmt.Println(r.Call(common.Address{}, input))
}
