package builder

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

type builder struct {
	config   *builderConfig
	txns     []*types.Transaction
	receipts []*types.Receipt
	state    *state.StateDB
	gasPool  *core.GasPool
	gasUsed  *uint64
}

type builderConfig struct {
	preState *state.StateDB
	header   *types.Header
	config   *params.ChainConfig
	context  core.ChainContext
	vmConfig vm.Config
}

func newBuilder(config *builderConfig) *builder {
	gp := core.GasPool(config.header.GasLimit)
	var gasUsed uint64

	return &builder{
		config:  config,
		state:   config.preState.Copy(),
		gasPool: &gp,
		gasUsed: &gasUsed,
	}
}

func (b *builder) AddTransaction(txn *types.Transaction) error {
	dummyAuthor := common.Address{}

	receipt, err := core.ApplyTransaction(b.config.config, b.config.context, &dummyAuthor, b.gasPool, b.state, b.config.header, txn, b.gasUsed, b.config.vmConfig)
	if err != nil {
		return err
	}

	b.txns = append(b.txns, txn)
	b.receipts = append(b.receipts, receipt)

	return nil
}

func (b *builder) Finalize() (*types.Block, error) {
	root, err := b.state.Commit(true)
	if err != nil {
		return nil, err
	}

	header := b.config.header
	header.Root = root
	header.GasUsed = *b.gasUsed

	block := types.NewBlock(header, b.txns, nil, b.receipts, trie.NewStackTrie(nil))
	return block, nil
}
