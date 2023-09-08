package backends

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"golang.org/x/exp/slices"
)

type LocalConfidentialStore struct {
	lock sync.Mutex
	bids map[suave.BidId]ACData
}

func NewLocalConfidentialStore() *LocalConfidentialStore {
	return &LocalConfidentialStore{
		bids: make(map[suave.BidId]ACData),
	}
}

func (s *LocalConfidentialStore) Start() error {
	return nil
}

func (s *LocalConfidentialStore) Stop() error {
	return nil
}

type ACData struct {
	bid     suave.Bid
	dataMap map[string][]byte
}

// This function is *trusted* and not available directly through precompiles
// In particular wrt bid id not being maliciously crafted
func (s *LocalConfidentialStore) InitializeBid(bid suave.Bid) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, found := s.bids[bid.Id]
	if found {
		return suave.BidAlreadyPresentError
	}

	s.bids[bid.Id] = ACData{bid, make(map[string][]byte)}

	return nil
}

func (s *LocalConfidentialStore) FetchEngineBidById(bidId suave.BidId) (suave.Bid, error) {
	bidData, found := s.bids[bidId]
	if !found {
		return suave.Bid{}, errors.New("bid not found")
	}

	return bidData.bid, nil
}

func (s *LocalConfidentialStore) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	bidAcd, found := s.bids[bidId]
	if !found {
		return suave.Bid{}, errors.New("bid not initialized")
	}

	if !slices.Contains(bidAcd.bid.AllowedPeekers, caller) {
		return suave.Bid{}, fmt.Errorf("%x not allowed to store %s on %x", caller, key, bidId)
	}

	bidAcd.dataMap[key] = append(make([]byte, 0, len(value)), value...)

	defer log.Trace("CSSW", "caller", caller, "key", key, "value", value, "stored", s.bids[bidId].dataMap[key])
	return bidAcd.bid, nil
}

func (s *LocalConfidentialStore) Retrieve(bidId suave.BidId, caller common.Address, key string) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	bidAcd, found := s.bids[bidId]
	if !found {
		return []byte{}, fmt.Errorf("%v not found", bidId)
	}

	if !slices.Contains(bidAcd.bid.AllowedPeekers, caller) {
		return []byte{}, fmt.Errorf("%x not allowed to fetch %s on %x", caller, key, bidId)
	}

	data, found := bidAcd.dataMap[key]
	if !found {
		return []byte{}, fmt.Errorf("data for key %s not found", key)
	}

	log.Trace("CSRW", "caller", caller, "key", key, "data", data)
	return append(make([]byte, 0, len(data)), data...), nil
}
