package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDispatch_Example(t *testing.T) {
	d := NewDispatchTable()
	d.MustRegister(&testPrecompile{})

	out, err := d.packAndRun(nil, "testPrecompile", uint64(1))
	require.NoError(t, err)
	require.Equal(t, uint64(11), out[0])
}

type testPrecompile struct{}

func (t *testPrecompile) Do(ctx *SuaveContext, input uint64) (uint64, error) {
	return input + 10, nil
}

func (t *testPrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (t *testPrecompile) Address() common.Address {
	return common.Address{}
}
