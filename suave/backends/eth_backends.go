package backends

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/ethereum/go-ethereum/trie"
)

type EthMock struct{}

func (e *EthMock) BuildEthBlock(ctx context.Context, args *suave.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error) {
	block := types.NewBlock(&types.Header{GasUsed: 1000}, txs, nil, nil, trie.NewStackTrie(nil))
	return engine.BlockToExecutableData(block, big.NewInt(11000)), nil
}

type RemoteEthBackend struct {
	endpoint string
	client   *rpc.Client
}

func NewRemoteEthBackend(endpoint string) *RemoteEthBackend {
	return &RemoteEthBackend{
		endpoint: endpoint,
	}
}

func (e *RemoteEthBackend) BuildEthBlock(ctx context.Context, args *suave.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error) {
	if e.client == nil {
		// should lock
		var err error
		client, err := rpc.DialContext(ctx, e.endpoint)
		if err != nil {
			return nil, err
		}
		e.client = client
	}

	var result engine.ExecutionPayloadEnvelope
	err := e.client.CallContext(ctx, &result, "eth_buildEth2Block", args, txs)
	if err != nil {
		client := e.client
		e.client = nil
		client.Close()
		return nil, err
	}

	return &result, nil
}
