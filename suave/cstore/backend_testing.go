package cstore

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/stretchr/testify/require"
)

func testBackendStore(t *testing.T, store ConfidentialStorageBackend) {
	record := suave.DataRecord{
		Id:                  suave.RandomDataRecordId(),
		DecryptionCondition: 10,
		AllowedPeekers:      []common.Address{common.HexToAddress("0x424344")},
		Namespace:           "default:v0:ethBundles",
	}

	err := store.InitRecord(record)
	require.NoError(t, err)

	recordRes, err := store.FetchRecordByID(record.Id)
	require.NoError(t, err)
	require.Equal(t, record, recordRes)

	_, err = store.Store(record, record.AllowedPeekers[0], "xx", []byte{0x43, 0x14})
	require.NoError(t, err)

	retrievedData, err := store.Retrieve(record, record.AllowedPeekers[0], "xx")
	require.NoError(t, err)
	require.Equal(t, []byte{0x43, 0x14}, retrievedData)

	records := store.FetchRecordsByNamespaceAndBlock(10, "default:v0:ethBundles")
	require.Len(t, records, 1)
	require.Equal(t, record, records[0])
}
