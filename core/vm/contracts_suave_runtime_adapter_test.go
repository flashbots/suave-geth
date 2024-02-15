package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/stretchr/testify/require"
)

var _ SuaveRuntime = &mockRuntime{}

type mockRuntime struct {
}

func (m *mockRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, dataId types.DataId, namespace string, chainId string) ([]byte, []byte, error) {
	return []byte{0x1}, []byte{0x1}, nil
}

func (m *mockRuntime) confidentialInputs() ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) confidentialRetrieve(dataId types.DataId, key string) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) confidentialStore(dataId types.DataId, key string, data1 []byte) error {
	return nil
}

func (m *mockRuntime) ethcall(contractAddr common.Address, input1 []byte, chainId string) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) extractHint(bundleData []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) fetchDataRecords(cond uint64, namespace string) ([]types.DataRecord, error) {
	return []types.DataRecord{{}}, nil
}

func (m *mockRuntime) fillMevShareBundle(dataId types.DataId) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) newDataRecord(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, dataType string) (types.DataRecord, error) {
	return types.DataRecord{}, nil
}

func (m *mockRuntime) signEthTransaction(txn []byte, chainId string, signingKey string) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) signMessage(digest []byte, signingKey string) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) simulateBundle(bundleData []byte, chainId string) (uint64, error) {
	return 1, nil
}

func (m *mockRuntime) submitBundleJsonRPC(url string, method string, params []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) submitEthBlockToRelay(relayUrl string, builderDataRecord []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) doHTTPRequest(request types.HttpRequest) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) newBuilder() (string, error) {
	return "", nil
}

func (m *mockRuntime) simulateTransaction(session string, txn []byte, chainId string) (types.SimulateTransactionResult, error) {
	return types.SimulateTransactionResult{}, nil
}

func (m *mockRuntime) privateKeyGen() (string, error) {
	return "", nil
}

func TestRuntimeAdapter(t *testing.T) {
	adapter := &SuaveRuntimeAdapter{
		impl: &mockRuntime{},
	}

	for name, addr := range artifacts.SuaveMethods {
		abiMethod, ok := artifacts.SuaveAbi.Methods[name]
		if !ok {
			t.Fatalf("abi method '%s' not found", name)
		}

		inputVals := abi.GenerateRandomTypeForMethod(abiMethod)

		packedInput, err := abiMethod.Inputs.Pack(inputVals...)
		require.NoError(t, err)

		_, err = adapter.run(addr, packedInput)
		require.NoError(t, err)
	}
}
