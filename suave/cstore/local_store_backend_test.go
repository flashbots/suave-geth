package cstore

import (
	"testing"
)

func TestLocal_StoreSuite(t *testing.T) {
	store := NewLocalConfidentialStore()
	testBackendStore(t, store)
}
