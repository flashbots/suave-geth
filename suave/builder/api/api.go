package api

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core/types"
)

type API interface {
	NewSession(ctx context.Context) (string, error)
	AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) error
	Finalize(ctx context.Context, sessionId string) (*engine.ExecutionPayloadEnvelope, error)
}
