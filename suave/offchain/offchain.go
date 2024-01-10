package offchain

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/suave/datastore"
	"github.com/ipfs/go-cid"
)

type Datastore interface {
	Get(context.Context, cid.Cid) (io.Reader, error)
	Put(context.Context, io.Reader) (cid.Cid, error)
}

type Env struct {
	Store Datastore
}

func (env *Env) Start() error {
	if env.Store == nil {
		env.Store = &datastore.IPFS{}
	}

	return nil
}

func (env *Env) Stop() error {
	return nil
}
