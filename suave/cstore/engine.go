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
	InitRecord(record suave.DataRecord) error
	Store(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error)
	Retrieve(record suave.DataRecord, caller common.Address, key string) ([]byte, error)
	FetchRecordByID(suave.DataId) (suave.DataRecord, error)
	FetchRecordsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord
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
	DataRecord suave.DataRecord `json:"dataRecord"`
	Caller     common.Address   `json:"caller"`
	Key        string           `json:"key"`
	Value      suave.Bytes      `json:"value"`
}

type DASigner interface {
	Sign(account common.Address, data []byte) ([]byte, error)
	Sender(data []byte, signature []byte) (common.Address, error)
	LocalAddresses() []common.Address
}

type ChainSigner interface {
	Sender(tx *types.Transaction) (common.Address, error)
}

type CStoreEngine struct {
	ctx    context.Context
	cancel context.CancelFunc

	storage        ConfidentialStorageBackend
	transportTopic StoreTransportTopic

	daSigner    DASigner
	chainSigner ChainSigner

	storeUUID      uuid.UUID
	localAddresses map[common.Address]struct{}
}

// NewEngine creates a new instance of CStoreEngine.
func NewEngine(backend ConfidentialStorageBackend, transportTopic StoreTransportTopic, daSigner DASigner, chainSigner ChainSigner) *CStoreEngine {
	localAddresses := make(map[common.Address]struct{})
	for _, addr := range daSigner.LocalAddresses() {
		localAddresses[addr] = struct{}{}
	}

	return &CStoreEngine{
		storage:        backend,
		transportTopic: transportTopic,
		daSigner:       daSigner,
		chainSigner:    chainSigner,
		storeUUID:      uuid.New(),
		localAddresses: localAddresses,
	}
}

func (e *CStoreEngine) Reset() error {
	if local, ok := e.storage.(*LocalConfidentialStore); ok {
		// only allow reset for local store
		return local.Reset()
	}
	return nil
}

// NewTransactionalStore creates a new transactional store.
func (e *CStoreEngine) NewTransactionalStore(sourceTx *types.Transaction) *TransactionalStore {
	return &TransactionalStore{
		sourceTx:       sourceTx,
		engine:         e,
		pendingRecords: make(map[suave.DataId]suave.DataRecord),
	}
}

// Start initializes the CStoreEngine.
func (e *CStoreEngine) Start() error {
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

// Stop terminates the CStoreEngine.
func (e *CStoreEngine) Stop() error {
	if e.cancel == nil {
		return errors.New("confidential engine: Stop() called before Start()")
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

// Backend provides for testing purposes access to the underlying ConfidentialStorageBackend.
func (e *CStoreEngine) Backend() ConfidentialStorageBackend {
	return e.storage
}

// ProcessMessages handles incoming messages.
func (e *CStoreEngine) ProcessMessages() {
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

// InitRecord prepares a data record for storage.
func (e *CStoreEngine) InitRecord(record types.DataRecord, creationTx *types.Transaction) (suave.DataRecord, error) {
	// Share with all stores this node trusts
	record.AllowedStores = append(record.AllowedStores, e.daSigner.LocalAddresses()...)

	expectedId, err := calculateRecordId(record)
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("confidential engine: could not initialize new record: %w", err)
	}

	if isEmptyID(record.Id) {
		record.Id = expectedId
	} else if record.Id != expectedId {
		// True in some tests, might be time to rewrite them
		return suave.DataRecord{}, errors.New("confidential engine: incorrect record id passed")
	}

	initializedRecord := suave.DataRecord{
		Id:                  record.Id,
		Salt:                record.Salt,
		DecryptionCondition: record.DecryptionCondition,
		AllowedPeekers:      record.AllowedPeekers,
		AllowedStores:       record.AllowedStores,
		Version:             record.Version,
		CreationTx:          creationTx,
	}

	reocrdBytes, err := SerializeDataRecord(&initializedRecord)
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("confidential engine: could not hash record for signing: %w", err)
	}

	signingAccount, err := KettleAddressFromTransaction(creationTx)
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("confidential engine: could not recover execution node from creation transaction: %w", err)
	}

	initializedRecord.Signature, err = e.daSigner.Sign(signingAccount, reocrdBytes)
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("confidential engine: could not sign initialized record: %w", err)
	}

	return initializedRecord, nil
}

// FetchRecordByID retrieves a data record by its identifier.
func (e *CStoreEngine) FetchRecordByID(id suave.DataId) (suave.DataRecord, error) {
	return e.storage.FetchRecordByID(id)
}

// FetchRecordsByProtocolAndBlock fetches data records based on protocol and block number.
func (e *CStoreEngine) FetchRecordsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	return e.storage.FetchRecordsByProtocolAndBlock(blockNumber, namespace)
}

// Retrieve fetches data associated with a record.
func (e *CStoreEngine) Retrieve(id suave.DataId, caller common.Address, key string) ([]byte, error) {
	record, err := e.storage.FetchRecordByID(id)
	if err != nil {
		return []byte{}, fmt.Errorf("confidential engine: could not fetch record %x while retrieving: %w", id, err)
	}

	if !slices.Contains(record.AllowedPeekers, caller) && !slices.Contains(record.AllowedPeekers, suave.AllowedPeekerAny) {
		return []byte{}, fmt.Errorf("confidential engine: %x not allowed to retrieve %s on %x", caller, key, id)
	}

	return e.storage.Retrieve(record, caller, key)
}

// Finalize finalizes a transaction and updates the store.
func (e *CStoreEngine) Finalize(tx *types.Transaction, newRecords map[suave.DataId]suave.DataRecord, stores []StoreWrite) error {
	for _, record := range newRecords {
		err := e.storage.InitRecord(record)
		if err != nil {
			// TODO: deinitialize!
			return fmt.Errorf("confidential engine: store backend failed to initialize record: %w", err)
		}
	}

	for _, sw := range stores {
		if _, err := e.storage.Store(sw.DataRecord, sw.Caller, sw.Key, sw.Value); err != nil {
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

	msgBytes, err := SerializeDAMessage(&pwMsg)
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

// NewMessage processes a new incoming DAMessage.
func (e *CStoreEngine) NewMessage(message DAMessage) error {
	// Note the validation is a work in progress and not guaranteed to be correct!

	// Message-level validation
	msgBytes, err := SerializeDAMessage(&message)
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

	// DataRecord level validation

	for _, sw := range message.StoreWrites {
		expectedId, err := calculateRecordId(types.DataRecord{
			Id:                  sw.DataRecord.Id,
			Salt:                sw.DataRecord.Salt,
			DecryptionCondition: sw.DataRecord.DecryptionCondition,
			AllowedPeekers:      sw.DataRecord.AllowedPeekers,
			AllowedStores:       sw.DataRecord.AllowedStores,
			Version:             sw.DataRecord.Version,
		})
		if err != nil {
			return fmt.Errorf("confidential engine: could not calculate received records id: %w", err)
		}

		if expectedId != sw.DataRecord.Id {
			return fmt.Errorf("confidential engine: received records id (%x) does not match the expected (%x)", sw.DataRecord.Id, expectedId)
		}

		reocrdBytes, err := SerializeDataRecord(&sw.DataRecord)
		if err != nil {
			return fmt.Errorf("confidential engine: could not hash received record: %w", err)
		}
		recoveredRecordSigner, err := e.daSigner.Sender(reocrdBytes, sw.DataRecord.Signature)
		if err != nil {
			return fmt.Errorf("confidential engine: incorrect record signature: %w", err)
		}
		expectedRecordSigner, err := KettleAddressFromTransaction(sw.DataRecord.CreationTx)
		if err != nil {
			return fmt.Errorf("confidential engine: could not recover signer from record: %w", err)
		}
		if recoveredRecordSigner != expectedRecordSigner {
			return fmt.Errorf("confidential engine: record signer %x, expected %x", recoveredRecordSigner, expectedRecordSigner)
		}

		if !slices.Contains(sw.DataRecord.AllowedStores, recoveredMessageSigner) {
			return fmt.Errorf("confidential engine: sw signer %x not allowed to store on record %x", recoveredMessageSigner, sw.DataRecord.Id)
		}

		if !slices.Contains(sw.DataRecord.AllowedPeekers, sw.Caller) && !slices.Contains(sw.DataRecord.AllowedPeekers, suave.AllowedPeekerAny) {
			return fmt.Errorf("confidential engine: caller %x not allowed on record %x", sw.Caller, sw.DataRecord.Id)
		}

		// TODO: move to types.Sender()
		_, err = e.chainSigner.Sender(sw.DataRecord.CreationTx)
		if err != nil {
			return fmt.Errorf("confidential engine: creation tx for record id %x is not signed properly: %w", sw.DataRecord.Id, err)
		}
	}

	for _, sw := range message.StoreWrites {
		err = e.storage.InitRecord(sw.DataRecord)
		if err != nil {
			if !errors.Is(err, suave.ErrRecordAlreadyPresent) {
				log.Error("confidential engine: unexpected error while initializing record from transport: %w", err)
				continue // Don't abandon!
			}
		}

		_, err = e.storage.Store(sw.DataRecord, sw.Caller, sw.Key, sw.Value)
		if err != nil {
			log.Error("confidential engine: unexpected error while storing: %w", err)
			continue // Don't abandon!
		}
	}

	return nil
}

// SerializeDataRecord prepares a data record for signing.
func SerializeDataRecord(record *suave.DataRecord) ([]byte, error) {
	recordBytes, err := json.Marshal(suave.DataRecord{
		Id:                  record.Id,
		Salt:                record.Salt,
		DecryptionCondition: record.DecryptionCondition,
		AllowedPeekers:      record.AllowedPeekers,
		AllowedStores:       record.AllowedStores,
		Version:             record.Version,
		CreationTx:          record.CreationTx,
	})
	if err != nil {
		return []byte{}, err
	}

	return []byte(fmt.Sprintf("\x19Suave Signed Message:\n%d%s", len(recordBytes), string(recordBytes))), nil
}

// SerializeDAMessage prepares a DAMessage for signing.
func SerializeDAMessage(message *DAMessage) ([]byte, error) {
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

// KettleAddressFromTransaction returns address of kettle that executed confidential transaction
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

var recordUuidSpace = uuid.UUID{0x42}

func calculateRecordId(record types.DataRecord) (types.DataId, error) {
	copy(record.Id[:], emptyId[:])

	body, err := json.Marshal(record)
	if err != nil {
		return types.DataId{}, fmt.Errorf("could not marshal record to calculate its id: %w", err)
	}

	uuidv5 := uuid.NewSHA1(recordUuidSpace, body)
	copy(record.Id[:], uuidv5[:])

	return record.Id, nil
}

func RandomRecordId() types.DataId {
	return types.DataId(uuid.New())
}

// isEmptyID checks if the given DataId is empty.
func isEmptyID(id types.DataId) bool {
	return id == emptyId
}

// Mocks

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
