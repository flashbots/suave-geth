package consolelog

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestConsoleLog(t *testing.T) {
	data := emitConsoleLog("log(address)", common.Address{})
	val, err := decode(data)
	require.NoError(t, err)
	require.NotNil(t, val)
}

func emitConsoleLog(typStr string, args interface{}) []byte {
	// decode the type and encode the arguments
	typ, err := abi.NewTypeFromString(typStr[strings.Index(typStr, "("):])
	if err != nil {
		panic(err)
	}

	// pack the arguments
	data, err := typ.Pack(args)
	if err != nil {
		panic(err)
	}

	sig := crypto.Keccak256Hash([]byte(typStr))

	buf := make([]byte, 0)
	buf = append(buf, sig[:4]...)
	buf = append(buf, data...)

	return buf
}
