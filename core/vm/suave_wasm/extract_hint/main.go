package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"

	suave_lib "github.com/ethereum/go-ethereum/core/vm/suave_wasm/lib"
)

func main() {
	unpacked, err := suave_lib.UnpackInputs(artifacts.SuaveAbi.Methods["extractHint"].Inputs)
	if err != nil {
		suave_lib.Fail(err)
	}

	bundleBytes := unpacked[0].([]byte)

	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
		RefundPercent   int                `json:"percent"`
		MatchId         types.BidId        `json:"MatchId"`
	}{}

	err = json.Unmarshal(bundleBytes, &bundle)
	if err != nil {
		io.Copy(os.Stdout, bytes.NewBuffer([]byte(err.Error())))
		os.Exit(2)
	}

	tx := bundle.Txs[0]
	hint := struct {
		To   common.Address
		Data []byte
	}{
		To:   *tx.To(),
		Data: tx.Data(),
	}

	hintBytes, err := json.Marshal(hint)
	if err != nil {
		suave_lib.Fail(err)
	}

	suave_lib.ReturnBytes(hintBytes)
}
