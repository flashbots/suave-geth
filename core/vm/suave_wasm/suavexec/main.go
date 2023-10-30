package main

import (
	"bytes"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/suave_wasm/lib"
)

var (
	bid = types.BidId{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}
	key = "someKey"
)

func main() {
	data, err := lib.StoreRetrieve(bid, key)
	if err != nil {
		os.Exit(1)
	}

	io.Copy(os.Stdout, bytes.NewReader(data[:]))
}
