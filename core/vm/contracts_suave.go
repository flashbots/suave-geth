package vm

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

var (
	confStorePrecompileStoreMeter    = metrics.NewRegisteredMeter("suave/confstore/store", nil)
	confStorePrecompileRetrieveMeter = metrics.NewRegisteredMeter("suave/confstore/retrieve", nil)
)

var (
	isConfidentialAddress     = common.HexToAddress("0x42010000")
	confidentialInputsAddress = common.HexToAddress("0x42010001")

	confStoreStoreAddress    = common.HexToAddress("0x42020000")
	confStoreRetrieveAddress = common.HexToAddress("0x42020001")

	newBidAddress    = common.HexToAddress("0x42030000")
	fetchBidsAddress = common.HexToAddress("0x42030001")
)

/* General utility precompiles */

type isConfidentialPrecompile struct{}

func (c *isConfidentialPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *isConfidentialPrecompile) Name() string {
	return "isConfidential"
}

func (c *isConfidentialPrecompile) Address() common.Address {
	return isConfidentialAddress
}

func (c *isConfidentialPrecompile) Do(suaveContext *SuaveContext) ([]byte, error) {
	return []byte{0x1}, nil
}

type confidentialInputsPrecompile struct{}

func (c *confidentialInputsPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *confidentialInputsPrecompile) Address() common.Address {
	return confidentialInputsAddress
}

func (c *confidentialInputsPrecompile) Name() string {
	return "confidentialInputs"
}

func (c *confidentialInputsPrecompile) Do(suaveContext *SuaveContext) ([]byte, error) {
	return suaveContext.ConfidentialInputs, nil
}

/* Confidential store precompiles */

type confStoreStore struct {
}

func (c *confStoreStore) RequiredGas(input []byte) uint64 {
	return uint64(100 * len(input))
}

func (c *confStoreStore) Name() string {
	return "confidentialStoreStore"
}

func (c *confStoreStore) Address() common.Address {
	return confStoreStoreAddress
}

func (c *confStoreStore) Do(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
	return c.runImpl(suaveContext, bidId, key, data)
}

func (c *confStoreStore) runImpl(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
	bid, err := suaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
	if err != nil {
		return suave.ErrBidNotFound
	}

	log.Info("confStoreStore", "bidId", bidId, "key", key)

	caller, err := checkIsPrecompileCallAllowed(suaveContext, confStoreStoreAddress, bid)
	if err != nil {
		return err
	}

	if metrics.Enabled {
		confStorePrecompileStoreMeter.Mark(int64(len(data)))
	}

	_, err = suaveContext.Backend.ConfidentialStore.Store(bidId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

type confStoreRetrieve struct{}

func (c *confStoreRetrieve) RequiredGas(input []byte) uint64 {
	return 100
}

func (c *confStoreRetrieve) Address() common.Address {
	return confStoreRetrieveAddress
}

func (c *confStoreRetrieve) Do(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
	return c.runImpl(suaveContext, bidId, key)
}

func (c *confStoreRetrieve) Name() string {
	return "confidentialStoreRetrieve"
}

func (c *confStoreRetrieve) runImpl(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
	bid, err := suaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
	if err != nil {
		return nil, suave.ErrBidNotFound
	}

	caller, err := checkIsPrecompileCallAllowed(suaveContext, confStoreRetrieveAddress, bid)
	if err != nil {
		return nil, err
	}

	data, err := suaveContext.Backend.ConfidentialStore.Retrieve(bidId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
	}

	return data, nil
}

/* Bid precompiles */

type newBid struct {
}

func (c *newBid) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *newBid) Name() string {
	return "newBid"
}

func (c *newBid) Address() common.Address {
	return newBidAddress
}

func (c *newBid) Do(suaveContext *SuaveContext, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, version string) (*types.Bid, error) {
	return c.runImpl(suaveContext, version, decryptionCondition, allowedPeekers, allowedStores)
}

func (c *newBid) runImpl(suaveContext *SuaveContext, version string, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address) (*types.Bid, error) {
	if suaveContext.ConfidentialComputeRequestTx == nil {
		panic("newBid: source transaction not present")
	}

	bid, err := suaveContext.Backend.ConfidentialStore.InitializeBid(types.Bid{
		Salt:                suave.RandomBidId(),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		AllowedStores:       allowedStores,
		Version:             version, // TODO : make generic
	})
	if err != nil {
		return nil, err
	}

	return &bid, nil
}

type fetchBids struct {
}

func (c *fetchBids) Name() string {
	return "fetchBids"
}

func (c *fetchBids) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *fetchBids) Address() common.Address {
	return fetchBidsAddress
}

func (c *fetchBids) Do(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
	return c.runImpl(suaveContext, targetBlock, namespace)
}

func (c *fetchBids) runImpl(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
	bids1 := suaveContext.Backend.ConfidentialStore.FetchBidsByProtocolAndBlock(targetBlock, namespace)

	bids := make([]types.Bid, 0, len(bids1))
	for _, bid := range bids1 {
		bids = append(bids, bid.ToInnerBid())
	}

	return bids, nil
}

func mustParseAbi(data string) abi.ABI {
	inoutAbi, err := abi.JSON(strings.NewReader(data))
	if err != nil {
		panic(err.Error())
	}

	return inoutAbi
}

func mustParseMethodAbi(data string, method string) abi.Method {
	inoutAbi := mustParseAbi(data)
	return inoutAbi.Methods[method]
}

func formatPeekerError(format string, args ...any) ([]byte, error) {
	err := fmt.Errorf(format, args...)
	return []byte(err.Error()), err
}
