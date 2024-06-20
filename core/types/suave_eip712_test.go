package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCCREIP712(t *testing.T) {
	to := common.Address{0x2}

	ccr := &ConfidentialComputeRecord{
		GasPrice: big.NewInt(0),
		To:       &to,
		Value:    big.NewInt(0),
		ChainID:  big.NewInt(0),
	}

	_, err := ccr.EIP712Hash()
	require.NoError(t, err)
}
