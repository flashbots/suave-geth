package cstore

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

var _ ConfidentialStorageBackend = &LocalConfidentialStore{}

type LocalConfidentialStore struct {
	lock    sync.Mutex
	records map[suave.DataId]suave.DataRecord
	dataMap map[string][]byte
	index   map[string][]suave.DataId
}

func NewLocalConfidentialStore() *LocalConfidentialStore {
	return &LocalConfidentialStore{
		records: make(map[suave.DataId]suave.DataRecord),
		dataMap: make(map[string][]byte),
		index:   make(map[string][]suave.DataId),
	}
}

func (l *LocalConfidentialStore) Reset() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.records = make(map[suave.DataId]suave.DataRecord)
	l.dataMap = make(map[string][]byte)
	l.index = make(map[string][]suave.DataId)

	return nil
}

func (l *LocalConfidentialStore) Stop() error {
	return nil
}

func (l *LocalConfidentialStore) InitRecord(record suave.DataRecord) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	_, found := l.records[record.Id]
	if found {
		return suave.ErrRecordAlreadyPresent
	}

	l.records[record.Id] = record

	// index the record by (protocol, block number)
	indexKey := fmt.Sprintf("protocol-%s-bn-%d", record.Version, record.DecryptionCondition)
	recordIds := l.index[indexKey]
	recordIds = append(recordIds, record.Id)
	l.index[indexKey] = recordIds

	return nil
}

func (l *LocalConfidentialStore) Store(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.dataMap[fmt.Sprintf("%x-%s", record.Id, key)] = append(make([]byte, 0, len(value)), value...)

	defer log.Trace("CSSW", "caller", caller, "key", key, "value", value, "stored", l.dataMap[fmt.Sprintf("%x-%s", record.Id, key)])
	return record, nil
}

func (l *LocalConfidentialStore) Retrieve(record suave.DataRecord, caller common.Address, key string) ([]byte, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	data, found := l.dataMap[fmt.Sprintf("%x-%s", record.Id, key)]
	if !found {
		return []byte{}, fmt.Errorf("data for key %s not found", key)
	}

	log.Trace("CSRW", "caller", caller, "key", key, "data", data)
	return append(make([]byte, 0, len(data)), data...), nil
}

func (l *LocalConfidentialStore) FetchRecordByID(dataId suave.DataId) (suave.DataRecord, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	bid, found := l.records[dataId]
	if !found {
		return suave.DataRecord{}, errors.New("record not found")
	}

	return bid, nil
}

func (l *LocalConfidentialStore) FetchRecordsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	l.lock.Lock()
	defer l.lock.Unlock()

	indexKey := fmt.Sprintf("protocol-%s-bn-%d", namespace, blockNumber)
	bidIDs, ok := l.index[indexKey]
	if !ok {
		return nil
	}

	res := []suave.DataRecord{}
	for _, id := range bidIDs {
		bid, found := l.records[id]
		if found {
			res = append(res, bid)
		}
	}

	return res
}
