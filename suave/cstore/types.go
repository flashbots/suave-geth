package cstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/google/uuid"
)

type Bytes = hexutil.Bytes
type BidId = types.BidId

type Bid struct {
	Id                  types.BidId
	Salt                types.BidId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
	CreationTx          *types.Transaction
	Signature           []byte
}

func (b *Bid) ToInnerBid() types.Bid {
	return types.Bid{
		Id:                  b.Id,
		Salt:                b.Salt,
		DecryptionCondition: b.DecryptionCondition,
		AllowedPeekers:      b.AllowedPeekers,
		AllowedStores:       b.AllowedStores,
		Version:             b.Version,
	}
}

type MEVMBid = types.Bid

type BuildBlockArgs = types.BuildBlockArgs

var ConfStoreAllowedAny common.Address = common.HexToAddress("0x42")

var (
	ErrBidAlreadyPresent = errors.New("bid already present")
	ErrUnsignedFinalize  = errors.New("finalize called with unsigned transaction, refusing to propagate")
)

type DASigner interface {
	Sign(account common.Address, data []byte) ([]byte, error)
	Sender(data []byte, signature []byte) (common.Address, error)
	LocalAddresses() []common.Address
}

type ChainSigner interface {
	Sender(tx *types.Transaction) (common.Address, error)
}

type ConfidentialStoreBackend interface {
	node.Lifecycle

	InitializeBid(bid Bid) error
	Store(bid Bid, caller common.Address, key string, value []byte) (Bid, error)
	Retrieve(bid Bid, caller common.Address, key string) ([]byte, error)
	FetchBidById(BidId) (Bid, error)
	FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []Bid
}

type ConfidentialEthBackend interface {
	BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
	BuildEthBlockFromBundles(ctx context.Context, args *BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error)
}

type StoreTransportTopic interface {
	node.Lifecycle
	Subscribe() (<-chan DAMessage, context.CancelFunc)
	Publish(DAMessage)
}

type DAMessage struct {
	SourceTx    *types.Transaction `json:"sourceTx"`
	StoreWrites []StoreWrite       `json:"storeWrites"`
	StoreUUID   uuid.UUID          `json:"storeUUID"`
	Signature   Bytes              `json:"signature"`
}

type StoreWrite struct {
	Bid    Bid            `json:"bid"`
	Caller common.Address `json:"caller"`
	Key    string         `json:"key"`
	Value  Bytes          `json:"value"`
}

var bidUuidSpace = uuid.UUID{0x42}
var emptyId [16]byte

func calculateBidId(bid types.Bid) (types.BidId, error) {
	copy(bid.Id[:], emptyId[:])

	body, err := json.Marshal(bid)
	if err != nil {
		return types.BidId{}, fmt.Errorf("could not marshal bid to calculate its id: %w", err)
	}

	uuidv5 := uuid.NewSHA1(bidUuidSpace, body)
	copy(bid.Id[:], uuidv5[:])

	return bid.Id, nil
}

func RandomBidId() types.BidId {
	return types.BidId(uuid.New())
}

func MustEncode[T any](data T) []byte {
	res, err := json.Marshal(data)
	if err != nil {
		panic(err.Error())
	}
	return res
}

func MustDecode[T any](data []byte) T {
	var t T
	if err := json.Unmarshal(data, &t); err != nil {
		panic(err.Error())
	}
	return t
}
