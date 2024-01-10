package datastore

import (
	"context"
	"io"
	"sync"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/kubo/client/rpc"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/coreiface/options"
	"github.com/multiformats/go-multihash"
)

type IPFS struct {
	once sync.Once
	API  iface.BlockAPI
}

func (c *IPFS) init() (err error) {
	c.once.Do(func() {
		if c.API == nil {
			var api *rpc.HttpApi
			if api, err = rpc.NewLocalApi(); err == nil {
				c.API = api.Block()
			}
		}
	})
	return
}

func (c *IPFS) Get(ctx context.Context, cid cid.Cid) (io.Reader, error) {
	if err := c.init(); err != nil {
		return nil, err
	}

	p := path.FromCid(cid)
	return c.API.Get(ctx, p)
}

func (c *IPFS) Put(ctx context.Context, r io.Reader) (cid.Cid, error) {
	if err := c.init(); err != nil {
		return cid.Cid{}, err
	}

	bs, err := c.API.Put(ctx, r,
		// options.Block.Pin(false),  // TODO:  refcounting
		options.Block.Hash(multihash.BLAKE3, 512))
	if err != nil {
		return cid.Cid{}, err
	}

	return bs.Path().RootCid(), nil
}
