package cstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

type ConfidentialStoreEngine struct {
	ctx    context.Context
	cancel context.CancelFunc

	backend        ConfidentialStoreBackend
	transportTopic StoreTransportTopic

	daSigner    DASigner
	chainSigner ChainSigner

	storeUUID      uuid.UUID
	localAddresses map[common.Address]struct{}
}

type Config struct {
}

func NewConfidentialStoreEngineWithConfig(config *Config) {
	// TODO, config pattern?
}

func NewConfidentialStoreEngine(backend ConfidentialStoreBackend, transportTopic StoreTransportTopic, daSigner DASigner, chainSigner ChainSigner) *ConfidentialStoreEngine {
	localAddresses := make(map[common.Address]struct{})
	for _, addr := range daSigner.LocalAddresses() {
		localAddresses[addr] = struct{}{}
	}

	return &ConfidentialStoreEngine{
		backend:        backend,
		transportTopic: transportTopic,
		daSigner:       daSigner,
		chainSigner:    chainSigner,
		storeUUID:      uuid.New(),
		localAddresses: localAddresses,
	}
}

func (e *ConfidentialStoreEngine) NewTransactionalStore(sourceTx *types.Transaction) *TransactionalStore {
	return &TransactionalStore{
		sourceTx:    sourceTx,
		engine:      e,
		pendingBids: make(map[BidId]Bid),
	}
}

func (e *ConfidentialStoreEngine) Start() error {
	if err := e.backend.Start(); err != nil {
		return err
	}

	if err := e.transportTopic.Start(); err != nil {
		return err
	}

	if e.cancel != nil {
		e.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.ctx = ctx
	go e.ProcessMessages()

	return nil
}

func (e *ConfidentialStoreEngine) Stop() error {
	if e.cancel == nil {
		return errors.New("Confidential engine: Stop() called before Start()")
	}

	e.cancel()

	if err := e.transportTopic.Stop(); err != nil {
		log.Warn("Confidential engine: error while stopping transport", "err", err)
	}

	if err := e.backend.Stop(); err != nil {
		log.Warn("Confidential engine: error while stopping transport", "err", err)
	}

	return nil
}

// For testing purposes!
func (e *ConfidentialStoreEngine) Backend() ConfidentialStoreBackend {
	return e.backend
}

func (e *ConfidentialStoreEngine) ProcessMessages() {
	ch, cancel := e.transportTopic.Subscribe()
	defer cancel()

	for {
		select {
		case <-e.ctx.Done(): // Stop() called
			return
		case msg := <-ch:
			err := e.NewMessage(msg)
			if err != nil {
				log.Info("could not process new store message", "err", err)
			} else {
				log.Info("Message processed", "msg", msg)
			}
		}
	}
}

func (e *ConfidentialStoreEngine) InitializeBid(bid types.Bid, creationTx *types.Transaction) (Bid, error) {
	expectedId, err := calculateBidId(bid)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine: could not initialize new bid: %w", err)
	}

	if bid.Id == emptyId {
		bid.Id = expectedId
	} else if bid.Id != expectedId {
		// True in some tests, might be time to rewrite them
		return Bid{}, errors.New("confidential engine:incorrect bid id passed")
	}

	initializedBid := Bid{
		Id:                  bid.Id,
		Salt:                bid.Salt,
		DecryptionCondition: bid.DecryptionCondition,
		AllowedPeekers:      bid.AllowedPeekers,
		AllowedStores:       bid.AllowedStores,
		Version:             bid.Version,
		CreationTx:          creationTx,
	}

	bidBytes, err := SerializeBidForSigning(&initializedBid)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine: could not hash bid for signing: %w", err)
	}

	signingAccount, err := ExecutionNodeFromTransaction(creationTx)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine: could not recover execution node from creation transaction: %w", err)
	}

	initializedBid.Signature, err = e.daSigner.Sign(signingAccount, bidBytes)
	if err != nil {
		return Bid{}, fmt.Errorf("confidential engine: could not sign initialized bid: %w", err)
	}

	return initializedBid, nil
}

func (e *ConfidentialStoreEngine) FetchBidById(bidId BidId) (Bid, error) {
	return e.backend.FetchBidById(bidId)
}

func (e *ConfidentialStoreEngine) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []Bid {
	return e.backend.FetchBidsByProtocolAndBlock(blockNumber, namespace)
}

func (e *ConfidentialStoreEngine) Retrieve(bidId BidId, caller common.Address, key string) ([]byte, error) {
	bid, err := e.backend.FetchBidById(bidId)
	if err != nil {
		return []byte{}, fmt.Errorf("confidential engine: could not fetch bid %x while retrieving: %w", bidId, err)
	}

	if !slices.Contains(bid.AllowedPeekers, caller) {
		return []byte{}, fmt.Errorf("confidential engine: %x not allowed to retrieve %s on %x", caller, key, bidId)
	}

	return e.backend.Retrieve(bid, caller, key)
}

func (e *ConfidentialStoreEngine) Finalize(tx *types.Transaction, newBids map[BidId]Bid, stores []StoreWrite) error {
	//
	for _, bid := range newBids {
		err := e.backend.InitializeBid(bid)
		if err != nil {
			// TODO: deinitialize!
			return fmt.Errorf("confidential engine: store backend failed to initialize bid: %w", err)
		}
	}

	for _, sw := range stores {
		if _, err := e.backend.Store(sw.Bid, sw.Caller, sw.Key, sw.Value); err != nil {
			// TODO: deinitialize and deStore!
			return fmt.Errorf("failed to store data: %w", err)
		}
	}

	// Sign and propagate the message
	pwMsg := DAMessage{
		SourceTx:    tx,
		StoreWrites: stores,
		StoreUUID:   e.storeUUID,
	}

	if _, sigErr := e.chainSigner.Sender(tx); sigErr != nil {
		log.Info("confidential engine: refusing to send writes based on unsigned transaction", "hash", tx.Hash().Hex(), "err", sigErr)
		return ErrUnsignedFinalize
	}

	msgBytes, err := SerializeMessageForSigning(&pwMsg)
	if err != nil {
		return fmt.Errorf("confidential engine: could not hash message for signing: %w", err)
	}

	signingAccount, err := ExecutionNodeFromTransaction(tx)
	if err != nil {
		return fmt.Errorf("confidential engine: could not recover execution node from source transaction: %w", err)
	}

	pwMsg.Signature, err = e.daSigner.Sign(signingAccount, msgBytes)
	if err != nil {
		return fmt.Errorf("confidential engine: could not sign message: %w", err)
	}

	// TODO: avoid marshalling twice
	go e.transportTopic.Publish(pwMsg)

	return nil
}

func (e *ConfidentialStoreEngine) NewMessage(message DAMessage) error {
	// Note the validation is a work in progress and not guaranteed to be correct!

	// Message-level validation
	msgBytes, err := SerializeMessageForSigning(&message)
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

	if message.StoreUUID == e.storeUUID {
		if _, found := e.localAddresses[recoveredMessageSigner]; found {
			return nil
		}
		// Message from self!
		log.Info("Confidential engine: message is spoofing our storeUUID, processing anyway", "message", message)
	}

	_, err = e.chainSigner.Sender(message.SourceTx)
	if err != nil {
		return fmt.Errorf("confidential engine: source tx for message is not signed properly: %w", err)
	}

	// TODO: check if message.SourceTx is valid and insert it into the mempool!

	// Bid level validation

	for _, sw := range message.StoreWrites {
		expectedId, err := calculateBidId(types.Bid{
			Id:                  sw.Bid.Id,
			Salt:                sw.Bid.Salt,
			DecryptionCondition: sw.Bid.DecryptionCondition,
			AllowedPeekers:      sw.Bid.AllowedPeekers,
			AllowedStores:       sw.Bid.AllowedStores,
			Version:             sw.Bid.Version,
		})
		if err != nil {
			return fmt.Errorf("confidential engine: could not calculate received bids id: %w", err)
		}

		if expectedId != sw.Bid.Id {
			return fmt.Errorf("confidential engine: received bids id (%x) does not match the expected (%x)", sw.Bid.Id, expectedId)
		}

		bidBytes, err := SerializeBidForSigning(&sw.Bid)
		if err != nil {
			return fmt.Errorf("confidential engine: could not hash received bid: %w", err)
		}
		recoveredBidSigner, err := e.daSigner.Sender(bidBytes, sw.Bid.Signature)
		if err != nil {
			return fmt.Errorf("confidential engine: incorrect bid signature: %w", err)
		}
		expectedBidSigner, err := ExecutionNodeFromTransaction(sw.Bid.CreationTx)
		if err != nil {
			return fmt.Errorf("confidential engine: could not recover signer from bid: %w", err)
		}
		if recoveredBidSigner != expectedBidSigner {
			return fmt.Errorf("confidential engine: bid signer %x, expected %x", recoveredBidSigner, expectedBidSigner)
		}

		if !slices.Contains(sw.Bid.AllowedStores, recoveredMessageSigner) {
			return fmt.Errorf("confidential engine: sw signer %x not allowed to store on bid %x", recoveredMessageSigner, sw.Bid.Id)
		}

		if !slices.Contains(sw.Bid.AllowedPeekers, sw.Caller) {
			return fmt.Errorf("confidential engine: caller %x not allowed on bid %x", sw.Caller, sw.Bid.Id)
		}

		// TODO: move to types.Sender()
		_, err = e.chainSigner.Sender(sw.Bid.CreationTx)
		if err != nil {
			return fmt.Errorf("confidential engine: creation tx for bid id %x is not signed properly: %w", sw.Bid.Id, err)
		}
	}

	for _, sw := range message.StoreWrites {
		err = e.backend.InitializeBid(sw.Bid)
		if err != nil {
			if !errors.Is(err, ErrBidAlreadyPresent) {
				log.Error("confidential engine: unexpected error while initializing bid from transport: %w", err)
				continue // Don't abandon!
			}
		}

		_, err = e.backend.Store(sw.Bid, sw.Caller, sw.Key, sw.Value)
		if err != nil {
			log.Error("confidential engine: unexpected error while storing: %w", err)
			continue // Don't abandon!
		}
	}

	return nil
}

func SerializeBidForSigning(bid *Bid) ([]byte, error) {
	bidBytes, err := json.Marshal(Bid{
		Id:                  bid.Id,
		Salt:                bid.Salt,
		DecryptionCondition: bid.DecryptionCondition,
		AllowedPeekers:      bid.AllowedPeekers,
		AllowedStores:       bid.AllowedStores,
		Version:             bid.Version,
		CreationTx:          bid.CreationTx,
	})
	if err != nil {
		return []byte{}, err
	}

	return []byte(fmt.Sprintf("\x19Suave Signed Message:\n%d%s", len(bidBytes), string(bidBytes))), nil
}

func SerializeMessageForSigning(message *DAMessage) ([]byte, error) {
	msgBytes, err := json.Marshal(DAMessage{
		SourceTx:    message.SourceTx,
		StoreWrites: message.StoreWrites,
		StoreUUID:   message.StoreUUID,
		Signature:   nil,
	})
	if err != nil {
		return []byte{}, err
	}

	return []byte(fmt.Sprintf("\x19Suave Signed Message:\n%d%s", len(msgBytes), string(msgBytes))), nil
}

type MockTransport struct{}

func (MockTransport) Start() error { return nil }
func (MockTransport) Stop() error  { return nil }

func (MockTransport) Subscribe() (<-chan DAMessage, context.CancelFunc) {
	return nil, func() {}
}
func (MockTransport) Publish(DAMessage) {}

type MockSigner struct{}

func (MockSigner) Sign(account common.Address, data []byte) ([]byte, error) {
	return account.Bytes(), nil
}

func (MockSigner) Sender(data []byte, signature []byte) (common.Address, error) {
	return common.BytesToAddress(signature), nil
}

func (MockSigner) LocalAddresses() []common.Address {
	return []common.Address{}
}

type MockChainSigner struct{}

func (MockChainSigner) Sender(tx *types.Transaction) (common.Address, error) {
	if tx == nil {
		return common.Address{}, nil
	}

	return types.NewSuaveSigner(tx.ChainId()).Sender(tx)
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
