package cstore

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func TestTransactionalStore2(t *testing.T) {
	store := NewLocalConfidentialStore()
	tstore := NewTransactionalStore2(store)

	testBid, err := tstore.InitRecord(types.DataRecord{
		Salt:                RandomRecordId(),
		DecryptionCondition: 46,
		AllowedStores:       []common.Address{{0x42}},
		AllowedPeekers:      []common.Address{{0x43}},
		Version:             "v0-test",
	})
	require.NoError(t, err)

	_, err = tstore.Store(testBid.Id, testBid.AllowedPeekers[0], "xx", []byte{0x44})
	require.NoError(t, err)

	tfetchedBid, err := tstore.FetchRecordByID(testBid.Id)
	require.NoError(t, err)
	require.Equal(t, testBid, tfetchedBid.ToInnerRecord())

	require.Empty(t, tstore.FetchRecordsByProtocolAndBlock(45, "v0-test"))
	require.Empty(t, tstore.FetchRecordsByProtocolAndBlock(46, "v1-test"))

	tfetchedBids := tstore.FetchRecordsByProtocolAndBlock(46, "v0-test")
	require.Equal(t, 1, len(tfetchedBids))
	require.Equal(t, testBid, tfetchedBids[0].ToInnerRecord())

	_, err = tstore.Retrieve(testBid.Id, testBid.AllowedPeekers[0], "xy")
	require.Error(t, err)

	_, err = tstore.Retrieve(suave.RandomDataRecordId(), testBid.AllowedPeekers[0], "xx")
	require.Error(t, err)

	_, err = tstore.Retrieve(testBid.Id, testBid.AllowedStores[0], "xx")
	require.Error(t, err)

	tretrieved, err := tstore.Retrieve(testBid.Id, testBid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x44}, tretrieved)

	// Not finalized, engine should return empty
	_, err = store.FetchRecordByID(testBid.Id)
	require.Error(t, err)
	require.Empty(t, store.FetchRecordsByProtocolAndBlock(46, "v0-test"))
	_, err = store.Retrieve(suave.DataRecord{Id: testBid.Id}, testBid.AllowedPeekers[0], "xx")
	require.Error(t, err)
}
