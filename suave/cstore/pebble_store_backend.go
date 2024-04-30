package cstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type PebbleStoreBackend struct {
	ctx    context.Context
	cancel context.CancelFunc
	dbPath string
	db     *pebble.DB
}

var recordByBlockAndProtocolIndexDbKey = func(blockNumber uint64, namespace string) []byte {
	return []byte(fmt.Sprintf("records-block-%d-ns-%s", blockNumber, namespace))
}

type recordByBlockAndProtocolIndexType = []types.DataId

func NewPebbleStoreBackend(dbPath string) (*PebbleStoreBackend, error) {
	// TODO: should we check sanity in the constructor?
	backend := &PebbleStoreBackend{
		dbPath: dbPath,
	}

	return backend, backend.start()
}

func (b *PebbleStoreBackend) start() error {
	if b.cancel != nil {
		b.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	b.ctx = ctx

	db, err := pebble.Open(b.dbPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("could not open pebble database at %s: %w", b.dbPath, err)
	}

	go func() {
		<-ctx.Done()
		db.Close()
	}()

	b.db = db

	return nil
}

func (b *PebbleStoreBackend) Stop() error {
	b.cancel()
	return nil
}

// InitRecord prepares a data record for storage.
func (b *PebbleStoreBackend) InitRecord(record suave.DataRecord) error {
	key := []byte(formatRecordKey(record.Id))

	_, closer, err := b.db.Get(key)
	if !errors.Is(err, pebble.ErrNotFound) {
		if err == nil {
			closer.Close()
		}
		return nil
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = b.db.Set(key, data, nil)
	if err != nil {
		return err
	}

	// index update
	var currentValues recordByBlockAndProtocolIndexType

	dbBlockProtoIndexKey := recordByBlockAndProtocolIndexDbKey(record.DecryptionCondition, record.Version)
	rawCurrentValues, closer, err := b.db.Get(dbBlockProtoIndexKey)
	if err != nil {
		if !errors.Is(err, pebble.ErrNotFound) {
			return err
		}
	} else if err == nil {
		err = json.Unmarshal(rawCurrentValues, &currentValues)
		closer.Close()
		if err != nil {
			return err
		}
	}

	currentValues = append(currentValues, record.Id)
	rawUpdatedValues, err := json.Marshal(currentValues)
	if err != nil {
		return err
	}

	return b.db.Set(dbBlockProtoIndexKey, rawUpdatedValues, nil)
}

// FetchRecordByID retrieves a data record by its identifier.
func (b *PebbleStoreBackend) FetchRecordByID(dataId suave.DataId) (suave.DataRecord, error) {
	key := []byte(formatRecordKey(dataId))

	recordData, closer, err := b.db.Get(key)
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("record %x not found: %w", dataId, err)
	}

	var record suave.DataRecord
	err = json.Unmarshal(recordData, &record)
	closer.Close()
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("could not unmarshal stored record: %w", err)
	}

	return record, nil
}

func (b *PebbleStoreBackend) Store(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
	storeKey := []byte(formatRecordValueKey(record.Id, key))
	return record, b.db.Set(storeKey, value, nil)
}

// Retrieve fetches data associated with a record.
func (b *PebbleStoreBackend) Retrieve(record suave.DataRecord, caller common.Address, key string) ([]byte, error) {
	storeKey := []byte(formatRecordValueKey(record.Id, key))
	data, closer, err := b.db.Get(storeKey)
	if err != nil {
		return nil, fmt.Errorf("could not fetch data for record %x and key %s: %w", record.Id, key, err)
	}
	ret := make([]byte, len(data))
	copy(ret, data)
	closer.Close()
	return ret, nil
}

func (b *PebbleStoreBackend) FetchRecordsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	dbBlockProtoIndexKey := recordByBlockAndProtocolIndexDbKey(blockNumber, namespace)
	rawCurrentValues, closer, err := b.db.Get(dbBlockProtoIndexKey)
	if err != nil {
		return nil
	}

	var currentRecordIds recordByBlockAndProtocolIndexType
	err = json.Unmarshal(rawCurrentValues, &currentRecordIds)
	closer.Close()
	if err != nil {
		return nil
	}

	records := []suave.DataRecord{}
	for _, dataId := range currentRecordIds {
		record, err := b.FetchRecordByID(dataId)
		if err == nil {
			records = append(records, record)
		}
	}

	return records
}
