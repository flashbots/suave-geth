package cstore

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func TestTransactionalStore(t *testing.T) {
	engine := NewConfidentialStoreEngine(NewLocalConfidentialStore(), MockTransport{}, MockSigner{}, MockChainSigner{})

	testKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	dummyCreationTx, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: common.Address{0x42},
		},
	}), types.NewSuaveSigner(new(big.Int)), testKey)
	require.NoError(t, err)

	tstore := engine.NewTransactionalStore(dummyCreationTx)

	testBid, err := tstore.InitializeBid(types.Bid{
		Salt:                RandomBidId(),
		DecryptionCondition: 46,
		AllowedStores:       []common.Address{{0x42}},
		AllowedPeekers:      []common.Address{{0x43}},
		Version:             "v0-test",
	})
	require.NoError(t, err)

	_, err = tstore.Store(testBid.Id, testBid.AllowedPeekers[0], "xx", []byte{0x44})
	require.NoError(t, err)

	tfetchedBid, err := tstore.FetchBidById(testBid.Id)
	require.NoError(t, err)
	require.Equal(t, testBid, tfetchedBid.ToInnerBid())

	require.Empty(t, tstore.FetchBidsByProtocolAndBlock(45, "v0-test"))
	require.Empty(t, tstore.FetchBidsByProtocolAndBlock(46, "v1-test"))

	tfetchedBids := tstore.FetchBidsByProtocolAndBlock(46, "v0-test")
	require.Equal(t, 1, len(tfetchedBids))
	require.Equal(t, testBid, tfetchedBids[0].ToInnerBid())

	_, err = tstore.Retrieve(testBid.Id, testBid.AllowedPeekers[0], "xy")
	require.Error(t, err)

	_, err = tstore.Retrieve(suave.RandomBidId(), testBid.AllowedPeekers[0], "xx")
	require.Error(t, err)

	_, err = tstore.Retrieve(testBid.Id, testBid.AllowedStores[0], "xx")
	require.Error(t, err)

	tretrieved, err := tstore.Retrieve(testBid.Id, testBid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x44}, tretrieved)

	// Not finalized, engine should return empty
	_, err = engine.FetchBidById(testBid.Id)
	require.Error(t, err)
	require.Empty(t, engine.FetchBidsByProtocolAndBlock(46, "v0-test"))
	_, err = engine.Retrieve(testBid.Id, testBid.AllowedPeekers[0], "xx")
	require.Error(t, err)

	require.NoError(t, tstore.Finalize())

	efetchedBid, err := engine.FetchBidById(testBid.Id)
	require.NoError(t, err)
	require.Equal(t, testBid, efetchedBid.ToInnerBid())

	efetchedBids := engine.FetchBidsByProtocolAndBlock(46, "v0-test")
	require.Equal(t, 1, len(efetchedBids))
	require.Equal(t, testBid, efetchedBids[0].ToInnerBid())

	eretrieved, err := engine.Retrieve(testBid.Id, testBid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x44}, eretrieved)
}
