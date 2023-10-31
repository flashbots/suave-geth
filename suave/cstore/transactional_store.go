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
	engine   *ConfidentialStoreEngine

	pendingLock   sync.Mutex
	pendingBids   map[suave.BidId]suave.Bid
	pendingWrites []StoreWrite
}

func (s *TransactionalStore) FetchBidById(bidId suave.BidId) (suave.Bid, error) {
	s.pendingLock.Lock()
	bid, ok := s.pendingBids[bidId]
	s.pendingLock.Unlock()

	if ok {
		return bid, nil
	}

	return s.engine.FetchBidById(bidId)
}

func (s *TransactionalStore) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
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

func (s *TransactionalStore) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	bid, err := s.FetchBidById(bidId)
	if err != nil {
		return suave.Bid{}, err
	}

	if !slices.Contains(bid.AllowedPeekers, caller) && !slices.Contains(bid.AllowedPeekers, suave.AllowedPeekerAny) {
		return suave.Bid{}, fmt.Errorf("confidential store transaction: %x not allowed to store %s on %x", caller, key, bidId)
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

func (s *TransactionalStore) Retrieve(bidId suave.BidId, caller common.Address, key string) ([]byte, error) {
	bid, err := s.FetchBidById(bidId)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(bid.AllowedPeekers, caller) && !slices.Contains(bid.AllowedPeekers, suave.AllowedPeekerAny) {
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
