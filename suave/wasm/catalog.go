package wasm

import (
	_ "embed"

	"github.com/ethereum/go-ethereum/common"
)

var libraries = map[common.Address][]byte{
	{0x1, 0x2, 0x3}: addLib,
}

//go:embed catalog/add/add.wasm
var addLib []byte
