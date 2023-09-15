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
	ctx    context.Context
	cancel context.CancelFunc

	backend ConfidentialStoreBackend
	pubsub  PubSub
	mempool MempoolBackend

	daSigner    DASigner
	chainSigner ChainSigner
}

func (e *ConfidentialStoreEngine) Start() error {
	if err := e.backend.Start(); err != nil {
		return err
	}

	if err := e.pubsub.Start(); err != nil {
		return err
	}

	if err := e.mempool.Start(); err != nil {
		return err
	}

	if e.cancel != nil {
		e.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.ctx = ctx
	go e.Subscribe(ctx)

	return nil
}

func (e *ConfidentialStoreEngine) Stop() error {
	if e.cancel == nil {
		panic("Stop() called before Start()")
	}

	e.cancel()

	e.mempool.Stop()

	if err := e.pubsub.Stop(); err != nil {
		return err
	}

	if err := e.backend.Stop(); err != nil {
		// todo: wrap the error
		e.pubsub.Stop()
		return err
	}

	return nil
}

type DASigner interface {
	Sign(account common.Address, data []byte) ([]byte, error)
	Sender(data []byte, signature []byte) (common.Address, error)
}

type ChainSigner interface {
	Sender(tx *types.Transaction) (common.Address, error)
}

func NewConfidentialStoreEngine(backend ConfidentialStoreBackend, pubsub PubSub, mempool MempoolBackend, daSigner DASigner, chainSigner ChainSigner) (*ConfidentialStoreEngine, error) {
	engine := &ConfidentialStoreEngine{
		backend:     backend,
		pubsub:      pubsub,
		mempool:     mempool,
		daSigner:    daSigner,
		chainSigner: chainSigner,
	}

	return engine, nil
}

func (e *ConfidentialStoreEngine) Subscribe(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := e.pubsub.Subscribe(innerCtx)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				log.Error("Engine: pubsub channel closed, reopenning")
				cancel()
				innerCtx, cancel = context.WithCancel(ctx)
				defer cancel()

				ch = e.pubsub.Subscribe(innerCtx)
				continue
			}
			err := e.NewMessage(msg)
			if err != nil {
				log.Info("could not process new store message", "err", err)
			} else {
				log.Info("Message processed", "msg", msg)
			}
		}
	}
}

func ExecutionNodeFromTransaction(tx *types.Transaction) (common.Address, error) {
	innerExecutedTx, ok := types.CastTxInner[*types.SuaveTransaction](tx)
	if ok {
		return innerExecutedTx.ExecutionNode, nil
	}

	innerRequestTx, ok := types.CastTxInner[*types.ConfidentialComputeRequest](tx)
	if ok {
		return innerRequestTx.ExecutionNode, nil
	}

	return common.Address{}, fmt.Errorf("transaction is not of confidential type")
}

func (e *ConfidentialStoreEngine) InitializeBid(bid types.Bid, creationTx *types.Transaction) (types.Bid, error) {
	if bid.Salt == emptyId {
		bid.Salt = RandomBidId()
	}

	expectedId, err := calculateBidId(bid)
	if err != nil {
		return types.Bid{}, fmt.Errorf("confidential engine: could not initialize new bid: %w", err)
	}

	if bid.Id == emptyId {
		bid.Id = expectedId
	} else if bid.Id != expectedId {
		// True in some tests, might be time to rewrite them
		return types.Bid{}, errors.New("confidential engine:incorrect bid id passed")
	}

	daBid := Bid{
		Id:                  bid.Id,
		Salt:                bid.Salt,
		DecryptionCondition: bid.DecryptionCondition,
		AllowedPeekers:      bid.AllowedPeekers,
		AllowedStores:       bid.AllowedStores,
		Version:             bid.Version,
		CreationTx:          creationTx,
	}

	bidBytes, err := SerializeBidForSigning(daBid)
	if err != nil {
		return types.Bid{}, fmt.Errorf("confidential engine: could not hash bid for signing: %w", err)
	}

	signingAccount, err := ExecutionNodeFromTransaction(creationTx)
	if err != nil {
		return types.Bid{}, fmt.Errorf("confidential engine: could not recover execution node from creation transaction: %w", err)
	}

	daBid.Signature, err = e.daSigner.Sign(signingAccount, bidBytes)
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

	msgBytes, err := SerializeMessageForSigning(msg)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine: could not hash message for signing: %w", err)
	}

	signingAccount, err := ExecutionNodeFromTransaction(sourceTx)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine: could not recover execution node from source transaction: %w", err)
	}

	msg.Signature, err = e.daSigner.Sign(signingAccount, msgBytes)
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

	innerBid := types.Bid{
		Id:                  message.Bid.Id,
		Salt:                message.Bid.Salt,
		AllowedPeekers:      message.Bid.AllowedPeekers,
		AllowedStores:       message.Bid.AllowedStores,
		DecryptionCondition: message.Bid.DecryptionCondition,
		Version:             message.Bid.Version,
	}

	expectedId, err := calculateBidId(innerBid)

	if err != nil {
		return fmt.Errorf("confidential engine: could not calculate received bids id: %w", err)
	}

	if expectedId != message.Bid.Id {
		return fmt.Errorf("confidential engine: received bids id (%x) does not match the expected (%x)", message.Bid.Id, expectedId)
	}

	msgBytes, err := SerializeMessageForSigning(message)
	if err != nil {
		return fmt.Errorf("confidential engine: could not hash received message: %w", err)
	}
	recoveredMessageSigner, err := e.daSigner.Sender(msgBytes, message.Signature)
	if err != nil {
		return fmt.Errorf("confidential engine: incorrect message signature: %w", err)
	}
	expectedMessageSigner, err := ExecutionNodeFromTransaction(message.SourceTx)
	if err != nil {
		return fmt.Errorf("confidential engine: could not recover signer from message: %w", err)
	}
	if recoveredMessageSigner != expectedMessageSigner {
		return fmt.Errorf("confidential engine: message signer %x, expected %x", recoveredMessageSigner, expectedMessageSigner)
	}

	bidBytes, err := SerializeBidForSigning(message.Bid)
	if err != nil {
		return fmt.Errorf("confidential engine: could not hash received bid: %w", err)
	}
	recoveredBidSigner, err := e.daSigner.Sender(bidBytes, message.Bid.Signature)
	if err != nil {
		return fmt.Errorf("confidential engine: incorrect bid signature: %w", err)
	}
	expectedBidSigner, err := ExecutionNodeFromTransaction(message.Bid.CreationTx)
	if err != nil {
		return fmt.Errorf("confidential engine: could not recover signer from bid: %w", err)
	}
	if recoveredBidSigner != expectedBidSigner {
		return fmt.Errorf("confidential engine: bid signer %x, expected %x", recoveredBidSigner, expectedBidSigner)
	}

	if !slices.Contains(message.Bid.AllowedStores, recoveredMessageSigner) {
		return fmt.Errorf("confidential engine: message signer %x not allowed to store on bid %x", recoveredMessageSigner, message.Bid.Id)
	}

	if !slices.Contains(message.Bid.AllowedPeekers, message.Caller) {
		return fmt.Errorf("confidential engine: caller %x not allowed on bid %x", message.Caller, message.Bid.Id)
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
	if err != nil {
		if !errors.Is(err, ErrBidAlreadyPresent) {
			return fmt.Errorf("unexpected error while initializing bid from transport: %w", err)
		}
	} else {
		e.mempool.SubmitBid(innerBid)
	}

	_, err = e.backend.Store(message.Bid.Id, message.Caller, message.Key, message.Value)
	if err != nil {
		panic(fmt.Errorf("unexpected error while storing, the message was not validated properly: %w (%v)", err, message.Caller))
	}

	return nil
}

func SerializeBidForSigning(bid Bid) ([]byte, error) {
	bid.Signature = []byte{}

	bidBytes, err := json.Marshal(bid)
	if err != nil {
		return []byte{}, err
	}

	return []byte(fmt.Sprintf("\x19Suave Signed Message:\n%d%s", len(bidBytes), string(bidBytes))), nil
}

func SerializeMessageForSigning(message DAMessage) ([]byte, error) {
	message.Signature = []byte{}

	msgBytes, err := json.Marshal(message)
	if err != nil {
		return []byte{}, err
	}

	return []byte(fmt.Sprintf("\x19Suave Signed Message:\n%d%s", len(msgBytes), string(msgBytes))), nil
}

type MockPubSub struct{}

func (MockPubSub) Start() error { return nil }
func (MockPubSub) Stop() error  { return nil }

func (MockPubSub) Subscribe(context.Context) <-chan DAMessage { return nil }
func (MockPubSub) Publish(DAMessage)                          {}

type MockSigner struct{}

func (MockSigner) Sign(account common.Address, data []byte) ([]byte, error) {
	return account.Bytes(), nil
}

func (MockSigner) Sender(data []byte, signature []byte) (common.Address, error) {
	return common.BytesToAddress(signature), nil
}

type MockChainSigner struct{}

func (MockChainSigner) Sender(tx *types.Transaction) (common.Address, error) {
	if tx == nil {
		return common.Address{}, nil
	}

	return *tx.To(), nil
}

type MockMempool struct{}

func (MockMempool) Start() error { return nil }
func (MockMempool) Stop() error  { return nil }

func (MockMempool) SubmitBid(types.Bid) error { return nil }

func (MockMempool) FetchBidById(BidId) (types.Bid, error) { return types.Bid{}, nil }
func (MockMempool) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []types.Bid {
	return nil
}
