package cstore

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func TestRedisTransport(t *testing.T) {
	mr := miniredis.RunT(t)

	redisPubSub := NewRedisPubSubTransport(mr.Addr())
	require.NoError(t, redisPubSub.Start())
	t.Cleanup(func() { redisPubSub.Stop() })

	msgSub, cancel := redisPubSub.Subscribe()
	t.Cleanup(cancel)

	daMsg := DAMessage{
		StoreWrites: []StoreWrite{{
			Bid: suave.Bid{
				Id:                  suave.BidId{0x42},
				DecryptionCondition: uint64(13),
				AllowedPeekers:      []common.Address{{0x41, 0x39}},
				Version:             string("vv"),
			},
			Value: suave.Bytes{},
		}},
		Signature: []byte{},
	}

	redisPubSub.Publish(daMsg)

	select {
	case msg := <-msgSub:
		require.Equal(t, daMsg, msg)
	case <-time.After(100 * time.Millisecond):
		t.Error("did not receive expected message")
	}

	select {
	case <-msgSub:
		t.Error("received an expected message")
	case <-time.After(5 * time.Millisecond):
	}

	daMsg.StoreWrites[0].Bid.Id[0] = 0x43
	redisPubSub.Publish(daMsg)

	select {
	case msg := <-msgSub:
		require.Equal(t, daMsg, msg)
	case <-time.After(100 * time.Millisecond):
		t.Error("did not receive expected message")
	}
}

func TestEngineOnRedis(t *testing.T) {
	mrStore1 := miniredis.RunT(t)
	mrStore2 := miniredis.RunT(t)
	mrPubSub := mrStore1

	redisPubSub1 := NewRedisPubSubTransport(mrPubSub.Addr())
	redisStoreBackend1, _ := NewRedisStoreBackend(mrStore1.Addr())

	engine1 := NewConfidentialStoreEngine(redisStoreBackend1, redisPubSub1, MockSigner{}, MockChainSigner{})
	require.NoError(t, engine1.Start())
	t.Cleanup(func() { engine1.Stop() })

	redisPubSub2 := NewRedisPubSubTransport(mrPubSub.Addr())
	redisStoreBackend2, _ := NewRedisStoreBackend(mrStore2.Addr())

	engine2 := NewConfidentialStoreEngine(redisStoreBackend2, redisPubSub2, MockSigner{}, MockChainSigner{})
	require.NoError(t, engine2.Start())
	t.Cleanup(func() { engine2.Stop() })

	testKey, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	dummyCreationTx, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: common.Address{},
		},
	}), types.NewSuaveSigner(new(big.Int)), testKey)
	require.NoError(t, err)

	// Make sure a store to engine1 is propagated to endine2 through redis->miniredis transport
	bid, err := engine1.InitializeBid(types.Bid{
		DecryptionCondition: uint64(13),
		AllowedPeekers:      []common.Address{{0x41, 0x39}},
		AllowedStores:       []common.Address{{}},
		Version:             string("vv"),
	}, dummyCreationTx)
	require.NoError(t, err)

	redisPubSub3 := NewRedisPubSubTransport(mrPubSub.Addr())
	require.NoError(t, redisPubSub3.Start())
	t.Cleanup(func() { redisPubSub3.Stop() })

	// Do not subscribe on redisPubSub1 or 2! That would cause one of the subscribers to not receive the message, I think
	subch, cancel := redisPubSub3.Subscribe()
	t.Cleanup(cancel)

	// Trigger propagation
	err = engine1.Finalize(dummyCreationTx, nil, []StoreWrite{{
		Bid:    bid,
		Caller: bid.AllowedPeekers[0],
		Key:    "xx",
		Value:  []byte{0x43, 0x14},
	}})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	submittedBid := suave.Bid{
		Id:                  bid.Id,
		Salt:                bid.Salt,
		DecryptionCondition: bid.DecryptionCondition,
		AllowedPeekers:      bid.AllowedPeekers,
		AllowedStores:       bid.AllowedStores,
		Version:             bid.Version,
		CreationTx:          dummyCreationTx,
	}

	var nilAddress common.Address
	submittedBid.Signature = nilAddress.Bytes()

	submittedBidJson, err := json.Marshal(submittedBid)
	require.NoError(t, err)

	// require.NoError(t, engine1.Finalize(dummyCreationTx))

	select {
	case msg := <-subch:
		rececivedBidJson, err := json.Marshal(msg.StoreWrites[0].Bid)
		require.NoError(t, err)

		require.Equal(t, submittedBidJson, rececivedBidJson)
		require.Equal(t, "xx", msg.StoreWrites[0].Key)
		require.Equal(t, suave.Bytes{0x43, 0x14}, msg.StoreWrites[0].Value)
		require.Equal(t, bid.AllowedPeekers[0], msg.StoreWrites[0].Caller)
	case <-time.After(20 * time.Millisecond):
		t.Error("did not receive expected message")
	}

	retrievedData, err := engine2.Retrieve(bid.Id, bid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)

	fetchedBid, err := redisStoreBackend2.FetchBidById(bid.Id)
	require.NoError(t, err)

	fetchedBidJson, err := json.Marshal(fetchedBid)
	require.NoError(t, err)

	require.Equal(t, submittedBidJson, fetchedBidJson)
}
