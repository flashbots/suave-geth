package datastore

import (
	"context"
	"io"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/coreiface/options"
	"github.com/multiformats/go-multihash"
)

type IPFS struct {
	API iface.CoreAPI
}

func (c *IPFS) Get(ctx context.Context, cid cid.Cid) (io.Reader, error) {
	p := path.FromCid(cid)
	return c.API.Block().Get(ctx, p)
}

func (c *IPFS) Put(ctx context.Context, r io.Reader) (cid.Cid, error) {
	bs, err := c.API.Block().Put(ctx, r,
		// options.Block.Pin(false),  // TODO:  refcounting
		options.Block.Hash(multihash.BLAKE3, 512))
	if err != nil {
		return cid.Cid{}, err
	}

	return bs.Path().RootCid(), nil
}
