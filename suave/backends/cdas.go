package backends

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

type LocalConfidentialStore struct {
	lock    sync.Mutex
	bids    map[suave.BidId]suave.Bid
	dataMap map[string][]byte
}

func NewLocalConfidentialStore() *LocalConfidentialStore {
	return &LocalConfidentialStore{
		bids:    make(map[suave.BidId]suave.Bid),
		dataMap: make(map[string][]byte),
	}
}

func (s *LocalConfidentialStore) Start() error {
	return nil
}

func (s *LocalConfidentialStore) Stop() error {
	return nil
}

// This function is *trusted* and not available directly through precompiles
// In particular wrt bid id not being maliciously crafted
func (s *LocalConfidentialStore) InitializeBid(bid suave.Bid) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, found := s.bids[bid.Id]
	if found {
		return suave.ErrBidAlreadyPresent
	}

	s.bids[bid.Id] = bid

	return nil
}

func (s *LocalConfidentialStore) FetchEngineBidById(bidId suave.BidId) (suave.Bid, error) {
	bid, found := s.bids[bidId]
	if !found {
		return suave.Bid{}, errors.New("bid not found")
	}

	return bid, nil
}

func (s *LocalConfidentialStore) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.dataMap[fmt.Sprintf("%x-%s", bid.Id, key)] = append(make([]byte, 0, len(value)), value...)

	defer log.Trace("CSSW", "caller", caller, "key", key, "value", value, "stored", s.dataMap[fmt.Sprintf("%x-%s", bid.Id, key)])
	return bid, nil
}

func (s *LocalConfidentialStore) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, found := s.dataMap[fmt.Sprintf("%x-%s", bid.Id, key)]
	if !found {
		return []byte{}, fmt.Errorf("data for key %s not found", key)
	}

	log.Trace("CSRW", "caller", caller, "key", key, "data", data)
	return append(make([]byte, 0, len(data)), data...), nil
}
