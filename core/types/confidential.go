package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/suave/apitypes"
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
}

type ConfidentialComputeRequest2 struct {
	// Message is the message we are signed with the EIP-712 envelope
	Message json.RawMessage `json:"message"`
	// Signature is the signature of the message with the EIP-712 envelope
	Signature []byte `json:"signature"`
}

func (c *ConfidentialComputeRequest2) txType() byte {
	return 0x69
}

func (c *ConfidentialComputeRequest2) copy() TxData {
	// lets be lazy here for now
	raw, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	cpy := &ConfidentialComputeRequest2{}
	err = json.Unmarshal(raw, cpy)
	if err != nil {
		panic(err)
	}
	return cpy
}

func (c *ConfidentialComputeRequest2) GetRecord() ConfidentialComputeRecord {
	var record ConfidentialComputeRecord
	if err := json.Unmarshal(c.Message, &record); err != nil {
		panic(err)
	}
	return record
}

func (c *ConfidentialComputeRequest2) chainID() *big.Int {
	return big.NewInt(1)
}

func (c *ConfidentialComputeRequest2) accessList() AccessList {
	return AccessList{}
}

func (c *ConfidentialComputeRequest2) data() []byte {
	return c.GetRecord().Data
}

func (c *ConfidentialComputeRequest2) gas() uint64 {
	return c.GetRecord().Gas
}

func (c *ConfidentialComputeRequest2) gasPrice() *big.Int {
	return c.GetRecord().GasPrice
}

func (c *ConfidentialComputeRequest2) gasTipCap() *big.Int {
	return big.NewInt(1)
}
func (c *ConfidentialComputeRequest2) gasFeeCap() *big.Int {
	return big.NewInt(1)
}
func (c *ConfidentialComputeRequest2) value() *big.Int {
	return c.GetRecord().Value
}
func (c *ConfidentialComputeRequest2) nonce() uint64 {
	return c.GetRecord().Nonce
}
func (c *ConfidentialComputeRequest2) to() *common.Address {
	return c.GetRecord().To
}
func (c *ConfidentialComputeRequest2) blobGas() uint64 {
	return 0
}
func (c *ConfidentialComputeRequest2) blobGasFeeCap() *big.Int {
	return nil
}
func (c *ConfidentialComputeRequest2) blobHashes() []common.Hash {
	return nil
}

func (c *ConfidentialComputeRequest2) rawSignatureValues() (v, r, s *big.Int) {
	return nil, nil, nil
}
func (c *ConfidentialComputeRequest2) setSignatureValues(chainID, v, r, s *big.Int) {
	panic("x")
}
func (c *ConfidentialComputeRequest2) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	panic("x")
}

type SuaveTransaction struct {
	ConfidentialComputeRequest ConfidentialComputeRequest2 `json:"confidentialComputeRequest" gencodec:"required"`
	ConfidentialComputeResult  []byte                      `json:"confidentialComputeResult" gencodec:"required"`

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

func (msg *ConfidentialComputeRecord) Recover(signature []byte) common.Address {
	signHash, _, err := apitypes.TypedDataAndHash(msg.BuildConfidentialRecordEIP712Envelope())
	if err != nil {
		panic(err)
	}
	result, err := crypto.Ecrecover(signHash, signature)
	if err != nil {
		panic(err)
	}

	// THIS WORKS!
	var signer common.Address
	copy(signer[:], crypto.Keccak256(result[1:])[12:])

	return signer
}

func (msg *ConfidentialComputeRecord) BuildConfidentialRecordEIP712Envelope() apitypes.TypedData {
	typ := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
			},
			"ConfidentialRecord": []apitypes.Type{
				{Name: "nonce", Type: "uint64"},
				{Name: "gasPrice", Type: "uint256"},
				{Name: "gas", Type: "uint64"},
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "data", Type: "bytes"},
				{Name: "kettleAddress", Type: "address"},
				{Name: "confidentialInputsHash", Type: "bytes32"},
			},
		},
		Domain: apitypes.TypedDataDomain{
			Name: "ConfidentialRecord",
		},
		PrimaryType: "ConfidentialRecord",
		Message: apitypes.TypedDataMessage{
			"nonce":                  msg.Nonce,
			"gasPrice":               msg.GasPrice,
			"gas":                    msg.Gas,
			"to":                     msg.To,
			"value":                  msg.Value,
			"data":                   msg.Data,
			"kettleAddress":          msg.KettleAddress,
			"confidentialInputsHash": msg.ConfidentialInputsHash,
		},
	}
	return typ
}
