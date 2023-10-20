package vm

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	suave "github.com/ethereum/go-ethereum/suave/core"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func InstantiateHostModule(ctx context.Context, r wazero.Runtime, suaveCtx *SuaveContext) (api.Closer, context.Context, error) {
	sx, err := wazergo.Instantiate(ctx, r, HostModule,
		withConfidentialStoreBackend(suaveCtx))
	return sx, wazergo.WithModuleInstance(ctx, sx), err
}

// Declare the host module from a set of exported functions.
var HostModule wazergo.HostModule[*EngineModule] = functions{
	"initializeBid":               wazergo.F3((*EngineModule).InitializeBid),
	"storeRetrieve":               wazergo.F4((*EngineModule).StoreRetrieve),
	"storePut":                    wazergo.F3((*EngineModule).StorePut),
	"fetchBidById":                wazergo.F3((*EngineModule).FetchBidById),
	"FetchBidsByProtocolAndBlock": wazergo.F3((*EngineModule).FetchBidsByProtocolAndBlock),
}

// The `functions` type impements `HostModule[*Module]`, providing the
// module name, map of exported functions, and the ability to create instances
// of the module type.
type functions wazergo.Functions[*EngineModule]

func (f functions) Name() string {
	return "suavexec"
}

func (f functions) Functions() wazergo.Functions[*EngineModule] {
	return (wazergo.Functions[*EngineModule])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*EngineModule, error) {
	mod := &EngineModule{
		// ...
	}
	wazergo.Configure(mod, opts...)
	return mod, nil
}

type Option = wazergo.Option[*EngineModule]

func withConfidentialStoreBackend(suaveCtx *SuaveContext) Option {
	return wazergo.OptionFunc(func(m *EngineModule) {
		m.SuaveContext = suaveCtx
	})
}

// EngineModule will be the Go type we use to maintain the state of our module
// instances.
type EngineModule struct {
	SuaveContext *SuaveContext
}

func (EngineModule) Close(context.Context) error {
	return nil
}

func (m EngineModule) InitializeBid(ctx context.Context, raw_bidID types.Bytes, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error {
	return types.Fail(errors.New("not implemented"))
}

func (m EngineModule) StorePut(ctx context.Context, jsonWrite types.String, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error {
	return types.Fail(errors.New("not implemented"))
}

func (m EngineModule) FetchBidById(ctx context.Context, raw_bidID types.Bytes, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error {
	return types.Fail(errors.New("not implemented"))
}

func (m EngineModule) FetchBidsByProtocolAndBlock(ctx context.Context, jsonSelector types.String, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error {
	return types.Fail(errors.New("not implemented"))
}

func (m EngineModule) StoreRetrieve(ctx context.Context, key types.String, raw_bidID types.Bytes, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error {
	var bidId suave.BidId
	if copy(bidId[:], raw_bidID) != 16 {
		return types.Fail(errors.New("invalid size for bidID"))
	}

	if len(m.SuaveContext.CallerStack) == 0 {
		n.Store(types.Uint32(copy(buf, []byte("not allowed in this suaveContext"))))
		return types.Fail(errors.New("not allowed in this suaveContext"))
	}

	log.Info("confStoreRetrieve", "bidId", bidId, "key", key)

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(m.SuaveContext.CallerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := m.SuaveContext.CallerStack[i]; _c != nil && *_c != confStoreRetrieveAddress {
			caller = *_c
			break
		}
	}

	data, err := m.SuaveContext.Backend.ConfidentialStore.Retrieve(bidId, caller, string(key))
	if err != nil {
		n.Store(types.Uint32(copy(buf, types.Bytes(err.Error()))))
		return types.Fail(err)
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
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
