package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
		RefundPercent   int                `json:"percent"`
		MatchId         [16]byte           `json:"MatchId"`
	}{}

	if err := json.NewDecoder(os.Stdin).Decode(&bundle); err != nil {
		die(err)
	}

	tx := bundle.Txs[0]
	hint := struct {
		To   common.Address
		Data []byte
	}{
		To:   *tx.To(),
		Data: tx.Data(),
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(hint); err != nil {
		die(err)
	}

	_, err := io.Copy(os.Stdout, buf)
	die(err)
}

func die(err error) {
	if err == nil {
		os.Exit(0)
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
