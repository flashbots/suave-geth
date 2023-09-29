package vm

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

// ConfidentialStore represents the API for the confidential store
// required by Suave runtime.
type ConfidentialStore interface {
	Start() error
	Stop() error
	InitializeBid(bid types.Bid, creationTx *types.Transaction) (types.Bid, error)
	Store(bidId suave.BidId, sourceTx *types.Transaction, caller common.Address, key string, value []byte) (suave.Bid, error)
	Retrieve(bid types.BidId, caller common.Address, key string) ([]byte, error)
	FetchBidById(suave.BidId) (types.Bid, error)
	FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []types.Bid
}

type SuaveContext struct {
	// TODO: MEVM access to Backend should be restricted to only the necessary functions!
	Backend                      *SuaveExecutionBackend
	ConfidentialComputeRequestTx *types.Transaction
	ConfidentialInputs           []byte
	CallerStack                  []*common.Address
}

type SuaveExecutionBackend struct {
	ConfidentialStoreEngine ConfidentialStore
	MempoolBackend          suave.MempoolBackend
	ConfidentialEthBackend  suave.ConfidentialEthBackend
}

func (b *SuaveExecutionBackend) Start() error {
	if err := b.ConfidentialStoreEngine.Start(); err != nil {
		return err
	}
	return nil
}

func (b *SuaveExecutionBackend) Stop() error {
	b.MempoolBackend.Stop()
	b.ConfidentialStoreEngine.Stop()

	return nil
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
	contract     SuavePrecompiledContract
}

func NewSuavePrecompiledContractWrapper(addr common.Address, suaveContext *SuaveContext, contract SuavePrecompiledContract) *SuavePrecompiledContractWrapper {
	return &SuavePrecompiledContractWrapper{addr: addr, suaveContext: suaveContext, contract: contract}
}

func (p *SuavePrecompiledContractWrapper) RequiredGas(input []byte) uint64 {
	return p.contract.RequiredGas(input)
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

	switch p.addr {
	case isConfidentialAddress:
		return (&isConfidentialPrecompile{}).RunConfidential(p.suaveContext, input)

	case confidentialInputsAddress:
		return (&confidentialInputsPrecompile{}).RunConfidential(p.suaveContext, input)

	case confStoreStoreAddress:
		return stub.confidentialStoreStore(input)

	case confStoreRetrieveAddress:
		return stub.confidentialStoreRetrieve(input)

	case newBidAddress:
		return stub.newBid(input)

	case fetchBidsAddress:
		return stub.fetchBids(input)

	case extractHintAddress:
		return stub.extractHint(input)

	case simulateBundleAddress:
		return stub.simulateBundle(input)

	case buildEthBlockAddress:
		return stub.buildEthBlock(input)

	case submitEthBlockBidToRelayAddress:
		return stub.submitEthBlockBidToRelay(input)
	}

	return nil, fmt.Errorf("precompile %s not found", p.addr)
}
