package vm

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"
)

func (b *suaveRuntime) encodeTransaction(txn types.TransactionArgs) ([]byte, error) {
	return (&encodeTransactionPrecompile{}).encodeTransaction(txn)
}

func (b *suaveRuntime) decodeTransaction(txn []byte) (types.TransactionArgs, error) {
	return (&decodeTransactionPrecompile{}).decodeTransaction(txn)
}

func (b *suaveRuntime) marshalBundle(bundle types.Bundle) ([]byte, error) {
	return (&marshalBundlePrecompile{}).marshalBundle(bundle)
}

func (b *suaveRuntime) unmarshalBundle(bundle []byte) (types.Bundle, error) {
	return (&unmarshalBundlePrecompile{}).unmarshalBundle(bundle)
}

type encodeTransactionPrecompile struct{}

type AbiTransactionArgs struct {
	Type                      uint64         "json:\"Type\""
	ChainID                   uint64         "json:\"ChainID\""
	Nonce                     uint64         "json:\"Nonce\""
	To                        common.Address "json:\"To\""
	Gas                       uint64         "json:\"Gas\""
	GasPrice                  uint64         "json:\"GasPrice\""
	MaxPriorityFeePerGas      uint64         "json:\"MaxPriorityFeePerGas\""
	MaxFeePerGas              uint64         "json:\"MaxFeePerGas\""
	MaxFeePerDataGas          uint64         "json:\"MaxFeePerDataGas\""
	Value                     uint64         "json:\"Value\""
	Input                     []uint8        "json:\"Input\""
	AccessList                []uint8        "json:\"AccessList\""
	BlobVersionedHashes       []uint8        "json:\"BlobVersionedHashes\""
	ExecutionNode             common.Address "json:\"ExecutionNode\""
	ConfidentialInputsHash    [32]uint8      "json:\"ConfidentialInputsHash\""
	ConfidentialInputs        []uint8        "json:\"ConfidentialInputs\""
	Wrapped                   []uint8        "json:\"Wrapped\""
	ConfidentialComputeResult []uint8        "json:\"ConfidentialComputeResult\""
	V                         []uint8        "json:\"V\""
	R                         []uint8        "json:\"R\""
	S                         []uint8        "json:\"S\""
}

func (*encodeTransactionPrecompile) RequiredGas(input []byte) uint64 { return 1000 }
func (p *encodeTransactionPrecompile) Run(input []byte) ([]byte, error) {
	mAbi := artifacts.SuaveAbi.Methods[artifacts.PrecompileAddressToName(encodeTransactionAddr)]
	unpackedArgs, err := mAbi.Inputs.Unpack(input)
	if err != nil {
		return nil, err
	}
	txArgs := unpackedArgs[0].(struct {
		Type                      uint64         "json:\"Type\""
		ChainID                   uint64         "json:\"ChainID\""
		Nonce                     uint64         "json:\"Nonce\""
		To                        common.Address "json:\"To\""
		Gas                       uint64         "json:\"Gas\""
		GasPrice                  uint64         "json:\"GasPrice\""
		MaxPriorityFeePerGas      uint64         "json:\"MaxPriorityFeePerGas\""
		MaxFeePerGas              uint64         "json:\"MaxFeePerGas\""
		MaxFeePerDataGas          uint64         "json:\"MaxFeePerDataGas\""
		Value                     uint64         "json:\"Value\""
		Input                     []uint8        "json:\"Input\""
		AccessList                []uint8        "json:\"AccessList\""
		BlobVersionedHashes       []uint8        "json:\"BlobVersionedHashes\""
		ExecutionNode             common.Address "json:\"ExecutionNode\""
		ConfidentialInputsHash    [32]uint8      "json:\"ConfidentialInputsHash\""
		ConfidentialInputs        []uint8        "json:\"ConfidentialInputs\""
		Wrapped                   []uint8        "json:\"Wrapped\""
		ConfidentialComputeResult []uint8        "json:\"ConfidentialComputeResult\""
		V                         []uint8        "json:\"V\""
		R                         []uint8        "json:\"R\""
		S                         []uint8        "json:\"S\""
	})
	txnBytes, err := p.encodeTransaction(types.TransactionArgs{
		Type:                      txArgs.Type,
		ChainID:                   txArgs.ChainID,
		Nonce:                     txArgs.Nonce,
		To:                        txArgs.To,
		Gas:                       txArgs.Gas,
		GasPrice:                  txArgs.GasPrice,
		MaxPriorityFeePerGas:      txArgs.MaxPriorityFeePerGas,
		MaxFeePerGas:              txArgs.MaxFeePerGas,
		MaxFeePerDataGas:          txArgs.MaxFeePerDataGas,
		Value:                     txArgs.Value,
		Input:                     txArgs.Input,
		AccessList:                txArgs.AccessList,
		BlobVersionedHashes:       txArgs.BlobVersionedHashes,
		ExecutionNode:             txArgs.ExecutionNode,
		ConfidentialInputsHash:    txArgs.ConfidentialInputsHash,
		ConfidentialInputs:        txArgs.ConfidentialInputs,
		Wrapped:                   txArgs.Wrapped,
		ConfidentialComputeResult: txArgs.ConfidentialComputeResult,
		V:                         txArgs.V,
		R:                         txArgs.R,
		S:                         txArgs.S,
	})
	if err != nil {
		return nil, err
	}
	return txnBytes, nil
}

func (*encodeTransactionPrecompile) encodeTransaction(txn types.TransactionArgs) ([]byte, error) {
	rpcTx := types.TxJson{
		Type: hexutil.Uint64(txn.Type),

		ChainID:              (*hexutil.Big)(big.NewInt(int64(txn.ChainID))),
		Nonce:                (*hexutil.Uint64)(&txn.Nonce),
		To:                   &txn.To,
		Gas:                  (*hexutil.Uint64)(&txn.Gas),
		GasPrice:             (*hexutil.Big)(big.NewInt(int64(txn.GasPrice))),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(int64(txn.MaxPriorityFeePerGas))),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(int64(txn.MaxFeePerGas))),
		MaxFeePerDataGas:     (*hexutil.Big)(big.NewInt(int64(txn.MaxPriorityFeePerGas))),
		Value:                (*hexutil.Big)(big.NewInt(int64(txn.Value))),
		Input:                (*hexutil.Bytes)(&txn.Input),
		/* AccessList                *AccessList      `json:"accessList,omitempty"`
		/ BlobVersionedHashes       []common.Hash    `json:"blobVersionedHashes,omitempty"`
		/ KettleAddress             *common.Address  `json:"kettleAddress,omitempty"`
		/ ConfidentialInputsHash    *common.Hash     `json:"confidentialInputsHash,omitempty"`
		/ ConfidentialInputs        *hexutil.Bytes   `json:"confidentialInputs,omitempty"`
		/ Wrapped                   *json.RawMessage `json:"wrapped,omitempty"`
		/ ConfidentialComputeResult *hexutil.Bytes   `json:"confidentialComputeResult,omitempty"`
		*/
		V: &hexutil.Big{},
		R: &hexutil.Big{},
		S: &hexutil.Big{},
	}
	// There probably are ways to avoid marshaling twice
	rpcMarshaledBytes, err := json.Marshal(rpcTx)
	if err != nil {
		return nil, err
	}
	tx := types.Transaction{}
	err = tx.UnmarshalJSON(rpcMarshaledBytes)
	if err != nil {
		return nil, err
	}
	return tx.MarshalBinary()
}

type decodeTransactionPrecompile struct{}

func (*decodeTransactionPrecompile) RequiredGas(input []byte) uint64 { return 1000 }

func (p *decodeTransactionPrecompile) Run(input []byte) ([]byte, error) {
	mAbi := artifacts.SuaveAbi.Methods[artifacts.PrecompileAddressToName(decodeTransactionAddr)]
	unpackedArgs, err := mAbi.Inputs.Unpack(input)
	if err != nil {
		return nil, err
	}
	txArgs, err := p.decodeTransaction(unpackedArgs[0].([]byte))
	if err != nil {
		return nil, err
	}
	return mAbi.Outputs.Pack(txArgs)
}

func (*decodeTransactionPrecompile) decodeTransaction(txnBytes []byte) (types.TransactionArgs, error) {
	tx := types.Transaction{}
	err := tx.UnmarshalBinary(txnBytes)
	if err != nil {
		return types.TransactionArgs{}, err
	}

	var toAddr common.Address
	if tx.To() != nil {
		toAddr = *tx.To()
	}

	txArgs := types.TransactionArgs{
		Type:       uint64(tx.Type()),
		Nonce:      tx.Nonce(),
		To:         toAddr,
		Gas:        tx.Gas(),
		Input:      tx.Data(),
		AccessList: nil,
		/*
			/ BlobVersionedHashes       []byte
			/ ExecutionNode             common.Address
			/ ConfidentialInputsHash    common.Hash
			/ ConfidentialInputs        []byte
			/ Wrapped                   []byte
			/ ConfidentialComputeResult []byte
			V                         []byte
			R                         []byte
			S                         []byte
		*/
	}

	if tx.ChainId() != nil {
		txArgs.ChainID = tx.ChainId().Uint64()
	}

	if tx.GasPrice() != nil {
		txArgs.GasPrice = tx.GasPrice().Uint64()
	}

	if tx.GasTipCap() != nil {
		txArgs.MaxPriorityFeePerGas = tx.GasTipCap().Uint64()
	}

	if tx.GasFeeCap() != nil {
		txArgs.MaxFeePerGas = tx.GasFeeCap().Uint64()
	}

	if tx.BlobGasFeeCap() != nil {
		txArgs.MaxFeePerDataGas = tx.BlobGasFeeCap().Uint64()
	}

	if tx.Value() != nil {
		txArgs.Value = tx.Value().Uint64()
	}

	return txArgs, nil
}

type marshalBundlePrecompile struct{}

func (*marshalBundlePrecompile) RequiredGas(input []byte) uint64 { return 1000 }

func (p *marshalBundlePrecompile) Run(input []byte) ([]byte, error) {
	mAbi := artifacts.SuaveAbi.Methods[artifacts.PrecompileAddressToName(marshalBundleAddr)]
	unpackedArgs, err := mAbi.Inputs.Unpack(input)
	if err != nil {
		return nil, err
	}
	bundleBytes, err := p.marshalBundle(unpackedArgs[0].(types.Bundle))
	if err != nil {
		return nil, err
	}
	return bundleBytes, nil
}

func (*marshalBundlePrecompile) marshalBundle(bundle types.Bundle) ([]byte, error) {
	txs := make([]hexutil.Bytes, len(bundle.Txs))
	for i, tx := range bundle.Txs {
		txs[i] = hexutil.Bytes(tx)
	}
	return json.Marshal(types.RpcSBundle{
		BlockNumber:     (*hexutil.Big)(big.NewInt(int64(bundle.BlockNumber))),
		Txs:             txs,
		RevertingHashes: bundle.RevertingHashes,
	})
}

type unmarshalBundlePrecompile struct{}

func (*unmarshalBundlePrecompile) RequiredGas(input []byte) uint64 { return 1000 }

func (p *unmarshalBundlePrecompile) Run(input []byte) ([]byte, error) {
	mAbi := artifacts.SuaveAbi.Methods[artifacts.PrecompileAddressToName(unmarshalBundleAddr)]
	unpackedArgs, err := mAbi.Inputs.Unpack(input)
	if err != nil {
		return nil, err
	}
	bundle, err := p.unmarshalBundle(unpackedArgs[0].([]byte))
	if err != nil {
		return nil, err
	}
	return mAbi.Outputs.Pack(bundle)
}

func (*unmarshalBundlePrecompile) unmarshalBundle(bundleBytes []byte) (types.Bundle, error) {
	rpcBundle := types.RpcSBundle{}
	err := json.Unmarshal(bundleBytes, &rpcBundle)
	if err != nil {
		return types.Bundle{}, err
	}

	txs := make([][]byte, len(rpcBundle.Txs))
	for i, tx := range rpcBundle.Txs {
		txs[i] = ([]byte)(tx)
	}
	return types.Bundle{
		BlockNumber:     rpcBundle.BlockNumber.ToInt().Uint64(),
		Txs:             txs,
		RevertingHashes: rpcBundle.RevertingHashes,
	}, nil
}
