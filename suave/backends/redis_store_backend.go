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
	"golang.org/x/exp/slices"
)

type RedisStoreBackend struct {
	ctx    context.Context
	client *redis.Client
}

func NewRedisStoreBackend(ctx context.Context, redisURI string) (*RedisStoreBackend, error) {
	client, err := connectRedis(redisURI)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		client.Close()
	}()

	return &RedisStoreBackend{
		ctx:    ctx,
		client: client,
	}, nil
}

func (r *RedisStoreBackend) InitializeBid(bid suave.Bid) error {
	key := formatRedisBidKey(bid.Id)

	err := r.client.Get(r.ctx, key).Err()
	if !errors.Is(err, redis.Nil) {
		return suave.BidAlreadyPresentError
	}

	data, err := json.Marshal(bid)
	if err != nil {
		return err
	}

	err = r.client.Set(r.ctx, key, string(data), time.Second).Err()
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

func (r *RedisStoreBackend) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	bid, err := r.FetchEngineBidById(bidId)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("could not fetch bid from redis: %w", err)
	}

	if !slices.Contains(bid.AllowedPeekers, caller) {
		return suave.Bid{}, fmt.Errorf("%x not allowed to store %s on %x", caller, key, bidId)
	}

	storeKey := formatRedisBidValueKey(bid, key)
	err = r.client.Set(r.ctx, storeKey, string(value), time.Second).Err()
	if err != nil {
		return suave.Bid{}, fmt.Errorf("unexpected redis error: %w", err)
	}

	return bid, nil
}

func (r *RedisStoreBackend) Retrieve(bidId suave.BidId, caller common.Address, key string) ([]byte, error) {
	bid, err := r.FetchEngineBidById(bidId)
	if err != nil {
		return []byte{}, errors.New("bid not present yet")
	}

	if !slices.Contains(bid.AllowedPeekers, caller) {
		return []byte{}, fmt.Errorf("%x not allowed to fetch %s on %x", caller, key, bidId)
	}

	storeKey := formatRedisBidValueKey(bid, key)
	data, err := r.client.Get(r.ctx, storeKey).Bytes()
	if err != nil {
		return []byte{}, fmt.Errorf("unexpected redis error: %w, %s, %v", err, storeKey, r.client.Keys(context.TODO(), "*").String())
	}

	return data, nil
}
