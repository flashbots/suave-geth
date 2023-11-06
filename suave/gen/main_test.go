package main

import (
	"testing"

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
		actual := encodeTypeToGolang(c.name, true, true)
		require.Equal(t, c.expected, actual)
	}
}
