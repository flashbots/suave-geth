package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type SuaveContext struct {
	// TODO: MEVM access to Backend should be restricted to only the necessary functions!
	Backend                      *SuaveExecutionBackend
	ConfidentialComputeRequestTx *types.Transaction
	ConfidentialInputs           []byte
	CallerStack                  []*common.Address
}

type SuaveExecutionBackend struct {
	ConfidentialStoreEngine *suave.ConfidentialStoreEngine
	MempoolBackend          suave.MempoolBackend
	ConfidentialEthBackend  suave.ConfidentialEthBackend
}

func (b *SuaveExecutionBackend) Start() error {
	if err := b.ConfidentialStoreEngine.Start(); err != nil {
		return err
	}

	if err := b.MempoolBackend.Start(); err != nil {
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
