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

	pendingLock   sync.Mutex
	pendingBids   map[suave.DataId]suave.DataRecord
	pendingWrites []StoreWrite
}

func (s *TransactionalStore) FetchBidByID(dataId suave.DataId) (suave.DataRecord, error) {
	s.pendingLock.Lock()
	bid, ok := s.pendingBids[dataId]
	s.pendingLock.Unlock()

	if ok {
		return bid, nil
	}

	return s.engine.FetchBidByID(dataId)
}

func (s *TransactionalStore) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	bids := s.engine.FetchBidsByProtocolAndBlock(blockNumber, namespace)

	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	for _, bid := range s.pendingBids {
		if bid.Version == namespace && bid.DecryptionCondition == blockNumber {
			bids = append(bids, bid)
		}
	}

	return bids
}

func (s *TransactionalStore) Store(dataId suave.DataId, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
	record, err := s.FetchBidByID(dataId)
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

func (s *TransactionalStore) Retrieve(dataId suave.DataId, caller common.Address, key string) ([]byte, error) {
	record, err := s.FetchBidByID(dataId)
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

func (s *TransactionalStore) InitializeBid(rawRecord types.DataRecord) (types.DataRecord, error) {
	bid, err := s.engine.InitRecord(rawRecord, s.sourceTx)
	if err != nil {
		return types.DataRecord{}, err
	}

	s.pendingLock.Lock()
	_, found := s.pendingBids[bid.Id]
	if found {
		s.pendingLock.Unlock()
		return types.DataRecord{}, errors.New("bid with this id already exists")
	}
	s.pendingBids[bid.Id] = bid
	s.pendingLock.Unlock()

	return bid.ToInnerBid(), nil
}

func (s *TransactionalStore) Finalize() error {
	return s.engine.Finalize(s.sourceTx, s.pendingBids, s.pendingWrites)
}
