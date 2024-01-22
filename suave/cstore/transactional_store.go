package cstore

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"golang.org/x/exp/slices"
)

type TransactionalStore struct {
	sourceTx *types.Transaction
	engine   *CStoreEngine

	pendingLock    sync.Mutex
	pendingRecords map[suave.DataId]suave.DataRecord
	pendingWrites  []StoreWrite
}

// FetchRecordByID retrieves a data record by its identifier.
func (s *TransactionalStore) FetchRecordByID(dataId suave.DataId) (suave.DataRecord, error) {
	s.pendingLock.Lock()
	record, ok := s.pendingRecords[dataId]
	s.pendingLock.Unlock()

	if ok {
		return record, nil
	}

	return s.engine.FetchRecordByID(dataId)
}

func (s *TransactionalStore) FetchRecordsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	records := s.engine.FetchRecordsByProtocolAndBlock(blockNumber, namespace)

	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	for _, record := range s.pendingRecords {
		if record.Version == namespace && record.DecryptionCondition == blockNumber {
			records = append(records, record)
		}
	}

	return records
}

func (s *TransactionalStore) Store(dataId suave.DataId, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
	record, err := s.FetchRecordByID(dataId)
	if err != nil {
		return suave.DataRecord{}, err
	}

	if !slices.Contains(record.AllowedPeekers, caller) && !slices.Contains(record.AllowedPeekers, suave.AllowedPeekerAny) {
		return suave.DataRecord{}, fmt.Errorf("confidential store transaction: %x not allowed to store %s on %x", caller, key, dataId)
	}

	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	s.pendingWrites = append(s.pendingWrites, StoreWrite{
		DataRecord: record,
		Caller:     caller,
		Key:        key,
		Value:      common.CopyBytes(value),
	})

	return record, nil
}

// Retrieve fetches data associated with a record.
func (s *TransactionalStore) Retrieve(dataId suave.DataId, caller common.Address, key string) ([]byte, error) {
	record, err := s.FetchRecordByID(dataId)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(record.AllowedPeekers, caller) && !slices.Contains(record.AllowedPeekers, suave.AllowedPeekerAny) {
		return nil, fmt.Errorf("confidential store transaction: %x not allowed to retrieve %s on %x", caller, key, dataId)
	}

	s.pendingLock.Lock()

	for _, sw := range s.pendingWrites {
		if sw.DataRecord.Id == record.Id && sw.Key == key {
			s.pendingLock.Unlock()
			return common.CopyBytes(sw.Value), nil
		}
	}

	s.pendingLock.Unlock()
	return s.engine.Retrieve(dataId, caller, key)
}

// InitRecord prepares a data record for storage.
func (s *TransactionalStore) InitRecord(rawRecord types.DataRecord) (types.DataRecord, error) {
	if s.sourceTx == nil {
		return types.DataRecord{}, errors.New("confidential store transaction: no source transaction")
	}

	record, err := s.engine.InitRecord(rawRecord, s.sourceTx)
	if err != nil {
		return types.DataRecord{}, err
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

func (s *TransactionalStore) Finalize() error {
	return s.engine.Finalize(s.sourceTx, s.pendingRecords, s.pendingWrites)
}
