package blockbuilding

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suavesdk"
)

// sdk for block building
type Builder struct {
	*suavesdk.Suapp

	txnsTopic *suavesdk.Topic
}

func NewBuilder(opts ...suavesdk.Option) *Builder {
	suapp := suavesdk.NewSuapp(
		opts...,
	)

	topic := suapp.NewTopic("txns")

	return &Builder{
		Suapp:     suapp,
		txnsTopic: topic,
	}
}

func (b *Builder) AddTxn(txn []byte) {
	b.txnsTopic.Publish(txn)
}

func (b *Builder) StreamTxns(handle func(txn *types.Transaction)) {
	b.txnsTopic.Subscribe(func(data []byte) {
		txn := types.Transaction{}
		if err := json.Unmarshal(data, &txn); err != nil {
			return
		}

		handle(&txn)
	})
}
