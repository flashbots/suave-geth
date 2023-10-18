package vm

import "github.com/ethereum/go-ethereum/accounts/abi"

var bidIdsAbi = mustParseMethodAbi(`[{"inputs": [{ "type": "bytes16[]" }], "name": "bidids", "outputs":[], "type": "function"}]`, "bidids")

func mustParseMethodAbi(data string, method string) abi.Method {
	inoutAbi := mustParseAbi(data)
	return inoutAbi.Methods[method]
}
