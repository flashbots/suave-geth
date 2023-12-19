package vm

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func (m *mockSuaveBackend) NewSession(ctx context.Context) (string, error) {
	return "", nil
}

func (m *mockSuaveBackend) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error) {
	return &types.SimulateTransactionResult{}, nil
}

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

type httpTestHandler struct{}

func (h *httpTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if val := r.Header.Get("a"); val != "" {
		w.Write([]byte(val))
		return
	}
	if val := r.Header.Get("fail"); val != "" {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if r.Method == "POST" {
		w.Write([]byte("ok"))
		return
	}
	w.Write([]byte("ok1"))
}

func TestSuave_HttpRequest_Basic(t *testing.T) {
	s := &suaveRuntime{
		suaveContext: &SuaveContext{
			Backend: &SuaveExecutionBackend{
				ExternalWhitelist: []string{"127.0.0.1"},
			},
		},
	}

	srv := httptest.NewServer(&httpTestHandler{})
	defer srv.Close()

	cases := []struct {
		req  types.HttpRequest
		err  bool
		resp []byte
	}{
		{
			// url not set
			req: types.HttpRequest{},
			err: true,
		},
		{
			// method not supported
			req: types.HttpRequest{Url: srv.URL},
			err: true,
		},
		{
			// url not allowed
			req: types.HttpRequest{Url: "http://example.com", Method: "GET"},
			err: true,
		},
		{
			// incorrect header format
			req: types.HttpRequest{Url: srv.URL, Method: "GET", Headers: []string{"a"}},
			err: true,
		},
		{
			// POST request
			req:  types.HttpRequest{Url: srv.URL, Method: "POST"},
			resp: []byte("ok"),
		},
		{
			// GET request
			req:  types.HttpRequest{Url: srv.URL, Method: "GET"},
			resp: []byte("ok1"),
		},
		{
			// GET request with headers
			req:  types.HttpRequest{Url: srv.URL, Method: "GET", Headers: []string{"a:b"}},
			resp: []byte("b"),
		},
		{
			// POST request with headers
			req:  types.HttpRequest{Url: srv.URL, Method: "POST", Headers: []string{"a:c"}},
			resp: []byte("c"),
		},
		{
			// POST request with headers with multiple :
			req:  types.HttpRequest{Url: srv.URL, Method: "POST", Headers: []string{"a:c:d"}},
			resp: []byte("c:d"),
		},
		{
			// POST with error
			req: types.HttpRequest{Url: srv.URL, Method: "POST", Headers: []string{"fail:1"}},
			err: true,
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			resp, err := s.doHTTPRequest(c.req)
			if c.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, c.resp, resp)
			}
		})
	}
}
