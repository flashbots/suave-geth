package cstore

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func testBackendStore(t *testing.T, store ConfidentialStorageBackend) {
	bid := suave.Bid{
		Id:                  suave.RandomBidId(),
		DecryptionCondition: 10,
		AllowedPeekers:      []common.Address{common.HexToAddress("0x424344")},
		Version:             "default:v0:ethBundles",
	}

	err := store.InitializeBid(bid)
	require.NoError(t, err)

	bidRes, err := store.FetchBidById(bid.Id)
	require.NoError(t, err)
	require.Equal(t, bid, bidRes)

	_, err = store.Store(bid, bid.AllowedPeekers[0], "xx", []byte{0x43, 0x14})
	require.NoError(t, err)

	retrievedData, err := store.Retrieve(bid, bid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)

	bids := store.FetchBidsByProtocolAndBlock(10, "default:v0:ethBundles")
	require.Len(t, bids, 1)
	require.Equal(t, bid, bids[0])
}
