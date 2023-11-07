package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestCCRequestToRecord(t *testing.T) {
	testKey, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)

	signer := NewSuaveSigner(new(big.Int))
	unsignedTx := NewTx(&ConfidentialComputeRequest{
		ConfidentialComputeRecord: ConfidentialComputeRecord{
			KettleAddress: crypto.PubkeyToAddress(testKey.PublicKey),
		},
		ConfidentialInputs: []byte{0x46},
	})
	signedTx, err := SignTx(unsignedTx, signer, testKey)
	require.NoError(t, err)

	recoveredSender, err := signer.Sender(signedTx)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(testKey.PublicKey), recoveredSender)

	marshalledTxBytes, err := signedTx.MarshalBinary()
	require.NoError(t, err)

	unmarshalledTx := new(Transaction)
	require.NoError(t, unmarshalledTx.UnmarshalBinary(marshalledTxBytes))

	recoveredUnmarshalledSender, err := signer.Sender(unmarshalledTx)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(testKey.PublicKey), recoveredUnmarshalledSender)

	signedRequestInner, ok := CastTxInner[*ConfidentialComputeRequest](unmarshalledTx)
	require.True(t, ok)

	recoveredRecordSender, err := signer.Sender(NewTx(&signedRequestInner.ConfidentialComputeRecord))
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(testKey.PublicKey), recoveredRecordSender)
}

func TestCCR(t *testing.T) {
	testKey, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)

	signer := NewSuaveSigner(new(big.Int))
	unsignedTx := NewTx(&ConfidentialComputeRequest{})
	signedTx, err := SignTx(unsignedTx, signer, testKey)
	require.NoError(t, err)

	recoveredSender, err := signer.Sender(signedTx)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(testKey.PublicKey), recoveredSender)

	marshalledTxBytes, err := signedTx.MarshalBinary()
	require.NoError(t, err)

	unmarshalledTx := new(Transaction)
	require.NoError(t, unmarshalledTx.UnmarshalBinary(marshalledTxBytes))

	recoveredUnmarshalledSender, err := signer.Sender(unmarshalledTx)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(testKey.PublicKey), recoveredUnmarshalledSender)
}

func TestSuaveTx(t *testing.T) {
	testKey, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	require.NoError(t, err)

	signer := NewSuaveSigner(new(big.Int))

	signedCCR, err := SignTx(NewTx(&ConfidentialComputeRecord{
		KettleAddress: crypto.PubkeyToAddress(testKey.PublicKey),
	}), signer, testKey)
	require.NoError(t, err)

	signedInnerCCR, ok := CastTxInner[*ConfidentialComputeRecord](signedCCR)
	require.True(t, ok)

	unsignedTx := NewTx(&SuaveTransaction{
		ConfidentialComputeRequest: *signedInnerCCR,
	})
	signedTx, err := SignTx(unsignedTx, signer, testKey)
	require.NoError(t, err)

	recoveredSender, err := signer.Sender(signedTx)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(testKey.PublicKey), recoveredSender)

	marshalledTxBytes, err := signedTx.MarshalBinary()
	require.NoError(t, err)

	unmarshalledTx := new(Transaction)
	require.NoError(t, unmarshalledTx.UnmarshalBinary(marshalledTxBytes))

	recoveredUnmarshalledSender, err := signer.Sender(unmarshalledTx)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(testKey.PublicKey), recoveredUnmarshalledSender)
}
