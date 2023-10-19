package cstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedis_StoreSuite(t *testing.T) {
	store := NewRedisStoreBackend("")
	require.NoError(t, store.Start())

	testBackendStore(t, store)
}
