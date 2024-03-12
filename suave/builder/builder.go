package builder

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
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

func (b *builder) AddTransaction(txn *types.Transaction) (*types.SimulateTransactionResult, error) {
	return b.AddTransactions([]*types.Transaction{txn}, make(map[common.Hash]struct{}))
}

func (b *builder) AddTransactions(txs types.Transactions, revertingHashes map[common.Hash]struct{}) (*types.SimulateTransactionResult, error) {
	vmConfig := vm.Config{
		NoBaseFee: true,
	}

	snap := b.state.Snapshot()
	txnsSnap := b.txns
	receiptsSnap := b.receipts
	result := &types.SimulateTransactionResult{}

	for _, txn := range txs {
		dummyAuthor := common.Address{}
		b.state.SetTxContext(txn.Hash(), len(b.txns))
		receipt, err := core.ApplyTransaction(b.config.config, b.config.context, &dummyAuthor, b.gasPool, b.state, b.config.header, txn, b.gasUsed, vmConfig)
		if err != nil {
			if _, ok := revertingHashes[txn.Hash()]; ok {
				// if the transaction is allowed to revert, continue.
				continue
			}

			b.state.RevertToSnapshot(snap)
			b.txns = txnsSnap
			b.receipts = receiptsSnap
			return &types.SimulateTransactionResult{
				Success: false,
				Error:   err.Error(),
			}, err
		}

		b.txns = append(b.txns, txn)
		b.receipts = append(b.receipts, receipt)

		for _, log := range receipt.Logs {
			result.Logs = append(result.Logs, &types.SimulatedLog{
				Addr:   log.Address,
				Topics: log.Topics,
				Data:   log.Data,
			})
		}
	}
	return result, nil
}

func checkInclusion(currentBlockNumber *big.Int, bundle *types.SBundle) error {
	if bundle.BlockNumber != nil && bundle.MaxBlock != nil && bundle.BlockNumber.Cmp(bundle.MaxBlock) > 0 {
		return types.ErrInvalidInclusionRange
	}

	// check inclusion target if BlockNumber is set
	if bundle.BlockNumber != nil {
		if bundle.MaxBlock == nil && currentBlockNumber.Cmp(bundle.BlockNumber) != 0 {
			return types.ErrInvalidBlockNumber
		}

		if bundle.MaxBlock != nil {
			if currentBlockNumber.Cmp(bundle.MaxBlock) > 0 {
				return types.ErrExceedsMaxBlock
			}

			if currentBlockNumber.Cmp(bundle.BlockNumber) < 0 {
				return types.ErrInvalidBlockNumber
			}
		}
	}

	// check if the bundle has transactions
	if bundle.Txs == nil || bundle.Txs.Len() == 0 {
		return types.ErrEmptyTxs
	}

	return nil
}

// TODO: consider revertingHashes as a map[common.Hash]struct{} in the future
func getRevertHashMap(bundle *types.SBundle) map[common.Hash]struct{} {
	m := make(map[common.Hash]struct{})
	for _, hash := range bundle.RevertingHashes {
		m[hash] = struct{}{}
	}
	return m
}

func (b *builder) AddBundle(bundle *types.SBundle) (*types.SimulateBundleResult, error) {
	return b.AddBundles([]*types.SBundle{bundle})
}

func (b *builder) AddBundles(bundles []*types.SBundle) (*types.SimulateBundleResult, error) {
	snap := b.state.Snapshot()
	txnsSnap := b.txns
	receiptsSnap := b.receipts

	var simResults []*types.SimulateTransactionResult

	var err error
	for _, bundle := range bundles {
		err = checkInclusion(b.config.header.Number, bundle)
		if err != nil {
			break
		}
		revertingHashes := getRevertHashMap(bundle)
		txResult, err := b.AddTransactions(bundle.Txs, revertingHashes)
		if err != nil {
			break
		}
		simResults = append(simResults, txResult)
	}

	if err != nil {
		b.state.RevertToSnapshot(snap)
		b.txns = txnsSnap
		b.receipts = receiptsSnap
		return &types.SimulateBundleResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &types.SimulateBundleResult{
		Success:                    true,
		SimulateTransactionResults: simResults,
	}, nil
}
