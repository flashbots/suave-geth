package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/eip712"
)

func (msg *ConfidentialComputeRecord) EIP712Hash() (common.Hash, error) {
	hash, _, err := eip712.TypedDataAndHash(CCREIP712Envelope(msg))

	hash32 := common.Hash{}
	copy(hash32[:], hash[:])

	return hash32, err
}

func CCREIP712Envelope(msg *ConfidentialComputeRecord) eip712.TypedData {
	return eip712.TypedData{
		Types: eip712.Types{
			"EIP712Domain": []eip712.Type{
				{Name: "name", Type: "string"},
				{Name: "verifyingContract", Type: "address"},
			},
			"ConfidentialRecord": []eip712.Type{
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
		Domain: eip712.TypedDataDomain{
			Name:              "ConfidentialRecord",
			VerifyingContract: msg.KettleAddress.Hex(),
		},
		PrimaryType: "ConfidentialRecord",
		Message: eip712.TypedDataMessage{
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
}
