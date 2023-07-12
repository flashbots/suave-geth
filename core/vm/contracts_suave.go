package vm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	builderSpec "github.com/attestantio/go-builder-client/spec"

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

	newBidAddress      = common.HexToAddress("0x42030000")
	fetchBidsAddress   = common.HexToAddress("0x42030001")
	extractHintAddress = common.HexToAddress("0x42100037")

	simulateBundleAddress = common.HexToAddress("0x42100000")
	buildEthBlockAddress  = common.HexToAddress("0x42100001")
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

func (c *isOffchainPrecompile) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
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

func (c *confidentialInputsPrecompile) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
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

func (c *confStoreStore) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
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

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(backend.callerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := backend.callerStack[i]; _c != nil && *_c != confStoreStoreAddress {
			caller = *_c
			break
		}
	}

	_, err = backend.ConfiendialStoreBackend.Store(bidId, caller, key, data)
	if err != nil {
		return []byte(err.Error()), err
	}

	return nil, nil
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

func (c *confStoreRetrieve) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
	if len(backend.callerStack) == 0 {
		return []byte("not allowed"), errors.New("not allowed in this context")
	}

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bidId := unpacked[0].([16]byte)
	key := unpacked[1].(string)

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

	data, err := backend.ConfiendialStoreBackend.Retrieve(bidId, caller, key)
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

func (c *newBid) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}
	version := unpacked[2].(string)

	decryptionCondition := unpacked[0].(uint64)
	allowedPeekers := unpacked[1].([]common.Address)
	bid := suave.Bid{
		Id:                  suave.BidId(uuid.New()),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		Version:             version, // TODO : make generic
	}

	bid, err = backend.ConfiendialStoreBackend.Initialize(bid, "", nil)
	if err != nil {
		return []byte(err.Error()), err
	}

	err = backend.MempoolBackned.SubmitBid(bid)
	if err != nil {
		return []byte(err.Error()), err
	}

	return c.inoutAbi.Outputs.Pack(bid)
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

func (c *fetchBids) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	targetBlock := unpacked[0].(uint64)
	namespace := unpacked[1].(string)

	bids := backend.MempoolBackned.FetchBidsByProtocolAndBlock(targetBlock, namespace)

	return c.inoutAbi.Outputs.Pack(bids)
}

/* Eth precompiles */

type simulateBundle struct {
}

func (c *simulateBundle) RequiredGas(input []byte) uint64 {
	// Should be proportional to bundle gas limit
	return 10000
}

func (c *simulateBundle) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *simulateBundle) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
	}{}
	err := json.Unmarshal(input, &bundle)
	if err != nil {
		return []byte(err.Error()), err
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

	envelope, err := backend.OffchainEthBackend.BuildEthBlock(ctx, nil, bundle.Txs)
	if err != nil {
		return []byte(err.Error()), err
	}

	if envelope.ExecutionPayload.GasUsed == 0 {
		return nil, errors.New("transaction not applied correctly")
	}

	egp := new(big.Int).Div(envelope.BlockValue, big.NewInt(int64(envelope.ExecutionPayload.GasUsed)))

	// Return the EGP
	egpBytes, err := abi.Arguments{abi.Argument{Type: abi.Type{T: abi.UintTy, Size: 64}}}.Pack(egp.Uint64())

	if err != nil {
		return []byte(err.Error()), err
	}

	return egpBytes, nil
}

type buildEthBlock struct {
}

func (c *buildEthBlock) RequiredGas(input []byte) uint64 {
	// Should be proportional to bundle gas limit
	return 10000
}

func (c *buildEthBlock) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *buildEthBlock) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
	unpacked, err := buildBlockPrecompileAbi.Methods["buildEthBlock"].Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	blockArgsRaw := unpacked[0].(struct {
		Parent       [32]uint8      "json:\"parent\""
		Timestamp    uint64         "json:\"timestamp\""
		FeeRecipient common.Address "json:\"feeRecipient\""
		GasLimit     uint64         "json:\"gasLimit\""
		Random       [32]uint8      "json:\"random\""
		Withdrawals  []struct {
			Index     uint64         "json:\"index\""
			Validator uint64         "json:\"validator\""
			Address   common.Address "json:\"Address\""
			Amount    uint64         "json:\"amount\""
		} "json:\"withdrawals\""
	})
	blockArgs := types.BuildBlockArgs{
		Parent:       blockArgsRaw.Parent,
		Timestamp:    blockArgsRaw.Timestamp,
		FeeRecipient: blockArgsRaw.FeeRecipient,
		GasLimit:     blockArgsRaw.GasLimit,
		Random:       blockArgsRaw.Random,
		Withdrawals:  types.Withdrawals{},
	}

	for _, w := range blockArgsRaw.Withdrawals {
		blockArgs.Withdrawals = append(blockArgs.Withdrawals, &types.Withdrawal{
			Index:     w.Index,
			Validator: w.Validator,
			Address:   w.Address,
			Amount:    w.Amount,
		})
	}

	bidId := unpacked[1].(suave.BidId)
	namespace := unpacked[2].(string)
	var bidIds = []suave.BidId{}
	// first check for merged bid, else assume regular bid
	if mergedBidsBytes, err := backend.ConfiendialStoreBackend.Retrieve(bidId, buildEthBlockAddress, namespace+":mergedBids"); err == nil {
		bidIdsAbi := mustParseMethodAbi(`[{"inputs": [{ "type": "bytes16[]" }], "name": "bidids", "outputs":[], "type": "function"}]`, "bidids")
		unpacked, err := bidIdsAbi.Inputs.Unpack(mergedBidsBytes)
		if err != nil {
			return []byte(err.Error()), err
		}
		bidIds = unpacked[0].([]suave.BidId)
	} else {
		bidIds = append(bidIds, bidId)
	}

	var txs types.Transactions
	var bundles []types.SBundle
	idToBundle := make(map[suave.BidId]types.SBundle)
	var zero [16]byte
	for _, bidId := range bidIds {
		bundleBytes, err := backend.ConfiendialStoreBackend.Retrieve(bidId, buildEthBlockAddress, namespace+":ethBundles")
		if err != nil {
			return []byte(err.Error()), err
		}

		bundle := types.SBundle{}
		if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
			return []byte(err.Error()), err
		}
		txs = append(txs, bundle.Txs...)
		idToBundle[bidId] = bundle
		if bundle.MatchId != zero {
			bundles = append(bundles, bundle)
		}

	}

	var mergedBundles []types.SBundle
	for _, b := range bundles {
		// hack: merge relevant bundles
		// need to create a mergeBid precompile imo
		match, success := idToBundle[b.MatchId]
		if success {
			var mergedTxs types.Transactions
			mergedTxs = append(mergedTxs, match.Txs...)
			mergedTxs = append(mergedTxs, b.Txs...)
			b.Txs = mergedTxs
			b.RefundPercent = match.RefundPercent
		}

		mergedBundles = append(mergedBundles, b)
	}

	envelope, err := backend.OffchainEthBackend.BuildEthBlockFromBundles(context.TODO(), &blockArgs, mergedBundles)
	if err != nil {
		return []byte(err.Error()), err
	}

	// envelope, err := backend.OffchainEthBackend.BuildEthBlock(context.TODO(), &blockArgs, txs)
	// if err != nil {
	// 	return []byte(err.Error()), err
	// }

	/*
		"github.com/attestantio/go-builder-client/api/capella"
		builderSpec "github.com/attestantio/go-builder-client/spec"
		consensusspec "github.com/attestantio/go-eth2-client/spec"
		boostCommon "github.com/flashbots/mev-boost-relay/common"

		profit, overflow := uint256.FromBig(envelope.BlockValue)
		if overflow {
			return nil, errors.New("overflow")
		}

		header, err := boostCommon.CapellaPayloadToPayloadHeader( envelope.ExecutionPayload)
		if err != nil {
			return nil, err
		}

		builderBid := builderSpec.VersionedSignedBuilderBid{
			Version: consensusspec.DataVersionCapella,
			Capella: &capella.SignedBuilderBid{
				Message: *capella.BuilderBid{
					Header: header,
					Value:  profit,
					Pubkey: phase0.BLSPubKey{},
				},
				Signature: phase0.BLSSignature{},
			},
		}
	*/

	builderBid := builderSpec.VersionedSignedBuilderBid{}
	bidBytes, err := json.Marshal(builderBid)
	if err != nil {
		return []byte(err.Error()), err
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return []byte(err.Error()), err
	}

	return buildBlockPrecompileAbi.Methods["buildEthBlock"].Outputs.Pack(bidBytes, envelopeBytes)
}

type extractHint struct{}

func (c *extractHint) RequiredGas(input []byte) uint64 {
	return 10000
}

func (c *extractHint) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *extractHint) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
	unpacked, err := peekerPrecompileAbi.Methods["extractHint"].Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bundleBytes := unpacked[0].([]byte)
	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
		RefundPercent   int                `json:"percent"`
		MatchId         [16]byte           `json:"MatchId"`
	}{}

	err = json.Unmarshal(bundleBytes, &bundle)
	if err != nil {
		return []byte(err.Error()), err
	}

	tx := bundle.Txs[0]
	hint := struct {
		To   common.Address
		Data []byte
	}{
		To:   *tx.To(),
		Data: tx.Data(),
	}

	hintBytes, err := json.Marshal(hint)
	if err != nil {
		return []byte(err.Error()), err
	}
	return hintBytes, nil
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
