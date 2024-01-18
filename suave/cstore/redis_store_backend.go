package cstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/go-redis/redis/v8"
)

var _ ConfidentialStorageBackend = &RedisStoreBackend{}

var (
	formatRecordKey = func(dataId suave.DataId) string {
		return fmt.Sprintf("record-%x", dataId)
	}

	formatRecordValueKey = func(dataId suave.DataId, key string) string {
		return fmt.Sprintf("record-data-%x-%s", dataId, key)
	}

	ffStoreTTL = 24 * time.Hour
)

type RedisStoreBackend struct {
	ctx      context.Context
	cancel   context.CancelFunc
	redisUri string
	client   *redis.Client
	local    *miniredis.Miniredis
}

func NewRedisStoreBackend(redisUri string) (*RedisStoreBackend, error) {
	r := &RedisStoreBackend{
		cancel:   nil,
		redisUri: redisUri,
	}

	if err := r.start(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RedisStoreBackend) start() error {
	if r.redisUri == "" {
		// create a mini-redis instance
		localRedis, err := miniredis.Run()
		if err != nil {
			return err
		}
		r.local = localRedis
		r.redisUri = localRedis.Addr()
	}

	if r.cancel != nil {
		r.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.ctx = ctx

	client, err := connectRedis(r.redisUri)
	if err != nil {
		return err
	}
	r.client = client

	err = r.InitRecord(mempoolConfidentialStoreRecord)
	if err != nil && !errors.Is(err, suave.ErrRecordAlreadyPresent) {
		return fmt.Errorf("mempool: could not initialize: %w", err)
	}

	return nil
}

func (r *RedisStoreBackend) Stop() error {
	if r.cancel == nil || r.client == nil {
		return errors.New("redis store: Stop() called before Start()")
	}

	if r.local != nil {
		r.local.Close()
	}
	r.cancel()
	r.client.Close()

	return nil
}

// InitRecord prepares a data record for storage.
func (r *RedisStoreBackend) InitRecord(record suave.DataRecord) error {
	key := formatRecordKey(record.Id)

	err := r.client.Get(r.ctx, key).Err()
	if !errors.Is(err, redis.Nil) {
		return suave.ErrRecordAlreadyPresent
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	err = r.client.Set(r.ctx, key, string(data), ffStoreTTL).Err()
	if err != nil {
		return err
	}

	err = r.indexRecord(record)
	if err != nil {
		return err
	}

	return nil
}

// FetchRecordByID retrieves a data record by its identifier.
func (r *RedisStoreBackend) FetchRecordByID(dataId suave.DataId) (suave.DataRecord, error) {
	key := formatRecordKey(dataId)

	data, err := r.client.Get(r.ctx, key).Bytes()
	if err != nil {
		return suave.DataRecord{}, err
	}

	var record suave.DataRecord
	err = json.Unmarshal(data, &record)
	if err != nil {
		return suave.DataRecord{}, err
	}

	return record, nil
}

func (r *RedisStoreBackend) Store(record suave.DataRecord, caller common.Address, key string, value []byte) (suave.DataRecord, error) {
	storeKey := formatRecordValueKey(record.Id, key)
	err := r.client.Set(r.ctx, storeKey, string(value), ffStoreTTL).Err()
	if err != nil {
		return suave.DataRecord{}, fmt.Errorf("unexpected redis error: %w", err)
	}

	return record, nil
}

// Retrieve fetches data associated with a record.
func (r *RedisStoreBackend) Retrieve(record suave.DataRecord, caller common.Address, key string) ([]byte, error) {
	storeKey := formatRecordValueKey(record.Id, key)
	data, err := r.client.Get(r.ctx, storeKey).Bytes()
	if err != nil {
		return []byte{}, fmt.Errorf("unexpected redis error: %w, %s, %v", err, storeKey, r.client.Keys(context.TODO(), "*").String())
	}

	return data, nil
}

var (
	mempoolConfStoreId             = types.DataId{0x39}
	mempoolConfStoreAddr           = common.HexToAddress("0x39")
	mempoolConfidentialStoreRecord = suave.DataRecord{Id: mempoolConfStoreId, AllowedPeekers: []common.Address{mempoolConfStoreAddr}}
)

func (r *RedisStoreBackend) indexRecord(record suave.DataRecord) error {
	defer log.Info("record submitted", "record", record, "store", r.Store)

	var recordsByBlockAndProtocol []suave.DataId
	recordsByBlockAndProtocolBytes, err := r.Retrieve(mempoolConfidentialStoreRecord, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", record.Version, record.DecryptionCondition))
	if err == nil {
		recordsByBlockAndProtocol = suave.MustDecode[[]suave.DataId](recordsByBlockAndProtocolBytes)
	}
	// store record by block number and by protocol + block number
	recordsByBlockAndProtocol = append(recordsByBlockAndProtocol, record.Id)

	r.Store(mempoolConfidentialStoreRecord, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", record.Version, record.DecryptionCondition), suave.MustEncode(recordsByBlockAndProtocol))

	return nil
}

func (r *RedisStoreBackend) FetchRecordsByProtocolAndBlock(blockNumber uint64, namespace string) []suave.DataRecord {
	recordsByProtocolBytes, err := r.Retrieve(mempoolConfidentialStoreRecord, mempoolConfStoreAddr, fmt.Sprintf("protocol-%s-bn-%d", namespace, blockNumber))
	if err != nil {
		return nil
	}

	res := []suave.DataRecord{}

	recordIDs := suave.MustDecode[[]suave.DataId](recordsByProtocolBytes)
	for _, id := range recordIDs {
		record, err := r.FetchRecordByID(id)
		if err != nil {
			continue
		}
		res = append(res, record)
	}

	// defer log.Info("records fetched", "records", string(recordsByProtocolBytes))
	return res
}
