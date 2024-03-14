package main

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suavesdk"
	"github.com/ethereum/go-ethereum/suavesdk/blockbuilding"
)

func main() {
	// THIS ONE IS A BUILDER
	b := &builder{}

	builder := blockbuilding.NewBuilder(
		suavesdk.WithFunction(b),
	)

	builder.StreamTxns(b.HandleTxn)
}

type builder struct {
}

func (b *builder) HandleTxn(txn *types.Transaction) {
	// go crazy
}
