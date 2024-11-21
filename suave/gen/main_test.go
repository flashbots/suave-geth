package main

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/stretchr/testify/require"
)

func TestToAddressName(t *testing.T) {
	cases := []struct {
		name     string
		expected string
	}{
		{"newBid", "NEW_BID"},
		{"confidentialRetrieve", "CONFIDENTIAL_RETRIEVE"},
	}

	for _, c := range cases {
		actual := toAddressName(c.name)
		require.Equal(t, c.expected, actual)
	}
}

func TestEncodeTypeToGolang(t *testing.T) {
	cases := []struct {
		name     string
		expected string
	}{
		{"uint256", "*big.Int"},
		{"address", "common.Address"},
		{"bool", "bool"},
		{"bytes", "[]byte"},
		{"bytes32", "common.Hash"},
		{"bytes16", "[16]byte"},
		{"string", "string"},
		{"address[]", "[]common.Address"},
		{"Bid", "Bid"},
		{"Bid[]", "[]*Bid"},
	}

	for _, c := range cases {
		actual := encodeTypeToGolang(c.name, true, true, false)
		require.Equal(t, c.expected, actual)
	}
}

func TestDecodeABI_PeekerReverted(t *testing.T) {
	errMsg, err := hex.DecodeString("75fff4670000000000000000000000000000000000000000000000000000000042100000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000036261640000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	errorEvnt := artifacts.SuaveAbi.Errors["PeekerReverted"]
	vals, err := errorEvnt.Inputs.Unpack(errMsg[4:])
	require.NoError(t, err)

	addr := vals[0].(common.Address)
	reason := vals[1].([]byte)

	require.Equal(t, addr.String(), "0x0000000000000000000000000000000042100000")
	require.Equal(t, string(reason), "bad")
}
