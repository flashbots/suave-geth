package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/stretchr/testify/require"
)

func TestEncodeTransaction(t *testing.T) {
	pAbi := artifacts.SuaveAbi.Methods["encodeTransaction"]
	packedInput, err := pAbi.Inputs.Pack(types.TransactionArgs{
		Type:     types.LegacyTxType,
		ChainID:  7331,
		Nonce:    15,
		To:       common.Address{0x5, 0x4, 0x3, 0x2, 0x1, 0x0},
		Gas:      21000,
		GasPrice: 1000,
		Value:    1000,
		Input:    []byte{0x71},
	})
	require.NoError(t, err)
	outp, err := (&encodeTransactionPrecompile{}).Run(packedInput)
	require.NoError(t, err)

	recoveredTx := types.Transaction{}
	require.NoError(t, recoveredTx.UnmarshalBinary(outp))

	require.Equal(t, uint8(types.LegacyTxType), recoveredTx.Type())
	require.Equal(t, uint64(15), recoveredTx.Nonce())
	require.Equal(t, common.Address{0x5, 0x4, 0x3, 0x2, 0x1, 0x0}, *recoveredTx.To())
	require.Equal(t, uint64(21000), recoveredTx.Gas())
	require.Equal(t, uint64(1000), recoveredTx.GasPrice().Uint64())
	require.Equal(t, uint64(1000), recoveredTx.Value().Uint64())
	require.Equal(t, []byte{0x71}, recoveredTx.Data())
}

func TestDecodeTransaction(t *testing.T) {
	pAbi := artifacts.SuaveAbi.Methods["decodeTransaction"]

	tx := types.NewTransaction(15, common.Address{0x5, 0x4, 0x3, 0x2, 0x1, 0x0}, big.NewInt(10), 21000, big.NewInt(100), []byte{0x16})
	inputTxBytes, err := tx.MarshalBinary()
	require.NoError(t, err)

	packedInput, err := pAbi.Inputs.Pack(inputTxBytes)
	require.NoError(t, err)
	outp, err := (&decodeTransactionPrecompile{}).Run(packedInput)
	require.NoError(t, err)

	unpackedOutput, err := pAbi.Outputs.Unpack(outp)
	require.NoError(t, err)

	recoveredTxArgs := unpackedOutput[0].(struct {
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

	require.Equal(t, uint64(types.LegacyTxType), recoveredTxArgs.Type)
	require.Equal(t, uint64(15), recoveredTxArgs.Nonce)
	require.Equal(t, common.Address{0x5, 0x4, 0x3, 0x2, 0x1, 0x0}, recoveredTxArgs.To)
	require.Equal(t, uint64(21000), recoveredTxArgs.Gas)
	require.Equal(t, uint64(100), recoveredTxArgs.GasPrice)
	require.Equal(t, uint64(10), recoveredTxArgs.Value)
	require.Equal(t, []byte{0x16}, recoveredTxArgs.Input)
}
