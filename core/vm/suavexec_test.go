package vm

import (
	"bytes"
	"context"
	_ "embed"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed internal/suavexec/main.wasm
var testHostCallSrc []byte

var wantAddr = common.Address{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}

func TestHostCall(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Instantiate the Wazero runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true))
	defer r.Close(ctx)

	// Instantiate WASI.
	sys, err := wasi.Instantiate(ctx, r)
	require.NoError(t, err)
	defer sys.Close(ctx)

	// Instantiate suavexec host module
	// We first need to instantiate a mock SuaveExecutionBackend
	b := &SuaveExecutionBackend{
		ConfidentialStoreBackend: &MockConfidentialStoreBackend{t},
		callerStack:              []*common.Address{&wantAddr},
	}

	sx, ctx, err := InstantiateHostModule(ctx, r, b)
	require.NoError(t, err)
	defer sx.Close(ctx)

	// Compile the WASM bytecode to Wazero IR.
	ir, err := r.CompileModule(ctx, testHostCallSrc)
	require.NoError(t, err)
	defer ir.Close(ctx)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	mod, err := r.InstantiateModule(ctx, ir, wazero.NewModuleConfig().
		WithStdout(stdout).
		WithStderr(stderr))
	require.NoError(t, err, stderr.String())
	defer mod.Close(ctx)

	assert.Equal(t, "test data", stdout.String())
}

type MockConfidentialStoreBackend struct {
	*testing.T
}

func (MockConfidentialStoreBackend) Initialize(bid suave.Bid, key string, value []byte) (suave.Bid, error) {
	panic("NOT IMPLEMENTED")
}

func (MockConfidentialStoreBackend) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	panic("NOT IMPLEMENTED")
}

func (b MockConfidentialStoreBackend) Retrieve(bid suave.BidId, caller common.Address, key string) ([]byte, error) {
	wantBid := [16]byte{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}

	assert.Equal(b.T, wantBid, bid)
	assert.Equal(b.T, wantAddr, caller)
	assert.Equal(b.T, "someKey", key)

	return []byte("test data"), nil
}
