package vm

import (
	"bytes"
	"context"
	_ "embed"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed suave_wasm/suavexec.wasm
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
	suaveCtx := &SuaveContext{
		// TODO: MEVM access to Backend should be restricted to only the necessary functions!
		Backend: &SuaveExecutionBackend{
			ConfidentialStore: &MockConfidentialStoreBackend{t},
		},
		ConfidentialComputeRequestTx: nil, //*types.Transaction
		CallerStack:                  []*common.Address{&wantAddr},
	}

	sx, ctx, err := InstantiateHostModule(ctx, r, suaveCtx)
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

func (MockConfidentialStoreBackend) InitializeBid(bid types.Bid) (types.Bid, error) {
	panic("NOT IMPLEMENTED")
}

func (MockConfidentialStoreBackend) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	panic("NOT IMPLEMENTED")
}

var wantBid = types.BidId{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}

func (b MockConfidentialStoreBackend) Retrieve(bid suave.BidId, caller common.Address, key string) ([]byte, error) {

	assert.Equal(b.T, wantBid, bid)
	assert.Equal(b.T, wantAddr, caller)
	assert.Equal(b.T, "someKey", key)

	return []byte("test data"), nil
}

func (b MockConfidentialStoreBackend) FetchBidById(bid suave.BidId) (suave.Bid, error) {

	assert.Equal(b.T, wantBid, bid)
	return suave.Bid{
		AllowedPeekers: []common.Address{wantAddr},
	}, nil
}

func (b MockConfidentialStoreBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	panic("NOT IMPLEMENTED")
}
