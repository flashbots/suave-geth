package cstore

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	suave "github.com/ethereum/go-ethereum/suave/core"
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
	OnStore func(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error)
}

func (*FakeStoreBackend) Start() error { return nil }
func (*FakeStoreBackend) Stop() error  { return nil }

func (*FakeStoreBackend) InitializeBid(bid suave.Bid) error { return nil }
func (*FakeStoreBackend) FetchEngineBidById(bidId suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, errors.New("not implemented")
}

func (b *FakeStoreBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	return b.OnStore(bid, caller, key, value)
}
func (*FakeStoreBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (*FakeStoreBackend) FetchBidById(suave.BidId) (suave.Bid, error) {
	return suave.Bid{}, nil
}

func (*FakeStoreBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	return nil
}

func (*FakeStoreBackend) SubmitBid(types.Bid) error {
	return nil
}

func TestOwnMessageDropping(t *testing.T) {
	var wasCalled *bool = new(bool)
	fakeStore := FakeStoreBackend{OnStore: func(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
		*wasCalled = true
		return bid, nil
	}}

	fakeDaSigner := FakeDASigner{localAddresses: []common.Address{{0x42}}}
	engine := NewConfidentialStoreEngine(&fakeStore, MockTransport{}, fakeDaSigner, MockChainSigner{})

	testKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// testKeyAddress := crypto.PubkeyToAddress(testKey.PublicKey)
	dummyCreationTx, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: common.Address{0x42},
		},
	}), types.NewSuaveSigner(new(big.Int)), testKey)
	require.NoError(t, err)

	bidId, err := calculateBidId(types.Bid{
		AllowedStores:  []common.Address{{0x42}},
		AllowedPeekers: []common.Address{{}},
	})
	require.NoError(t, err)
	testBid := suave.Bid{
		Id:             bidId,
		CreationTx:     dummyCreationTx,
		AllowedStores:  []common.Address{{0x42}},
		AllowedPeekers: []common.Address{{}},
	}

	testBidBytes, err := SerializeBidForSigning(&testBid)
	require.NoError(t, err)

	testBid.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, testBidBytes)
	require.NoError(t, err)

	*wasCalled = false

	daMessage := DAMessage{
		SourceTx:    dummyCreationTx,
		StoreUUID:   engine.storeUUID,
		StoreWrites: []StoreWrite{{Bid: testBid}},
	}

	daMessageBytes, err := SerializeMessageForSigning(&daMessage)
	require.NoError(t, err)

	daMessage.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, daMessageBytes)
	require.NoError(t, err)

	*wasCalled = false
	err = engine.NewMessage(daMessage)
	require.NoError(t, err)
	// require.True(t, *wasCalled)

	daMessage.StoreUUID = uuid.New()
	daMessageBytes, err = SerializeMessageForSigning(&daMessage)
	require.NoError(t, err)

	daMessage.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, daMessageBytes)
	require.NoError(t, err)

	*wasCalled = false
	err = engine.NewMessage(daMessage)
	require.NoError(t, err)
	require.True(t, *wasCalled)
}
