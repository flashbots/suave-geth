package backends

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

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

type ACData struct {
	bid     suave.Bid
	dataMap map[string][]byte
}

// Optional key, if provided will initialize with the value
// This function is *trusted* and not available directly through precompiles
// In particular wrt bid id not being maliciously crafted
func (s *LocalConfidentialStore) Initialize(bid suave.Bid, key string, value []byte) (suave.Bid, error) {
	if bid.Id == [16]byte{} {
		bid.Id = [16]byte(uuid.New())
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	_, found := s.bids[bid.Id]
	if found {
		return suave.Bid{}, errors.New("bid already present")
	}
	acData := ACData{bid, make(map[string][]byte)}
	if key != "" {
		acData.dataMap[key] = value
	}
	s.bids[bid.Id] = acData

	return bid, nil
}

func (s *LocalConfidentialStore) Store(bidId suave.BidId, caller common.Address, key string, value []byte) (suave.Bid, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	bidAcd, found := s.bids[bidId]
	if !found {
		return suave.Bid{}, errors.New("bid not initialized")
	}

	if !slices.Contains(bidAcd.bid.AllowedPeekers, caller) {
		return suave.Bid{}, errors.New("not allowed")
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
		return []byte{}, errors.New("not allowed")
	}

	data, found := bidAcd.dataMap[key]
	if !found {
		return []byte{}, errors.New("data not found")
	}

	log.Trace("CSRW", "caller", caller, "key", key, "data", data)
	return append(make([]byte, 0, len(data)), data...), nil
}
