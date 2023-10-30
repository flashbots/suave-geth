package wasi

import "github.com/ethereum/go-ethereum/core/types"

type StoreHostFnArgs struct {
	BidId types.BidId
	Key   string
	Value []byte
}

type RetrieveHostFnArgs struct {
	BidId types.BidId
	Key   string
}

type FetchBidByProtocolFnArgs struct {
	BlockNumber uint64
	Namespace   string
}
