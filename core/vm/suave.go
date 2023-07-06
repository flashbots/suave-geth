package vm

import (
	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type SuaveOffchainBackend struct {
	ConfiendialStoreBackend suave.ConfiendialStoreBackend
	MempoolBackned          suave.MempoolBackend
	OffchainEthBackend      suave.OffchainEthBackend
	confidentialInputs      []byte
	callerStack             []*common.Address
}

func NewRuntimeSuaveOffchainBackend(evm *EVM, caller common.Address) *SuaveOffchainBackend {
	if !evm.Config.IsOffchain {
		return nil
	}

	return &SuaveOffchainBackend{
		ConfiendialStoreBackend: evm.suaveOffchainBackend.ConfiendialStoreBackend,
		MempoolBackned:          evm.suaveOffchainBackend.MempoolBackned,
		OffchainEthBackend:      evm.suaveOffchainBackend.OffchainEthBackend,
		confidentialInputs:      evm.suaveOffchainBackend.confidentialInputs,
		callerStack:             append(evm.suaveOffchainBackend.callerStack, &caller),
	}
}

// Implements PrecompiledContract for Offchain smart contracts
type OffchainPrecompiledContractWrapper struct {
	backend  *SuaveOffchainBackend
	contract SuavePrecompiledContract
}

func NewOffchainPrecompiledContractWrapper(backend *SuaveOffchainBackend, contract SuavePrecompiledContract) *OffchainPrecompiledContractWrapper {
	return &OffchainPrecompiledContractWrapper{backend: backend, contract: contract}
}

func (p *OffchainPrecompiledContractWrapper) RequiredGas(input []byte) uint64 {
	return p.contract.RequiredGas(input)
}

func (p *OffchainPrecompiledContractWrapper) Run(input []byte) ([]byte, error) {
	return p.contract.RunOffchain(p.backend, input)
}
