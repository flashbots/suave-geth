package api

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

type API interface {
	NewSession(ctx context.Context) (string, error)
	AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error)
	AddTransactions(ctx context.Context, sessionId string, txs types.Transactions) ([]*types.SimulateTransactionResult, error)
	AddBundles(ctx context.Context, sessionId string, bundles []*types.SBundle) ([]*types.SimulateBundleResult, error)
}
