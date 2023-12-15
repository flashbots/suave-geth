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

var bidByBlockAndProtocolIndexDbKey = func(blockNumber uint64, namespace string) []byte {
	return []byte(fmt.Sprintf("bids-block-%d-ns-%s", blockNumber, namespace))
}

type bidByBlockAndProtocolIndexType = []types.DataId

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

func (b *PebbleStoreBackend) InitRecord(record suave.DataRecord) error {
	key := []byte(formatRecordKey(record.Id))

	_, closer, err := b.db.Get(key)
	if !errors.Is(err, pebble.ErrNotFound) {
		if err == nil {
			closer.Close()
		}
		return suave.ErrBidAlreadyPresent
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
	var currentValues bidByBlockAndProtocolIndexType

	dbBlockProtoIndexKey := bidByBlockAndProtocolIndexDbKey(record.DecryptionCondition, record.Version)
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

func (b *PebbleStoreBackend) FetchBidByID(dataId suave.DataId) (suave.DataRecord, error) {
	key := []byte(formatRecordKey(dataId))

	bidData, closer, err := b.db.Get(key)
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("record %x not found: %w", dataId, err)
	}

	var record suave.DataRecord
	err = json.Unmarshal(bidData, &record)
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

func (b *PebbleStoreBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	dbBlockProtoIndexKey := bidByBlockAndProtocolIndexDbKey(blockNumber, namespace)
	rawCurrentValues, closer, err := b.db.Get(dbBlockProtoIndexKey)
	if err != nil {
		return nil
	}

	var currentBidIds bidByBlockAndProtocolIndexType
	err = json.Unmarshal(rawCurrentValues, &currentBidIds)
	closer.Close()
	if err != nil {
		return nil
	}

	bids := []suave.DataRecord{}
	for _, dataId := range currentBidIds {
		record, err := b.FetchBidByID(dataId)
		if err == nil {
			bids = append(bids, record)
		}
	}

	return bids
}
