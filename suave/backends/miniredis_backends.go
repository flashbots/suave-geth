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
	"golang.org/x/exp/slices"
)

var (
	miniredisUpsertTopic = "store:upsert"

	formatMiniredisBidKey = func(bidId suave.BidId) string {
		return fmt.Sprintf("bid-%x", bidId)
	}

	formatMiniredisBidValueKey = func(bid suave.Bid, key string) string {
		return fmt.Sprintf("bid-data-%x-%s", bid.Id, key) // TODO: should also include the hash of the bid at least
	}
)

type MiniredisBackend struct {
	ctx        context.Context
	cancel     context.CancelFunc
	client     *miniredis.Miniredis
	subscriber *miniredis.Subscriber
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
		return err
	}
	r.client = client

	subscriber := client.NewSubscriber()
	subscriber.Subscribe(miniredisUpsertTopic)
	r.subscriber = subscriber

	return nil
}

func (r *MiniredisBackend) Stop() error {
	if r.cancel == nil || r.subscriber == nil || r.client == nil {
		panic("Stop() called before Start()")
	}

	r.cancel()
	r.subscriber.Close()
	r.client.Close()

	return nil
}

func (r *MiniredisBackend) Subscribe(ctx context.Context) <-chan suave.DAMessage {
	if r.cancel == nil || r.subscriber == nil || r.client == nil {
		panic("Subscribe() called before Start()")
	}

	ch := make(chan suave.DAMessage, 16)

	go func() {
		for rmsg := range r.subscriber.Messages() {
			var msg suave.DAMessage
			err := json.Unmarshal([]byte(rmsg.Message), &msg)
			if err != nil {
				log.Debug("could not parse message from subscription", "err", err, "msg", rmsg.Message)
			}

			select {
			case <-ctx.Done():
				return
			case <-r.ctx.Done():
				return
			case ch <- msg:
				continue
			default:
				log.Warn("dropping transport message due to channel being blocked")
				continue
			}
		}
	}()

	return ch
}

func (r *MiniredisBackend) Publish(message suave.DAMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		panic(fmt.Errorf("could not marshal message: %w", err))
	}

	r.subscriber.Publish(miniredisUpsertTopic, string(data))
}

func (r *MiniredisBackend) InitializeBid(bid suave.Bid) error {
	key := formatMiniredisBidKey(bid.Id)

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
	key := formatMiniredisBidKey(bidId)

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

func (r *MiniredisBackend) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	bid, err := r.FetchEngineBidById(bidId)
	if err != nil {
		return suave.Bid{}, errors.New("bid not present yet")
	}

	if !slices.Contains(bid.AllowedPeekers, caller) {
		return suave.Bid{}, fmt.Errorf("%x not allowed to store %s on %x", caller, key, bidId)
	}

	storeKey := formatMiniredisBidValueKey(bid, key)
	err = r.client.Set(storeKey, string(value))
	if err != nil {
		return suave.Bid{}, fmt.Errorf("unexpected redis error: %w", err)
	}

	return bid, nil
}

func (r *MiniredisBackend) Retrieve(bidId suave.BidId, caller common.Address, key string) ([]byte, error) {
	bid, err := r.FetchEngineBidById(bidId)
	if err != nil {
		return []byte{}, errors.New("bid not present yet")
	}

	if !slices.Contains(bid.AllowedPeekers, caller) {
		return []byte{}, fmt.Errorf("%x not allowed to fetch %s on %x", caller, key, bidId)
	}

	storeKey := formatMiniredisBidValueKey(bid, key)
	data, err := r.client.Get(storeKey)
	if err != nil {
		return []byte{}, fmt.Errorf("unexpected redis error: %w", err)
	}

	return []byte(data), nil
}
