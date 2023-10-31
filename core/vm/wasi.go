package vm

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/tetratelabs/wazero"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	cache = wazero.NewCompilationCache()

	//go:embed suave_wasm/suavexec.wasm
	helloworld         []byte
	wasmHelloWorldAddr = extractHintAddress // common.HexToAddress("0x67000001")

	//go:embed suave_wasm/store_put.wasm
	wasmStorePutBytecode []byte

	//go:embed suave_wasm/store_retrieve.wasm
	wasmRetrieveBytecode []byte

	//go:embed suave_wasm/extract_hint.wasm
	wasmExtractHintBytecode []byte
)

type WasiPrecompileWrapper struct {
	bytecode []byte
}

func (w *WasiPrecompileWrapper) RequiredGas(input []byte) uint64 {
	return 0
}

func (w *WasiPrecompileWrapper) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this context")
}

func (w *WasiPrecompileWrapper) RunConfidential(suaveCtx *SuaveContext, input []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Instantiate the Wazero runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithCompilationCache(cache))
	defer r.Close(ctx)

	// Instantiate WASI.
	sys, err := wasi.Instantiate(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("init wasi: %w", err)
	}
	defer sys.Close(ctx)

	sx, ctx, err := InstantiateHostModule(ctx, r, suaveCtx)
	if err != nil {
		return nil, fmt.Errorf("init host module: %w", err)
	}
	defer sx.Close(ctx)

	// Compile the WASM bytecode to Wazero IR.
	ir, err := r.CompileModule(ctx, w.bytecode)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	defer ir.Close(ctx)

	stdin := bytes.NewBuffer(input)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	mod, err := r.InstantiateModule(ctx, ir, wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(stdout).
		WithStderr(stderr))
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("init module: %w. Output: %s", err, string(stdout.Bytes()))
	}
	defer mod.Close(ctx)

	return stdout.Bytes(), nil
}
