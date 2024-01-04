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
	return runtime, nil
}

func (r *Runtime) Register(addr common.Address, code []byte) error {
	mod, err := r.wasmRuntime.Instantiate(context.Background(), code)
	if err != nil {
		return err
	}
	if _, ok := mod.ExportedFunctionDefinitions()["export"]; !ok {
		return fmt.Errorf("exported function not found")
	}

	r.libraries[addr] = mod
	return nil
}

func (r *Runtime) Call(addr common.Address, input []byte) ([]byte, error) {
	mod, ok := r.libraries[addr]
	if !ok {
		return nil, fmt.Errorf("library not found")
	}

	exportMethod := mod.ExportedFunction("export")

	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")

	// Allocate Memory for the input
	results, err := malloc.Call(context.Background(), uint64(len(input)))
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory")
	}
	namePosition := results[0]

	// This pointer is managed by TinyGo,
	// but TinyGo is unaware of external usage.
	// So, we have to free it when finished
	defer free.Call(context.Background(), namePosition)

	// Copy input to memory
	if !mod.Memory().Write(uint32(namePosition), input) {
		return nil, fmt.Errorf("failed to write memory")
	}

	// Now, we can call the export method
	// with the position and the size of "Bob Morane"
	// the result type is []uint64
	result, err := exportMethod.Call(context.Background(), namePosition, uint64(len(input)))
	if err != nil {
		return nil, fmt.Errorf("failed to call export method: %v", err)
	}

	// Extract the position and size of the returned value
	valuePosition := uint32(result[0] >> 32)
	valueSize := uint32(result[0])

	// Read the value from the memory
	bytes, ok := mod.Memory().Read(valuePosition, valueSize)
	if !ok {
		return nil, fmt.Errorf("failed to read memory")
	}

	fmt.Println("-- otuptu --")
	fmt.Println(bytes)

	return bytes, nil
}
