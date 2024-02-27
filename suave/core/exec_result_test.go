package suave

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestExecResult_ABIEncoding(t *testing.T) {
	cases := []*ExecResult{
		{
			Logs: []*types.Log{
				{
					Address: common.Address{0x1},
				},
			},
		},
	}

	for _, c := range cases {
		data, err := c.EncodeABI()
		if err != nil {
			t.Errorf("Error encoding ABI: %v", err)
		}
		decoded := new(ExecResult)
		if err := decoded.DecodeABI(data); err != nil {
			t.Errorf("Error decoding ABI: %v", err)
		}
		if !c.Equal(decoded) {
			t.Errorf("Decoded result is not equal to original: %v != %v", c, decoded)
		}
	}
}
