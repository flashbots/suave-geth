package backends

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type MiniredisBackend struct {
	ctx    context.Context
	cancel context.CancelFunc
	client *miniredis.Miniredis
}

func NewMiniredisBackend() *MiniredisBackend {
	return &MiniredisBackend{}
}

func (r *MiniredisBackend) Start() error {
	if r.cancel != nil {
		r.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.ctx = ctx

	client, err := miniredis.Run()
	if err != nil {
		r.cancel()
		return err
	}
	r.client = client

	return nil
}

func (r *MiniredisBackend) Stop() error {
	if r.cancel == nil || r.client == nil {
		return errors.New("Minireddis: Stop() called before Start()")
	}

	r.cancel()
	r.client.Close()

	return nil
}

func (r *MiniredisBackend) Subscribe() (<-chan suave.DAMessage, context.CancelFunc) {
	ctx, cancel := context.WithCancel(r.ctx)

	ch := make(chan suave.DAMessage, 16)

	subscriber := r.client.NewSubscriber()
	subscriber.Subscribe(redisUpsertTopic)

	go func() {
		defer close(ch)
		defer subscriber.Close()

		for {
			var msg suave.DAMessage
			select {
			case <-ctx.Done():
				log.Info("Miniredis: closing subscription")
				return
			case rmsg := <-subscriber.Messages():
				err := json.Unmarshal([]byte(rmsg.Message), &msg)
				if err != nil {
					log.Debug("could not parse message from subscription", "err", err, "msg", rmsg.Message)
					continue
				}
			}

			select {
			case <-ctx.Done():
				log.Info("Miniredis: closing subscription")
				return
			case ch <- msg:
				continue
			default:
				log.Warn("dropping transport message due to channel being blocked")
				continue
			}
		}
	}()

	return ch, cancel
}

func (r *MiniredisBackend) Publish(message suave.DAMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		panic(fmt.Errorf("could not marshal message: %w", err))
	}

	r.client.Publish(redisUpsertTopic, string(data))
}

func (r *MiniredisBackend) InitializeBid(bid suave.Bid) error {
	key := formatRedisBidKey(bid.Id)

	_, err := r.client.Get(key)
	if !errors.Is(err, miniredis.ErrKeyNotFound) {
		return suave.ErrBidAlreadyPresent
	}

	data, err := json.Marshal(bid)
	if err != nil {
		return err
	}

	r.client.Set(key, string(data))

	return nil
}

func (r *MiniredisBackend) FetchEngineBidById(bidId suave.BidId) (suave.Bid, error) {
	key := formatRedisBidKey(bidId)

	data, err := r.client.Get(key)
	if err != nil {
		return suave.Bid{}, err
	}

	var bid suave.Bid
	err = json.Unmarshal([]byte(data), &bid)
	if err != nil {
		return suave.Bid{}, err
	}

	return bid, nil
}

func (r *MiniredisBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	storeKey := formatRedisBidValueKey(bid.Id, key)
	err := r.client.Set(storeKey, string(value))
	if err != nil {
		return suave.Bid{}, fmt.Errorf("unexpected redis error: %w", err)
	}

	return bid, nil
}

func (r *MiniredisBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	storeKey := formatRedisBidValueKey(bid.Id, key)
	data, err := r.client.Get(storeKey)
	if err != nil {
		return []byte{}, fmt.Errorf("unexpected redis error: %w", err)
	}

	return []byte(data), nil
}
