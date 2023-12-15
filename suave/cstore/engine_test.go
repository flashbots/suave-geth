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
	OnStore func(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error)
}

func (*FakeStoreBackend) Start() error { return nil }
func (*FakeStoreBackend) Stop() error  { return nil }

func (*FakeStoreBackend) InitRecord(record suave.DataRecord) error { return nil }
func (*FakeStoreBackend) FetchBidByID(id suave.DataId) (suave.DataRecord, error) {
	return suave.DataRecord{}, errors.New("not implemented")
}

func (b *FakeStoreBackend) Store(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
	return b.OnStore(record, caller, key, value)
}
func (*FakeStoreBackend) Retrieve(record suave.DataRecord, caller common.Address, key string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (*FakeStoreBackend) FetchBidById(suave.DataId) (suave.DataRecord, error) {
	return suave.DataRecord{}, nil
}

func (*FakeStoreBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	return nil
}

func (*FakeStoreBackend) SubmitDataRecord(types.DataRecord) error {
	return nil
}

///

func TestOwnMessageDropping(t *testing.T) {
	var wasCalled *bool = new(bool)
	fakeStore := FakeStoreBackend{OnStore: func(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
		*wasCalled = true
		return record, nil
	}}

	fakeDaSigner := FakeDASigner{localAddresses: []common.Address{{0x42}}}
	engine := NewEngine(&fakeStore, MockTransport{}, fakeDaSigner, MockChainSigner{})

	testKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// testKeyAddress := crypto.PubkeyToAddress(testKey.PublicKey)
	dummyCreationTx, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: common.Address{0x42},
		},
	}), types.NewSuaveSigner(new(big.Int)), testKey)
	require.NoError(t, err)

	recordId, err := calculateRecordId(types.DataRecord{
		AllowedStores:  []common.Address{{0x42}},
		AllowedPeekers: []common.Address{{}},
	})
	require.NoError(t, err)
	testRecord := suave.DataRecord{
		Id:             recordId,
		CreationTx:     dummyCreationTx,
		AllowedStores:  []common.Address{{0x42}},
		AllowedPeekers: []common.Address{{}},
	}

	testRecordBytes, err := SerializeDataRecord(&testRecord)
	require.NoError(t, err)

	testRecord.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, testRecordBytes)
	require.NoError(t, err)

	*wasCalled = false

	daMessage := DAMessage{
		SourceTx:    dummyCreationTx,
		StoreUUID:   engine.storeUUID,
		StoreWrites: []StoreWrite{{DataRecord: testRecord}},
	}

	daMessageBytes, err := SerializeDAMessage(&daMessage)
	require.NoError(t, err)

	daMessage.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, daMessageBytes)
	require.NoError(t, err)

	*wasCalled = false
	err = engine.NewMessage(daMessage)
	require.NoError(t, err)
	// require.True(t, *wasCalled)

	daMessage.StoreUUID = uuid.New()
	daMessageBytes, err = SerializeDAMessage(&daMessage)
	require.NoError(t, err)

	daMessage.Signature, err = fakeDaSigner.Sign(common.Address{0x42}, daMessageBytes)
	require.NoError(t, err)

	*wasCalled = false
	err = engine.NewMessage(daMessage)
	require.NoError(t, err)
	require.True(t, *wasCalled)
}
