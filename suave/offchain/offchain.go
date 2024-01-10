package offchain

import (
	"context"
	"io"

	"github.com/ethereum/go-ethereum/suave/datastore"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"
)

type Datastore interface {
	Get(context.Context, cid.Cid) (io.Reader, error)
	Put(context.Context, io.Reader) (cid.Cid, error)
}

type Env struct {
	Core  iface.CoreAPI
	Store Datastore
}

func (env *Env) Start() (err error) {
	if env.Store == nil {
		// Import IPFS into the off-chain environment
		if env.Core, err = rpc.NewLocalApi(); err == nil {
			env.Store = &datastore.IPFS{
				API: env.Core,
			}
		}

	}

	return
}

func (env *Env) Stop() error {
	return nil
}
