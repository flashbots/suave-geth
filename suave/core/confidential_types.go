package suave

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func BuildConfidentialRecordEIP712Envelope(msg *types.ConfidentialComputeRecord) apitypes.TypedData {
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
