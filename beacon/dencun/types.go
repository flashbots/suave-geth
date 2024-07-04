// Copyright 2022 The go-ethereum Authors
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

package dencun

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/trie"
)

// PayloadAttributes describes the environment context in which a block should
// be built.
type PayloadAttributes struct {
	Timestamp             uint64              `json:"timestamp"             gencodec:"required"`
	Random                common.Hash         `json:"prevRandao"            gencodec:"required"`
	SuggestedFeeRecipient common.Address      `json:"suggestedFeeRecipient" gencodec:"required"`
	Withdrawals           []*types.Withdrawal `json:"withdrawals"`
	BeaconRoot            *common.Hash        `json:"parentBeaconBlockRoot"`
}

// ExecutableData is the data necessary to execute an EL payload.
type ExecutableData struct {
	ParentHash    common.Hash         `json:"parentHash"    gencodec:"required"`
	FeeRecipient  common.Address      `json:"feeRecipient"  gencodec:"required"`
	StateRoot     common.Hash         `json:"stateRoot"     gencodec:"required"`
	ReceiptsRoot  common.Hash         `json:"receiptsRoot"  gencodec:"required"`
	LogsBloom     []byte              `json:"logsBloom"     gencodec:"required"`
	Random        common.Hash         `json:"prevRandao"    gencodec:"required"`
	Number        uint64              `json:"blockNumber"   gencodec:"required"`
	GasLimit      uint64              `json:"gasLimit"      gencodec:"required"`
	GasUsed       uint64              `json:"gasUsed"       gencodec:"required"`
	Timestamp     uint64              `json:"timestamp"     gencodec:"required"`
	ExtraData     []byte              `json:"extraData"     gencodec:"required"`
	BaseFeePerGas *big.Int            `json:"baseFeePerGas" gencodec:"required"`
	BlockHash     common.Hash         `json:"blockHash"     gencodec:"required"`
	Transactions  [][]byte            `json:"transactions"  gencodec:"required"`
	Withdrawals   []*types.Withdrawal `json:"withdrawals"`
	BlobGasUsed   *uint64             `json:"blobGasUsed"`
	ExcessBlobGas *uint64             `json:"excessBlobGas"`
}

type ExecutionPayloadEnvelope struct {
	ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
	BlockValue       *big.Int        `json:"blockValue"  gencodec:"required"`
	BlobsBundle      *BlobsBundleV1  `json:"blobsBundle"`
	Override         bool            `json:"shouldOverrideBuilder"`
}

// MarshalJSON marshals as JSON.
func (e ExecutionPayloadEnvelope) MarshalJSON() ([]byte, error) {
	type ExecutionPayloadEnvelope struct {
		ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
		BlockValue       *hexutil.Big    `json:"blockValue"  gencodec:"required"`
		BlobsBundle      *BlobsBundleV1  `json:"blobsBundle"`
		Override         bool            `json:"shouldOverrideBuilder"`
	}
	var enc ExecutionPayloadEnvelope
	enc.ExecutionPayload = e.ExecutionPayload
	enc.BlockValue = (*hexutil.Big)(e.BlockValue)
	enc.BlobsBundle = e.BlobsBundle
	enc.Override = e.Override
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (e *ExecutionPayloadEnvelope) UnmarshalJSON(input []byte) error {
	type ExecutionPayloadEnvelope struct {
		ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
		BlockValue       *hexutil.Big    `json:"blockValue"  gencodec:"required"`
		BlobsBundle      *BlobsBundleV1  `json:"blobsBundle"`
		Override         *bool           `json:"shouldOverrideBuilder"`
	}
	var dec ExecutionPayloadEnvelope
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ExecutionPayload == nil {
		return errors.New("missing required field 'executionPayload' for ExecutionPayloadEnvelope")
	}
	e.ExecutionPayload = dec.ExecutionPayload
	if dec.BlockValue == nil {
		return errors.New("missing required field 'blockValue' for ExecutionPayloadEnvelope")
	}
	e.BlockValue = (*big.Int)(dec.BlockValue)
	if dec.BlobsBundle != nil {
		e.BlobsBundle = dec.BlobsBundle
	}
	if dec.Override != nil {
		e.Override = *dec.Override
	}
	return nil
}

type BlobsBundleV1 struct {
	Commitments []hexutil.Bytes `json:"commitments"`
	Proofs      []hexutil.Bytes `json:"proofs"`
	Blobs       []hexutil.Bytes `json:"blobs"`
}

type PayloadStatusV1 struct {
	Status          string       `json:"status"`
	LatestValidHash *common.Hash `json:"latestValidHash"`
	ValidationError *string      `json:"validationError"`
}

type TransitionConfigurationV1 struct {
	TerminalTotalDifficulty *hexutil.Big   `json:"terminalTotalDifficulty"`
	TerminalBlockHash       common.Hash    `json:"terminalBlockHash"`
	TerminalBlockNumber     hexutil.Uint64 `json:"terminalBlockNumber"`
}

// PayloadID is an identifier of the payload build process
type PayloadID [8]byte

func (b PayloadID) String() string {
	return hexutil.Encode(b[:])
}

func (b PayloadID) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b *PayloadID) UnmarshalText(input []byte) error {
	err := hexutil.UnmarshalFixedText("PayloadID", input, b[:])
	if err != nil {
		return fmt.Errorf("invalid payload id %q: %w", input, err)
	}
	return nil
}

type ForkChoiceResponse struct {
	PayloadStatus PayloadStatusV1 `json:"payloadStatus"`
	PayloadID     *PayloadID      `json:"payloadId"`
}

type ForkchoiceStateV1 struct {
	HeadBlockHash      common.Hash `json:"headBlockHash"`
	SafeBlockHash      common.Hash `json:"safeBlockHash"`
	FinalizedBlockHash common.Hash `json:"finalizedBlockHash"`
}

func encodeTransactions(txs []*types.Transaction) [][]byte {
	var enc = make([][]byte, len(txs))
	for i, tx := range txs {
		enc[i], _ = tx.MarshalBinary()
	}
	return enc
}

func decodeTransactions(enc [][]byte) ([]*types.Transaction, error) {
	var txs = make([]*types.Transaction, len(enc))
	for i, encTx := range enc {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encTx); err != nil {
			return nil, fmt.Errorf("invalid transaction %d: %v", i, err)
		}
		txs[i] = &tx
	}
	return txs, nil
}

// ExecutableDataToBlock constructs a block from executable data.
// It verifies that the following fields:
//
//		len(extraData) <= 32
//		uncleHash = emptyUncleHash
//		difficulty = 0
//	 	if versionedHashes != nil, versionedHashes match to blob transactions
//
// and that the blockhash of the constructed block matches the parameters. Nil
// Withdrawals value will propagate through the returned block. Empty
// Withdrawals value must be passed via non-nil, length 0 value in params.
func ExecutableDataToBlock(params ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash) (*types.DencunBlock, error) {
	txs, err := decodeTransactions(params.Transactions)
	if err != nil {
		return nil, err
	}
	if len(params.ExtraData) > 32 {
		return nil, fmt.Errorf("invalid extradata length: %v", len(params.ExtraData))
	}
	if len(params.LogsBloom) != 256 {
		return nil, fmt.Errorf("invalid logsBloom length: %v", len(params.LogsBloom))
	}
	// Check that baseFeePerGas is not negative or too big
	if params.BaseFeePerGas != nil && (params.BaseFeePerGas.Sign() == -1 || params.BaseFeePerGas.BitLen() > 256) {
		return nil, fmt.Errorf("invalid baseFeePerGas: %v", params.BaseFeePerGas)
	}
	var blobHashes []common.Hash
	for _, tx := range txs {
		blobHashes = append(blobHashes, tx.BlobHashes()...)
	}
	if len(blobHashes) != len(versionedHashes) {
		return nil, fmt.Errorf("invalid number of versionedHashes: %v blobHashes: %v", versionedHashes, blobHashes)
	}
	for i := 0; i < len(blobHashes); i++ {
		if blobHashes[i] != versionedHashes[i] {
			return nil, fmt.Errorf("invalid versionedHash at %v: %v blobHashes: %v", i, versionedHashes, blobHashes)
		}
	}
	// Only set withdrawalsRoot if it is non-nil. This allows CLs to use
	// ExecutableData before withdrawals are enabled by marshaling
	// Withdrawals as the json null value.
	var withdrawalsRoot *common.Hash
	if params.Withdrawals != nil {
		h := types.DeriveSha(types.Withdrawals(params.Withdrawals), trie.NewStackTrie(nil))
		withdrawalsRoot = &h
	}
	header := &types.DencunHeader{
		ParentHash:       params.ParentHash,
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         params.FeeRecipient,
		Root:             params.StateRoot,
		TxHash:           types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash:      params.ReceiptsRoot,
		Bloom:            types.BytesToBloom(params.LogsBloom),
		Difficulty:       common.Big0,
		Number:           new(big.Int).SetUint64(params.Number),
		GasLimit:         params.GasLimit,
		GasUsed:          params.GasUsed,
		Time:             params.Timestamp,
		BaseFee:          params.BaseFeePerGas,
		Extra:            params.ExtraData,
		MixDigest:        params.Random,
		WithdrawalsHash:  withdrawalsRoot,
		ExcessBlobGas:    params.ExcessBlobGas,
		BlobGasUsed:      params.BlobGasUsed,
		ParentBeaconRoot: beaconRoot,
	}
	block := types.NewBlockWithDencunHeader(header).WithBody(txs, nil /* uncles */).WithWithdrawals(params.Withdrawals)
	if block.Hash() != params.BlockHash {
		return nil, fmt.Errorf("blockhash mismatch, want %x, got %x", params.BlockHash, block.Hash())
	}
	return block, nil
}

// BlobTxSidecar contains the blobs of a blob transaction.
type BlobTxSidecar struct {
	Blobs       []kzg4844.Blob       // Blobs needed by the blob pool
	Commitments []kzg4844.Commitment // Commitments needed by the blob pool
	Proofs      []kzg4844.Proof      // Proofs needed by the blob pool
}

// BlockToExecutableData constructs the ExecutableData structure by filling the
// fields from the given block. It assumes the given block is post-merge block.
func BlockToExecutableData(block *types.Block, fees *big.Int, sidecars []*BlobTxSidecar) *ExecutionPayloadEnvelope {
	data := &ExecutableData{
		BlockHash:     block.Hash(),
		ParentHash:    block.ParentHash(),
		FeeRecipient:  block.Coinbase(),
		StateRoot:     block.Root(),
		Number:        block.NumberU64(),
		GasLimit:      block.GasLimit(),
		GasUsed:       block.GasUsed(),
		BaseFeePerGas: block.BaseFee(),
		Timestamp:     block.Time(),
		ReceiptsRoot:  block.ReceiptHash(),
		LogsBloom:     block.Bloom().Bytes(),
		Transactions:  encodeTransactions(block.Transactions()),
		Random:        block.MixDigest(),
		ExtraData:     block.Extra(),
		Withdrawals:   block.Withdrawals(),
		BlobGasUsed:   nil,
		ExcessBlobGas: nil,
	}
	bundle := BlobsBundleV1{
		Commitments: make([]hexutil.Bytes, 0),
		Blobs:       make([]hexutil.Bytes, 0),
		Proofs:      make([]hexutil.Bytes, 0),
	}
	for _, sidecar := range sidecars {
		for j := range sidecar.Blobs {
			bundle.Blobs = append(bundle.Blobs, hexutil.Bytes(sidecar.Blobs[j][:]))
			bundle.Commitments = append(bundle.Commitments, hexutil.Bytes(sidecar.Commitments[j][:]))
			bundle.Proofs = append(bundle.Proofs, hexutil.Bytes(sidecar.Proofs[j][:]))
		}
	}
	return &ExecutionPayloadEnvelope{ExecutionPayload: data, BlockValue: fees, BlobsBundle: &bundle, Override: false}
}

// ExecutionPayloadBodyV1 is used in the response to GetPayloadBodiesByHashV1 and GetPayloadBodiesByRangeV1
type ExecutionPayloadBodyV1 struct {
	TransactionData []hexutil.Bytes     `json:"transactions"`
	Withdrawals     []*types.Withdrawal `json:"withdrawals"`
}

// var _ = (*executableDataMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (e ExecutableData) MarshalJSON() ([]byte, error) {
	type ExecutableData struct {
		ParentHash    common.Hash         `json:"parentHash"    gencodec:"required"`
		FeeRecipient  common.Address      `json:"feeRecipient"  gencodec:"required"`
		StateRoot     common.Hash         `json:"stateRoot"     gencodec:"required"`
		ReceiptsRoot  common.Hash         `json:"receiptsRoot"  gencodec:"required"`
		LogsBloom     hexutil.Bytes       `json:"logsBloom"     gencodec:"required"`
		Random        common.Hash         `json:"prevRandao"    gencodec:"required"`
		Number        hexutil.Uint64      `json:"blockNumber"   gencodec:"required"`
		GasLimit      hexutil.Uint64      `json:"gasLimit"      gencodec:"required"`
		GasUsed       hexutil.Uint64      `json:"gasUsed"       gencodec:"required"`
		Timestamp     hexutil.Uint64      `json:"timestamp"     gencodec:"required"`
		ExtraData     hexutil.Bytes       `json:"extraData"     gencodec:"required"`
		BaseFeePerGas *hexutil.Big        `json:"baseFeePerGas" gencodec:"required"`
		BlockHash     common.Hash         `json:"blockHash"     gencodec:"required"`
		Transactions  []hexutil.Bytes     `json:"transactions"  gencodec:"required"`
		Withdrawals   []*types.Withdrawal `json:"withdrawals"`
		BlobGasUsed   *hexutil.Uint64     `json:"blobGasUsed"`
		ExcessBlobGas *hexutil.Uint64     `json:"excessBlobGas"`
	}
	var enc ExecutableData
	enc.ParentHash = e.ParentHash
	enc.FeeRecipient = e.FeeRecipient
	enc.StateRoot = e.StateRoot
	enc.ReceiptsRoot = e.ReceiptsRoot
	enc.LogsBloom = e.LogsBloom
	enc.Random = e.Random
	enc.Number = hexutil.Uint64(e.Number)
	enc.GasLimit = hexutil.Uint64(e.GasLimit)
	enc.GasUsed = hexutil.Uint64(e.GasUsed)
	enc.Timestamp = hexutil.Uint64(e.Timestamp)
	enc.ExtraData = e.ExtraData
	enc.BaseFeePerGas = (*hexutil.Big)(e.BaseFeePerGas)
	enc.BlockHash = e.BlockHash
	if e.Transactions != nil {
		enc.Transactions = make([]hexutil.Bytes, len(e.Transactions))
		for k, v := range e.Transactions {
			enc.Transactions[k] = v
		}
	}
	enc.Withdrawals = e.Withdrawals
	enc.BlobGasUsed = (*hexutil.Uint64)(e.BlobGasUsed)
	enc.ExcessBlobGas = (*hexutil.Uint64)(e.ExcessBlobGas)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (e *ExecutableData) UnmarshalJSON(input []byte) error {
	type ExecutableData struct {
		ParentHash    *common.Hash        `json:"parentHash"    gencodec:"required"`
		FeeRecipient  *common.Address     `json:"feeRecipient"  gencodec:"required"`
		StateRoot     *common.Hash        `json:"stateRoot"     gencodec:"required"`
		ReceiptsRoot  *common.Hash        `json:"receiptsRoot"  gencodec:"required"`
		LogsBloom     *hexutil.Bytes      `json:"logsBloom"     gencodec:"required"`
		Random        *common.Hash        `json:"prevRandao"    gencodec:"required"`
		Number        *hexutil.Uint64     `json:"blockNumber"   gencodec:"required"`
		GasLimit      *hexutil.Uint64     `json:"gasLimit"      gencodec:"required"`
		GasUsed       *hexutil.Uint64     `json:"gasUsed"       gencodec:"required"`
		Timestamp     *hexutil.Uint64     `json:"timestamp"     gencodec:"required"`
		ExtraData     *hexutil.Bytes      `json:"extraData"     gencodec:"required"`
		BaseFeePerGas *hexutil.Big        `json:"baseFeePerGas" gencodec:"required"`
		BlockHash     *common.Hash        `json:"blockHash"     gencodec:"required"`
		Transactions  []hexutil.Bytes     `json:"transactions"  gencodec:"required"`
		Withdrawals   []*types.Withdrawal `json:"withdrawals"`
		BlobGasUsed   *hexutil.Uint64     `json:"blobGasUsed"`
		ExcessBlobGas *hexutil.Uint64     `json:"excessBlobGas"`
	}
	var dec ExecutableData
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for ExecutableData")
	}
	e.ParentHash = *dec.ParentHash
	if dec.FeeRecipient == nil {
		return errors.New("missing required field 'feeRecipient' for ExecutableData")
	}
	e.FeeRecipient = *dec.FeeRecipient
	if dec.StateRoot == nil {
		return errors.New("missing required field 'stateRoot' for ExecutableData")
	}
	e.StateRoot = *dec.StateRoot
	if dec.ReceiptsRoot == nil {
		return errors.New("missing required field 'receiptsRoot' for ExecutableData")
	}
	e.ReceiptsRoot = *dec.ReceiptsRoot
	if dec.LogsBloom == nil {
		return errors.New("missing required field 'logsBloom' for ExecutableData")
	}
	e.LogsBloom = *dec.LogsBloom
	if dec.Random == nil {
		return errors.New("missing required field 'prevRandao' for ExecutableData")
	}
	e.Random = *dec.Random
	if dec.Number == nil {
		return errors.New("missing required field 'blockNumber' for ExecutableData")
	}
	e.Number = uint64(*dec.Number)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for ExecutableData")
	}
	e.GasLimit = uint64(*dec.GasLimit)
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for ExecutableData")
	}
	e.GasUsed = uint64(*dec.GasUsed)
	if dec.Timestamp == nil {
		return errors.New("missing required field 'timestamp' for ExecutableData")
	}
	e.Timestamp = uint64(*dec.Timestamp)
	if dec.ExtraData == nil {
		return errors.New("missing required field 'extraData' for ExecutableData")
	}
	e.ExtraData = *dec.ExtraData
	if dec.BaseFeePerGas == nil {
		return errors.New("missing required field 'baseFeePerGas' for ExecutableData")
	}
	e.BaseFeePerGas = (*big.Int)(dec.BaseFeePerGas)
	if dec.BlockHash == nil {
		return errors.New("missing required field 'blockHash' for ExecutableData")
	}
	e.BlockHash = *dec.BlockHash
	if dec.Transactions == nil {
		return errors.New("missing required field 'transactions' for ExecutableData")
	}
	e.Transactions = make([][]byte, len(dec.Transactions))
	for k, v := range dec.Transactions {
		e.Transactions[k] = v
	}
	if dec.Withdrawals != nil {
		e.Withdrawals = dec.Withdrawals
	}
	if dec.BlobGasUsed != nil {
		e.BlobGasUsed = (*uint64)(dec.BlobGasUsed)
	}
	if dec.ExcessBlobGas != nil {
		e.ExcessBlobGas = (*uint64)(dec.ExcessBlobGas)
	}
	return nil
}
