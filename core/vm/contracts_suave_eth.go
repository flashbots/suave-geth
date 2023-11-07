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

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
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

func (s *suaveRuntime) signEthTransaction(txn []byte, chainId string, signingKey string) ([]byte, error) {
	key, err := crypto.HexToECDSA(signingKey)
	if err != nil {
		return nil, fmt.Errorf("key not formatted properly: %w", err)
	}

	chainIdInt, err := hexutil.DecodeBig(chainId)
	if err != nil {
		return nil, fmt.Errorf("chainId not formatted properly: %w", err)
	}

	var tx types.Transaction
	err = tx.UnmarshalBinary(txn)
	if err != nil {
		return nil, fmt.Errorf("txn not formatted properly: %w", err)
	}

	signer := types.LatestSignerForChainID(chainIdInt)

	signedTx, err := types.SignTx(&tx, signer, key)
	if err != nil {
		return nil, fmt.Errorf("could not sign: %w", err)
	}

	signedBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("could not encode signed transaction: %w", err)
	}

	return signedBytes, nil
}

func (b *suaveRuntime) simulateBundle(input []byte) (uint64, error) {
	var bundle types.SBundle
	err := json.Unmarshal(input, &bundle)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

	envelope, err := b.suaveContext.Backend.ConfidentialEthBackend.BuildEthBlock(ctx, nil, bundle.Txs)
	if err != nil {
		return 0, err
	}

	if envelope.ExecutionPayload.GasUsed == 0 {
		return 0, err
	}

	egp := new(big.Int).Div(envelope.BlockValue, big.NewInt(int64(envelope.ExecutionPayload.GasUsed)))
	return egp.Uint64(), nil
}

func (b *suaveRuntime) extractHint(bundleBytes []byte) ([]byte, error) {
	var bundle types.SBundle
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

func (b *suaveRuntime) ethcall(contractAddr common.Address, input []byte) ([]byte, error) {
	return b.suaveContext.Backend.ConfidentialEthBackend.Call(context.Background(), contractAddr, input)
}

func (b *suaveRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, bidId types.BidId, namespace string) ([]byte, []byte, error) {
	bidIds := [][16]byte{}
	// first check for merged bid, else assume regular bid
	if mergedBidsBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(bidId, buildEthBlockAddr, "default:v0:mergedBids"); err == nil {
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

		bid, err := b.suaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
		if err != nil {
			return nil, nil, fmt.Errorf("could not fetch bid id %v: %w", bidId, err)
		}

		if _, err := checkIsPrecompileCallAllowed(b.suaveContext, buildEthBlockAddr, bid); err != nil {
			return nil, nil, err
		}

		bidsToMerge[i] = bid.ToInnerBid()
	}

	var mergedBundles []types.SBundle
	for _, bid := range bidsToMerge {
		switch bid.Version {
		case "mevshare:v0:matchBids":
			// fetch the matched ids and merge the bundle
			matchedBundleIdsBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(bid.Id, buildEthBlockAddr, "mevshare:v0:mergedBids")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bid ids data for bid %v, from cdas: %w", bid, err)
			}

			unpackedBidIds, err := bidIdsAbi.Inputs.Unpack(matchedBundleIdsBytes)
			if err != nil {
				return nil, nil, fmt.Errorf("could not unpack bid ids data for bid %v, from cdas: %w", bid, err)
			}

			matchBidIds := unpackedBidIds[0].([][16]byte)

			userBundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(matchBidIds[0], buildEthBlockAddr, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for bidId %v: %w", matchBidIds[0], err)
			}

			var userBundle types.SBundle
			if err := json.Unmarshal(userBundleBytes, &userBundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal user bundle data for bidId %v: %w", matchBidIds[0], err)
			}

			matchBundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(matchBidIds[1], buildEthBlockAddr, "mevshare:v0:ethBundles")
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
			bundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(bid.Id, buildEthBlockAddr, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}

			var bundle types.SBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal bundle data for bidId %v, from cdas: %w", bid.Id, err)
			}
			mergedBundles = append(mergedBundles, bundle)
		case "default:v0:ethBundles":
			bundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(bid.Id, buildEthBlockAddr, "default:v0:ethBundles")
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
	envelope, err := b.suaveContext.Backend.ConfidentialEthBackend.BuildEthBlockFromBundles(context.TODO(), &blockArgs, mergedBundles)
	if err != nil {
		return nil, nil, fmt.Errorf("could not build eth block: %w", err)
	}

	log.Info("built block from bundles", "payload", *envelope.ExecutionPayload)

	payload, err := executableDataToCapellaExecutionPayload(envelope.ExecutionPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("could not format execution payload as capella payload: %w", err)
	}

	blsPk, err := bls.PublicKeyFromSecretKey(b.suaveContext.Backend.EthBlockSigningKey)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get bls pubkey: %w", err)
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
	signature, err := ssz.SignMessage(&blockBidMsg, builderSigningDomain, b.suaveContext.Backend.EthBlockSigningKey)
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

func (b *suaveRuntime) submitEthBlockBidToRelay(relayUrl string, builderBidJson []byte) ([]byte, error) {
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

func (c *suaveRuntime) submitBundleJsonRPC(url string, method string, params []byte) ([]byte, error) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(3*time.Second))
	defer cancel()

	request := map[string]interface{}{
		"id":      json.RawMessage([]byte("1")),
		"jsonrpc": "2.0",
		"method":  method,
		"params":  []interface{}{json.RawMessage(params)},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	hashedBody := crypto.Keccak256Hash(body).Hex()
	sig, err := crypto.Sign(accounts.TextHash([]byte(hashedBody)), c.suaveContext.Backend.EthBundleSigningKey)
	if err != nil {
		return nil, err
	}

	signature := crypto.PubkeyToAddress(c.suaveContext.Backend.EthBundleSigningKey.PublicKey).Hex() + ":" + hexutil.Encode(sig)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return formatPeekerError("could not prepare request to relay: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Flashbots-Signature", signature)

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return formatPeekerError("could not send request to relay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return formatPeekerError("request failed with code %d", resp.StatusCode)
		}

		return formatPeekerError("request failed with code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil, nil
}

func (c *suaveRuntime) fillMevShareBundle(bidId types.BidId) ([]byte, error) {
	bid, err := c.suaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
	if err != nil {
		return nil, err
	}

	if _, err := checkIsPrecompileCallAllowed(c.suaveContext, fillMevShareBundleAddr, bid); err != nil {
		return nil, err
	}

	matchedBundleIdsBytes, err := c.confidentialRetrieve(bidId, "mevshare:v0:mergedBids")
	if err != nil {
		return nil, err
	}

	unpackedBidIds, err := bidIdsAbi.Inputs.Unpack(matchedBundleIdsBytes)
	if err != nil {
		return nil, fmt.Errorf("could not unpack bid ids data for bid %v, from cdas: %w", bid, err)
	}

	matchBidIds := unpackedBidIds[0].([][16]byte)

	userBundleBytes, err := c.confidentialRetrieve(matchBidIds[0], "mevshare:v0:ethBundles")
	if err != nil {
		return nil, fmt.Errorf("could not retrieve bundle data for bidId %v: %w", matchBidIds[0], err)
	}

	var userBundle types.SBundle
	if err := json.Unmarshal(userBundleBytes, &userBundle); err != nil {
		return nil, fmt.Errorf("could not unmarshal user bundle data for bidId %v: %w", matchBidIds[0], err)
	}

	matchBundleBytes, err := c.confidentialRetrieve(matchBidIds[1], "mevshare:v0:ethBundles")
	if err != nil {
		return nil, fmt.Errorf("could not retrieve match bundle data for bidId %v: %w", matchBidIds[1], err)
	}

	var matchBundle types.SBundle
	if err := json.Unmarshal(matchBundleBytes, &matchBundle); err != nil {
		return nil, fmt.Errorf("could not unmarshal match bundle data for bidId %v: %w", matchBidIds[1], err)
	}

	shareBundle := &types.RPCMevShareBundle{
		Version: "v0.1",
	}

	shareBundle.Inclusion.Block = hexutil.EncodeUint64(bid.DecryptionCondition)

	for _, tx := range append(userBundle.Txs, matchBundle.Txs...) {
		txBytes, err := tx.MarshalBinary()
		if err != nil {
			return nil, fmt.Errorf("could not marshal transaction: %w", err)
		}

		shareBundle.Body = append(shareBundle.Body, struct {
			Tx        string `json:"tx"`
			CanRevert bool   `json:"canRevert"`
		}{Tx: hexutil.Encode(txBytes)})
	}

	for i := range userBundle.Txs {
		refundPercent := 10
		if userBundle.RefundPercent != nil {
			refundPercent = *userBundle.RefundPercent
		}
		shareBundle.Validity.Refund = append(shareBundle.Validity.Refund, struct {
			BodyIdx int `json:"bodyIdx"`
			Percent int `json:"percent"`
		}{
			BodyIdx: i,
			Percent: refundPercent,
		})
	}

	return json.Marshal(shareBundle)
}
