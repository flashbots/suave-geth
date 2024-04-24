package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMossDispatcher(t *testing.T) {
	addr1 := common.Address{0x1}

	d := NewDispatchTable()
	d.MustRegister(&backend{})

	require.Equal(t, d.methods[addr1]["do"].method.Sig, "do(uint64)")

	out, err := d.packAndRun(addr1, "do", uint64(1))
	require.NoError(t, err)
	require.Equal(t, uint64(11), out[0])
}

type backend struct{}

func (b *backend) Do(input uint64) (uint64, error) {
	return input + 10, nil
}

func (b *backend) Address() common.Address {
	return common.Address{0x1}
}
