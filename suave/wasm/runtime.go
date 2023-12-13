package wasm

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type Runtime struct {
	wasmRuntime wazero.Runtime
	libraries   map[common.Address]api.Module
}

func NewRuntime() (*Runtime, error) {
	ctx := context.Background()
	wasmRuntime := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, wasmRuntime)

	// TODO: load the host functions (aka basic precompiles)

	runtime := &Runtime{
		wasmRuntime: wasmRuntime,
		libraries:   map[common.Address]api.Module{},
	}

	// Load the catalog
	for addr, lib := range libraries {
		mod, err := wasmRuntime.Instantiate(ctx, lib)
		if err != nil {
			return nil, err
		}
		for _, method := range mod.ExportedFunctionDefinitions() {
			fmt.Println(method)
		}
		runtime.libraries[addr] = mod
	}

	return runtime, nil
}

func (r *Runtime) Call(addr common.Address) error {
	lib, ok := r.libraries[addr]
	if !ok {
		return fmt.Errorf("library not found")
	}

	fmt.Println(lib)
	return nil
}
