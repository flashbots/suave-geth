package vm

import (
	"errors"
	"fmt"

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
	isConfidentialAddress               = common.HexToAddress("0x42010000")
	errIsConfidentialInvalidInputLength = errors.New("invalid input length")

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

func (c *isConfidentialPrecompile) Run(input []byte) ([]byte, error) {
	if len(input) == 1 {
		// The precompile was called *directly* confidentially, and the result was cached - return 1
		if input[0] == 0x01 {
			return []byte{0x01}, nil
		} else {
			return nil, errors.New("incorrect value passed in")
		}
	}

	if len(input) > 1 {
		return nil, errIsConfidentialInvalidInputLength
	}

	return []byte{0x00}, nil
}

func (c *isConfidentialPrecompile) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	if len(input) != 0 {
		return nil, errIsConfidentialInvalidInputLength
	}
	return []byte{0x01}, nil
}

type confidentialInputsPrecompile struct{}

func (c *confidentialInputsPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *confidentialInputsPrecompile) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confidentialInputsPrecompile) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	return suaveContext.ConfidentialInputs, nil
}

/* Confidential store precompiles */

func confStoreStoreImpl(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
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

func confStoreRetrieveImpl(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
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

func newBidImpl(suaveContext *SuaveContext, version string, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address) (*types.Bid, error) {
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

func fetchBidsImpl(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
	bids1 := suaveContext.Backend.ConfidentialStore.FetchBidsByProtocolAndBlock(targetBlock, namespace)

	bids := make([]types.Bid, 0, len(bids1))
	for _, bid := range bids1 {
		bids = append(bids, bid.ToInnerBid())
	}

	return bids, nil
}

func formatPeekerError(format string, args ...any) ([]byte, error) {
	err := fmt.Errorf(format, args...)
	return []byte(err.Error()), err
}

type suaveRuntime struct {
	suaveContext *SuaveContext
}

var _ SuaveRuntime = &suaveRuntime{}

func (b *suaveRuntime) ethcall(contractAddr common.Address, input []byte) ([]byte, error) {
	return ethCallPrecompileImpl(b.suaveContext, contractAddr, input)
}

func (b *suaveRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, bid types.BidId, namespace string) ([]byte, []byte, error) {
	return buildEthBlockImpl(b.suaveContext, blockArgs, bid, namespace)
}

func (b *suaveRuntime) confidentialInputs() ([]byte, error) {
	return nil, nil
}

func (b *suaveRuntime) confidentialStoreRetrieve(bidId types.BidId, key string) ([]byte, error) {
	return confStoreRetrieveImpl(b.suaveContext, bidId, key)
}

func (b *suaveRuntime) confidentialStoreStore(bidId types.BidId, key string, data []byte) error {
	return confStoreStoreImpl(b.suaveContext, bidId, key, data)
}

func (b *suaveRuntime) extractHint(bundleData []byte) ([]byte, error) {
	return extractHintImpl(b.suaveContext, bundleData)
}

func (b *suaveRuntime) fetchBids(cond uint64, namespace string) ([]types.Bid, error) {
	bids, err := fetchBidsImpl(b.suaveContext, cond, namespace)
	if err != nil {
		return nil, err
	}
	return bids, nil
}

func (b *suaveRuntime) newBid(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, BidType string) (types.Bid, error) {
	bid, err := newBidImpl(b.suaveContext, BidType, decryptionCondition, allowedPeekers, allowedStores)
	if err != nil {
		return types.Bid{}, err
	}
	return *bid, nil
}

func (b *suaveRuntime) signEthTransaction(txn []byte, chainId string, signingKey string) ([]byte, error) {
	return signEthTransactionImpl(txn, chainId, signingKey)
}

func (b *suaveRuntime) simulateBundle(bundleData []byte) (uint64, error) {
	num, err := simulateBundleImpl(b.suaveContext, bundleData)
	if err != nil {
		return 0, err
	}
	return num.Uint64(), nil
}

func (b *suaveRuntime) submitEthBlockBidToRelay(relayUrl string, builderBid []byte) ([]byte, error) {
	return submitEthBlockBidToRelayImpl(b.suaveContext, relayUrl, builderBid)
}

func (b *suaveRuntime) fillMevShareBundle(bidId types.BidId) ([]byte, error) {
	return fillMevShareBundleImpl(b.suaveContext, bidId)
}

func (b *suaveRuntime) submitBundleJsonRPC(url string, method string, params []byte) ([]byte, error) {
	return submitBundleJsonRPCImpl(b.suaveContext, url, method, params)
}
