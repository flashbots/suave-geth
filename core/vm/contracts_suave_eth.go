package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/holiman/uint256"

	builderCapella "github.com/attestantio/go-builder-client/api/capella"
	builderV1 "github.com/attestantio/go-builder-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	specCapella "github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	boostUtils "github.com/flashbots/go-boost-utils/utils"
)

var (
	simulateBundleAddress           = common.HexToAddress("0x42100000")
	extractHintAddress              = common.HexToAddress("0x42100037")
	buildEthBlockAddress            = common.HexToAddress("0x42100001")
	submitEthBlockBidToRelayAddress = common.HexToAddress("0x42100002")
)

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
		return formatPeekerError("could not unmarshal bundle: %w", err)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

	envelope, err := backend.OffchainEthBackend.BuildEthBlock(ctx, nil, bundle.Txs)
	if err != nil {
		return formatPeekerError("could not build eth block for bundle simulation: %w", err)
	}

	if envelope.ExecutionPayload.GasUsed == 0 {
		return formatPeekerError("transaction not applied correctly: %v", envelope)
	}

	egp := new(big.Int).Div(envelope.BlockValue, big.NewInt(int64(envelope.ExecutionPayload.GasUsed)))

	// Return the EGP
	egpBytes, err := precompilesAbi.Methods["simulateBundle"].Outputs.Pack(egp.Uint64())

	if err != nil {
		return formatPeekerError("could not pack egp %v: %w", egp, err)
	}

	return egpBytes, nil
}

type extractHint struct{}

func (c *extractHint) RequiredGas(input []byte) uint64 {
	return 10000
}

func (c *extractHint) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *extractHint) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
	unpacked, err := precompilesAbi.Methods["extractHint"].Inputs.Unpack(input)
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
	unpacked, err := precompilesAbi.Methods["buildEthBlock"].Inputs.Unpack(input)
	if err != nil {
		return formatPeekerError("could not unpack inputs: %w", err)
	}

	// blockArgs := unpacked[0].(types.BuildBlockArgs)
	blockArgsRaw := unpacked[0].(struct {
		Slot           uint64         "json:\"slot\""
		ProposerPubkey []uint8        "json:\"proposerPubkey\""
		Parent         [32]uint8      "json:\"parent\""
		Timestamp      uint64         "json:\"timestamp\""
		FeeRecipient   common.Address "json:\"feeRecipient\""
		GasLimit       uint64         "json:\"gasLimit\""
		Random         [32]uint8      "json:\"random\""
		Withdrawals    []struct {
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

	inputBidId := unpacked[1].(suave.BidId)

	bidIds := []suave.BidId{}
	// first check for merged bid, else assume regular bid
	if mergedBidsBytes, err := backend.ConfiendialStoreBackend.Retrieve(inputBidId, buildEthBlockAddress, "default:v0:mergedBids"); err == nil {
		unpacked, err := bidIdsAbi.Inputs.Unpack(mergedBidsBytes)

		if err != nil {
			return formatPeekerError("could not unpack merged bid ids: %w", err)
		}
		log.Info("x", "x", unpacked, "x", mergedBidsBytes)
		bidIds = unpacked[0].([]suave.BidId)
	} else {
		bidIds = append(bidIds, inputBidId)
	}

	var bidsToMerge = make([]suave.Bid, len(bidIds))
	for i, bidId := range bidIds {
		bidsToMerge[i], err = backend.MempoolBackend.FetchBidById(bidId)
		if err != nil {
			return formatPeekerError("could not fetch bid id %v: %w", bidId, err)
		}
	}

	var mergedBundles []types.SBundle
	for _, bid := range bidsToMerge {
		switch bid.Version {
		case "mevshare:v0:matchBids":
			// fetch the matched ids and merge the bundle
			matchedBundleIdsBytes, err := backend.ConfiendialStoreBackend.Retrieve(bid.Id, buildEthBlockAddress, "mevshare:v0:mergedBids")
			if err != nil {
				return formatPeekerError("could not retrieve bid ids data for bid %v, from cdas: %w", bid, err)
			}

			unpackedBidIds, err := bidIdsAbi.Inputs.Unpack(matchedBundleIdsBytes)
			if err != nil {
				return formatPeekerError("could not unpack bid ids data for bid %v, from cdas: %w", bid, err)
			}

			matchBidIds := unpackedBidIds[0].([]suave.BidId)

			userBundleBytes, err := backend.ConfiendialStoreBackend.Retrieve(matchBidIds[0], buildEthBlockAddress, "mevshare:v0:ethBundles")
			if err != nil {
				return formatPeekerError("could not retrieve bundle data for bidId %v: %w", matchBidIds[0], err)
			}

			var userBundle types.SBundle
			if err := json.Unmarshal(userBundleBytes, &userBundle); err != nil {
				return formatPeekerError("could not unmarshal user bundle data for bidId %v: %w", matchBidIds[0], err)
			}

			matchBundleBytes, err := backend.ConfiendialStoreBackend.Retrieve(matchBidIds[1], buildEthBlockAddress, "mevshare:v0:ethBundles")
			if err != nil {
				return formatPeekerError("could not retrieve match bundle data for bidId %v: %w", matchBidIds[1], err)
			}

			var matchBundle types.SBundle
			if err := json.Unmarshal(matchBundleBytes, &matchBundle); err != nil {
				return formatPeekerError("could not unmarshal match bundle data for bidId %v: %w", matchBidIds[1], err)
			}

			userBundle.Txs = append(userBundle.Txs, matchBundle.Txs...)

			mergedBundles = append(mergedBundles, userBundle)

		case "mevshare:v0:unmatchedBundles":
			bundleBytes, err := backend.ConfiendialStoreBackend.Retrieve(bid.Id, buildEthBlockAddress, "mevshare:v0:ethBundles")
			if err != nil {
				return formatPeekerError("could not retrieve bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}

			var bundle types.SBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return formatPeekerError("could not unmarshal bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}
			mergedBundles = append(mergedBundles, bundle)
		case "default:v0:ethBundles":
			bundleBytes, err := backend.ConfiendialStoreBackend.Retrieve(bid.Id, buildEthBlockAddress, "default:v0:ethBundles")
			if err != nil {
				return formatPeekerError("could not retrieve bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}

			var bundle types.SBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return formatPeekerError("could not unmarshal bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}
			mergedBundles = append(mergedBundles, bundle)
		default:
			return formatPeekerError("unknown bid version %s", bid.Version)
		}
	}

	log.Info("requesting a block be built", "mergedBundles", mergedBundles)
	envelope, err := backend.OffchainEthBackend.BuildEthBlockFromBundles(context.TODO(), &blockArgs, mergedBundles)
	if err != nil {
		return formatPeekerError("could not build eth block: %w", err)
	}

	log.Info("built block from bundles", "payload", *envelope.ExecutionPayload)

	payload, err := executableDataToCapellaExecutionPayload(envelope.ExecutionPayload)
	if err != nil {
		return formatPeekerError("could not format execution payload as capella payload: %w", err)
	}

	// really should not be generated here
	blsSk, blsPk, err := bls.GenerateNewKeypair()
	if err != nil {
		return formatPeekerError("could not generate new bls key pair: %w", err)
	}

	pk, err := boostUtils.BlsPublicKeyToPublicKey(blsPk)
	if err != nil {
		return formatPeekerError("could not format bls pubkey as bytes: %w", err)
	}

	value, overflow := uint256.FromBig(envelope.BlockValue)
	if overflow {
		return formatPeekerError("block value %v overflows", *envelope.BlockValue)
	}
	var proposerPubkey [48]byte
	copy(proposerPubkey[:], blockArgsRaw.ProposerPubkey)

	blockBidMsg := builderV1.BidTrace{
		Slot:                 blockArgsRaw.Slot,
		ParentHash:           payload.ParentHash,
		BlockHash:            payload.BlockHash,
		BuilderPubkey:        pk,
		ProposerPubkey:       phase0.BLSPubKey(proposerPubkey),
		ProposerFeeRecipient: bellatrix.ExecutionAddress(blockArgsRaw.FeeRecipient),
		GasLimit:             envelope.ExecutionPayload.GasLimit,
		GasUsed:              envelope.ExecutionPayload.GasUsed,
		Value:                value,
	}

	// hardcoded for goerli, should be passed in with the inputs
	genesisForkVersion := phase0.Version{0x00, 0x00, 0x10, 0x20}
	builderSigningDomain := ssz.ComputeDomain(ssz.DomainTypeAppBuilder, genesisForkVersion, phase0.Root{})
	signature, err := ssz.SignMessage(&blockBidMsg, builderSigningDomain, blsSk)
	if err != nil {
		return formatPeekerError("could not sign builder bid: %w", err)
	}

	bidRequest := builderCapella.SubmitBlockRequest{
		Message:          &blockBidMsg,
		ExecutionPayload: payload,
		Signature:        signature,
	}

	bidBytes, err := bidRequest.MarshalJSON()
	if err != nil {
		return formatPeekerError("could not marshal builder bid request: %w", err)
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return formatPeekerError("could not marshal payload envelope: %w", err)
	}

	return precompilesAbi.Methods["buildEthBlock"].Outputs.Pack(bidBytes, envelopeBytes)
}

type submitEthBlockBidToRelay struct {
}

func (c *submitEthBlockBidToRelay) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *submitEthBlockBidToRelay) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *submitEthBlockBidToRelay) RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error) {
	unpacked, err := precompilesAbi.Methods["submitEthBlockBidToRelay"].Inputs.Unpack(input)
	if err != nil {
		return formatPeekerError("could not unpack inputs: %w", err)
	}

	relayUrl := unpacked[0].(string)
	builderBidJson := unpacked[1].([]byte)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(3*time.Second))
	defer cancel()

	endpoint := relayUrl + "/relay/v1/builder/blocks"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(builderBidJson))
	if err != nil {
		return formatPeekerError("could not prepare request to relay: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return formatPeekerError("could not send request to relay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode > 299 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return formatPeekerError("could not read error response body for status code %d: %w", resp.StatusCode, err)
		}

		return formatPeekerError("relay request failed with code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil, nil
}

func executableDataToCapellaExecutionPayload(data *engine.ExecutableData) (*specCapella.ExecutionPayload, error) {
	transactionData := make([]bellatrix.Transaction, len(data.Transactions))
	for i, tx := range data.Transactions {
		transactionData[i] = bellatrix.Transaction(tx)
	}

	withdrawalData := make([]*specCapella.Withdrawal, len(data.Withdrawals))
	for i, wd := range data.Withdrawals {
		withdrawalData[i] = &specCapella.Withdrawal{
			Index:          specCapella.WithdrawalIndex(wd.Index),
			ValidatorIndex: phase0.ValidatorIndex(wd.Validator),
			Address:        bellatrix.ExecutionAddress(wd.Address),
			Amount:         phase0.Gwei(wd.Amount),
		}
	}

	baseFeePerGas := new(boostTypes.U256Str)
	err := baseFeePerGas.FromBig(data.BaseFeePerGas)
	if err != nil {
		return nil, err
	}

	return &specCapella.ExecutionPayload{
		ParentHash:    [32]byte(data.ParentHash),
		FeeRecipient:  [20]byte(data.FeeRecipient),
		StateRoot:     [32]byte(data.StateRoot),
		ReceiptsRoot:  [32]byte(data.ReceiptsRoot),
		LogsBloom:     types.BytesToBloom(data.LogsBloom),
		PrevRandao:    [32]byte(data.Random),
		BlockNumber:   data.Number,
		GasLimit:      data.GasLimit,
		GasUsed:       data.GasUsed,
		Timestamp:     data.Timestamp,
		ExtraData:     data.ExtraData,
		BaseFeePerGas: *baseFeePerGas,
		BlockHash:     [32]byte(data.BlockHash),
		Transactions:  transactionData,
		Withdrawals:   withdrawalData,
	}, nil
}
