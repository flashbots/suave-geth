package offchain

import (
	"bytes"
	"context"
	"io"

	"github.com/ipfs/boxo/path"
	"github.com/ipfs/go-cid"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/ipfs/kubo/core/coreiface/options"
	"github.com/multiformats/go-multihash"
)

type Blockstore struct {
	API iface.BlockAPI
}

func (b Blockstore) Get(ctx context.Context, cid cid.Cid) ([]byte, error) {
	r, err := b.API.Get(ctx, path.FromCid(cid))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(r)
}

func (b Blockstore) Put(ctx context.Context, p []byte) (cid.Cid, error) {
	bs, err := b.API.Put(ctx, bytes.NewReader(p),
		// options.Block.Pin(false),  // TODO:  refcounting
		options.Block.Hash(multihash.BLAKE3, 512))
	if err != nil {
		return cid.Cid{}, err
	}

	return bs.Path().RootCid(), nil
}
