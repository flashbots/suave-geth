package lib

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/ethereum/go-ethereum/core/types"
	suave_wasi "github.com/ethereum/go-ethereum/suave/wasi"
)

type hostJsonFn = func(inputOffset, inputSize, outputOffset, outputSize, n uint32) uint32

func call_hostFn[In any, Out any](data In, fn hostJsonFn) (Out, error) {
	var out Out

	jsonInputBytes, err := json.Marshal(data)
	if err != nil {
		return out, fmt.Errorf("failed to marshal: %w", err)
	}

	inputBidOffset := bytesToPointer(jsonInputBytes)
	inputBidSize := uint32(len(jsonInputBytes))

	bufOffset := bytesToPointer(dataBuf)
	bufSize := uint32(len(dataBuf))

	n := uint32(0)
	nOffset := uint32(uintptr(unsafe.Pointer(&n)))

	errno := fn(inputBidOffset, inputBidSize, bufOffset, bufSize, nOffset)
	if errno != 0 {
		return out, fmt.Errorf("host function failed: %s", string(dataBuf[:n]))
	}

	err = json.Unmarshal(dataBuf[:n], &out)
	if err != nil {
		return out, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return out, nil
}

func InitializeBid(rawBid types.Bid) (types.Bid, error) {
	return call_hostFn[types.Bid, types.Bid](rawBid, initializeBid)
}

func Store(bidId types.BidId, key string, value []byte) (types.EngineBid, error) {
	return call_hostFn[suave_wasi.StoreHostFnArgs, types.EngineBid](suave_wasi.StoreHostFnArgs{bidId, key, value}, storePut)
}

func StoreRetrieve(bidId types.BidId, key string) ([]byte, error) {
	return call_hostFn[suave_wasi.RetrieveHostFnArgs, []byte](suave_wasi.RetrieveHostFnArgs{bidId, key}, storeRetrieve)
}

func FetchBidById(bidId types.BidId) (types.EngineBid, error) {
	return call_hostFn[types.BidId, types.EngineBid](bidId, fetchBidById)
}

func FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) ([]types.EngineBid, error) {
	return call_hostFn[suave_wasi.FetchBidByProtocolFnArgs, []types.EngineBid](suave_wasi.FetchBidByProtocolFnArgs{blockNumber, namespace}, fetchBidsByProtocolAndBlock)
}

var (
	dataBuf = make([]byte, 1024*256) // max 256KB (TODO!)
)

//go:wasmimport suavexec initializeBid
//go:noescape
func initializeBid(inputOffset, inputSize, outputOffset, outputSize, n uint32) uint32

//go:wasmimport suavexec storePut
//go:noescape
func storePut(inputOffset, inputSize, outputOffset, outputSize, n uint32) uint32

//go:wasmimport suavexec storeRetrieve
//go:noescape
func storeRetrieve(inputOffset, inputSize, outputOffset, outputSize, n uint32) uint32

//go:wasmimport suavexec fetchBidById
//go:noescape
func fetchBidById(inputOffset, inputSize, outputOffset, outputSize, n uint32) uint32

//go:wasmimport suavexec fetchBidsByProtocolAndBlock
//go:noescape
func fetchBidsByProtocolAndBlock(inputOffset, inputSize, outputOffset, outputSize, n uint32) uint32
