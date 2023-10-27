package cstore

import (
	"testing"
)

func TestRedis_StoreSuite(t *testing.T) {
	store, _ := NewRedisStoreBackend("")
	testBackendStore(t, store)
}
