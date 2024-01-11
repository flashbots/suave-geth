package offchain

import (
	"context"
	"io"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/coreiface/options"
	"github.com/multiformats/go-multihash"
)

type Datastore struct {
	API iface.BlockAPI
}

func (d Datastore) Get(ctx context.Context, cid cid.Cid) (io.Reader, error) {
	p := path.FromCid(cid)
	return d.API.Get(ctx, p)
}

func (d Datastore) Put(ctx context.Context, r io.Reader) (cid.Cid, error) {
	bs, err := d.API.Put(ctx, r,
		// options.Block.Pin(false),  // TODO:  refcounting
		options.Block.Hash(multihash.BLAKE3, 512))
	if err != nil {
		return cid.Cid{}, err
	}

	return bs.Path().RootCid(), nil
}
