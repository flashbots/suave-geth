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

var (
	formatPebbleBidKey      = formatRedisBidKey
	formatPebbleBidValueKey = formatRedisBidValueKey
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

type bidByBlockAndProtocolIndexType = []types.BidId

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

func (b *PebbleStoreBackend) InitializeBid(bid suave.Bid) error {
	key := []byte(formatPebbleBidKey(bid.Id))

	_, closer, err := b.db.Get(key)
	if !errors.Is(err, pebble.ErrNotFound) {
		if err == nil {
			closer.Close()
		}
		return suave.ErrBidAlreadyPresent
	}

	data, err := json.Marshal(bid)
	if err != nil {
		return err
	}

	err = b.db.Set(key, data, nil)
	if err != nil {
		return err
	}

	// index update
	var currentValues bidByBlockAndProtocolIndexType

	dbBlockProtoIndexKey := bidByBlockAndProtocolIndexDbKey(bid.DecryptionCondition, bid.Version)
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

	currentValues = append(currentValues, bid.Id)
	rawUpdatedValues, err := json.Marshal(currentValues)
	if err != nil {
		return err
	}

	return b.db.Set(dbBlockProtoIndexKey, rawUpdatedValues, nil)
}

func (b *PebbleStoreBackend) FetchBidById(bidId suave.BidId) (suave.Bid, error) {
	key := []byte(formatPebbleBidKey(bidId))

	bidData, closer, err := b.db.Get(key)
	if err != nil {
		return suave.Bid{}, fmt.Errorf("bid %x not found: %w", bidId, err)
	}

	var bid suave.Bid
	err = json.Unmarshal(bidData, &bid)
	closer.Close()
	if err != nil {
		return suave.Bid{}, fmt.Errorf("could not unmarshal stored bid: %w", err)
	}

	return bid, nil
}

func (b *PebbleStoreBackend) Store(bid suave.Bid, caller common.Address, key string, value []byte) (suave.Bid, error) {
	storeKey := []byte(formatPebbleBidValueKey(bid.Id, key))
	return bid, b.db.Set(storeKey, value, nil)
}

func (b *PebbleStoreBackend) Retrieve(bid suave.Bid, caller common.Address, key string) ([]byte, error) {
	storeKey := []byte(formatPebbleBidValueKey(bid.Id, key))
	data, closer, err := b.db.Get(storeKey)
	if err != nil {
		return nil, fmt.Errorf("could not fetch data for bid %x and key %s: %w", bid.Id, key, err)
	}
	ret := make([]byte, len(data))
	copy(ret, data)
	closer.Close()
	return ret, nil
}

func (b *PebbleStoreBackend) FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.Bid {
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

	bids := []suave.Bid{}
	for _, bidId := range currentBidIds {
		bid, err := b.FetchBidById(bidId)
		if err == nil {
			bids = append(bids, bid)
		}
	}

	return bids
}
