package vm

import (
	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type SuaveExecutionBackend struct {
	ConfiendialStoreBackend suave.ConfiendialStoreBackend
	MempoolBackend          suave.MempoolBackend
	OffchainEthBackend      suave.OffchainEthBackend
	confidentialInputs      []byte
	callerStack             []*common.Address
}

func NewRuntimeSuaveExecutionBackend(evm *EVM, caller common.Address) *SuaveExecutionBackend {
	if !evm.Config.IsOffchain {
		return nil
	}

	return &SuaveExecutionBackend{
		ConfiendialStoreBackend: evm.suaveExecutionBackend.ConfiendialStoreBackend,
		MempoolBackend:          evm.suaveExecutionBackend.MempoolBackend,
		OffchainEthBackend:      evm.suaveExecutionBackend.OffchainEthBackend,
		confidentialInputs:      evm.suaveExecutionBackend.confidentialInputs,
		callerStack:             append(evm.suaveExecutionBackend.callerStack, &caller),
	}
}

// Implements PrecompiledContract for Offchain smart contracts
type SuavePrecompiledContractWrapper struct {
	backend  *SuaveExecutionBackend
	contract SuavePrecompiledContract
}

func NewSuavePrecompiledContractWrapper(backend *SuaveExecutionBackend, contract SuavePrecompiledContract) *SuavePrecompiledContractWrapper {
	return &SuavePrecompiledContractWrapper{backend: backend, contract: contract}
}

func (p *SuavePrecompiledContractWrapper) RequiredGas(input []byte) uint64 {
	return p.contract.RequiredGas(input)
}

func (p *SuavePrecompiledContractWrapper) Run(input []byte) ([]byte, error) {
	return p.contract.RunOffchain(p.backend, input)
}
