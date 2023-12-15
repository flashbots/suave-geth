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

func (m *mockSuaveBackend) InitializeBid(record suave.DataRecord) error {
	return nil
}

func (m *mockSuaveBackend) Store(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
	return suave.DataRecord{}, nil
}

func (m *mockSuaveBackend) Retrieve(record suave.DataRecord, caller common.Address, key string) ([]byte, error) {
	return nil, nil
}

func (m *mockSuaveBackend) SubmitBid(types.DataRecord) error {
	return nil
}

func (m *mockSuaveBackend) FetchEngineBidById(suave.DataId) (suave.DataRecord, error) {
	return suave.DataRecord{}, nil
}

func (m *mockSuaveBackend) FetchBidById(suave.DataId) (suave.DataRecord, error) {
	return suave.DataRecord{}, nil
}

func (m *mockSuaveBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
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
	confEngine := cstore.NewEngine(confStore, &cstore.MockTransport{}, cstore.MockSigner{}, cstore.MockChainSigner{})

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

	d5, err := b.newDataRecord(5, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	d10, err := b.newDataRecord(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	d10b, err := b.newDataRecord(10, []common.Address{{0x1}}, nil, "a")
	require.NoError(t, err)

	cases := []struct {
		cond        uint64
		namespace   string
		dataRecords []types.DataRecord
	}{
		{0, "a", []types.DataRecord{}},
		{5, "a", []types.DataRecord{d5}},
		{10, "a", []types.DataRecord{d10, d10b}},
		{11, "a", []types.DataRecord{}},
	}

	for _, c := range cases {
		dRecords, err := b.fetchDataRecords(c.cond, c.namespace)
		require.NoError(t, err)

		require.ElementsMatch(t, c.dataRecords, dRecords)
	}
}

func TestSuave_ConfStoreWorkflow(t *testing.T) {
	b := newTestBackend(t)

	callerAddr := common.Address{0x1}
	data := []byte{0x1}

	// cannot store a value for a dataRecord that does not exist
	err := b.confidentialStore(types.DataId{}, "key", data)
	require.Error(t, err)

	dataRecord, err := b.newDataRecord(5, []common.Address{callerAddr}, nil, "a")
	require.NoError(t, err)

	// cannot store the dataRecord if the caller is not allowed to
	err = b.confidentialStore(dataRecord.Id, "key", data)
	require.Error(t, err)

	// now, the caller is allowed to store the dataRecord
	b.suaveContext.CallerStack = append(b.suaveContext.CallerStack, &callerAddr)
	err = b.confidentialStore(dataRecord.Id, "key", data)
	require.NoError(t, err)

	val, err := b.confidentialRetrieve(dataRecord.Id, "key")
	require.NoError(t, err)
	require.Equal(t, data, val)

	// cannot retrieve the value if the caller is not allowed to
	b.suaveContext.CallerStack = []*common.Address{}
	_, err = b.confidentialRetrieve(dataRecord.Id, "key")
	require.Error(t, err)
}
