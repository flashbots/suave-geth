package vm

/*
var (
	//go:embed internal/main.wasm
	src   []byte
	cache = wazero.NewCompilationCache()
)

func (c *extractHint) runImpl(backend *SuaveExecutionBackend, bundleBytes []byte) ([]byte, error) {
	// return c.runImplOld(backend, bundleBytes)

	// Enforce a maximum execution time for the WASM code.
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

	// Compile the WASM bytecode to Wazero IR.
	ir, err := r.CompileModule(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	defer ir.Close(ctx)

	stdin := bytes.NewBuffer(bundleBytes)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	mod, err := r.InstantiateModule(ctx, ir, wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(stdout).
		WithStderr(stderr))
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("init module: %w", err)
	}
	defer mod.Close(ctx)

	return stdout.Bytes(), nil
}
*/
