package suave

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type Bytes = hexutil.Bytes
type BidId = [32]byte

type Bid struct {
	Id                  BidId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
	CreationTx          *types.Transaction
	Signature           []byte
}

type MEVMBid = types.Bid

type BuildBlockArgs = types.BuildBlockArgs

var ConfStoreAllowedAny common.Address = common.HexToAddress("0x42")

var BidAlreadyPresentError = errors.New("bid already present")

type ConfidentialStoreBackend interface {
	InitializeBid(bid Bid) error
	FetchEngineBidById(bidId BidId) (Bid, error)
	Store(bidId BidId, caller common.Address, key string, value []byte) (Bid, error)
	Retrieve(bidId BidId, caller common.Address, key string) ([]byte, error)
}

type MempoolBackend interface {
	SubmitBid(types.Bid) error
	FetchBidById(BidId) (types.Bid, error)
	FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []types.Bid
}

type OffchainEthBackend interface {
	BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
	BuildEthBlockFromBundles(ctx context.Context, args *BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error)
}

type PubSub interface {
	Subscribe() <-chan DAMessage
	Publish(DAMessage)
}

type DAMessage struct {
	Bid       Bid                `json:"bid"`
	SourceTx  *types.Transaction `json:"sourceTx"`
	Caller    common.Address     `json:"caller"`
	Key       string             `json:"key"`
	Value     Bytes              `json:"value"`
	Signature Bytes              `json:"signature"`
}
