package vm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

var (
	isOffchainAddress               = common.HexToAddress("0x42010000")
	errIsOffchainInvalidInputLength = errors.New("invalid input length")

	confidentialInputsAddress = common.HexToAddress("0x42010001")

	confStoreStoreAddress    = common.HexToAddress("0x42020000")
	confStoreRetrieveAddress = common.HexToAddress("0x42020001")

	newBidAddress    = common.HexToAddress("0x42030000")
	fetchBidsAddress = common.HexToAddress("0x42030001")
)

/* General utility precompiles */

type isOffchainPrecompile struct{}

func (c *isOffchainPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *isOffchainPrecompile) Run(input []byte) ([]byte, error) {
	if len(input) == 1 {
		// The precompile was called *directly* off-chain, and the result was cached - return 1
		if input[0] == 0x01 {
			return []byte{0x01}, nil
		} else {
			return nil, errors.New("incorrect value passed in")
		}
	}

	if len(input) > 1 {
		return nil, errIsOffchainInvalidInputLength
	}

	return []byte{0x00}, nil
}

func (c *isOffchainPrecompile) RunOffchain(backend *SuaveExecutionBackend, input []byte) ([]byte, error) {
	if len(input) != 0 {
		return nil, errIsOffchainInvalidInputLength
	}
	return []byte{0x01}, nil
}

type confidentialInputsPrecompile struct{}

func (c *confidentialInputsPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *confidentialInputsPrecompile) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this context")
}

func (c *confidentialInputsPrecompile) RunOffchain(backend *SuaveExecutionBackend, input []byte) ([]byte, error) {
	return backend.confidentialInputs, nil
}

/* Confidential store precompiles */

type confStoreStore struct {
	inoutAbi abi.Method
}

func newConfStoreStore() *confStoreStore {
	inoutAbi := mustParseMethodAbi(`[{"inputs":[{"type":"bytes16"}, {"type":"string"}, {"type":"bytes"}],"name":"store","outputs":[],"stateMutability":"nonpayable","type":"function"}]`, "store")

	return &confStoreStore{inoutAbi}
}

func (c *confStoreStore) RequiredGas(input []byte) uint64 {
	return uint64(100 * len(input))
}

func (c *confStoreStore) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this context")
}

func (c *confStoreStore) RunOffchain(backend *SuaveExecutionBackend, input []byte) ([]byte, error) {
	if len(backend.callerStack) == 0 {
		return []byte("not allowed"), errors.New("not allowed in this context")
	}

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bidId := unpacked[0].([16]byte)
	key := unpacked[1].(string)
	data := unpacked[2].([]byte)

	if err := c.runImpl(backend, bidId, key, data); err != nil {
		return []byte(err.Error()), err
	}
	return nil, nil
}

func (c *confStoreStore) runImpl(backend *SuaveExecutionBackend, bidId [16]byte, key string, data []byte) error {
	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(backend.callerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := backend.callerStack[i]; _c != nil && *_c != confStoreStoreAddress {
			caller = *_c
			break
		}
	}

	_, err := backend.ConfidentialStoreBackend.Store(bidId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

type confStoreRetrieve struct {
	inoutAbi abi.Method
}

func newConfStoreRetrieve() *confStoreRetrieve {
	inoutAbi := mustParseMethodAbi(`[{"inputs":[{"type":"bytes16"}, {"type":"string"}],"name":"retrieve","outputs":[{"type":"bytes"}],"stateMutability":"nonpayable","type":"function"}]`, "retrieve")

	return &confStoreRetrieve{inoutAbi}
}

func (c *confStoreRetrieve) RequiredGas(input []byte) uint64 {
	return 100
}

func (c *confStoreRetrieve) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this context")
}

func (c *confStoreRetrieve) RunOffchain(backend *SuaveExecutionBackend, input []byte) ([]byte, error) {
	if len(backend.callerStack) == 0 {
		return []byte("not allowed"), errors.New("not allowed in this context")
	}

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bidId := unpacked[0].([16]byte)
	key := unpacked[1].(string)

	return c.runImpl(backend, bidId, key)
}

func (c *confStoreRetrieve) runImpl(backend *SuaveExecutionBackend, bidId [16]byte, key string) ([]byte, error) {
	log.Info("confStoreRetrieve", "bidId", bidId, "key", key)

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(backend.callerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := backend.callerStack[i]; _c != nil && *_c != confStoreRetrieveAddress {
			caller = *_c
			break
		}
	}

	data, err := backend.ConfidentialStoreBackend.Retrieve(bidId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	return data, nil
}

/* Bid precompiles */

type newBid struct {
	inoutAbi abi.Method
}

func newNewBid() *newBid {
	inoutAbi := mustParseMethodAbi(`[{ "inputs": [ { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "string", "name": "BidType", "type": "string" } ], "name": "newBid", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid", "name": "", "type": "tuple" } ], "stateMutability": "view", "type": "function" }]`, "newBid")

	return &newBid{inoutAbi}
}

func (c *newBid) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *newBid) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *newBid) RunOffchain(backend *SuaveExecutionBackend, input []byte) ([]byte, error) {
	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}
	version := unpacked[2].(string)

	decryptionCondition := unpacked[0].(uint64)
	allowedPeekers := unpacked[1].([]common.Address)

	bid, err := c.runImpl(backend, version, decryptionCondition, allowedPeekers)
	if err != nil {
		return []byte(err.Error()), err
	}

	return c.inoutAbi.Outputs.Pack(bid)
}

func (c *newBid) runImpl(backend *SuaveExecutionBackend, version string, decryptionCondition uint64, allowedPeekers []common.Address) (*suave.Bid, error) {
	bid := suave.Bid{
		Id:                  suave.BidId(uuid.New()),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		Version:             version, // TODO : make generic
	}

	bid, err := backend.ConfidentialStoreBackend.Initialize(bid, "", nil)
	if err != nil {
		return nil, err
	}

	err = backend.MempoolBackend.SubmitBid(bid)
	if err != nil {
		return nil, err
	}

	return &bid, nil
}

type fetchBids struct {
	inoutAbi abi.Method
}

func newFetchBids() *fetchBids {
	inoutAbi := mustParseMethodAbi(`[ { "inputs": [ { "internalType": "uint64", "name": "cond", "type": "uint64" }, { "internalType": "string", "name": "namespace", "type": "string" } ], "name": "fetchBids", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "string", "name": "version", "type": "string" } ], "internalType": "struct Suave.Bid[]", "name": "", "type": "tuple[]" } ], "stateMutability": "view", "type": "function" } ]`, "fetchBids")

	return &fetchBids{inoutAbi}
}

func (c *fetchBids) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *fetchBids) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *fetchBids) RunOffchain(backend *SuaveExecutionBackend, input []byte) ([]byte, error) {
	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	targetBlock := unpacked[0].(uint64)
	namespace := unpacked[1].(string)

	bids, err := c.runImpl(backend, targetBlock, namespace)
	if err != nil {
		return []byte(err.Error()), err
	}

	return c.inoutAbi.Outputs.Pack(bids)
}

func (c *fetchBids) runImpl(backend *SuaveExecutionBackend, targetBlock uint64, namespace string) ([]suave.Bid, error) {
	bids := backend.MempoolBackend.FetchBidsByProtocolAndBlock(targetBlock, namespace)
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

type backendImpl struct {
	backend *SuaveExecutionBackend
}

var _ BackendImpl = &backendImpl{}

func (b *backendImpl) buildEthBlock(blockArgs types.BuildBlockArgs, bid types.BidId, namespace string) ([]byte, []byte, error) {
	return (&buildEthBlock{}).runImpl(b.backend, blockArgs, bid, namespace)
}

func (b *backendImpl) confidentialInputs() ([]byte, error) {
	return nil, nil
}

func (b *backendImpl) confidentialStoreRetrieve(bidId types.BidId, key string) ([]byte, error) {
	return (&confStoreRetrieve{}).runImpl(b.backend, bidId, key)
}

func (b *backendImpl) confidentialStoreStore(bidId types.BidId, key string, data []byte) error {
	return (&confStoreStore{}).runImpl(b.backend, bidId, key, data)
}

func (b *backendImpl) extractHint(bundleData []byte) ([]byte, error) {
	return (&extractHint{}).runImpl(b.backend, bundleData)
}

func (b *backendImpl) fetchBids(cond uint64, namespace string) ([]types.Bid, error) {
	bids, err := (&fetchBids{}).runImpl(b.backend, cond, namespace)
	if err != nil {
		return nil, err
	}
	return bids, nil
}

func (b *backendImpl) newBid(decryptionCondition uint64, allowedPeekers []common.Address, BidType string) (types.Bid, error) {
	bid, err := (&newBid{}).runImpl(b.backend, BidType, decryptionCondition, allowedPeekers)
	if err != nil {
		return types.Bid{}, err
	}
	return *bid, nil
}

func (b *backendImpl) simulateBundle(bundleData []byte) (uint64, error) {
	num, err := (&simulateBundle{}).runImpl(b.backend, bundleData)
	if err != nil {
		return 0, err
	}
	return num.Uint64(), nil
}

func (b *backendImpl) submitEthBlockBidToRelay(relayUrl string, builderBid []byte) ([]byte, error) {
	return (&submitEthBlockBidToRelay{}).runImpl(b.backend, relayUrl, builderBid)
}
