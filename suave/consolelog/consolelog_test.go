package consolelog

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo/abi"
)

func TestConsoleLog(t *testing.T) {
	cases := []struct {
		typStr string
		args   interface{}
	}{
		{
			"log(address)", []interface{}{common.Address{}},
		},
		{
			"log(bytes)", []interface{}{[]byte{0x1, 0x2, 0x3}},
		},
	}

	for _, c := range cases {
		data := emitConsoleLog(c.typStr, c.args)
		val, err := decode(data)
		require.NoError(t, err)
		require.NotNil(t, val)
	}
}

func emitConsoleLog(typStr string, args interface{}) []byte {
	// decode the type and encode the arguments
	typ, err := abi.NewType(typStr[strings.Index(typStr, "("):])
	if err != nil {
		panic(err)
	}

	// pack the arguments
	data, err := typ.Encode(args)
	if err != nil {
		panic(err)
	}

	sig := crypto.Keccak256Hash([]byte(typStr))

	buf := make([]byte, 0)
	buf = append(buf, sig[:4]...)
	buf = append(buf, data...)

	return buf
}
