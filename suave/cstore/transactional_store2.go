package cstore

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

// TransactionalStore2 introduces a transactional layer on top of a ConfidentialStorageBackend
// Unlike TransactionalStore, it does not require a transaction to be started, and it does
// not do any caller authentication for the data.
type TransactionalStore2 struct {
	storage ConfidentialStorageBackend

	pendingLock    sync.Mutex
	pendingRecords map[suave.DataId]suave.DataRecord
	pendingWrites  []StoreWrite
}

func NewTransactionalStore2(storage ConfidentialStorageBackend) *TransactionalStore2 {
	return &TransactionalStore2{
		storage:        storage,
		pendingRecords: make(map[suave.DataId]suave.DataRecord),
	}
}

// FetchRecordByID retrieves a data record by its identifier.
func (s *TransactionalStore2) FetchRecordByID(dataId suave.DataId) (suave.DataRecord, error) {
	s.pendingLock.Lock()
	record, ok := s.pendingRecords[dataId]
	s.pendingLock.Unlock()
	if ok {
		return record, nil
	}

	return s.storage.FetchRecordByID(dataId)
}

func (s *TransactionalStore2) FetchRecordsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	records := s.storage.FetchRecordsByProtocolAndBlock(blockNumber, namespace)

	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	for _, record := range s.pendingRecords {
		if record.Version == namespace && record.DecryptionCondition == blockNumber {
			records = append(records, record)
		}
	}

	return records
}

func (s *TransactionalStore2) Store(dataId suave.DataId, key string, value []byte) (suave.DataRecord, error) {
	record, err := s.FetchRecordByID(dataId)
	if err != nil {
		return suave.DataRecord{}, err
	}

	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	s.pendingWrites = append(s.pendingWrites, StoreWrite{
		DataRecord: record,
		Caller:     common.Address{},
		Key:        key,
		Value:      common.CopyBytes(value),
	})

	return record, nil
}

// Retrieve fetches data associated with a record.
func (s *TransactionalStore2) Retrieve(dataId suave.DataId, key string) ([]byte, error) {
	record, err := s.FetchRecordByID(dataId)
	if err != nil {
		return nil, err
	}

	s.pendingLock.Lock()

	for _, sw := range s.pendingWrites {
		if sw.DataRecord.Id == record.Id && sw.Key == key {
			s.pendingLock.Unlock()
			return common.CopyBytes(sw.Value), nil
		}
	}

	s.pendingLock.Unlock()
	return s.storage.Retrieve(suave.DataRecord{Id: dataId}, common.Address{}, key)
}

// InitRecord prepares a data record for storage.
func (s *TransactionalStore2) InitRecord(rawRecord types.DataRecord) (types.DataRecord, error) {
	expectedId, err := calculateRecordId(rawRecord)
	if err != nil {
		return types.DataRecord{}, fmt.Errorf("confidential engine: could not initialize new record: %w", err)
	}

	if isEmptyID(rawRecord.Id) {
		rawRecord.Id = expectedId
	} else if rawRecord.Id != expectedId {
		// True in some tests, might be time to rewrite them
		return types.DataRecord{}, errors.New("confidential engine: incorrect record id passed")
	}

	record := suave.DataRecord{
		Id:                  rawRecord.Id,
		Salt:                rawRecord.Salt,
		DecryptionCondition: rawRecord.DecryptionCondition,
		AllowedPeekers:      rawRecord.AllowedPeekers,
		AllowedStores:       rawRecord.AllowedStores,
		Version:             rawRecord.Version,
	}

	s.pendingLock.Lock()
	_, found := s.pendingRecords[record.Id]
	if found {
		s.pendingLock.Unlock()
		return types.DataRecord{}, errors.New("record with this id already exists")
	}
	s.pendingRecords[record.Id] = record
	s.pendingLock.Unlock()

	return record.ToInnerRecord(), nil
}
