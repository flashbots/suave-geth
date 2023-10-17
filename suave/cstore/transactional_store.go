package cstore

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/exp/slices"
)

type TransactionalStore struct {
	sourceTx *types.Transaction
	engine   *ConfidentialStoreEngine

	pendingLock   sync.Mutex
	pendingBids   map[BidId]Bid
	pendingWrites []StoreWrite
}

func (s *TransactionalStore) FetchBidById(bidId BidId) (Bid, error) {
	s.pendingLock.Lock()
	bid, ok := s.pendingBids[bidId]
	s.pendingLock.Unlock()

	if ok {
		return bid, nil
	}

	return s.engine.FetchBidById(bidId)
}

func (s *TransactionalStore) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []Bid {
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

func (s *TransactionalStore) Store(bidId BidId, caller common.Address, key string, value []byte) (Bid, error) {
	bid, err := s.FetchBidById(bidId)
	if err != nil {
		return Bid{}, err
	}

	if !slices.Contains(bid.AllowedPeekers, caller) {
		return Bid{}, fmt.Errorf("confidential store transaction: %x not allowed to store %s on %x", caller, key, bidId)
	}

	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	s.pendingWrites = append(s.pendingWrites, StoreWrite{
		Bid:    bid,
		Caller: caller,
		Key:    key,
		Value:  common.CopyBytes(value),
	})

	return bid, nil
}

func (s *TransactionalStore) Retrieve(bidId BidId, caller common.Address, key string) ([]byte, error) {
	bid, err := s.FetchBidById(bidId)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(bid.AllowedPeekers, caller) {
		return nil, fmt.Errorf("confidential store transaction: %x not allowed to retrieve %s on %x", caller, key, bidId)
	}

	s.pendingLock.Lock()

	for _, sw := range s.pendingWrites {
		if sw.Bid.Id == bid.Id && sw.Key == key {
			s.pendingLock.Unlock()
			return common.CopyBytes(sw.Value), nil
		}
	}

	s.pendingLock.Unlock()
	return s.engine.Retrieve(bidId, caller, key)
}

func (s *TransactionalStore) InitializeBid(rawBid types.Bid) (types.Bid, error) {
	bid, err := s.engine.InitializeBid(rawBid, s.sourceTx)
	if err != nil {
		return types.Bid{}, err
	}

	s.pendingLock.Lock()
	_, found := s.pendingBids[bid.Id]
	if found {
		s.pendingLock.Unlock()
		return types.Bid{}, errors.New("bid with this id already exists")
	}
	s.pendingBids[bid.Id] = bid
	s.pendingLock.Unlock()

	return bid.ToInnerBid(), nil
}

func (s *TransactionalStore) Finalize() error {
	return s.engine.Finalize(s.sourceTx, s.pendingBids, s.pendingWrites)
}
