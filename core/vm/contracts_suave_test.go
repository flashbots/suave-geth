package vm

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

type mockSuaveBackend struct {
}

func (m *mockSuaveBackend) Initialize(bid suave.Bid, key string, value []byte) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) Retrieve(bid suave.BidId, caller common.Address, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockSuaveBackend) SubmitBid(suave.Bid) error {
	return nil
}

func (m *mockSuaveBackend) FetchBidById(suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	return nil
}

func (m *mockSuaveBackend) BuildEthBlock(ctx context.Context, args *suave.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

func (m *mockSuaveBackend) BuildEthBlockFromBundles(ctx context.Context, args *suave.BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error) {
	return nil, nil
}

var dummyBlockContext = BlockContext{
	CanTransfer: func(StateDB, common.Address, *big.Int) bool { return true },
	Transfer:    func(StateDB, common.Address, common.Address, *big.Int) {},
	BlockNumber: big.NewInt(0),
}

func TestSuavePrecompileStub(t *testing.T) {
	// This test ensures that the Suave precompile stubs work as expected
	// for encoding/decoding.
	mockSuaveBackend := &mockSuaveBackend{}
	suaveBackend := &SuaveExecutionBackend{
		ConfiendialStoreBackend: mockSuaveBackend,
		MempoolBackend:          mockSuaveBackend,
		OffchainEthBackend:      mockSuaveBackend,
	}

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	vmenv := NewOffchainEVM(suaveBackend, dummyBlockContext, TxContext{}, statedb, params.AllEthashProtocolChanges, Config{IsOffchain: true})

	methods := map[string]common.Address{
		"extractHint":               extractHintAddress,
		"fetchBids":                 fetchBidsAddress,
		"newBid":                    newBidAddress,
		"simulateBundle":            simulateBundleAddress,
		"submitEthBlockBidToRelay":  submitEthBlockBidToRelayAddress,
		"buildEthBlock":             buildEthBlockAddress,
		"confidentialStoreRetrieve": confStoreRetrieveAddress,
		"confidentialStoreStore":    confStoreStoreAddress,
	}

	// The objective of the unit test is to make sure that the encoding of the precompile
	// inputs works as expected from the ABI specification. Thus, we will skip any errors
	// that are generated by the logic of the precompile.
	// Note: Once code generated is in place, we can remove this and only test the
	// encodings in isolation outside the logic.
	expectedErrors := []string{
		// json error when the precompile expects to decode a json object encoded as []byte
		// in the precompile input.
		"invalid character",
		// error from a precompile that expects to make an http request from an input value.
		"could not send request to relay",
		// error in 'buildEthBlock' when it expects to retrieve bids in abi format from the
		// confidential store.
		"could not unpack merged bid ids",
	}

	for name, addr := range methods {
		abiMethod, ok := artifacts.SuaveAbi.Methods[name]
		if !ok {
			t.Fatalf("abi method '%s' not found", name)
		}

		inputVals := abi.GenerateRandomTypeForMethod(abiMethod)

		packedInput, err := abiMethod.Inputs.Pack(inputVals...)
		require.NoError(t, err)

		_, _, err = vmenv.Call(AccountRef(common.Address{}), addr, packedInput, 100000000, big.NewInt(0))
		if err != nil {
			found := false
			for _, expectedError := range expectedErrors {
				if strings.Contains(err.Error(), expectedError) {
					found = true
				}
			}
			if !found {
				t.Fatal(err)
			}
		}
	}

	// error if there are methods in the abi that are not tested
	for name := range artifacts.SuaveAbi.Methods {
		if _, ok := methods[name]; !ok {
			t.Fatalf("abi method '%s' not tested", name)
		}
	}
}
