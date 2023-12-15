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

func (b *suaveRuntime) confidentialStore(dataId types.DataId, key string, data []byte) error {
	record, err := b.suaveContext.Backend.ConfidentialStore.FetchRecordByID(dataId)
	if err != nil {
		return suave.ErrRecordNotFound
	}

	log.Debug("confStore", "dataId", dataId, "key", key)

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialStoreAddr, record)
	if err != nil {
		return err
	}

	if metrics.Enabled {
		confStorePrecompileStoreMeter.Mark(int64(len(data)))
	}

	_, err = b.suaveContext.Backend.ConfidentialStore.Store(dataId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

func (b *suaveRuntime) confidentialRetrieve(dataId types.DataId, key string) ([]byte, error) {
	record, err := b.suaveContext.Backend.ConfidentialStore.FetchRecordByID(dataId)
	if err != nil {
		return nil, suave.ErrRecordNotFound
	}

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialRetrieveAddr, record)
	if err != nil {
		return nil, err
	}

	data, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(dataId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
	}

	return data, nil
}

/* Data Record precompiles */

func (b *suaveRuntime) newDataRecord(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, RecordType string) (types.DataRecord, error) {
	if b.suaveContext.ConfidentialComputeRequestTx == nil {
		panic("newRecord: source transaction not present")
	}

	record, err := b.suaveContext.Backend.ConfidentialStore.InitRecord(types.DataRecord{
		Salt:                suave.RandomDataRecordId(),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		AllowedStores:       allowedStores,
		Version:             RecordType, // TODO : make generic
	})
	if err != nil {
		return types.DataRecord{}, err
	}

	return record, nil
}

func (b *suaveRuntime) fetchDataRecords(targetBlock uint64, namespace string) ([]types.DataRecord, error) {
	records1 := b.suaveContext.Backend.ConfidentialStore.FetchRecordsByProtocolAndBlock(targetBlock, namespace)

	records := make([]types.DataRecord, 0, len(records1))
	for _, record := range records1 {
		records = append(records, record.ToInnerRecord())
	}

	return records, nil
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
