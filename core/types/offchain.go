package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type OffchainTx struct {
	ExecutionNode common.Address `json:"executionNode" gencodec:"required"`
	Wrapped       Transaction    `json:"wrapped" gencodec:"required"`
	ChainID       *big.Int       // Overwrite the wrapped chain id, since different txs handle it differently. Also probably should overwrite V,R,S
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *OffchainTx) copy() TxData {
	cpy := &OffchainTx{
		ExecutionNode: tx.ExecutionNode,
		Wrapped:       tx.Wrapped,
		ChainID:       new(big.Int),
	}

	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}

	return cpy
}

// accessors for innerTx.
func (tx *OffchainTx) txType() byte {
	return OffchainTxType
}

func (tx *OffchainTx) data() []byte {
	return tx.Wrapped.Data()
}

// Rest is carried over from wrapped tx
func (tx *OffchainTx) chainID() *big.Int         { return tx.ChainID }
func (tx *OffchainTx) accessList() AccessList    { return tx.Wrapped.inner.accessList() }
func (tx *OffchainTx) gas() uint64               { return tx.Wrapped.inner.gas() }
func (tx *OffchainTx) gasFeeCap() *big.Int       { return tx.Wrapped.inner.gasFeeCap() }
func (tx *OffchainTx) gasTipCap() *big.Int       { return tx.Wrapped.inner.gasTipCap() }
func (tx *OffchainTx) gasPrice() *big.Int        { return tx.Wrapped.inner.gasFeeCap() }
func (tx *OffchainTx) value() *big.Int           { return tx.Wrapped.inner.value() }
func (tx *OffchainTx) nonce() uint64             { return tx.Wrapped.inner.nonce() }
func (tx *OffchainTx) to() *common.Address       { return tx.Wrapped.inner.to() }
func (tx *OffchainTx) blobGas() uint64           { return tx.Wrapped.inner.blobGas() }
func (tx *OffchainTx) blobGasFeeCap() *big.Int   { return tx.Wrapped.inner.blobGasFeeCap() }
func (tx *OffchainTx) blobHashes() []common.Hash { return tx.Wrapped.inner.blobHashes() }

func (tx *OffchainTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return tx.Wrapped.inner.effectiveGasPrice(dst, baseFee)
}

func (tx *OffchainTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.Wrapped.inner.rawSignatureValues()
}

func (tx *OffchainTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID = new(big.Int).Set(chainID)
	tx.Wrapped.inner.setSignatureValues(chainID, v, r, s)
}

type OffchainExecutedTx struct {
	ExecutionNode  common.Address `json:"executionNode" gencodec:"required"`
	Wrapped        Transaction    `json:"wrapped" gencodec:"required"`
	OffchainResult []byte         // Should post-execution transaction be its own transaction type / be the main off-chain transaction type?

	// ExecutionNode's signature
	ChainID *big.Int
	V       *big.Int
	R       *big.Int
	S       *big.Int
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *OffchainExecutedTx) copy() TxData {
	cpy := &OffchainExecutedTx{
		ExecutionNode:  tx.ExecutionNode,
		Wrapped:        tx.Wrapped,
		OffchainResult: common.CopyBytes(tx.OffchainResult),
		ChainID:        new(big.Int),
		V:              new(big.Int),
		R:              new(big.Int),
		S:              new(big.Int),
	}

	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}

	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}

	return cpy
}

// accessors for innerTx.
func (tx *OffchainExecutedTx) txType() byte {
	return OffchainExecutedTxType
}

func (tx *OffchainExecutedTx) data() []byte {
	return tx.OffchainResult
}

// Rest is carried over from wrapped tx
func (tx *OffchainExecutedTx) chainID() *big.Int         { return tx.ChainID }
func (tx *OffchainExecutedTx) accessList() AccessList    { return tx.Wrapped.inner.accessList() }
func (tx *OffchainExecutedTx) gas() uint64               { return tx.Wrapped.inner.gas() }
func (tx *OffchainExecutedTx) gasFeeCap() *big.Int       { return tx.Wrapped.inner.gasFeeCap() }
func (tx *OffchainExecutedTx) gasTipCap() *big.Int       { return tx.Wrapped.inner.gasTipCap() }
func (tx *OffchainExecutedTx) gasPrice() *big.Int        { return tx.Wrapped.inner.gasFeeCap() }
func (tx *OffchainExecutedTx) value() *big.Int           { return tx.Wrapped.inner.value() }
func (tx *OffchainExecutedTx) nonce() uint64             { return tx.Wrapped.inner.nonce() }
func (tx *OffchainExecutedTx) to() *common.Address       { return tx.Wrapped.inner.to() }
func (tx *OffchainExecutedTx) blobGas() uint64           { return tx.Wrapped.inner.blobGas() }
func (tx *OffchainExecutedTx) blobGasFeeCap() *big.Int   { return tx.Wrapped.inner.blobGasFeeCap() }
func (tx *OffchainExecutedTx) blobHashes() []common.Hash { return tx.Wrapped.inner.blobHashes() }

func (tx *OffchainExecutedTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return tx.Wrapped.inner.effectiveGasPrice(dst, baseFee)
}

func (tx *OffchainExecutedTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *OffchainExecutedTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}
