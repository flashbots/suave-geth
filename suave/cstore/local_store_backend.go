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
	bids    map[suave.BidId]suave.Bid
	dataMap map[string][]byte
	index   map[string][]suave.BidId
}

func NewLocalConfidentialStore() *LocalConfidentialStore {
	return &LocalConfidentialStore{
		bids:    make(map[suave.BidId]suave.Bid),
		dataMap: make(map[string][]byte),
		index:   make(map[string][]suave.BidId),
	}
}

func (l *LocalConfidentialStore) Stop() error {
	return nil
}

func (l *LocalConfidentialStore) InitializeBid(bid suave.Bid) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	_, found := l.bids[bid.Id]
	if found {
		return suave.ErrBidAlreadyPresent
	}

	l.bids[bid.Id] = bid

	// index the bid by (protocol, block number)
	indexKey := fmt.Sprintf("protocol-%s-bn-%d", bid.Version, bid.DecryptionCondition)
	bidIds := l.index[indexKey]
	bidIds = append(bidIds, bid.Id)
	l.index[indexKey] = bidIds

	return nil
}

func (l *LocalConfidentialStore) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.dataMap[fmt.Sprintf("%x-%s", bid.Id, key)] = append(make([]byte, 0, len(value)), value...)

	defer log.Trace("CSSW", "caller", caller, "key", key, "value", value, "stored", l.dataMap[fmt.Sprintf("%x-%s", bid.Id, key)])
	return bid, nil
}

func (l *LocalConfidentialStore) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	data, found := l.dataMap[fmt.Sprintf("%x-%s", bid.Id, key)]
	if !found {
		return []byte{}, fmt.Errorf("data for key %s not found", key)
	}

	log.Trace("CSRW", "caller", caller, "key", key, "data", data)
	return append(make([]byte, 0, len(data)), data...), nil
}

func (l *LocalConfidentialStore) FetchBidById(bidId suave.BidId) (suave.Bid, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	bid, found := l.bids[bidId]
	if !found {
		return suave.Bid{}, errors.New("bid not found")
	}

	return bid, nil
}

func (l *LocalConfidentialStore) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
	l.lock.Lock()
	defer l.lock.Unlock()

	indexKey := fmt.Sprintf("protocol-%s-bn-%d", namespace, blockNumber)
	bidIDs, ok := l.index[indexKey]
	if !ok {
		return nil
	}

	res := []suave.Bid{}
	for _, id := range bidIDs {
		bid, found := l.bids[id]
		if found {
			res = append(res, bid)
		}
	}

	return res
}
