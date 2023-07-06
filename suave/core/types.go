package suave

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type Bytes = hexutil.Bytes
type BidId = [16]byte

type Bid struct {
	Id                  BidId            `json:"id"`
	DecryptionCondition uint64           `json:"decryptionCondition"` // For now simply the block number. Should be either derived from the source contract, or be a contract itself
	AllowedPeekers      []common.Address `json:"allowedPeekers"`
}

var ConfStoreAllowedAny common.Address = common.HexToAddress("0x42")

type ConfiendialStoreBackend interface {
	Initialize(bid Bid, key string, value []byte) (Bid, error)
	Store(bidId BidId, caller common.Address, key string, value []byte) (Bid, error)
	Retrieve(bid BidId, caller common.Address, key string) ([]byte, error)
}

type MempoolBackend interface {
	SubmitBid(Bid) error
	FetchBids(blockNumber uint64) []Bid
	FetchBidById(BidId) (Bid, error)
}

type OffchainEthBackend interface {
	BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
}

type BuildBlockArgs = types.BuildBlockArgs
