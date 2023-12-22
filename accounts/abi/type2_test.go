package abi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewType2FromString(t *testing.T) {
	type struct1 struct {
		A uint64
		B []struct {
			X struct {
				C uint64
			}
			D uint64
		}
		E struct {
			F uint64
			G uint64
		}
	}

	cases := []struct {
		input string
		want  string
		obj   interface{}
	}{
		{
			"uint64[]",
			"uint64[]",
			[]uint64{1, 2, 3},
		},
		{
			"tuple(a uint64, b uint64)",
			"(uint64,uint64)",
			&struct {
				A uint64
				B uint64
			}{A: 1, B: 2},
		},
		{
			// empty tuple
			"tuple()",
			"()",
			&struct{}{},
		},
		{
			// anonymous type
			"tuple(uint64,tuple(uint64),uint64)",
			"(uint64,(uint64),uint64)",
			&struct {
				Arg0 uint64
				Arg1 struct {
					Arg0 uint64
				}
				Arg2 uint64
			}{Arg0: 1, Arg1: struct{ Arg0 uint64 }{Arg0: 2}, Arg2: 3},
		},
		{
			"tuple(a uint64, b tuple[](x tuple(c uint64), d uint64), e tuple(f uint64, g uint64))",
			"(uint64,((uint64),uint64)[],(uint64,uint64))",
			&struct1{
				A: 1,
				B: []struct {
					X struct {
						C uint64
					}
					D uint64
				}{
					{
						X: struct {
							C uint64
						}{C: 2},
						D: 3,
					},
					{
						X: struct {
							C uint64
						}{C: 4},
						D: 5,
					},
				},
				E: struct {
					F uint64
					G uint64
				}{F: 6, G: 7},
			},
		},
	}

	for _, c := range cases {
		typ, err := NewTypeFromString(c.input)

		require.NoError(t, err)
		require.Equal(t, c.want, typ.String())

		_, err = typ.Pack(c.obj)
		require.NoError(t, err)
	}
}
