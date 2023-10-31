package vm

import (
	"crypto/ecdsa"
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/flashbots/go-boost-utils/bls"
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
	if !isPrecompileAllowed && precompile != confStoreStoreAddress && precompile != confStoreRetrieveAddress {
		return common.Address{}, fmt.Errorf("precompile %s (%x) not allowed on %x", precompile, precompile, bid.Id)
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

	return common.Address{}, fmt.Errorf("no caller of %s (%x) is allowed on %x", precompile, precompile, bid.Id)
}
