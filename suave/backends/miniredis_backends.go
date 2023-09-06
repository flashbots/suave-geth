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
	"github.com/google/uuid"
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
	client     *miniredis.Miniredis
	subscriber *miniredis.Subscriber
}

func NewMiniredisBackend(ctx context.Context) (*MiniredisBackend, error) {
	client, err := miniredis.Run()
	if err != nil {
		return nil, err
	}

	subscriber := client.NewSubscriber()
	subscriber.Subscribe(miniredisUpsertTopic)

	go func() {
		<-ctx.Done()
		subscriber.Close()
		client.Close()
	}()

	return &MiniredisBackend{
		ctx:        ctx,
		client:     client,
		subscriber: subscriber,
	}, nil
}

func (r *MiniredisBackend) Subscribe() <-chan suave.DAMessage {
	ch := make(chan suave.DAMessage, 16)

	go func() {
		for rmsg := range r.subscriber.Messages() {
			var msg suave.DAMessage
			err := json.Unmarshal([]byte(rmsg.Message), &msg)
			if err != nil {
				log.Debug("could not parse message from subscription", "err", err, "msg", rmsg.Message)
			}

			select {
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

func (r *MiniredisBackend) InitializeBid(bid suave.Bid) (suave.Bid, error) {
	if bid.Id == [16]byte{} {
		bid.Id = [16]byte(uuid.New())
	}

	key := formatMiniredisBidKey(bid.Id)

	_, err := r.client.Get(key)
	if !errors.Is(err, miniredis.ErrKeyNotFound) {
		return suave.Bid{}, suave.BidAlreadyPresentError
	}

	data, err := json.Marshal(bid)
	if err != nil {
		return suave.Bid{}, err
	}

	r.client.Set(key, string(data))

	return bid, nil
}

func (r *MiniredisBackend) FetchBidById(bidId suave.BidId) (suave.Bid, error) {
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
	bid, err := r.FetchBidById(bidId)
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
	bid, err := r.FetchBidById(bidId)
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
