package vm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/beacon/dencun"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/holiman/uint256"

	builderDeneb "github.com/attestantio/go-builder-client/api/deneb"
	builderV1 "github.com/attestantio/go-builder-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	specCapella "github.com/attestantio/go-eth2-client/spec/capella"
	specDeneb "github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
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

func (b *suaveRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, dataID types.DataId, namespace string) ([]byte, []byte, error) {
	dataIDs := [][16]byte{}
	// first check for merged record, else assume regular record
	if mergedDataRecordsBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(dataID, buildEthBlockAddr, "default:v0:mergedDataRecords"); err == nil {
		unpacked, err := dataIDsAbi.Inputs.Unpack(mergedDataRecordsBytes)

		if err != nil {
			return nil, nil, fmt.Errorf("could not unpack merged record ids: %w", err)
		}
		dataIDs = unpacked[0].([][16]byte)
	} else {
		dataIDs = append(dataIDs, dataID)
	}

	var recordsToMerge = make([]types.DataRecord, len(dataIDs))
	for i, dataID := range dataIDs {
		var err error

		record, err := b.suaveContext.Backend.ConfidentialStore.FetchRecordByID(dataID)
		if err != nil {
			return nil, nil, fmt.Errorf("could not fetch record id %v: %w", dataID, err)
		}

		if _, err := checkIsPrecompileCallAllowed(b.suaveContext, buildEthBlockAddr, record); err != nil {
			return nil, nil, err
		}

		recordsToMerge[i] = record.ToInnerRecord()
	}

	var mergedBundles []types.SBundle
	for _, record := range recordsToMerge {
		switch record.Version {
		case "mevshare:v0:matchDataRecords":
			// fetch the matched ids and merge the bundle
			matchedBundleIdsBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(record.Id, buildEthBlockAddr, "mevshare:v0:mergedDataRecords")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve record ids data for record %v, from cdas: %w", record, err)
			}

			unpackeddataIDs, err := dataIDsAbi.Inputs.Unpack(matchedBundleIdsBytes)
			if err != nil {
				return nil, nil, fmt.Errorf("could not unpack record ids data for record %v, from cdas: %w", record, err)
			}

			matchdataIDs := unpackeddataIDs[0].([][16]byte)

			userBundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(matchdataIDs[0], buildEthBlockAddr, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for dataID %v: %w", matchdataIDs[0], err)
			}

			var userBundle types.SBundle
			if err := json.Unmarshal(userBundleBytes, &userBundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal user bundle data for dataID %v: %w", matchdataIDs[0], err)
			}

			matchBundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(matchdataIDs[1], buildEthBlockAddr, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve match bundle data for dataID %v: %w", matchdataIDs[1], err)
			}

			var matchBundle types.SBundle
			if err := json.Unmarshal(matchBundleBytes, &matchBundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal match bundle data for dataID %v: %w", matchdataIDs[1], err)
			}

			userBundle.Txs = append(userBundle.Txs, matchBundle.Txs...)

			mergedBundles = append(mergedBundles, userBundle)

		case "mevshare:v0:unmatchedBundles":
			bundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(record.Id, buildEthBlockAddr, "mevshare:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for dataID %v, from cdas: %w", record.Id, err)
			}

			var bundle types.SBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal bundle data for dataID %v, from cdas: %w", record.Id, err)
			}
			mergedBundles = append(mergedBundles, bundle)
		case "default:v0:ethBundles":
			bundleBytes, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(record.Id, buildEthBlockAddr, "default:v0:ethBundles")
			if err != nil {
				return nil, nil, fmt.Errorf("could not retrieve bundle data for dataID %v, from cdas: %w", record.Id, err)
			}

			var bundle types.SBundle
			if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
				return nil, nil, fmt.Errorf("could not unmarshal bundle data for dataID %v, from cdas: %w", record.Id, err)
			}
			mergedBundles = append(mergedBundles, bundle)
		default:
			return nil, nil, fmt.Errorf("unknown record version %s", record.Version)
		}
	}

	log.Info("requesting a block be built", "mergedBundles", mergedBundles)

	envelope, err := b.suaveContext.Backend.ConfidentialEthBackend.BuildEthBlockFromBundles(context.TODO(), &blockArgs, mergedBundles)

	if err != nil {
		return nil, nil, fmt.Errorf("could not build eth block: %w", err)
	}

	log.Info("built block from bundles", "payload", *envelope.ExecutionPayload)

	payload, err := executableDataToDenebExecutionPayload(envelope.ExecutionPayload)
	if err != nil {
		log.Warn("failed to generate execution payload from executable data",
			"reason", err)
		return nil, nil, fmt.Errorf("could not format execution payload as deneb payload: %w", err)
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

	// hardcoded for holesky, should be passed in with the inputs
	genesisForkVersion := phase0.Version{0x01, 0x01, 0x70, 0x00}
	builderSigningDomain := ssz.ComputeDomain(ssz.DomainTypeAppBuilder, genesisForkVersion, phase0.Root{})
	signature, err := ssz.SignMessage(&blockBidMsg, builderSigningDomain, b.suaveContext.Backend.EthBlockSigningKey)
	if err != nil {
		return nil, nil, fmt.Errorf("could not sign builder record: %w", err)
	}

	bidRequest := builderDeneb.SubmitBlockRequest{
		Message:          &blockBidMsg,
		ExecutionPayload: payload,
		Signature:        signature,
		BlobsBundle:      &builderDeneb.BlobsBundle{},
	}

	bidBytes, err := bidRequest.MarshalJSON()
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal builder record request: %w", err)
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, nil, fmt.Errorf("could not marshal payload envelope: %w", err)
	}

	return bidBytes, envelopeBytes, nil
}

func (b *suaveRuntime) privateKeyGen(cryptoType types.CryptoSignature) (string, error) {
	if cryptoType == types.CryptoSignature_SECP256 {
		sk, err := crypto.GenerateKey()
		if err != nil {
			return "", fmt.Errorf("could not generate new a private key: %w", err)
		}
		return hex.EncodeToString(crypto.FromECDSA(sk)), nil
	} else if cryptoType == types.CryptoSignature_BLS {
		sk, err := bls.GenerateRandomSecretKey()
		if err != nil {
			return "", fmt.Errorf("could not generate new a private key: %w", err)
		}
		return hex.EncodeToString(sk.Marshal()), nil
	}

	return "", fmt.Errorf("unsupported crypto type %v", cryptoType)
}

func (b *suaveRuntime) submitEthBlockToRelay(relayUrl string, builderDataRecordJson []byte) ([]byte, error) {
	endpoint := relayUrl + "/relay/v1/builder/blocks"

	httpReq := types.HttpRequest{
		Method: http.MethodPost,
		Url:    endpoint,
		Body:   builderDataRecordJson,
		Headers: []string{
			"Content-Type:application/json",
			"Accept:application/json",
		},
	}

	resp, err := b.doHTTPRequest(httpReq)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func executableDataToDenebExecutionPayload(data *dencun.ExecutableData) (*specDeneb.ExecutionPayload, error) {
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

	baseFeePerGas := new(uint256.Int)
	if data.BaseFeePerGas == nil {
		return nil, errors.New("base fee per gas: not provided")
	} else if baseFeePerGas.SetFromBig(data.BaseFeePerGas) {
		return nil, errors.New("base fee per gas: overflow")
	}

	payload := &specDeneb.ExecutionPayload{
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
		BaseFeePerGas: baseFeePerGas,
		BlockHash:     [32]byte(data.BlockHash),
		Transactions:  transactionData,
		Withdrawals:   withdrawalData,
	}

	if data.BlobGasUsed != nil {
		payload.BlobGasUsed = *data.BlobGasUsed
	}
	if data.ExcessBlobGas != nil {
		payload.ExcessBlobGas = *data.ExcessBlobGas
	}

	return payload, nil
}

func (c *suaveRuntime) submitBundleJsonRPC(url string, method string, params []byte) ([]byte, error) {
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

	httpReq := types.HttpRequest{
		Method: http.MethodPost,
		Url:    url,
		Body:   body,
		Headers: []string{
			"Content-Type:application/json",
			"Accept:application/json",
			"X-Flashbots-Signature:" + signature,
		},
	}
	if _, err := c.doHTTPRequest(httpReq); err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *suaveRuntime) fillMevShareBundle(dataID types.DataId) ([]byte, error) {
	record, err := c.suaveContext.Backend.ConfidentialStore.FetchRecordByID(dataID)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	if _, err := checkIsPrecompileCallAllowed(c.suaveContext, fillMevShareBundleAddr, record); err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	matchedBundleIdsBytes, err := c.confidentialRetrieve(dataID, "mevshare:v0:mergedDataRecords")
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	unpackedDataIDs, err := dataIDsAbi.Inputs.Unpack(matchedBundleIdsBytes)
	if err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("could not unpack record ids data for record %v, from cdas: %w", record, err)
	}

	matchDataIDs := unpackedDataIDs[0].([][16]byte)

	userBundleBytes, err := c.confidentialRetrieve(matchDataIDs[0], "mevshare:v0:ethBundles")
	if err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("could not retrieve bundle data for dataID %v: %w", matchDataIDs[0], err)
	}

	var userBundle types.SBundle
	if err := json.Unmarshal(userBundleBytes, &userBundle); err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("could not unmarshal user bundle data for dataID %v: %w", matchDataIDs[0], err)
	}

	matchBundleBytes, err := c.confidentialRetrieve(matchDataIDs[1], "mevshare:v0:ethBundles")
	if err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("could not retrieve match bundle data for dataID %v: %w", matchDataIDs[1], err)
	}

	var matchBundle types.SBundle
	if err := json.Unmarshal(matchBundleBytes, &matchBundle); err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("could not unmarshal match bundle data for dataID %v: %w", matchDataIDs[1], err)
	}

	shareBundle := &types.RPCMevShareBundle{
		Version: "v0.1",
	}

	shareBundle.Inclusion.Block = hexutil.EncodeUint64(record.DecryptionCondition)
	shareBundle.Inclusion.MaxBlock = hexutil.EncodeUint64(record.DecryptionCondition + 25) // Assumes 25 block inclusion range

	for _, tx := range append(userBundle.Txs, matchBundle.Txs...) {
		txBytes, err := tx.MarshalBinary()
		if err != nil {
			fmt.Println(err.Error())
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
