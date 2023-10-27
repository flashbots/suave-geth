package vm

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var bidIdsAbi = mustParseMethodAbi(`[{"inputs": [{ "type": "bytes16[]" }], "name": "bidids", "outputs":[], "type": "function"}]`, "bidids")

func mustParseAbi(data string) abi.ABI {
	inoutAbi, err := abi.JSON(strings.NewReader(data))
	if err != nil {
		panic(err.Error())
	}

	return inoutAbi
}

func mustParseMethodAbi(data string, method string) abi.Method {
	inoutAbi := mustParseAbi(data)
	return inoutAbi.Methods[method]
}
