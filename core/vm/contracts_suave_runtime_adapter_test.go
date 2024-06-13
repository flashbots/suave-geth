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

func (m *mockRuntime) buildEthBlockTo(execNode string, blockArgs types.BuildBlockArgs, dataID types.DataId, relayUrl string) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (m *mockRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, dataId types.DataId, namespace string) ([]byte, []byte, error) {
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

func (m *mockRuntime) ethcall(contractAddr common.Address, input1 []byte) ([]byte, error) {
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

func (m *mockRuntime) signMessage(digest []byte, crypto types.CryptoSignature, signingKey string) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) simulateBundle(bundleData []byte) (uint64, error) {
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

func (m *mockRuntime) doHTTPRequests(request []types.HttpRequest) ([][]byte, error) {
	var byteSlices [][]byte
	byteSlices = append(byteSlices, []byte{0x1})
	return byteSlices, nil
}

func (m *mockRuntime) newBuilder() (string, error) {
	return "", nil
}

func (m *mockRuntime) simulateTransaction(session string, txn []byte) (types.SimulateTransactionResult, error) {
	return types.SimulateTransactionResult{}, nil
}

func (m *mockRuntime) privateKeyGen(crypto types.CryptoSignature) (string, error) {
	return "", nil
}

func (m *mockRuntime) contextGet(key string) ([]byte, error) {
	return nil, nil
}

func (m *mockRuntime) randomBytes(length uint8) ([]byte, error) {
	var bytes = make([]byte, length)
	for i := range bytes {
		bytes[i] = 0x1
	}
	return bytes, nil
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
