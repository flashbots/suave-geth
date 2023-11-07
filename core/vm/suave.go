package vm

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/flashbots/go-boost-utils/bls"
	"golang.org/x/exp/slices"
)

// ConfidentialStore represents the API for the confidential store
// required by Suave runtime.
type ConfidentialStore interface {
	InitializeBid(bid types.Bid) (types.Bid, error)
	Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error)
	Retrieve(bid types.BidId, caller common.Address, key string) ([]byte, error)
	FetchBidById(suave.BidId) (suave.Bid, error)
	FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid
}

type SuaveContext struct {
	// TODO: MEVM access to Backend should be restricted to only the necessary functions!
	Backend                      *SuaveExecutionBackend
	ConfidentialComputeRequestTx *types.Transaction
	ConfidentialInputs           []byte
	CallerStack                  []*common.Address
}

type SuaveExecutionBackend struct {
	EthBundleSigningKey    *ecdsa.PrivateKey
	EthBlockSigningKey     *bls.SecretKey
	ConfidentialStore      ConfidentialStore
	ConfidentialEthBackend suave.ConfidentialEthBackend
}

func NewRuntimeSuaveContext(evm *EVM, caller common.Address) *SuaveContext {
	if !evm.Config.IsConfidential {
		return nil
	}

	return &SuaveContext{
		Backend:                      evm.SuaveContext.Backend,
		ConfidentialComputeRequestTx: evm.SuaveContext.ConfidentialComputeRequestTx,
		ConfidentialInputs:           evm.SuaveContext.ConfidentialInputs,
		CallerStack:                  append(evm.SuaveContext.CallerStack, &caller),
	}
}

// Implements PrecompiledContract for confidential smart contracts
type SuavePrecompiledContractWrapper struct {
	addr         common.Address
	suaveContext *SuaveContext
}

func NewSuavePrecompiledContractWrapper(addr common.Address, suaveContext *SuaveContext) *SuavePrecompiledContractWrapper {
	return &SuavePrecompiledContractWrapper{addr: addr, suaveContext: suaveContext}
}

func (p *SuavePrecompiledContractWrapper) RequiredGas(input []byte) uint64 {
	// TODO: Figure out how to handle gas consumption of the precompiles
	return 1000
}

func (p *SuavePrecompiledContractWrapper) Run(input []byte) ([]byte, error) {
	stub := &SuaveRuntimeAdapter{
		impl: &suaveRuntime{
			suaveContext: p.suaveContext,
		},
	}

	if metrics.EnabledExpensive {
		precompileName := artifacts.PrecompileAddressToName(p.addr)
		metrics.GetOrRegisterMeter("suave/runtime/"+precompileName, nil).Mark(1)

		now := time.Now()
		defer func() {
			metrics.GetOrRegisterTimer("suave/runtime/"+precompileName+"/duration", nil).Update(time.Since(now))
		}()
	}

	if p.addr == isConfidentialAddress {
		// 'isConfidential' is a special precompile, redo as a function?
		return []byte{0x1}, nil
	}

	ret, err := stub.run(p.addr, input)
	if err != nil && ret == nil {
		ret = []byte(err.Error())
	}

	return ret, err
}

func isPrecompileAddr(addr common.Address) bool {
	if addr == isConfidentialAddress {
		return true
	}
	return slices.Contains(addrList, addr)
}

// Returns the caller
func checkIsPrecompileCallAllowed(suaveContext *SuaveContext, precompile common.Address, bid suave.Bid) (common.Address, error) {
	anyPeekerAllowed := slices.Contains(bid.AllowedPeekers, suave.AllowedPeekerAny)
	if anyPeekerAllowed {
		for i := len(suaveContext.CallerStack) - 1; i >= 0; i-- {
			caller := suaveContext.CallerStack[i]
			if caller != nil && *caller != precompile {
				return *caller, nil
			}
		}

		return precompile, nil
	}

	// In question!
	// For now both the precompile *and* at least one caller must be allowed to allow access to bid data
	// Alternative is to simply allow if any of the callers is allowed
	isPrecompileAllowed := slices.Contains(bid.AllowedPeekers, precompile)

	// Special case for confStore as those are implicitly allowed
	if !isPrecompileAllowed && precompile != confidentialStoreAddr && precompile != confidentialRetrieveAddr {
		return common.Address{}, fmt.Errorf("precompile %s (%x) not allowed on %x", artifacts.PrecompileAddressToName(precompile), precompile, bid.Id)
	}

	for i := len(suaveContext.CallerStack) - 1; i >= 0; i-- {
		caller := suaveContext.CallerStack[i]
		if caller == nil || *caller == precompile {
			continue
		}
		if slices.Contains(bid.AllowedPeekers, *caller) {
			return *caller, nil
		}
	}

	return common.Address{}, fmt.Errorf("no caller of %s (%x) is allowed on %x", artifacts.PrecompileAddressToName(precompile), precompile, bid.Id)
}
