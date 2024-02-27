package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ConfidentialComputeRecord struct {
	Nonce    uint64
	GasPrice *big.Int
	Gas      uint64
	To       *common.Address `rlp:"nil"`
	Value    *big.Int
	Data     []byte

	KettleAddress          common.Address
	ConfidentialInputsHash common.Hash

	ChainID *big.Int
	V, R, S *big.Int
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *ConfidentialComputeRecord) copy() TxData {
	cpy := &ConfidentialComputeRecord{
		Nonce:                  tx.Nonce,
		To:                     copyAddressPtr(tx.To),
		Data:                   common.CopyBytes(tx.Data),
		Gas:                    tx.Gas,
		KettleAddress:          tx.KettleAddress,
		ConfidentialInputsHash: tx.ConfidentialInputsHash,

		Value:    new(big.Int),
		GasPrice: new(big.Int),

		ChainID: new(big.Int),
		V:       new(big.Int),
		R:       new(big.Int),
		S:       new(big.Int),
	}

	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.GasPrice != nil {
		cpy.GasPrice.Set(tx.GasPrice)
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

func (tx *ConfidentialComputeRecord) txType() byte              { return ConfidentialComputeRecordTxType }
func (tx *ConfidentialComputeRecord) chainID() *big.Int         { return tx.ChainID }
func (tx *ConfidentialComputeRecord) accessList() AccessList    { return nil }
func (tx *ConfidentialComputeRecord) data() []byte              { return tx.Data }
func (tx *ConfidentialComputeRecord) gas() uint64               { return tx.Gas }
func (tx *ConfidentialComputeRecord) gasPrice() *big.Int        { return tx.GasPrice }
func (tx *ConfidentialComputeRecord) gasTipCap() *big.Int       { return tx.GasPrice }
func (tx *ConfidentialComputeRecord) gasFeeCap() *big.Int       { return tx.GasPrice }
func (tx *ConfidentialComputeRecord) value() *big.Int           { return tx.Value }
func (tx *ConfidentialComputeRecord) nonce() uint64             { return tx.Nonce }
func (tx *ConfidentialComputeRecord) to() *common.Address       { return tx.To }
func (tx *ConfidentialComputeRecord) blobGas() uint64           { return 0 }
func (tx *ConfidentialComputeRecord) blobGasFeeCap() *big.Int   { return nil }
func (tx *ConfidentialComputeRecord) blobHashes() []common.Hash { return nil }

func (tx *ConfidentialComputeRecord) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.GasPrice)
}

func (tx *ConfidentialComputeRecord) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *ConfidentialComputeRecord) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

type ConfidentialComputeRequest struct {
	ConfidentialComputeRecord
	ConfidentialInputs []byte
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *ConfidentialComputeRequest) copy() TxData {
	cpy := &ConfidentialComputeRequest{
		ConfidentialComputeRecord: *(tx.ConfidentialComputeRecord.copy().(*ConfidentialComputeRecord)),
		ConfidentialInputs:        tx.ConfidentialInputs,
	}

	return cpy
}

func (tx *ConfidentialComputeRequest) txType() byte              { return ConfidentialComputeRequestTxType }
func (tx *ConfidentialComputeRequest) chainID() *big.Int         { return tx.ChainID }
func (tx *ConfidentialComputeRequest) accessList() AccessList    { return nil }
func (tx *ConfidentialComputeRequest) data() []byte              { return tx.Data }
func (tx *ConfidentialComputeRequest) gas() uint64               { return tx.Gas }
func (tx *ConfidentialComputeRequest) gasPrice() *big.Int        { return tx.GasPrice }
func (tx *ConfidentialComputeRequest) gasTipCap() *big.Int       { return tx.GasPrice }
func (tx *ConfidentialComputeRequest) gasFeeCap() *big.Int       { return tx.GasPrice }
func (tx *ConfidentialComputeRequest) value() *big.Int           { return tx.Value }
func (tx *ConfidentialComputeRequest) nonce() uint64             { return tx.Nonce }
func (tx *ConfidentialComputeRequest) to() *common.Address       { return tx.To }
func (tx *ConfidentialComputeRequest) blobGas() uint64           { return 0 }
func (tx *ConfidentialComputeRequest) blobGasFeeCap() *big.Int   { return nil }
func (tx *ConfidentialComputeRequest) blobHashes() []common.Hash { return nil }

func (tx *ConfidentialComputeRequest) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.GasPrice)
}

func (tx *ConfidentialComputeRequest) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *ConfidentialComputeRequest) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}

type SuaveTransaction struct {
	ConfidentialComputeRequest ConfidentialComputeRecord `json:"confidentialComputeRequest" gencodec:"required"`
	ConfidentialComputeResult  []byte                    `json:"confidentialComputeResult" gencodec:"required"`

	// request KettleAddress's signature
	ChainID *big.Int
	V       *big.Int
	R       *big.Int
	S       *big.Int
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *SuaveTransaction) copy() TxData {
	cpy := &SuaveTransaction{
		ConfidentialComputeRequest: tx.ConfidentialComputeRequest,
		ConfidentialComputeResult:  common.CopyBytes(tx.ConfidentialComputeResult),
		ChainID:                    new(big.Int),
		V:                          new(big.Int),
		R:                          new(big.Int),
		S:                          new(big.Int),
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
func (tx *SuaveTransaction) txType() byte {
	return SuaveTxType
}

func (tx *SuaveTransaction) data() []byte {
	return tx.ConfidentialComputeResult
}

// Rest is carried over from wrapped tx
func (tx *SuaveTransaction) chainID() *big.Int { return tx.ChainID }
func (tx *SuaveTransaction) accessList() AccessList {
	return tx.ConfidentialComputeRequest.accessList()
}
func (tx *SuaveTransaction) gas() uint64 { return tx.ConfidentialComputeRequest.gas() }
func (tx *SuaveTransaction) gasFeeCap() *big.Int {
	return tx.ConfidentialComputeRequest.gasFeeCap()
}
func (tx *SuaveTransaction) gasTipCap() *big.Int {
	return tx.ConfidentialComputeRequest.gasTipCap()
}
func (tx *SuaveTransaction) gasPrice() *big.Int {
	return tx.ConfidentialComputeRequest.gasFeeCap()
}
func (tx *SuaveTransaction) value() *big.Int     { return tx.ConfidentialComputeRequest.value() }
func (tx *SuaveTransaction) nonce() uint64       { return tx.ConfidentialComputeRequest.nonce() }
func (tx *SuaveTransaction) to() *common.Address { return tx.ConfidentialComputeRequest.to() }
func (tx *SuaveTransaction) blobGas() uint64     { return tx.ConfidentialComputeRequest.blobGas() }
func (tx *SuaveTransaction) blobGasFeeCap() *big.Int {
	return tx.ConfidentialComputeRequest.blobGasFeeCap()
}
func (tx *SuaveTransaction) blobHashes() []common.Hash {
	return tx.ConfidentialComputeRequest.blobHashes()
}

func (tx *SuaveTransaction) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return tx.ConfidentialComputeRequest.effectiveGasPrice(dst, baseFee)
}

func (tx *SuaveTransaction) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *SuaveTransaction) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.ChainID, tx.V, tx.R, tx.S = chainID, v, r, s
}
