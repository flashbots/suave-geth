package suave

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type FakeDASigner struct {
	localAddresses []common.Address
}

func (FakeDASigner) Sign(account common.Address, data []byte) ([]byte, error) {
	return account.Bytes(), nil
}
func (FakeDASigner) Sender(data []byte, signature []byte) (common.Address, error) {
	return common.BytesToAddress(signature), nil
}
func (f FakeDASigner) LocalAddresses() []common.Address { return f.localAddresses }

type FakeStoreBackend struct {
	OnStore func(bid Bid, caller common.Address, key string, value []byte) (Bid, error)
}

func (*FakeStoreBackend) Start() error { return nil }
func (*FakeStoreBackend) Stop() error  { return nil }

func (*FakeStoreBackend) InitializeBid(bid Bid) error { return nil }
func (*FakeStoreBackend) FetchEngineBidById(bidId BidId) (Bid, error) {
	return Bid{}, errors.New("not implemented")
}

func (b *FakeStoreBackend) Store(bid Bid, caller common.Address, key string, value []byte) (Bid, error) {
	return b.OnStore(bid, caller, key, value)
}
func (*FakeStoreBackend) Retrieve(bid Bid, caller common.Address, key string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (*FakeStoreBackend) FetchBidById(BidId) (Bid, error) {
	return Bid{}, nil
}

func (*FakeStoreBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []Bid {
	return nil
}

func (*FakeStoreBackend) SubmitBid(types.Bid) error {
	return nil
}

func TestOwnMessageDropping(t *testing.T) {
	var wasCalled *bool = new(bool)
	fakeStore := FakeStoreBackend{OnStore: func(bid Bid, caller common.Address, key string, value []byte) (Bid, error) {
		*wasCalled = true
		return bid, nil
	}}

	fakeDaSigner := FakeDASigner{localAddresses: []common.Address{{0x42}}}
	engine, err := NewConfidentialStoreEngine(&fakeStore, MockTransport{}, fakeDaSigner, MockChainSigner{})
	require.NoError(t, err)

	dummyCreationTx := types.NewTx(&types.ConfidentialComputeRequest{
		ExecutionNode: common.Address{0x42},
		Wrapped:       *types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
	})

	bidId, err := calculateBidId(types.Bid{
		AllowedStores:  []common.Address{{0x42}},
		AllowedPeekers: []common.Address{{}},
	})
	require.NoError(t, err)
	testBid := Bid{
		Id:             bidId,
		CreationTx:     dummyCreationTx,
		AllowedStores:  []common.Address{{0x42}},
		AllowedPeekers: []common.Address{{}},
	}

	testBidBytes, err := SerializeBidForSigning(testBid)
	require.NoError(t, err)

	testBid.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, testBidBytes)
	require.NoError(t, err)

	*wasCalled = false

	daMessage := DAMessage{
		Bid:       testBid,
		SourceTx:  dummyCreationTx,
		StoreUUID: engine.storeUUID,
	}

	daMessageBytes, err := SerializeMessageForSigning(daMessage)
	require.NoError(t, err)

	daMessage.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, daMessageBytes)
	require.NoError(t, err)

	*wasCalled = false
	err = engine.NewMessage(daMessage)
	require.NoError(t, err)
	// require.True(t, *wasCalled)

	daMessage.StoreUUID = uuid.New()
	daMessageBytes, err = SerializeMessageForSigning(daMessage)
	require.NoError(t, err)

	daMessage.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, daMessageBytes)
	require.NoError(t, err)

	*wasCalled = false
	err = engine.NewMessage(daMessage)
	require.NoError(t, err)
	require.True(t, *wasCalled)
}
