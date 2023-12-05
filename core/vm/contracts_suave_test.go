package vm

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/ethereum/go-ethereum/suave/cstore"
	"github.com/stretchr/testify/require"
)

type mockSuaveBackend struct {
}

func (m *mockSuaveBackend) Start() error { return nil }
func (m *mockSuaveBackend) Stop() error  { return nil }

func (m *mockSuaveBackend) InitializeBid(bid suave.Bid) error {
	return nil
}

func (m *mockSuaveBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (m *mockSuaveBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockSuaveBackend) SubmitBid(types.Bid) error {
	return nil
}

func (m *mockSuaveBackend) FetchEngineBidById(suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, nil
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

func (m *mockSuaveBackend) Call(ctx context.Context, contractAddr common.Address, input []byte) ([]byte, error) {
	return nil, nil
}

func (m *mockSuaveBackend) Subscribe() (<-chan cstore.DAMessage, context.CancelFunc) {
	return nil, func() {}
}

func (m *mockSuaveBackend) Publish(cstore.DAMessage) {}

func newTestBackend(t *testing.T) *suaveRuntime {
	confStore := cstore.NewLocalConfidentialStore()
	confEngine := cstore.NewConfidentialStoreEngine(confStore, &cstore.MockTransport{}, cstore.MockSigner{}, cstore.MockChainSigner{})

	require.NoError(t, confEngine.Start())
	t.Cleanup(func() { confEngine.Stop() })

	reqTx := types.NewTx(&types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: common.Address{},
		},
	})

	b := &suaveRuntime{
		suaveContext: &SuaveContext{
			Backend: &SuaveExecutionBackend{
				ConfidentialStore:      confEngine.NewTransactionalStore(reqTx),
				ConfidentialEthBackend: &mockSuaveBackend{},
			},
			ConfidentialComputeRequestTx: reqTx,
		},
	}
	return b
}

func TestSuave_BidWorkflow(t *testing.T) {
	b := newTestBackend(t)

	bid5, err := b.newBid(5, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	bid10, err := b.newBid(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	bid10b, err := b.newBid(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	cases := []struct {
		cond      uint64
		namespace string
		bids      []types.Bid
	}{
		{0, "a", []types.Bid{}},
		{5, "a", []types.Bid{bid5}},
		{10, "a", []types.Bid{bid10, bid10b}},
		{11, "a", []types.Bid{}},
	}

	for _, c := range cases {
		bids, err := b.fetchBids(c.cond, c.namespace)
		require.NoError(t, err)

		require.ElementsMatch(t, c.bids, bids)
	}
}

func TestSuave_ConfStoreWorkflow(t *testing.T) {
	b := newTestBackend(t)

	callerAddr := common.Address{0x1}
	data := []byte{0x1}

	// cannot store a value for a bid that does not exist
	err := b.confidentialStore(types.BidId{}, "key", data)
	require.Error(t, err)

	bid, err := b.newBid(5, []common.Address{callerAddr}, nil, "a")
	require.NoError(t, err)

	// cannot store the bid if the caller is not allowed to
	err = b.confidentialStore(bid.Id, "key", data)
	require.Error(t, err)

	// now, the caller is allowed to store the bid
	b.suaveContext.CallerStack = append(b.suaveContext.CallerStack, &callerAddr)
	err = b.confidentialStore(bid.Id, "key", data)
	require.NoError(t, err)

	val, err := b.confidentialRetrieve(bid.Id, "key")
	require.NoError(t, err)
	require.Equal(t, data, val)

	// cannot retrieve the value if the caller is not allowed to
	b.suaveContext.CallerStack = []*common.Address{}
	_, err = b.confidentialRetrieve(bid.Id, "key")
	require.Error(t, err)
}
