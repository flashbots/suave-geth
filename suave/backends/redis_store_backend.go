package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/go-redis/redis/v8"
)

var (
	formatRedisBidKey = func(bidId suave.BidId) string {
		return fmt.Sprintf("bid-%x", bidId)
	}

	formatRedisBidValueKey = func(bidId suave.BidId, key string) string {
		return fmt.Sprintf("bid-data-%x-%s", bidId, key)
	}

	ffStoreTTL = 24 * time.Hour
)

type RedisStoreBackend struct {
	ctx      context.Context
	cancel   context.CancelFunc
	redisUri string
	client   *redis.Client
	local    *miniredis.Miniredis
}

func NewLocalConfidentialStore() *RedisStoreBackend {
	return NewRedisStoreBackend("")
}

func NewRedisStoreBackend(redisUri string) *RedisStoreBackend {
	r := &RedisStoreBackend{
		cancel:   nil,
		redisUri: redisUri,
	}
	return r
}

func (r *RedisStoreBackend) Start() error {
	if r.redisUri == "" {
		// create a mini-redis instance
		localRedis, err := miniredis.Run()
		if err != nil {
			return err
		}
		r.local = localRedis
		r.redisUri = localRedis.Addr()
	}

	if r.cancel != nil {
		r.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.ctx = ctx

	client, err := connectRedis(r.redisUri)
	if err != nil {
		return err
	}
	r.client = client

	err = r.InitializeBid(mempoolConfidentialStoreBid)
	if err != nil && !errors.Is(err, suave.ErrBidAlreadyPresent) {
		return fmt.Errorf("mempool: could not initialize: %w", err)
	}

	return nil
}

func (r *RedisStoreBackend) Stop() error {
	if r.cancel == nil || r.client == nil {
		return errors.New("Redis store: Stop() called before Start()")
	}

	if r.local != nil {
		r.local.Close()
	}
	r.cancel()
	r.client.Close()

	return nil
}

func (r *RedisStoreBackend) InitializeBid(bid suave.Bid) error {
	key := formatRedisBidKey(bid.Id)

	err := r.client.Get(r.ctx, key).Err()
	if !errors.Is(err, redis.Nil) {
		return suave.ErrBidAlreadyPresent
	}

	data, err := json.Marshal(bid)
	if err != nil {
		return err
	}

	err = r.client.Set(r.ctx, key, string(data), ffStoreTTL).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisStoreBackend) FetchEngineBidById(bidId suave.BidId) (suave.Bid, error) {
	key := formatRedisBidKey(bidId)

	data, err := r.client.Get(r.ctx, key).Bytes()
	if err != nil {
		return suave.Bid{}, err
	}

	var bid suave.Bid
	err = json.Unmarshal(data, &bid)
	if err != nil {
		return suave.Bid{}, err
	}

	return bid, nil
}

func (r *RedisStoreBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	storeKey := formatRedisBidValueKey(bid.Id, key)
	err := r.client.Set(r.ctx, storeKey, string(value), ffStoreTTL).Err()
	if err != nil {
		return suave.Bid{}, fmt.Errorf("unexpected redis error: %w", err)
	}

	return bid, nil
}

func (r *RedisStoreBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	storeKey := formatRedisBidValueKey(bid.Id, key)
	data, err := r.client.Get(r.ctx, storeKey).Bytes()
	if err != nil {
		return []byte{}, fmt.Errorf("unexpected redis error: %w, %s, %v", err, storeKey, r.client.Keys(context.TODO(), "*").String())
	}

	return data, nil
}

func (r *RedisStoreBackend) SubmitBid(bid types.Bid) error {
	defer log.Info("bid submitted", "bid", bid, "store", r.Store)

	var bidsByBlockAndProtocol []types.Bid
	bidsByBlockAndProtocolBytes, err := r.Retrieve(mempoolConfidentialStoreBid, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", bid.Version, bid.DecryptionCondition))
	if err == nil {
		bidsByBlockAndProtocol = suave.MustDecode[[]types.Bid](bidsByBlockAndProtocolBytes)
	}
	// store bid by block number and by protocol + block number
	bidsByBlockAndProtocol = append(bidsByBlockAndProtocol, bid)

	r.Store(mempoolConfidentialStoreBid, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", bid.Version, bid.DecryptionCondition), suave.MustEncode(bidsByBlockAndProtocol))

	return nil
}

func (r *RedisStoreBackend) FetchBidById(bidId suave.BidId) (types.Bid, error) {
	engineBid, err := r.FetchEngineBidById(bidId)
	if err != nil {
		log.Error("bid missing!", "id", bidId, "err", err)
		return types.Bid{}, errors.New("not found")
	}

	return types.Bid{
		Id:                  engineBid.Id,
		Salt:                engineBid.Salt,
		DecryptionCondition: engineBid.DecryptionCondition,
		AllowedPeekers:      engineBid.AllowedPeekers,
		AllowedStores:       engineBid.AllowedStores,
		Version:             engineBid.Version,
	}, nil
}

func (r *RedisStoreBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []types.Bid {
	bidsByProtocolBytes, err := r.Retrieve(mempoolConfidentialStoreBid, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", namespace, blockNumber))
	if err != nil {
		return nil
	}
	defer log.Info("bids fetched", "bids", string(bidsByProtocolBytes))
	return suave.MustDecode[[]types.Bid](bidsByProtocolBytes)
}
