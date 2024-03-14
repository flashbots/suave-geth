package suavesdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDispatch_Example(t *testing.T) {
	d := NewDispatchTable()
	d.MustRegister(&backend{})

	require.Equal(t, d.methods["do"].method.Sig, "do(uint64)")

	out, err := d.packAndRun("do", uint64(1))
	require.NoError(t, err)
	require.Equal(t, uint64(11), out[0])
}

type backend struct{}

func (b *backend) Do(input uint64) (uint64, error) {
	return input + 10, nil
}
