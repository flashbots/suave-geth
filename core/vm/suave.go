package vm

import (
	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type SuaveExecutionBackend struct {
	ConfidentialStoreBackend suave.ConfidentialStoreBackend
	MempoolBackend           suave.MempoolBackend
	ConfidentialEthBackend   suave.ConfidentialEthBackend
	confidentialInputs       []byte
	callerStack              []*common.Address
}

func NewRuntimeSuaveExecutionBackend(evm *EVM, caller common.Address) *SuaveExecutionBackend {
	if !evm.Config.IsConfidential {
		return nil
	}

	return &SuaveExecutionBackend{
		ConfidentialStoreBackend: evm.suaveExecutionBackend.ConfidentialStoreBackend,
		MempoolBackend:           evm.suaveExecutionBackend.MempoolBackend,
		ConfidentialEthBackend:   evm.suaveExecutionBackend.ConfidentialEthBackend,
		confidentialInputs:       evm.suaveExecutionBackend.confidentialInputs,
		callerStack:              append(evm.suaveExecutionBackend.callerStack, &caller),
	}
}

// Implements PrecompiledContract for confidential smart contracts
type SuavePrecompiledContractWrapper struct {
	addr     common.Address
	backend  *SuaveExecutionBackend
	contract SuavePrecompiledContract
}

func NewSuavePrecompiledContractWrapper(addr common.Address, backend *SuaveExecutionBackend, contract SuavePrecompiledContract) *SuavePrecompiledContractWrapper {
	return &SuavePrecompiledContractWrapper{addr: addr, backend: backend, contract: contract}
}

func (p *SuavePrecompiledContractWrapper) RequiredGas(input []byte) uint64 {
	return p.contract.RequiredGas(input)
}

func (p *SuavePrecompiledContractWrapper) Run(input []byte) ([]byte, error) {
	stub := &SuaveRuntimeAdapter{
		impl: &suaveRuntime{
			backend: p.backend,
		},
	}

	if p.addr == isConfidentialAddress {
		// 'isConfidential' is a special precompile
		return (&isConfidentialPrecompile{}).RunConfidential(p.backend, input)
	}

	return stub.run(p.addr, input)
}
