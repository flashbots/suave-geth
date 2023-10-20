package main

import (
	"github.com/ethereum/go-ethereum/suave/artifacts"

	suave_lib "github.com/ethereum/go-ethereum/core/vm/suave_wasm/lib"
)

func main() {
	unpacked, err := suave_lib.UnpackInputs(artifacts.SuaveAbi.Methods["confidentialStoreRetrieve"].Inputs)
	if err != nil {
		suave_lib.Fail(err)
	}

	bidId := unpacked[0].([16]byte)
	key := unpacked[1].(string)

	data, err := suave_lib.StoreRetrieve(bidId, key)
	if err != nil {
		suave_lib.Fail(err)
	}

	suave_lib.ReturnBytes(data)
}
