package main

import (
	"github.com/ethereum/go-ethereum/suave/artifacts"

	suave_lib "github.com/ethereum/go-ethereum/core/vm/suave_wasm/lib"
)

func main() {
	unpacked, err := suave_lib.UnpackInputs(artifacts.SuaveAbi.Methods["confidentialStoreStore"].Inputs)
	if err != nil {
		suave_lib.Fail(err)
	}

	bidId := unpacked[0].([16]byte)
	key := unpacked[1].(string)
	value := unpacked[2].([]byte)

	_, err = suave_lib.Store(bidId, key, value)
	if err != nil {
		suave_lib.Fail(err)
	}

	// no return
}
