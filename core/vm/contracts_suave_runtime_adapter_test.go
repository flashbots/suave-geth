package vm

import (
	"math/big"
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

func (m *mockRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, bidId types.BidId, namespace string) ([]byte, []byte, error) {
	return []byte{0x1}, []byte{0x1}, nil
}

func (m *mockRuntime) confidentialInputs() ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) confidentialRetrieve(bidId types.BidId, key string) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) confidentialStore(bidId types.BidId, key string, data1 []byte) error {
	return nil
}

func (m *mockRuntime) ethcall(contractAddr common.Address, input1 []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) extractHint(bundleData []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) fetchBids(cond uint64, namespace string) ([]types.Bid, error) {
	return []types.Bid{{}}, nil
}

func (m *mockRuntime) fillMevShareBundle(bidId types.BidId) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) newBid(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, bidType string) (types.Bid, error) {
	return types.Bid{}, nil
}

func (m *mockRuntime) signEthTransaction(txn []byte, chainId string, signingKey string) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) simulateBundle(bundleData []byte) (uint64, error) {
	return 1, nil
}

func (m *mockRuntime) submitBundleJsonRPC(url string, method string, params []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) submitEthBlockBidToRelay(relayUrl string, builderBid []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) randomUint() (*big.Int, error) {
	return big.NewInt(0x9001), nil
}

func (m *mockRuntime) secp256k1RecoverPubkey(_msg []byte, _sig []byte) ([]byte, error) {
	return []byte{0x1}, nil
}

func (m *mockRuntime) secp256k1VerifySignature(_msg []byte, _sig []byte, _pubkey []byte) (bool, error) {
	return true, nil
}

func (m *mockRuntime) secp256k1Sign(_msg []byte, _key []byte) ([]byte, error) {
	return []byte{0x1}, nil
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
