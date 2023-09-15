package backends

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func TestMiniredisTransport(t *testing.T) {
	mb := NewMiniredisBackend()
	require.NoError(t, mb.Start())
	t.Cleanup(func() { mb.Stop() })

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)

	msgSub := mb.Subscribe(ctx)

	daMsg := suave.DAMessage{
		Bid: suave.Bid{
			Id:                  suave.BidId{0x42},
			DecryptionCondition: uint64(13),
			AllowedPeekers:      []common.Address{{0x41, 0x39}},
			Version:             string("vv"),
		},
		Value:     []byte{},
		Signature: []byte{},
	}

	mb.Publish(daMsg)

	select {
	case msg := <-msgSub:
		require.Equal(t, daMsg, msg)
	case <-time.After(5 * time.Millisecond):
		t.Error("did not receive expected message")
	}

	select {
	case <-msgSub:
		t.Error("received an expected message")
	case <-time.After(5 * time.Millisecond):
	}

	daMsg.Bid.Id[0] = 0x43
	mb.Publish(daMsg)

	select {
	case msg := <-msgSub:
		require.Equal(t, daMsg, msg)
	case <-time.After(5 * time.Millisecond):
		t.Error("did not receive expected message")
	}
}

func TestMiniredisStore(t *testing.T) {
	mb := NewMiniredisBackend()
	require.NoError(t, mb.Start())
	t.Cleanup(func() { mb.Stop() })

	bid := suave.Bid{
		Id:                  suave.BidId{0x42},
		DecryptionCondition: uint64(13),
		AllowedPeekers:      []common.Address{{0x41, 0x39}},
		Version:             string("vv"),
	}

	err := mb.InitializeBid(bid)
	require.NoError(t, err)

	fetchedBid, err := mb.FetchEngineBidById(bid.Id)
	require.NoError(t, err)
	require.Equal(t, bid, fetchedBid)

	_, err = mb.Store(bid, bid.AllowedPeekers[0], "xx", []byte{0x43, 0x14})
	require.NoError(t, err)

	retrievedData, err := mb.Retrieve(bid, bid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)

	_, err = mb.Retrieve(bid, bid.AllowedPeekers[0], "xxy")
	require.Error(t, err)
}

func TestRedisTransport(t *testing.T) {
	mr := miniredis.RunT(t)

	redisPubSub := NewRedisPubSub(mr.Addr())
	require.NoError(t, redisPubSub.Start())
	t.Cleanup(func() { redisPubSub.Stop() })

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)

	msgSub := redisPubSub.Subscribe(ctx)

	daMsg := suave.DAMessage{
		Bid: suave.Bid{
			Id:                  suave.BidId{0x42},
			DecryptionCondition: uint64(13),
			AllowedPeekers:      []common.Address{{0x41, 0x39}},
			Version:             string("vv"),
		},
		Value:     suave.Bytes{},
		Signature: []byte{},
	}

	redisPubSub.Publish(daMsg)

	select {
	case msg := <-msgSub:
		require.Equal(t, daMsg, msg)
	case <-time.After(50 * time.Millisecond):
		t.Error("did not receive expected message")
	}

	select {
	case <-msgSub:
		t.Error("received an expected message")
	case <-time.After(5 * time.Millisecond):
	}

	daMsg.Bid.Id[0] = 0x43
	redisPubSub.Publish(daMsg)

	select {
	case msg := <-msgSub:
		require.Equal(t, daMsg, msg)
	case <-time.After(5 * time.Millisecond):
		t.Error("did not receive expected message")
	}
}

func TestRedisStore(t *testing.T) {
	mr := miniredis.RunT(t)

	redisStoreBackend := NewRedisStoreBackend(mr.Addr())
	redisStoreBackend.Start()
	t.Cleanup(func() { redisStoreBackend.Stop() })

	bid := suave.Bid{
		Id:                  suave.BidId{0x42},
		DecryptionCondition: uint64(13),
		AllowedPeekers:      []common.Address{{0x41, 0x39}},
		Version:             string("vv"),
	}

	err := redisStoreBackend.InitializeBid(bid)
	require.NoError(t, err)

	fetchedBid, err := redisStoreBackend.FetchEngineBidById(bid.Id)
	require.NoError(t, err)
	require.Equal(t, bid, fetchedBid)

	_, err = redisStoreBackend.Store(bid, bid.AllowedPeekers[0], "xx", []byte{0x43, 0x14})
	require.NoError(t, err)

	retrievedData, err := redisStoreBackend.Retrieve(bid, bid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)
}

func TestEngineOnRedis(t *testing.T) {
	mrStore1 := miniredis.RunT(t)
	mrStore2 := miniredis.RunT(t)
	mrPubSub := mrStore1

	redisPubSub1 := NewRedisPubSub(mrPubSub.Addr())
	redisStoreBackend1 := NewRedisStoreBackend(mrStore1.Addr())

	engine1, err := suave.NewConfidentialStoreEngine(redisStoreBackend1, redisPubSub1, suave.MockMempool{}, suave.MockSigner{}, suave.MockChainSigner{})
	require.NoError(t, err)

	require.NoError(t, engine1.Start())
	t.Cleanup(func() { engine1.Stop() })

	redisPubSub2 := NewRedisPubSub(mrPubSub.Addr())
	redisStoreBackend2 := NewRedisStoreBackend(mrStore2.Addr())

	engine2, err := suave.NewConfidentialStoreEngine(redisStoreBackend2, redisPubSub2, suave.MockMempool{}, suave.MockSigner{}, suave.MockChainSigner{})
	require.NoError(t, err)

	require.NoError(t, engine2.Start())
	t.Cleanup(func() { engine2.Stop() })

	dummyCreationTx := types.NewTx(&types.ConfidentialComputeRequest{
		ExecutionNode: common.Address{},
		Wrapped:       *types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
	})

	// Make sure a store to engine1 is propagated to endine2 through redis->miniredis transport
	bid, err := engine1.InitializeBid(types.Bid{
		DecryptionCondition: uint64(13),
		AllowedPeekers:      []common.Address{{0x41, 0x39}},
		AllowedStores:       []common.Address{common.Address{}},
		Version:             string("vv"),
	}, dummyCreationTx)
	require.NoError(t, err)

	redisPubSub3 := NewRedisPubSub(mrPubSub.Addr())
	require.NoError(t, redisPubSub3.Start())
	t.Cleanup(func() { redisPubSub3.Stop() })

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)

	// Do not subscribe on redisPubSub1 or 2! That would cause one of the subscribers to not receive the message, I think
	subch := redisPubSub3.Subscribe(ctx)

	// Trigger propagation
	_, err = engine1.Store(bid.Id, dummyCreationTx, bid.AllowedPeekers[0], "xx", []byte{0x43, 0x14})

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

	select {
	case msg := <-subch:
		rececivedBidJson, err := json.Marshal(msg.Bid)
		require.NoError(t, err)

		require.Equal(t, submittedBidJson, rececivedBidJson)
		require.Equal(t, "xx", msg.Key)
		require.Equal(t, suave.Bytes{0x43, 0x14}, msg.Value)
		require.Equal(t, bid.AllowedPeekers[0], msg.Caller)
	case <-time.After(20 * time.Millisecond):
		t.Error("did not receive expected message")
	}

	retrievedData, err := engine2.Retrieve(bid.Id, bid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)

	fetchedBid, err := redisStoreBackend2.FetchEngineBidById(bid.Id)
	require.NoError(t, err)

	fetchedBidJson, err := json.Marshal(fetchedBid)
	require.NoError(t, err)

	require.Equal(t, submittedBidJson, fetchedBidJson)
}
