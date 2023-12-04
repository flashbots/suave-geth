package api

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

var _ API = (*APIClient)(nil)

type APIClient struct {
	rpc *rpc.Client
}

func NewClient(endpoint string) (*APIClient, error) {
	clt, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return NewClientFromRPC(clt), nil
}

func NewClientFromRPC(rpc *rpc.Client) *APIClient {
	return &APIClient{rpc: rpc}
}

func (a *APIClient) NewSession(ctx context.Context) (string, error) {
	var id string
	err := a.rpc.CallContext(ctx, &id, "builder_newSession")
	return id, err
}

func (a *APIClient) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) error {
	err := a.rpc.CallContext(ctx, nil, "builder_addTransaction", sessionId, tx)
	return err
}

func (a *APIClient) Finalize(ctx context.Context, sessionId string) (*engine.ExecutionPayloadEnvelope, error) {
	var res *engine.ExecutionPayloadEnvelope
	err := a.rpc.CallContext(ctx, &res, "builder_finalize", sessionId)
	return res, err
}
