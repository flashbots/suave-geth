package vm

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/suave/artifacts"
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
		return (&isConfidentialPrecompile{}).RunConfidential(p.backend, input)

	case confidentialInputsAddress:
		return (&confidentialInputsPrecompile{}).RunConfidential(p.backend, input)

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
