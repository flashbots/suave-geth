package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/suave/artifacts"
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

func (c *simulateBundle) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	egp, err := c.runImpl(suaveContext, input)
	if err != nil {
		return []byte(err.Error()), err
	}

	return artifacts.SuaveAbi.Methods["simulateBundle"].Outputs.Pack(egp.Uint64())
}

func (c *simulateBundle) Do(suaveContext *SuaveContext, input []byte) (uint64, error) {
	res, err := c.runImpl(suaveContext, input)
	if err != nil {
		return 0, fmt.Errorf("could not simulate bundle: %w", err)
	}
	return res.Uint64(), nil
}

func (c *simulateBundle) runImpl(suaveContext *SuaveContext, input []byte) (*big.Int, error) {
	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
	}{}
	err := json.Unmarshal(input, &bundle)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

	envelope, err := suaveContext.Backend.ConfidentialEthBackend.BuildEthBlock(ctx, nil, bundle.Txs)
	if err != nil {
		return nil, err
	}

	if envelope.ExecutionPayload.GasUsed == 0 {
		return nil, err
	}

	egp := new(big.Int).Div(envelope.BlockValue, big.NewInt(int64(envelope.ExecutionPayload.GasUsed)))
	return egp, nil
}

type extractHint struct{}

func (c *extractHint) RequiredGas(input []byte) uint64 {
	return 10000
}

func (c *extractHint) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *extractHint) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	unpacked, err := artifacts.SuaveAbi.Methods["extractHint"].Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bundleBytes := unpacked[0].([]byte)

	return c.runImpl(suaveContext, bundleBytes)
}

func (c *extractHint) Do(suaveContext *SuaveContext, bundleBytes []byte) ([]byte, error) {
	return c.runImpl(suaveContext, bundleBytes)
}

func (c *extractHint) runImpl(suaveContext *SuaveContext, bundleBytes []byte) ([]byte, error) {
	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
		RefundPercent   int                `json:"percent"`
		MatchId         types.BidId        `json:"MatchId"`
	}{}

	err := json.Unmarshal(bundleBytes, &bundle)
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

func (c *buildEthBlock) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	unpacked, err := artifacts.SuaveAbi.Methods["buildEthBlock"].Inputs.Unpack(input)
	if err != nil {
		return formatPeekerError("could not unpack inputs: %w", err)
	}

	// blockArgs := unpacked[0].(types.BuildBlockArgs)
	blockArgsRaw := unpacked[0].(struct {
		Slot           uint64         "json:\"slot\""
		ProposerPubkey []uint8        "json:\"proposerPubkey\""
		Parent         common.Hash    "json:\"parent\""
		Timestamp      uint64         "json:\"timestamp\""
		FeeRecipient   common.Address "json:\"feeRecipient\""
		GasLimit       uint64         "json:\"gasLimit\""
		Random         common.Hash    "json:\"random\""
		Withdrawals    []struct {
			Index     uint64         "json:\"index\""
			Validator uint64         "json:\"validator\""
			Address   common.Address "json:\"Address\""
			Amount    uint64         "json:\"amount\""
		} "json:\"withdrawals\""
	})

	blockArgs := types.BuildBlockArgs{
		Slot:           blockArgsRaw.Slot,
		Parent:         blockArgsRaw.Parent,
		Timestamp:      blockArgsRaw.Timestamp,
		FeeRecipient:   blockArgsRaw.FeeRecipient,
		GasLimit:       blockArgsRaw.GasLimit,
		Random:         blockArgsRaw.Random,
		ProposerPubkey: blockArgsRaw.ProposerPubkey,
		Withdrawals:    types.Withdrawals{},
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

	bidBytes, envelopeBytes, err := c.runImpl(suaveContext, blockArgs, bidId, namespace)
	if err != nil {
		return formatPeekerError("could not unpack merged bid ids: %w", err)
	}

	return artifacts.SuaveAbi.Methods["buildEthBlock"].Outputs.Pack(bidBytes, envelopeBytes)
}

func (c *buildEthBlock) Do(suaveContext *SuaveContext, blockArgs types.BuildBlockArgs, bidId types.BidId, namespace string) ([]byte, []byte, error) {
	return c.runImpl(suaveContext, blockArgs, bidId, namespace)
}

func (c *buildEthBlock) runImpl(suaveContext *SuaveContext, blockArgs types.BuildBlockArgs, bidId types.BidId, namespace string) ([]byte, []byte, error) {
	caller := suaveContext.getCaller()

	bidIds := [][16]byte{}
	// first check for merged bid, else assume regular bid
	if mergedBidsBytes, err := suaveContext.Backend.ConfidentialStore.Retrieve(bidId, caller, "default:v0:mergedBids"); err == nil {
		unpacked, err := bidIdsAbi.Inputs.Unpack(mergedBidsBytes)

		if err != nil {
			return nil, nil, fmt.Errorf("could not unpack merged bid ids: %w", err)
		}
		bidIds = unpacked[0].([][16]byte)
	} else {
		bidIds = append(bidIds, bidId)
	}

	var bidsToMerge = make([]types.Bid, len(bidIds))
	for i, bidId := range bidIds {
		var err error

		bid, err := suaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
		if err != nil {
			return nil, nil, fmt.Errorf("could not fetch bid id %v: %w", bidId, err)
		}
		bidsToMerge[i] = bid.ToInnerBid()
	}

	var mergedBundles []types.SBundle
	for _, bid := range bidsToMerge {
		switch bid.Version {
		case "mevshare:v0:matchBids":
			// fetch the matched ids and merge the bundle
			matchedBundleIdsBytes, err := suaveContext.Backend.ConfidentialStore.Retrieve(bid.Id, caller, "mevshare:v0:mergedBids")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bid ids data for bid %v, from cdas: %w", bid, err)
			}

			unpackedBidIds, err := bidIdsAbi.Inputs.Unpack(matchedBundleIdsBytes)
			if err != nil {
				return nil, nil, fmt.Errorf("could not unpack bid ids data for bid %v, from cdas: %w", bid, err)
			}

			matchBidIds := unpackedBidIds[0].([][16]byte)

			userBundleBytes, err := suaveContext.Backend.ConfidentialStore.Retrieve(matchBidIds[0], caller, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for bidId %v: %w", matchBidIds[0], err)
			}

			var userBundle types.SBundle
			if err := json.Unmarshal(userBundleBytes, &userBundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal user bundle data for bidId %v: %w", matchBidIds[0], err)
			}

			matchBundleBytes, err := suaveContext.Backend.ConfidentialStore.Retrieve(matchBidIds[1], caller, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve match bundle data for bidId %v: %w", matchBidIds[1], err)
			}

			var matchBundle types.SBundle
			if err := json.Unmarshal(matchBundleBytes, &matchBundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal match bundle data for bidId %v: %w", matchBidIds[1], err)
			}

			userBundle.Txs = append(userBundle.Txs, matchBundle.Txs...)

			mergedBundles = append(mergedBundles, userBundle)

		case "mevshare:v0:unmatchedBundles":
			bundleBytes, err := suaveContext.Backend.ConfidentialStore.Retrieve(bid.Id, caller, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}

			var bundle types.SBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}
			mergedBundles = append(mergedBundles, bundle)
		case "default:v0:ethBundles":
			bundleBytes, err := suaveContext.Backend.ConfidentialStore.Retrieve(bid.Id, caller, "default:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}

			var bundle types.SBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}
			mergedBundles = append(mergedBundles, bundle)
		default:
			return nil, nil, fmt.Errorf("unknown bid version %s", bid.Version)
		}
	}

	log.Info("requesting a block be built", "mergedBundles", mergedBundles)
	envelope, err := suaveContext.Backend.ConfidentialEthBackend.BuildEthBlockFromBundles(context.TODO(), &blockArgs, mergedBundles)
	if err != nil {
		return nil, nil, fmt.Errorf("could not build eth block: %w", err)
	}

	log.Info("built block from bundles", "payload", *envelope.ExecutionPayload)

	payload, err := executableDataToCapellaExecutionPayload(envelope.ExecutionPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("could not format execution payload as capella payload: %w", err)
	}

	// really should not be generated here
	blsSk, blsPk, err := bls.GenerateNewKeypair()
	if err != nil {
		return nil, nil, fmt.Errorf("could not generate new bls key pair: %w", err)
	}

	pk, err := boostUtils.BlsPublicKeyToPublicKey(blsPk)
	if err != nil {
		return nil, nil, fmt.Errorf("could not format bls pubkey as bytes: %w", err)
	}

	value, overflow := uint256.FromBig(envelope.BlockValue)
	if overflow {
		return nil, nil, fmt.Errorf("block value %v overflows", *envelope.BlockValue)
	}
	var proposerPubkey [48]byte
	copy(proposerPubkey[:], blockArgs.ProposerPubkey)

	blockBidMsg := builderV1.BidTrace{
		Slot:                 blockArgs.Slot,
		ParentHash:           payload.ParentHash,
		BlockHash:            payload.BlockHash,
		BuilderPubkey:        pk,
		ProposerPubkey:       phase0.BLSPubKey(proposerPubkey),
		ProposerFeeRecipient: bellatrix.ExecutionAddress(blockArgs.FeeRecipient),
		GasLimit:             envelope.ExecutionPayload.GasLimit,
		GasUsed:              envelope.ExecutionPayload.GasUsed,
		Value:                value,
	}

	// hardcoded for goerli, should be passed in with the inputs
	genesisForkVersion := phase0.Version{0x00, 0x00, 0x10, 0x20}
	builderSigningDomain := ssz.ComputeDomain(ssz.DomainTypeAppBuilder, genesisForkVersion, phase0.Root{})
	signature, err := ssz.SignMessage(&blockBidMsg, builderSigningDomain, blsSk)
	if err != nil {
		return nil, nil, fmt.Errorf("could not sign builder bid: %w", err)
	}

	bidRequest := builderCapella.SubmitBlockRequest{
		Message:          &blockBidMsg,
		ExecutionPayload: payload,
		Signature:        signature,
	}

	bidBytes, err := bidRequest.MarshalJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal builder bid request: %w", err)
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal payload envelope: %w", err)
	}

	return bidBytes, envelopeBytes, nil
}

type submitEthBlockBidToRelay struct {
}

func (c *submitEthBlockBidToRelay) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *submitEthBlockBidToRelay) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *submitEthBlockBidToRelay) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	unpacked, err := artifacts.SuaveAbi.Methods["submitEthBlockBidToRelay"].Inputs.Unpack(input)
	if err != nil {
		return formatPeekerError("could not unpack inputs: %w", err)
	}

	relayUrl := unpacked[0].(string)
	builderBidJson := unpacked[1].([]byte)

	return c.runImpl(suaveContext, relayUrl, builderBidJson)
}

func (c *submitEthBlockBidToRelay) Do(suaveContext *SuaveContext, relayUrl string, builderBidJson []byte) ([]byte, error) {
	return c.runImpl(suaveContext, relayUrl, builderBidJson)
}

func (c *submitEthBlockBidToRelay) runImpl(suaveContext *SuaveContext, relayUrl string, builderBidJson []byte) ([]byte, error) {
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
