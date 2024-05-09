package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMossDispatcher(t *testing.T) {
	addr1 := common.Address{0x1}

	d := NewDispatchTable()
	d.MustRegister(&backend{})

	require.Equal(t, d.namespaces[addr1].methods["do"].method.Sig, "do(uint64)")

	out, err := d.packAndRun(addr1, "do", uint64(1))
	require.NoError(t, err)
	require.Equal(t, uint64(11), out[0])

	out, err = d.packAndRun(addr1, "do2")
	require.NoError(t, err)
	require.Equal(t, common.Address{0x2}, out[0])

	out, err = d.packAndRun(addr1, "do3")
	require.NoError(t, err)
	require.Equal(t, big.NewInt(3), out[0])

	_, err = d.packAndRun(addr1, "do4", Do4Input{Addr: common.Address{}})
	require.NoError(t, err)
}

type backend struct{}

func (b *backend) Do(input uint64) (uint64, error) {
	return input + 10, nil
}

func (b *backend) Do2() (common.Address, error) {
	return common.Address{0x2}, nil
}

func (b *backend) Do3() (*big.Int, error) {
	return big.NewInt(3), nil
}

type Do4Input struct {
	Addr common.Address
}

func (b *backend) Do4(input *Do4Input) error {
	return nil
}

func (b *backend) Address() common.Address {
	return common.Address{0x1}
}
