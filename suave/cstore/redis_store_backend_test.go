package cstore

import (
	"testing"
	"time"

	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func TestRedis_StoreSuite(t *testing.T) {
	store, err := NewRedisStoreBackend("", 0)
	require.NoError(t, err)

	testBackendStore(t, store)
}

func TestRedis_TTL_SingleEntry(t *testing.T) {
	store, err := NewRedisStoreBackend("", 1*time.Second)
	require.NoError(t, err)

	record1 := suave.DataRecord{
		Id:                  suave.RandomDataRecordId(),
		Version:             "a",
		DecryptionCondition: 1,
	}
	require.NoError(t, store.InitRecord(record1))

	record1Found, err := store.FetchRecordByID(record1.Id)
	require.NoError(t, err)
	require.Equal(t, record1, record1Found)

	vals := store.FetchRecordsByProtocolAndBlock(1, "a")
	require.Len(t, vals, 1)

	// Advance past the TTL
	store.local.FastForward(2 * time.Second)

	_, err = store.FetchRecordByID(record1.Id)
	require.Error(t, err)

	vals = store.FetchRecordsByProtocolAndBlock(1, "a")
	require.Len(t, vals, 0)
}

func TestRedis_TTL_MultipleEntries_SameIndex(t *testing.T) {
	store, err := NewRedisStoreBackend("", 2*time.Second)
	require.NoError(t, err)

	record1 := suave.DataRecord{
		Id:                  suave.RandomDataRecordId(),
		Version:             "a",
		DecryptionCondition: 1,
	}
	require.NoError(t, store.InitRecord(record1))

	vals := store.FetchRecordsByProtocolAndBlock(1, "a")
	require.Len(t, vals, 1)
	require.Equal(t, record1, vals[0])

	// Advance half the TTL time
	store.local.FastForward(1 * time.Second)

	// Add a new entry that refreshes the index entry
	record2 := suave.DataRecord{
		Id:                  suave.RandomDataRecordId(),
		Version:             "a",
		DecryptionCondition: 1,
	}
	require.NoError(t, store.InitRecord(record2))

	// Advance past the full TTL
	store.local.FastForward(1 * time.Second)

	_, err = store.FetchRecordByID(record1.Id)
	require.Error(t, err)

	_, err = store.FetchRecordByID(record2.Id)
	require.NoError(t, err)

	vals = store.FetchRecordsByProtocolAndBlock(1, "a")
	require.Len(t, vals, 1)
	require.Equal(t, record2, vals[0])
}

func TestRedis_Count(t *testing.T) {
	store, err := NewRedisStoreBackend("", 2*time.Second)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		record := suave.DataRecord{
			Id:                  suave.RandomDataRecordId(),
			Version:             "a",
			DecryptionCondition: 1,
		}
		require.NoError(t, store.InitRecord(record))
	}

	require.NotZero(t, store.count())
}
