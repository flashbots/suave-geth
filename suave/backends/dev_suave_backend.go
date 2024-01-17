package backends

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

type confidentialStore interface {
	Reset() error
}

// SuaveInternalBackend is a jsonrpc backend for internal suave testing
type SuaveInternalBackend struct {
	Cstore confidentialStore
}

func (d *SuaveInternalBackend) ResetConfStore(ctx context.Context) error {
	log.Info("Resetting Confidential Store")
	return d.Cstore.Reset()
}
