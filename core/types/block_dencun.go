// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package types contains data types related to Ethereum consensus.
package types

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type DencunHeader struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`
	Number      *big.Int       `json:"number"           gencodec:"required"`
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        uint64         `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`

	// WithdrawalsHash was added by EIP-4895 and is ignored in legacy headers.
	WithdrawalsHash *common.Hash `json:"withdrawalsRoot" rlp:"optional"`

	// BlobGasUsed was added by EIP-4844 and is ignored in legacy headers.
	BlobGasUsed *uint64 `json:"blobGasUsed" rlp:"optional"`

	// ExcessBlobGas was added by EIP-4844 and is ignored in legacy headers.
	ExcessBlobGas *uint64 `json:"excessBlobGas" rlp:"optional"`

	// ParentBeaconRoot was added by EIP-4788 and is ignored in legacy headers.
	ParentBeaconRoot *common.Hash `json:"parentBeaconBlockRoot" rlp:"optional"`
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *DencunHeader) Hash() common.Hash {
	return rlpHash(h)
}

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *DencunHeader) Size() common.StorageSize {
	var baseFeeBits int
	if h.BaseFee != nil {
		baseFeeBits = h.BaseFee.BitLen()
	}
	return headerSize + common.StorageSize(len(h.Extra)+(h.Difficulty.BitLen()+h.Number.BitLen()+baseFeeBits)/8)
}

// SanityCheck checks a few basic things -- these checks are way beyond what
// any 'sane' production values should hold, and can mainly be used to prevent
// that the unbounded fields are stuffed with junk data to add processing
// overhead
func (h *DencunHeader) SanityCheck() error {
	if h.Number != nil && !h.Number.IsUint64() {
		return fmt.Errorf("too large block number: bitlen %d", h.Number.BitLen())
	}
	if h.Difficulty != nil {
		if diffLen := h.Difficulty.BitLen(); diffLen > 80 {
			return fmt.Errorf("too large block difficulty: bitlen %d", diffLen)
		}
	}
	if eLen := len(h.Extra); eLen > 100*1024 {
		return fmt.Errorf("too large block extradata: size %d", eLen)
	}
	if h.BaseFee != nil {
		if bfLen := h.BaseFee.BitLen(); bfLen > 256 {
			return fmt.Errorf("too large base fee: bitlen %d", bfLen)
		}
	}
	return nil
}

// EmptyBody returns true if there is no additional 'body' to complete the header
// that is: no transactions, no uncles and no withdrawals.
func (h *DencunHeader) EmptyBody() bool {
	if h.WithdrawalsHash == nil {
		return h.TxHash == EmptyTxsHash && h.UncleHash == EmptyUncleHash
	}
	return h.TxHash == EmptyTxsHash && h.UncleHash == EmptyUncleHash && *h.WithdrawalsHash == EmptyWithdrawalsHash
}

// EmptyReceipts returns true if there are no receipts for this header/block.
func (h *DencunHeader) EmptyReceipts() bool {
	return h.ReceiptHash == EmptyReceiptsHash
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) together.
type DencunBody struct {
	Transactions []*Transaction
	Uncles       []*DencunHeader
	Withdrawals  []*Withdrawal `rlp:"optional"`
}

type DencunBlock struct {
	header       *DencunHeader
	uncles       []*DencunHeader
	transactions Transactions
	withdrawals  Withdrawals

	// caches
	hash atomic.Value
	size atomic.Value

	// These fields are used by package eth to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewDencunBlock(header *DencunHeader, txs []*Transaction, uncles []*DencunHeader, receipts []*Receipt, hasher TrieHasher) *DencunBlock {
	b := &DencunBlock{header: CopyDencunHeader(header)}

	// TODO: panic if len(txs) != len(receipts)
	if len(txs) == 0 {
		b.header.TxHash = EmptyTxsHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyReceiptsHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}

	if len(uncles) == 0 {
		b.header.UncleHash = EmptyUncleHash
	} else {
		b.header.UncleHash = CalcDencunUncleHash(uncles)
		b.uncles = make([]*DencunHeader, len(uncles))
		for i := range uncles {
			b.uncles[i] = CopyDencunHeader(uncles[i])
		}
	}

	return b
}

// NewBlockWithWithdrawals creates a new block with withdrawals. The input data
// is copied, changes to header and to the field values will not
// affect the block.
//
// The values of TxHash, UncleHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewDencunBlockWithWithdrawals(header *DencunHeader, txs []*Transaction, uncles []*DencunHeader, receipts []*Receipt, withdrawals []*Withdrawal, hasher TrieHasher) *DencunBlock {
	b := NewDencunBlock(header, txs, uncles, receipts, hasher)

	if withdrawals == nil {
		b.header.WithdrawalsHash = nil
	} else if len(withdrawals) == 0 {
		b.header.WithdrawalsHash = &EmptyWithdrawalsHash
	} else {
		h := DeriveSha(Withdrawals(withdrawals), hasher)
		b.header.WithdrawalsHash = &h
	}

	return b.WithWithdrawals(withdrawals)
}

func NewBlockWithDencunHeader(header *DencunHeader) *DencunBlock {
	return &DencunBlock{header: CopyDencunHeader(header)}
}

// CopyDencunHeader creates a deep copy of a block header.
func CopyDencunHeader(h *DencunHeader) *DencunHeader {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	if h.WithdrawalsHash != nil {
		cpy.WithdrawalsHash = new(common.Hash)
		*cpy.WithdrawalsHash = *h.WithdrawalsHash
	}
	if h.ExcessBlobGas != nil {
		cpy.ExcessBlobGas = new(uint64)
		*cpy.ExcessBlobGas = *h.ExcessBlobGas
	}
	if h.BlobGasUsed != nil {
		cpy.BlobGasUsed = new(uint64)
		*cpy.BlobGasUsed = *h.BlobGasUsed
	}
	if h.ParentBeaconRoot != nil {
		cpy.ParentBeaconRoot = new(common.Hash)
		*cpy.ParentBeaconRoot = *h.ParentBeaconRoot
	}
	return &cpy
}

// "external" block encoding. used for eth protocol, etc.
type extdencunblock struct {
	Header      *DencunHeader
	Txs         []*Transaction
	Uncles      []*DencunHeader
	Withdrawals []*Withdrawal `rlp:"optional"`
}

// DecodeRLP decodes the Ethereum
func (b *DencunBlock) DecodeRLP(s *rlp.Stream) error {
	var eb extdencunblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.uncles, b.transactions, b.withdrawals = eb.Header, eb.Uncles, eb.Txs, eb.Withdrawals
	b.size.Store(rlp.ListSize(size))
	return nil
}

// EncodeRLP serializes b into the Ethereum RLP block format.
func (b *DencunBlock) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extdencunblock{
		Header:      b.header,
		Txs:         b.transactions,
		Uncles:      b.uncles,
		Withdrawals: b.withdrawals,
	})
}

// TODO: copies

func (b *DencunBlock) Uncles() []*DencunHeader    { return b.uncles }
func (b *DencunBlock) Transactions() Transactions { return b.transactions }

func (b *DencunBlock) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *DencunBlock) Number() *big.Int     { return new(big.Int).Set(b.header.Number) }
func (b *DencunBlock) GasLimit() uint64     { return b.header.GasLimit }
func (b *DencunBlock) GasUsed() uint64      { return b.header.GasUsed }
func (b *DencunBlock) Difficulty() *big.Int { return new(big.Int).Set(b.header.Difficulty) }
func (b *DencunBlock) Time() uint64         { return b.header.Time }

func (b *DencunBlock) NumberU64() uint64        { return b.header.Number.Uint64() }
func (b *DencunBlock) MixDigest() common.Hash   { return b.header.MixDigest }
func (b *DencunBlock) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *DencunBlock) Bloom() Bloom             { return b.header.Bloom }
func (b *DencunBlock) Coinbase() common.Address { return b.header.Coinbase }
func (b *DencunBlock) Root() common.Hash        { return b.header.Root }
func (b *DencunBlock) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *DencunBlock) TxHash() common.Hash      { return b.header.TxHash }
func (b *DencunBlock) ReceiptHash() common.Hash { return b.header.ReceiptHash }
func (b *DencunBlock) UncleHash() common.Hash   { return b.header.UncleHash }
func (b *DencunBlock) Extra() []byte            { return common.CopyBytes(b.header.Extra) }

func (b *DencunBlock) BaseFee() *big.Int {
	if b.header.BaseFee == nil {
		return nil
	}
	return new(big.Int).Set(b.header.BaseFee)
}

func (b *DencunBlock) Withdrawals() Withdrawals {
	return b.withdrawals
}

func (b *DencunBlock) Header() *DencunHeader { return CopyDencunHeader(b.header) }

// Body returns the non-header content of the block.
func (b *DencunBlock) Body() *DencunBody { return &DencunBody{b.transactions, b.uncles, b.withdrawals} }

// Size returns the true RLP encoded storage size of the block, either by encoding
// and returning it, or returning a previously cached value.
func (b *DencunBlock) Size() uint64 {
	if size := b.size.Load(); size != nil {
		return size.(uint64)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(uint64(c))
	return uint64(c)
}

// SanityCheck can be used to prevent that unbounded fields are
// stuffed with junk data to add processing overhead
func (b *DencunBlock) SanityCheck() error {
	return b.header.SanityCheck()
}

func CalcDencunUncleHash(uncles []*DencunHeader) common.Hash {
	if len(uncles) == 0 {
		return EmptyUncleHash
	}
	return rlpHash(uncles)
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *DencunBlock) WithSeal(header *DencunHeader) *DencunBlock {
	cpy := *header

	return &DencunBlock{
		header:       &cpy,
		transactions: b.transactions,
		uncles:       b.uncles,
		withdrawals:  b.withdrawals,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *DencunBlock) WithBody(transactions []*Transaction, uncles []*DencunHeader) *DencunBlock {
	block := &DencunBlock{
		header:       CopyDencunHeader(b.header),
		transactions: make([]*Transaction, len(transactions)),
		uncles:       make([]*DencunHeader, len(uncles)),
	}
	copy(block.transactions, transactions)
	for i := range uncles {
		block.uncles[i] = CopyDencunHeader(uncles[i])
	}
	return block
}

// WithWithdrawals sets the withdrawal contents of a block, does not return a new block.
func (b *DencunBlock) WithWithdrawals(withdrawals []*Withdrawal) *DencunBlock {
	if withdrawals != nil {
		b.withdrawals = make([]*Withdrawal, len(withdrawals))
		copy(b.withdrawals, withdrawals)
	}
	return b
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *DencunBlock) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

type DencunBlocks []*DencunBlock
