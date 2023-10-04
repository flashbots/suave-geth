package vm

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func InstantiateHostModule(ctx context.Context, r wazero.Runtime, b *SuaveExecutionBackend) (api.Closer, context.Context, error) {
	sx, err := wazergo.Instantiate(ctx, r, HostModule,
		withConfidentialStoreBackend(b))
	return sx, wazergo.WithModuleInstance(ctx, sx), err
}

// Declare the host module from a set of exported functions.
var HostModule wazergo.HostModule[*Module] = functions{
	"retrieve": wazergo.F4((*Module).Retrieve),
}

// The `functions` type impements `HostModule[*Module]`, providing the
// module name, map of exported functions, and the ability to create instances
// of the module type.
type functions wazergo.Functions[*Module]

func (f functions) Name() string {
	return "suavexec"
}

func (f functions) Functions() wazergo.Functions[*Module] {
	return (wazergo.Functions[*Module])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*Module, error) {
	mod := &Module{
		// ...
	}
	wazergo.Configure(mod, opts...)
	return mod, nil
}

type Option = wazergo.Option[*Module]

func withConfidentialStoreBackend(b *SuaveExecutionBackend) Option {
	return wazergo.OptionFunc(func(m *Module) {
		m.Backend = b
	})
}

// Module will be the Go type we use to maintain the state of our module
// instances.
type Module struct {
	Backend *SuaveExecutionBackend
}

func (Module) Close(context.Context) error {
	return nil
}

func (m Module) Retrieve(ctx context.Context, key types.String, bidID types.Bytes, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error {
	var bid [16]byte
	if copy(bid[:], bidID) != 16 {
		return types.Fail(errors.New("invalid size for bidID"))
	}

	if len(m.Backend.callerStack) == 0 {
		return types.Fail(errors.New("not allowed in this context"))
	}

	log.Info("confStoreRetrieve", "bidId", bid, "key", key)

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(m.Backend.callerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := m.Backend.callerStack[i]; _c != nil && *_c != confStoreRetrieveAddress {
			caller = *_c
			break
		}
	}

	// Make the actual call to Retrieve.  We copy the resulting data
	// directly into the WASM process via direct memory access.  The
	// overhead for this is a copy operation at the WASM VM boundary.
	// This is loosely equivalent to the overhead of a syscall.
	data, err := m.Backend.ConfidentialStoreBackend.Retrieve(bid, caller, string(key))
	if err != nil {
		return types.Fail(err)
	}
	size := uint32(len(data))

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(size))
	}

	// Copy data into linear memory.  In the future, we can adopt
	// a strategy similar to Wetware's, which is to create a zero-
	// copy stream transport that works out of buffers in the guest's
	// linear memory.  For now, we assume a static buffer with sufficient
	// capacity.  We store the number of bytes written into a uint32 pointer
	// so that the guest knows how much of the buffer to return.
	n.Store(types.Uint32(copy(buf, data)))

	return types.OK
}
