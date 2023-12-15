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
	isConfidentialAddress = common.HexToAddress("0x42010000")
)

/* General utility precompiles */

func (b *suaveRuntime) confidentialInputs() ([]byte, error) {
	return b.suaveContext.ConfidentialInputs, nil
}

/* Confidential store precompiles */

func (b *suaveRuntime) confidentialStore(bidId types.DataId, key string, data []byte) error {
	bid, err := b.suaveContext.Backend.ConfidentialStore.FetchBidByID(bidId)
	if err != nil {
		return suave.ErrBidNotFound
	}

	log.Info("confStore", "bidId", bidId, "key", key)

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialStoreAddr, bid)
	if err != nil {
		return err
	}

	if metrics.Enabled {
		confStorePrecompileStoreMeter.Mark(int64(len(data)))
	}

	_, err = b.suaveContext.Backend.ConfidentialStore.Store(bidId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

func (b *suaveRuntime) confidentialRetrieve(bidId types.DataId, key string) ([]byte, error) {
	bid, err := b.suaveContext.Backend.ConfidentialStore.FetchBidByID(bidId)
	if err != nil {
		return nil, suave.ErrBidNotFound
	}

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialRetrieveAddr, bid)
	if err != nil {
		return nil, err
	}

	data, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(bidId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
	}

	return data, nil
}

/* Bid precompiles */

func (b *suaveRuntime) newDataRecord(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, BidType string) (types.DataRecord, error) {
	if b.suaveContext.ConfidentialComputeRequestTx == nil {
		panic("newBid: source transaction not present")
	}

	bid, err := b.suaveContext.Backend.ConfidentialStore.InitializeBid(types.DataRecord{
		Salt:                suave.RandomBidId(),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		AllowedStores:       allowedStores,
		Version:             BidType, // TODO : make generic
	})
	if err != nil {
		return types.DataRecord{}, err
	}

	return bid, nil
}

func (b *suaveRuntime) fetchDataRecords(targetBlock uint64, namespace string) ([]types.DataRecord, error) {
	bids1 := b.suaveContext.Backend.ConfidentialStore.FetchBidsByProtocolAndBlock(targetBlock, namespace)

	bids := make([]types.DataRecord, 0, len(bids1))
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

type suaveRuntime struct {
	suaveContext *SuaveContext
}

var _ SuaveRuntime = &suaveRuntime{}
