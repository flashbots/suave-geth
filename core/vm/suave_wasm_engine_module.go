package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	suave_wasi "github.com/ethereum/go-ethereum/suave/wasi"

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
	"initializeBid":               wazergo.F3(wrapCall((*EngineModule).InitializeBid)),
	"storeRetrieve":               wazergo.F3(wrapCall((*EngineModule).StoreRetrieve)),
	"storePut":                    wazergo.F3(wrapCall((*EngineModule).StorePut)),
	"fetchBidById":                wazergo.F3(wrapCall((*EngineModule).FetchBidById)),
	"FetchBidsByProtocolAndBlock": wazergo.F3(wrapCall((*EngineModule).FetchBidsByProtocolAndBlock)),
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

type hostFnType = func(m *EngineModule, ctx context.Context, inData types.Bytes, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error

func wrapCall[In any, Out any](handler func(*EngineModule, context.Context, In) (Out, error)) hostFnType {
	return func(m *EngineModule, ctx context.Context, inData types.Bytes, buf types.Bytes, n types.Pointer[types.Uint32]) types.Error {
		var inStruct In
		err := json.Unmarshal(inData, &inStruct)
		if err != nil {
			wrappedErr := fmt.Errorf("unable to unmarshal inputs: %w", err)
			n.Store(types.Uint32(copy(buf, types.Bytes(wrappedErr.Error()))))
			return types.Fail(wrappedErr)
		}

		out, err := handler(m, ctx, inStruct)
		if err != nil {
			n.Store(types.Uint32(copy(buf, types.Bytes(err.Error()))))
			return types.Fail(err)
		}

		data, err := json.Marshal(out)
		if err != nil {
			wrappedErr := fmt.Errorf("unable to marshal outputs: %w", err)
			n.Store(types.Uint32(copy(buf, types.Bytes(wrappedErr.Error()))))
			return types.Fail(wrappedErr)
		}

		n.Store(types.Uint32(copy(buf, data)))
		return types.OK
	}
}

func (m EngineModule) InitializeBid(ctx context.Context, bid ethtypes.Bid) (ethtypes.Bid, error) {
	return m.SuaveContext.Backend.ConfidentialStore.InitializeBid(bid)
}

func (m EngineModule) StoreRetrieve(ctx context.Context, args suave_wasi.RetrieveHostFnArgs) ([]byte, error) {
	if len(m.SuaveContext.CallerStack) == 0 {
		return nil, errors.New("caller stack not initialized, refusing to retrieve")
	}
	caller := m.SuaveContext.CallerStack[len(m.SuaveContext.CallerStack)-1]
	return m.SuaveContext.Backend.ConfidentialStore.Retrieve(args.BidId, *caller, args.Key)
}

func (m EngineModule) StorePut(ctx context.Context, args suave_wasi.StoreHostFnArgs) (ethtypes.EngineBid, error) {
	if len(m.SuaveContext.CallerStack) == 0 {
		return ethtypes.EngineBid{}, errors.New("caller stack not initialized, refusing to retrieve")
	}
	caller := m.SuaveContext.CallerStack[len(m.SuaveContext.CallerStack)-1]
	return m.SuaveContext.Backend.ConfidentialStore.Store(args.BidId, *caller, args.Key, args.Value)
}

func (m EngineModule) FetchBidById(ctx context.Context, bidId suave.BidId) (ethtypes.EngineBid, error) {
	return m.SuaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
}

func (m EngineModule) FetchBidsByProtocolAndBlock(ctx context.Context, args suave_wasi.FetchBidByProtocolFnArgs) ([]ethtypes.EngineBid, error) {
	return m.SuaveContext.Backend.ConfidentialStore.FetchBidsByProtocolAndBlock(args.BlockNumber, args.Namespace), nil
}
