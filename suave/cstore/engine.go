package cstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

// ConfidentialStorageBackend is the interface that must be implemented by a
// storage backend for the confidential storage engine.
type ConfidentialStorageBackend interface {
	InitializeBid(bid suave.Bid) error
	Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error)
	Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error)
	FetchBidById(suave.BidId) (suave.Bid, error)
	FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid
	Stop() error
}

// StoreTransportTopic is the interface that must be implemented by a
// transport engine for the confidential storage engine.
type StoreTransportTopic interface {
	node.Lifecycle
	Subscribe() (<-chan DAMessage, context.CancelFunc)
	Publish(DAMessage)
}

type DAMessage struct {
	SourceTx    *types.Transaction `json:"sourceTx"`
	StoreWrites []StoreWrite       `json:"storeWrites"`
	StoreUUID   uuid.UUID          `json:"storeUUID"`
	Signature   suave.Bytes        `json:"signature"`
}

type StoreWrite struct {
	Bid    suave.Bid      `json:"bid"`
	Caller common.Address `json:"caller"`
	Key    string         `json:"key"`
	Value  suave.Bytes    `json:"value"`
}

type DASigner interface {
	Sign(account common.Address, data []byte) ([]byte, error)
	Sender(data []byte, signature []byte) (common.Address, error)
	LocalAddresses() []common.Address
}

type ChainSigner interface {
	Sender(tx *types.Transaction) (common.Address, error)
}

type ConfidentialStoreEngine struct {
	ctx    context.Context
	cancel context.CancelFunc

	storage        ConfidentialStorageBackend
	transportTopic StoreTransportTopic

	daSigner    DASigner
	chainSigner ChainSigner

	storeUUID      uuid.UUID
	localAddresses map[common.Address]struct{}
}

func NewConfidentialStoreEngine(backend ConfidentialStorageBackend, transportTopic StoreTransportTopic, daSigner DASigner, chainSigner ChainSigner) *ConfidentialStoreEngine {
	localAddresses := make(map[common.Address]struct{})
	for _, addr := range daSigner.LocalAddresses() {
		localAddresses[addr] = struct{}{}
	}

	return &ConfidentialStoreEngine{
		storage:        backend,
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
		pendingBids: make(map[suave.BidId]suave.Bid),
	}
}

func (e *ConfidentialStoreEngine) Start() error {
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

	if err := e.storage.Stop(); err != nil {
		log.Warn("Confidential engine: error while stopping transport", "err", err)
	}

	return nil
}

// For testing purposes!
func (e *ConfidentialStoreEngine) Backend() ConfidentialStorageBackend {
	return e.storage
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

func (e *ConfidentialStoreEngine) InitializeBid(bid types.Bid, creationTx *types.Transaction) (suave.Bid, error) {
	// Share with all stores this node trusts
	bid.AllowedStores = append(bid.AllowedStores, e.daSigner.LocalAddresses()...)

	expectedId, err := calculateBidId(bid)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("confidential engine: could not initialize new bid: %w", err)
	}

	if bid.Id == emptyId {
		bid.Id = expectedId
	} else if bid.Id != expectedId {
		// True in some tests, might be time to rewrite them
		return suave.Bid{}, errors.New("confidential engine:incorrect bid id passed")
	}

	initializedBid := suave.Bid{
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
		return suave.Bid{}, fmt.Errorf("confidential engine: could not hash bid for signing: %w", err)
	}

	signingAccount, err := KettleAddressFromTransaction(creationTx)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("confidential engine: could not recover execution node from creation transaction: %w", err)
	}

	initializedBid.Signature, err = e.daSigner.Sign(signingAccount, bidBytes)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("confidential engine: could not sign initialized bid: %w", err)
	}

	return initializedBid, nil
}

func (e *ConfidentialStoreEngine) FetchBidById(bidId suave.BidId) (suave.Bid, error) {
	return e.storage.FetchBidById(bidId)
}

func (e *ConfidentialStoreEngine) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	return e.storage.FetchBidsByProtocolAndBlock(blockNumber, namespace)
}

func (e *ConfidentialStoreEngine) Retrieve(bidId suave.BidId, caller common.Address, key string) ([]byte, error) {
	bid, err := e.storage.FetchBidById(bidId)
	if err != nil {
		return []byte{}, fmt.Errorf("confidential engine: could not fetch bid %x while retrieving: %w", bidId, err)
	}

	if !slices.Contains(bid.AllowedPeekers, caller) && !slices.Contains(bid.AllowedPeekers, suave.AllowedPeekerAny) {
		return []byte{}, fmt.Errorf("confidential engine: %x not allowed to retrieve %s on %x", caller, key, bidId)
	}

	return e.storage.Retrieve(bid, caller, key)
}

func (e *ConfidentialStoreEngine) Finalize(tx *types.Transaction, newBids map[suave.BidId]suave.Bid, stores []StoreWrite) error {
	//
	for _, bid := range newBids {
		err := e.storage.InitializeBid(bid)
		if err != nil {
			// TODO: deinitialize!
			return fmt.Errorf("confidential engine: store backend failed to initialize bid: %w", err)
		}
	}

	for _, sw := range stores {
		if _, err := e.storage.Store(sw.Bid, sw.Caller, sw.Key, sw.Value); err != nil {
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
		return suave.ErrUnsignedFinalize
	}

	msgBytes, err := SerializeMessageForSigning(&pwMsg)
	if err != nil {
		return fmt.Errorf("confidential engine: could not hash message for signing: %w", err)
	}

	signingAccount, err := KettleAddressFromTransaction(tx)
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
	expectedMessageSigner, err := KettleAddressFromTransaction(message.SourceTx)
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
		expectedBidSigner, err := KettleAddressFromTransaction(sw.Bid.CreationTx)
		if err != nil {
			return fmt.Errorf("confidential engine: could not recover signer from bid: %w", err)
		}
		if recoveredBidSigner != expectedBidSigner {
			return fmt.Errorf("confidential engine: bid signer %x, expected %x", recoveredBidSigner, expectedBidSigner)
		}

		if !slices.Contains(sw.Bid.AllowedStores, recoveredMessageSigner) {
			return fmt.Errorf("confidential engine: sw signer %x not allowed to store on bid %x", recoveredMessageSigner, sw.Bid.Id)
		}

		if !slices.Contains(sw.Bid.AllowedPeekers, sw.Caller) && !slices.Contains(sw.Bid.AllowedPeekers, suave.AllowedPeekerAny) {
			return fmt.Errorf("confidential engine: caller %x not allowed on bid %x", sw.Caller, sw.Bid.Id)
		}

		// TODO: move to types.Sender()
		_, err = e.chainSigner.Sender(sw.Bid.CreationTx)
		if err != nil {
			return fmt.Errorf("confidential engine: creation tx for bid id %x is not signed properly: %w", sw.Bid.Id, err)
		}
	}

	for _, sw := range message.StoreWrites {
		err = e.storage.InitializeBid(sw.Bid)
		if err != nil {
			if !errors.Is(err, suave.ErrBidAlreadyPresent) {
				log.Error("confidential engine: unexpected error while initializing bid from transport: %w", err)
				continue // Don't abandon!
			}
		}

		_, err = e.storage.Store(sw.Bid, sw.Caller, sw.Key, sw.Value)
		if err != nil {
			log.Error("confidential engine: unexpected error while storing: %w", err)
			continue // Don't abandon!
		}
	}

	return nil
}

func SerializeBidForSigning(bid *suave.Bid) ([]byte, error) {
	bidBytes, err := json.Marshal(suave.Bid{
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

func KettleAddressFromTransaction(tx *types.Transaction) (common.Address, error) {
	innerExecutedTx, ok := types.CastTxInner[*types.SuaveTransaction](tx)
	if ok {
		return innerExecutedTx.ConfidentialComputeRequest.KettleAddress, nil
	}

	innerRequestTx, ok := types.CastTxInner[*types.ConfidentialComputeRequest](tx)
	if ok {
		return innerRequestTx.KettleAddress, nil
	}

	return common.Address{}, fmt.Errorf("transaction is not of confidential type")
}

var emptyId [16]byte

var bidUuidSpace = uuid.UUID{0x42}

func calculateBidId(bid types.Bid) (types.BidId, error) {
	copy(bid.Id[:], emptyId[:])

	body, err := json.Marshal(bid)
	if err != nil {
		return types.BidId{}, fmt.Errorf("could not marshal bid to calculate its id: %w", err)
	}

	uuidv5 := uuid.NewSHA1(bidUuidSpace, body)
	copy(bid.Id[:], uuidv5[:])

	return bid.Id, nil
}

func RandomBidId() types.BidId {
	return types.BidId(uuid.New())
}
