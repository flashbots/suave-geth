package suave

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/beacon/dencun"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	builder "github.com/ethereum/go-ethereum/suave/builder/api"
)

var AllowedPeekerAny = common.HexToAddress("0xC8df3686b4Afb2BB53e60EAe97EF043FE03Fb829") // "*"

type Bytes = hexutil.Bytes
type DataId = types.DataId

type DataRecord struct {
	Id                  types.DataId
	Salt                types.DataId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
	CreationTx          *types.Transaction
	Signature           []byte
}

func (b *DataRecord) ToInnerRecord() types.DataRecord {
	return types.DataRecord{
		Id:                  b.Id,
		Salt:                b.Salt,
		DecryptionCondition: b.DecryptionCondition,
		AllowedPeekers:      b.AllowedPeekers,
		AllowedStores:       b.AllowedStores,
		Version:             b.Version,
	}
}

type MEVMBid = types.DataRecord

type BuildBlockArgs = types.BuildBlockArgs

var ConfStoreAllowedAny common.Address = common.HexToAddress("0x42")

var (
	ErrRecordAlreadyPresent = errors.New("data record already present")
	ErrRecordNotFound       = errors.New("data record not found")
	ErrUnsignedFinalize     = errors.New("finalize called with unsigned transaction, refusing to propagate")
)

type ConfidentialStoreBackend interface {
	node.Lifecycle

	InitializeBid(record DataRecord) error
	Store(record DataRecord, caller common.Address, key string, value []byte) (DataRecord, error)
	Retrieve(record DataRecord, caller common.Address, key string) ([]byte, error)
	FetchBidById(DataId) (DataRecord, error)
	FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []DataRecord
}

type ConfidentialEthBackend interface {
	BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*dencun.ExecutionPayloadEnvelope, error)
	BuildEthBlockFromBundles(ctx context.Context, args *BuildBlockArgs, bundles []types.SBundle) (*dencun.ExecutionPayloadEnvelope, error)
	Call(ctx context.Context, contractAddr common.Address, input []byte) ([]byte, error)
	ChainID(ctx context.Context) (*big.Int, error)

	builder.API
}
