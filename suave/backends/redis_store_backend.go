package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/go-redis/redis/v8"
)

var ffStoreTTL = 24 * time.Hour

type RedisStoreBackend struct {
	ctx      context.Context
	cancel   context.CancelFunc
	redisUri string
	client   *redis.Client
}

func NewRedisStoreBackend(redisUri string) *RedisStoreBackend {
	return &RedisStoreBackend{
		cancel:   nil,
		redisUri: redisUri,
	}
}

func (r *RedisStoreBackend) Start() error {
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

	return nil
}

func (r *RedisStoreBackend) Stop() error {
	if r.cancel == nil || r.client == nil {
		panic("Stop() called before Start()")
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
