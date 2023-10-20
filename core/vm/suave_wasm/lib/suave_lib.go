package lib

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

/* Implements
type ConfidentialStore interface {
	InitializeBid(bid types.Bid) (types.Bid, error)
	Store(bidId suave.BidId, caller common.Address, key string, value []byte) (types.EngineBid, error)
	Retrieve(bid types.BidId, caller common.Address, key string) ([]byte, error)
	FetchBidById(suave.BidId) (types.EngineBid, error)
	FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []types.EngineBid
}
*/

func InitializeBid(rawBid types.Bid) (types.Bid, error) {
	jsonRawBid, err := json.Marshal(rawBid)
	if err != nil {
		return types.Bid{}, fmt.Errorf("failed to marshal: %w", err)
	}

	inputBidOffset := bytesToPointer(jsonRawBid)
	inputBidSize := uint32(len(jsonRawBid))

	bufOffset := bytesToPointer(dataBuf)
	bufSize := uint32(len(dataBuf))

	n := uint32(0)
	nOffset := uint32(uintptr(unsafe.Pointer(&n)))

	errno := initializeBid(inputBidOffset, inputBidSize, bufOffset, bufSize, nOffset)
	if errno != 0 {
		return types.Bid{}, fmt.Errorf("host bid initialize failed: %s", string(dataBuf[:n]))
	}

	var returnBid types.Bid
	err = json.Unmarshal(dataBuf[:n], &returnBid)
	if err != nil {
		return types.Bid{}, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return returnBid, nil
}

func Store(bidId types.BidId, key string, value []byte) (types.EngineBid, error) {
	jsonStore, err := json.Marshal(struct {
		BidId types.BidId
		Key   string
		Value []byte
	}{bidId, key, value})
	if err != nil {
		return types.EngineBid{}, fmt.Errorf("failed to marshal: %w", err)
	}

	inputOffset := bytesToPointer(jsonStore)
	inputSize := uint32(len(jsonStore))

	bufOffset := bytesToPointer(dataBuf)
	bufSize := uint32(len(dataBuf))

	n := uint32(0)
	nOffset := uint32(uintptr(unsafe.Pointer(&n)))

	errno := storePut(inputOffset, inputSize, bufOffset, bufSize, nOffset)
	if errno != 0 {
		return types.EngineBid{}, fmt.Errorf("host storePut failed: %s", string(dataBuf[:n]))
	}

	var returnBid types.EngineBid
	err = json.Unmarshal(dataBuf[:n], &returnBid)
	if err != nil {
		return types.EngineBid{}, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return returnBid, nil
}

func StoreRetrieve(bidId types.BidId, key string) ([]byte, error) {
	keyOffset := stringToPointer(key)
	keySize := uint32(len(key))

	bidOffset := bytesToPointer(bidId[:])
	bidSize := uint32(len(bidId))

	bufOffset := bytesToPointer(dataBuf)
	bufSize := uint32(len(dataBuf))

	n := uint32(0)
	nOffset := uint32(uintptr(unsafe.Pointer(&n)))

	errno := storeRetrieve(keyOffset, keySize, bidOffset, bidSize, bufOffset, bufSize, nOffset)
	if errno != 0 {
		return nil, fmt.Errorf("host storeRetrieve failed: %s", string(dataBuf[:n]))
	}

	return common.CopyBytes(dataBuf[:n]), nil
}

func FetchBidById(bidId types.BidId) (types.EngineBid, error) {
	inputBidOffset := bytesToPointer(bidId[:])

	bufOffset := bytesToPointer(dataBuf)
	bufSize := uint32(len(dataBuf))

	n := uint32(0)
	nOffset := uint32(uintptr(unsafe.Pointer(&n)))

	errno := fetchBidById(inputBidOffset, 16, bufOffset, bufSize, nOffset)
	if errno != 0 {
		return types.EngineBid{}, fmt.Errorf("host bid initialize failed: %s", string(dataBuf[:n]))
	}

	var returnBid types.EngineBid
	err := json.Unmarshal(dataBuf[:n], &returnBid)
	if err != nil {
		return types.EngineBid{}, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return returnBid, nil
}

func FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) ([]types.EngineBid, error) {
	selector, err := json.Marshal(struct {
		BlockNumber uint64
		Namespace   string
	}{blockNumber, namespace})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}

	inputOffset := bytesToPointer(selector)
	inputSize := uint32(len(selector))

	bufOffset := bytesToPointer(dataBuf)
	bufSize := uint32(len(dataBuf))

	n := uint32(0)
	nOffset := uint32(uintptr(unsafe.Pointer(&n)))

	errno := fetchBidsByProtocolAndBlock(inputOffset, inputSize, bufOffset, bufSize, nOffset)
	if errno != 0 {
		return nil, fmt.Errorf("host bid initialize failed: %s", string(dataBuf[:n]))
	}

	var returnBids []types.EngineBid
	err = json.Unmarshal(dataBuf[:n], &returnBids)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return returnBids, nil
}

var (
	dataBuf = make([]byte, 1024*256) // max 256KB (TODO!)
)

//go:wasmimport suavexec initializeBid
//go:noescape
func initializeBid(inputBidOffset, inputBidSize, outputBidOffset, outputBidSize, n uint32) uint32

//go:wasmimport suavexec storePut
//go:noescape
func storePut(inputOffset, inputSize, bufOffset, bufSize, n uint32) uint32

//go:wasmimport suavexec storeRetrieve
//go:noescape
func storeRetrieve(keyOffset, keySize, bidOffset, bidSize, bufOffset, bufSize, n uint32) uint32

//go:wasmimport suavexec fetchBidById
//go:noescape
func fetchBidById(bidOffset, bidSize, bufOffset, bufSize, n uint32) uint32

//go:wasmimport suavexec fetchBidsByProtocolAndBlock
//go:noescape
func fetchBidsByProtocolAndBlock(inputOffset, inputSize, bufOffset, bufSize, n uint32) uint32
