package backends

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func TestMiniredisTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	mb, err := NewMiniredisBackend(ctx)
	require.NoError(t, err)

	msgSub := mb.Subscribe()

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
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	mb, err := NewMiniredisBackend(ctx)
	require.NoError(t, err)

	bid := suave.Bid{
		Id:                  suave.BidId{0x42},
		DecryptionCondition: uint64(13),
		AllowedPeekers:      []common.Address{{0x41, 0x39}},
		Version:             string("vv"),
	}

	err = mb.InitializeBid(bid)
	require.NoError(t, err)

	fetchedBid, err := mb.FetchEngineBidById(bid.Id)
	require.NoError(t, err)
	require.Equal(t, bid, fetchedBid)

	_, err = mb.Store(bid.Id, bid.AllowedPeekers[0], "xx", []byte{0x43, 0x14})
	require.NoError(t, err)

	_, err = mb.Store(bid.Id, common.Address{0x41, 0x38}, "xxy", []byte{0x43, 0x15})
	require.Error(t, err)

	retrievedData, err := mb.Retrieve(bid.Id, bid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)

	_, err = mb.Retrieve(bid.Id, bid.AllowedPeekers[0], "xxy")
	require.Error(t, err)

	_, err = mb.Retrieve(bid.Id, common.Address{0x41, 0x38}, "xx")
	require.Error(t, err)
}

func TestRedisTransport(t *testing.T) {
	mr := miniredis.RunT(t)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	redisPubSub, err := NewRedisPubSub(ctx, mr.Addr())
	require.NoError(t, err)

	msgSub := redisPubSub.Subscribe()

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
	case <-time.After(5 * time.Millisecond):
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

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	redisStoreBackend, err := NewRedisStoreBackend(ctx, mr.Addr())
	require.NoError(t, err)

	bid := suave.Bid{
		Id:                  suave.BidId{0x42},
		DecryptionCondition: uint64(13),
		AllowedPeekers:      []common.Address{{0x41, 0x39}},
		Version:             string("vv"),
	}

	err = redisStoreBackend.InitializeBid(bid)
	require.NoError(t, err)

	fetchedBid, err := redisStoreBackend.FetchEngineBidById(bid.Id)
	require.NoError(t, err)
	require.Equal(t, bid, fetchedBid)

	_, err = redisStoreBackend.Store(bid.Id, bid.AllowedPeekers[0], "xx", []byte{0x43, 0x14})
	require.NoError(t, err)

	_, err = redisStoreBackend.Store(bid.Id, common.Address{0x41, 0x38}, "xxy", []byte{0x43, 0x15})
	require.Error(t, err)

	retrievedData, err := redisStoreBackend.Retrieve(bid.Id, bid.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)

	_, err = redisStoreBackend.Retrieve(bid.Id, bid.AllowedPeekers[0], "xxy")
	require.Error(t, err)

	_, err = redisStoreBackend.Retrieve(bid.Id, common.Address{0x41, 0x38}, "xx")
	require.Error(t, err)
}

func TestEngineOnRedis(t *testing.T) {
	mrStore1 := miniredis.RunT(t)
	mrStore2 := miniredis.RunT(t)
	mrPubSub := mrStore1

	ctx, cancel := context.WithCancel(context.TODO())
	t.Cleanup(cancel)

	redisPubSub1, err := NewRedisPubSub(ctx, mrPubSub.Addr())
	require.NoError(t, err)

	redisStoreBackend1, err := NewRedisStoreBackend(ctx, mrStore1.Addr())
	require.NoError(t, err)

	engine1, err := suave.NewConfidentialStoreEngine(redisStoreBackend1, redisPubSub1, suave.MockSigner{}, suave.MockChainSigner{})
	require.NoError(t, err)

	redisPubSub2, err := NewRedisPubSub(ctx, mrPubSub.Addr())
	require.NoError(t, err)

	redisStoreBackend2, err := NewRedisStoreBackend(ctx, mrStore2.Addr())
	require.NoError(t, err)

	engine2, err := suave.NewConfidentialStoreEngine(redisStoreBackend2, redisPubSub2, suave.MockSigner{}, suave.MockChainSigner{})
	require.NoError(t, err)

	go engine2.Subscribe(ctx)

	// Make sure a store to engine1 is propagated to endine2 through redis->miniredis transport
	bid, err := engine1.InitializeBid(types.Bid{
		DecryptionCondition: uint64(13),
		AllowedPeekers:      []common.Address{{0x41, 0x39}},
		AllowedStores:       []common.Address{common.Address{}},
		Version:             string("vv"),
	}, nil /* creation tx */)
	require.NoError(t, err)

	// Do not subscribe on redisPubSub2! That would cause one of the subscribers to not receive the message, I think
	subch := redisPubSub1.Subscribe()

	// Trigger propagation
	_, err = engine1.Store(bid.Id, nil /* source tx */, bid.AllowedPeekers[0], "xx", []byte{0x43, 0x14})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	submittedBid := suave.Bid{
		Id:                  bid.Id,
		DecryptionCondition: bid.DecryptionCondition,
		AllowedPeekers:      bid.AllowedPeekers,
		AllowedStores:       bid.AllowedStores,
		Version:             bid.Version,
		CreationTx:          nil,
	}

	var nilAddress common.Address
	submittedBid.Signature = nilAddress.Bytes()

	select {
	case msg := <-subch:
		require.Equal(t, submittedBid, msg.Bid)
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
	require.Equal(t, submittedBid, fetchedBid)
}
