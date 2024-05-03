package types

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

const DepositTxType = 0x7E

type DepositTx struct {
	// SourceHash uniquely identifies the source of the deposit
	SourceHash common.Hash
	// From is exposed through the types.Signer, not through TxData
	From common.Address
	// nil means contract creation
	To *common.Address `rlp:"nil"`
	// Mint is minted on L2, locked on L1, nil if no minting.
	Mint *big.Int `rlp:"nil"`
	// Value is transferred from L2 balance, executed after Mint (if any)
	Value *big.Int
	// gas limit
	Gas uint64
	// Field indicating if this transaction is exempt from the L2 gas limit.
	IsSystemTransaction bool
	// Normal Tx data
	Data []byte
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DepositTx) copy() TxData {
	cpy := &DepositTx{
		SourceHash:          tx.SourceHash,
		From:                tx.From,
		To:                  copyAddressPtr(tx.To),
		Mint:                nil,
		Value:               new(big.Int),
		Gas:                 tx.Gas,
		IsSystemTransaction: tx.IsSystemTransaction,
		Data:                common.CopyBytes(tx.Data),
	}
	if tx.Mint != nil {
		cpy.Mint = new(big.Int).Set(tx.Mint)
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	return cpy
}

// accessors for innerTx.
func (tx *DepositTx) txType() byte           { return DepositTxType }
func (tx *DepositTx) chainID() *big.Int      { return common.Big0 }
func (tx *DepositTx) accessList() AccessList { return nil }
func (tx *DepositTx) data() []byte           { return tx.Data }
func (tx *DepositTx) gas() uint64            { return tx.Gas }
func (tx *DepositTx) gasFeeCap() *big.Int    { return new(big.Int) }
func (tx *DepositTx) gasTipCap() *big.Int    { return new(big.Int) }
func (tx *DepositTx) gasPrice() *big.Int     { return new(big.Int) }
func (tx *DepositTx) value() *big.Int        { return tx.Value }
func (tx *DepositTx) nonce() uint64          { return 0 }
func (tx *DepositTx) to() *common.Address    { return tx.To }
func (tx *DepositTx) isSystemTx() bool       { return tx.IsSystemTransaction }

func (tx *DepositTx) blobGas() uint64           { return 0 }
func (tx *DepositTx) blobGasFeeCap() *big.Int   { return nil }
func (tx *DepositTx) blobHashes() []common.Hash { return nil }

func (tx *DepositTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(new(big.Int))
}

func (tx *DepositTx) effectiveNonce() *uint64 { return nil }

func (tx *DepositTx) rawSignatureValues() (v, r, s *big.Int) {
	return common.Big0, common.Big0, common.Big0
}

func (tx *DepositTx) setSignatureValues(chainID, v, r, s *big.Int) {
	// this is a noop for deposit transactions
}

func (tx *DepositTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *DepositTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}
