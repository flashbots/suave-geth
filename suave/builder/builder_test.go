package builder

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestBuilder_AddTxn_Simple(t *testing.T) {
	to := common.Address{0x01, 0x10, 0xab}

	mock := newMockBuilder(t)
	txn := mock.newTransfer(t, to, big.NewInt(1))

	_, err := mock.builder.AddTransaction(txn)
	require.NoError(t, err)

	mock.expect(t, expectedResult{
		txns: []*types.Transaction{
			txn,
		},
		balances: map[common.Address]*big.Int{
			to: big.NewInt(1),
		},
	})

	block, err := mock.builder.Finalize()
	require.NoError(t, err)

	require.Equal(t, uint64(21000), block.GasUsed())
	require.Len(t, block.Transactions(), 1)
	require.Equal(t, txn.Hash(), block.Transactions()[0].Hash())
}

func newMockBuilder(t *testing.T) *mockBuilder {
	// create a dummy header at 0
	header := &types.Header{
		Number:     big.NewInt(0),
		GasLimit:   1000000000000,
		Time:       1000,
		Difficulty: big.NewInt(1),
	}

	var stateRef *state.StateDB

	premineKey, _ := crypto.GenerateKey() // TODO: it would be nice to have it deterministic
	premineKeyAddr := crypto.PubkeyToAddress(premineKey.PublicKey)

	// create a state reference with at least one premined account
	// In order to test the statedb in isolation, we are going
	// to commit this pre-state to a memory database
	{
		db := state.NewDatabase(rawdb.NewMemoryDatabase())
		preState, err := state.New(types.EmptyRootHash, db, nil)
		require.NoError(t, err)

		preState.AddBalance(premineKeyAddr, big.NewInt(1000000000000000000))

		root, err := preState.Commit(true)
		require.NoError(t, err)

		stateRef, err = state.New(root, db, nil)
		require.NoError(t, err)
	}

	// for the sake of this test, we only need all the forks enabled
	chainConfig := params.SuaveChainConfig

	// Disable london so that we do not check gasFeeCap (TODO: Fix)
	chainConfig.LondonBlock = big.NewInt(100)

	m := &mockBuilder{
		premineKey:     premineKey,
		premineKeyAddr: premineKeyAddr,
		signer:         types.NewEIP155Signer(chainConfig.ChainID),
	}

	config := &builderConfig{
		header:   header,
		preState: stateRef,
		config:   chainConfig,
		context:  m, // m implements ChainContext with panics
		vmConfig: vm.Config{},
	}
	m.builder = newBuilder(config)

	return m
}

type mockBuilder struct {
	builder *builder

	// builtin private keys
	premineKey     *ecdsa.PrivateKey
	premineKeyAddr common.Address

	nextNonce uint64 // figure out a better way
	signer    types.Signer
}

func (m *mockBuilder) Engine() consensus.Engine {
	panic("TODO")
}

func (m *mockBuilder) GetHeader(common.Hash, uint64) *types.Header {
	panic("TODO")
}

func (m *mockBuilder) getNonce() uint64 {
	next := m.nextNonce
	m.nextNonce++
	return next
}

func (m *mockBuilder) newTransfer(t *testing.T, to common.Address, amount *big.Int) *types.Transaction {
	tx := types.NewTransaction(m.getNonce(), to, amount, 1000000, big.NewInt(1), nil)
	return m.newTxn(t, tx)
}

func (m *mockBuilder) newTxn(t *testing.T, tx *types.Transaction) *types.Transaction {
	// sign the transaction
	signature, err := crypto.Sign(m.signer.Hash(tx).Bytes(), m.premineKey)
	require.NoError(t, err)

	// include the signature in the transaction
	tx, err = tx.WithSignature(m.signer, signature)
	require.NoError(t, err)

	return tx
}

type expectedResult struct {
	txns     []*types.Transaction
	balances map[common.Address]*big.Int
}

func (m *mockBuilder) expect(t *testing.T, res expectedResult) {
	// validate txns
	if len(res.txns) != len(m.builder.txns) {
		t.Fatalf("expected %d txns, got %d", len(res.txns), len(m.builder.txns))
	}
	for indx, txn := range res.txns {
		if txn.Hash() != m.builder.txns[indx].Hash() {
			t.Fatalf("expected txn %d to be %s, got %s", indx, txn.Hash(), m.builder.txns[indx].Hash())
		}
	}

	// The receipts must be the same as the txns
	if len(res.txns) != len(m.builder.receipts) {
		t.Fatalf("expected %d receipts, got %d", len(res.txns), len(m.builder.receipts))
	}
	for indx, txn := range res.txns {
		if txn.Hash() != m.builder.receipts[indx].TxHash {
			t.Fatalf("expected receipt %d to be %s, got %s", indx, txn.Hash(), m.builder.receipts[indx].TxHash)
		}
	}

	// The gas left in the pool must be the header gas limit minus
	// the total gas consumed by all the transactions in the block.
	totalGasConsumed := uint64(0)
	for _, receipt := range m.builder.receipts {
		totalGasConsumed += receipt.GasUsed
	}
	if m.builder.gasPool.Gas() != m.builder.config.header.GasLimit-totalGasConsumed {
		t.Fatalf("expected gas pool to be %d, got %d", m.builder.config.header.GasLimit-totalGasConsumed, m.builder.gasPool.Gas())
	}

	// The 'gasUsed' must match the total gas consumed by all the transactions
	if *m.builder.gasUsed != totalGasConsumed {
		t.Fatalf("expected gas used to be %d, got %d", totalGasConsumed, m.builder.gasUsed)
	}

	// The state must match the expected balances
	for addr, expectedBalance := range res.balances {
		balance := m.builder.state.GetBalance(addr)
		if balance.Cmp(expectedBalance) != 0 {
			t.Fatalf("expected balance of %s to be %d, got %d", addr, expectedBalance, balance)
		}
	}
}
