package suave

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ConfidentialStoreEngine struct {
	backend ConfidentialStoreBackend
	pubsub  PubSub
}

func NewConfidentialStoreEngine(backend ConfidentialStoreBackend, pubsub PubSub) (*ConfidentialStoreEngine, error) {
	engine := &ConfidentialStoreEngine{
		backend: backend,
		pubsub:  pubsub,
	}

	return engine, nil
}

func (e *ConfidentialStoreEngine) Subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-e.pubsub.Subscribe():
			e.NewMessage(msg)
		}
	}
}

func (e *ConfidentialStoreEngine) InitializeBid(bid Bid) (Bid, error) {
	return e.backend.InitializeBid(bid)
}

func (e *ConfidentialStoreEngine) Store(bidId BidId, caller common.Address, key string, value []byte) (Bid, error) {
	bid, err := e.backend.FetchBidById(bidId)
	if err != nil {
		return Bid{}, err
	}
	msg := DAMessage{
		Bid:       bid,
		SourceTx:  types.Transaction{}, // TODO! maybe not needed here, but definitely needed in Initialize
		Caller:    caller,
		Key:       key,
		Value:     value,
		Signature: nil, // TODO!
	}

	e.pubsub.Publish(msg)

	return e.backend.Store(bidId, caller, key, value)
}

func (e *ConfidentialStoreEngine) Retrieve(bidId BidId, caller common.Address, key string) ([]byte, error) {
	return e.backend.Retrieve(bidId, caller, key)
}

func (e *ConfidentialStoreEngine) NewMessage(message DAMessage) {
	// TODO: validation

	bid, err := e.backend.InitializeBid(message.Bid)
	if err != nil && !errors.Is(err, BidAlreadyPresentError) {
		panic(fmt.Errorf("unexpected error while initializing bid from transport: %w", err))
	}

	_, err = e.backend.Store(bid.Id, message.Caller, message.Key, message.Value)
	if err != nil {
		panic(fmt.Errorf("unexpected error while storing, the message was not validated properly: %w (%v)", err, message.Caller))
	}
}

type MockPubSub struct{}

func (MockPubSub) Subscribe() <-chan DAMessage { return nil }
func (MockPubSub) Publish(DAMessage)           {}
