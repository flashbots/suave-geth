package offchain

import (
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"
)

type Env struct {
	IPFS iface.CoreAPI
}

func (env *Env) Start() (err error) {
	// Bind IPFS into the off-chain environment
	env.IPFS, err = rpc.NewLocalApi()
	return
}

func (env *Env) Stop() error {
	return nil
}

func (env *Env) Blockstore() Blockstore {
	return Blockstore{
		API: env.IPFS.Block(),
	}
}
