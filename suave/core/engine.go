package suave

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/exp/slices"
)

type ConfidentialStoreEngine struct {
	backend     ConfidentialStoreBackend
	pubsub      PubSub
	daSigner    DASigner
	chainSigner ChainSigner
}

type DASigner interface {
	Sign(tx *types.Transaction, data []byte) ([]byte, error)
	Sender(data []byte, tx *types.Transaction) (common.Address, error)
}

type ChainSigner interface {
	Sender(tx *types.Transaction) (common.Address, error)
}

func NewConfidentialStoreEngine(backend ConfidentialStoreBackend, pubsub PubSub, daSigner DASigner, chainSigner ChainSigner) (*ConfidentialStoreEngine, error) {
	engine := &ConfidentialStoreEngine{
		backend:     backend,
		pubsub:      pubsub,
		daSigner:    daSigner,
		chainSigner: chainSigner,
	}

	return engine, nil
}

func (e *ConfidentialStoreEngine) Subscribe(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-e.pubsub.Subscribe():
			err := e.NewMessage(msg)
			if err != nil {
				log.Info("could not process new store message: %w", err)
			}
		}
	}
}

func ExecutionNodeFromTransaction(tx *types.Transaction) (common.Address, error) {
	innerExecutedTx, ok := types.CastTxInner[*types.OffchainExecutedTx](tx)
	if ok {
		return innerExecutedTx.ExecutionNode, nil
	}

	innerRequestTx, ok := types.CastTxInner[*types.OffchainTx](tx)
	if ok {
		return innerRequestTx.ExecutionNode, nil
	}

	return common.Address{}, fmt.Errorf("transaction is not of confidential type")
}

func (e *ConfidentialStoreEngine) InitializeBid(bid types.Bid, creationTx *types.Transaction) (types.Bid, error) {
	expectedId, err := calculateBidId(bid)
	if err != nil {
		return types.Bid{}, fmt.Errorf("confidential engine: could not initialize new bid: %w", err)
	}

	var emptyId common.Hash
	if bid.Id == emptyId {
		bid.Id = expectedId
	} else if bid.Id != expectedId {
		// True in some tests, might be time to rewrite them
		return types.Bid{}, errors.New("confidential engine:incorrect bid id passed")
	}

	daBid := Bid{
		Id:                  bid.Id,
		DecryptionCondition: bid.DecryptionCondition,
		AllowedPeekers:      bid.AllowedPeekers,
		AllowedStores:       bid.AllowedStores,
		Version:             bid.Version,
		CreationTx:          creationTx,
	}

	bidBytes, err := json.Marshal(daBid)
	if err != nil {
		return types.Bid{}, fmt.Errorf("confidential engine: could not marshal message for signing: %w", err)
	}

	daBid.Signature, err = e.daSigner.Sign(creationTx, bidBytes)
	if err != nil {
		return types.Bid{}, fmt.Errorf("confidential engine: could not sign initialized bid: %w", err)
	}

	err = e.backend.InitializeBid(daBid)
	if err != nil {
		return types.Bid{}, fmt.Errorf("confidential engine: store backend failed to initialize bid: %w", err)
	}

	return bid, nil
}

func (e *ConfidentialStoreEngine) Store(bidId BidId, sourceTx *types.Transaction, caller common.Address, key string, value []byte) (Bid, error) {
	bid, err := e.backend.FetchEngineBidById(bidId)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine could not fetch bid: %w", err)
	}
	msg := DAMessage{
		Bid:      bid,
		SourceTx: sourceTx,
		Caller:   caller,
		Key:      key,
		Value:    value,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine could not marshal message for signing: %w", err)
	}

	msg.Signature, err = e.daSigner.Sign(sourceTx, msgBytes)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine: could not sign message: %w", err)
	}

	// TODO: avoid marshalling twice
	e.pubsub.Publish(msg)

	return e.backend.Store(bidId, caller, key, value)
}

func (e *ConfidentialStoreEngine) Retrieve(bidId BidId, caller common.Address, key string) ([]byte, error) {
	return e.backend.Retrieve(bidId, caller, key)
}

func (e *ConfidentialStoreEngine) NewMessage(message DAMessage) error {
	// Note the validation is a work in progress and not guaranteed to be correct!

	expectedId, err := calculateBidId(types.Bid{
		Id:                  message.Bid.Id,
		AllowedPeekers:      message.Bid.AllowedPeekers,
		AllowedStores:       message.Bid.AllowedStores,
		DecryptionCondition: message.Bid.DecryptionCondition,
		Version:             message.Bid.Version,
	})

	if err != nil {
		return fmt.Errorf("confidential engine: could not calculate received bids id: %w", err)
	}

	if expectedId != message.Bid.Id {
		return fmt.Errorf("confidential engine: received bids id (%x) does not match the expected (%x)", message.Bid.Id, expectedId)
	}

	messageSigner, err := e.daSigner.Sender(message.Signature, message.SourceTx)
	if err != nil {
		return fmt.Errorf("confidential engine: incorrect message signature: %w", err)
	}

	_, err = e.daSigner.Sender(message.Bid.Signature, message.Bid.CreationTx)
	if err != nil {
		return fmt.Errorf("confidential engine: incorrect message signature: %w", err)
	}

	if !slices.Contains(message.Bid.AllowedStores, messageSigner) {
		return fmt.Errorf("confidential engine: message signer %x not allowed to store on bid %x", messageSigner, message.Bid.Id)
	}

	if !slices.Contains(message.Bid.AllowedPeekers, message.Caller) {
		return fmt.Errorf("confidential engine: message signer %x not allowed to store on bid %x", messageSigner, message.Bid.Id)
	}

	// TODO: move to types.Sender()
	_, err = e.chainSigner.Sender(message.Bid.CreationTx)
	if err != nil {
		return fmt.Errorf("confidential engine: creation tx for bid id %x is not signed properly: %w", message.Bid.Id, err)
	}

	_, err = e.chainSigner.Sender(message.SourceTx)
	if err != nil {
		return fmt.Errorf("confidential engine: source tx for message is not signed properly: %w", err)
	}

	err = e.backend.InitializeBid(message.Bid)
	if err != nil && !errors.Is(err, BidAlreadyPresentError) {
		panic(fmt.Errorf("unexpected error while initializing bid from transport: %w", err))
	}

	_, err = e.backend.Store(message.Bid.Id, message.Caller, message.Key, message.Value)
	if err != nil {
		panic(fmt.Errorf("unexpected error while storing, the message was not validated properly: %w (%v)", err, message.Caller))
	}

	return nil
}

type MockPubSub struct{}

func (MockPubSub) Subscribe() <-chan DAMessage { return nil }
func (MockPubSub) Publish(DAMessage)           {}

type MockSigner struct{}

func (MockSigner) Sign(tx *types.Transaction, data []byte) ([]byte, error) {
	if tx == nil {
		nilAddr := common.Address{}
		return nilAddr.Bytes(), nil
	}

	executionNodeAddr, err := ExecutionNodeFromTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("mock signer: could not get execution node from source transaction: %w", err)
	}

	return executionNodeAddr.Bytes(), nil
}

func (MockSigner) Sender(data []byte, tx *types.Transaction) (common.Address, error) {
	return common.BytesToAddress(data[len(data)-20:]), nil
}

type MockChainSigner struct{}

func (MockChainSigner) Sender(tx *types.Transaction) (common.Address, error) {
	if tx == nil {
		return common.Address{}, nil
	}

	return *tx.To(), nil
}
