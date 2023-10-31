package cstore

import (
	"testing"
)

func TestPebbleStore(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewPebbleStoreBackend(tmpDir)
	testBackendStore(t, store)
}
