package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	builderCapella "github.com/attestantio/go-builder-client/api/capella"
	bellatrixSpec "github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/ethereum/go-ethereum/suave/sdk"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

func TestIsConfidential(t *testing.T) {
	fr := newFramework(t)
	defer fr.Close()

	suaveLib := sdk.GetContract(suave.RuntimeAddr, artifacts.SuaveAbi, fr.NewSDKClient())

	res, err := suaveLib.Call("isConfidential", nil)
	require.NoError(t, err)
	require.True(t, res[0].(bool))

	_, err = suaveLib.SendTransaction("isConfidential", nil, nil)
	require.NoError(t, err)

	block := fr.suethSrv.ProgressChain()
	require.Equal(t, 1, len(block.Transactions()))
}

func TestMempool(t *testing.T) {
	// t.Fatal("not implemented")
	fr := newFramework(t)
	defer fr.Close()

	suaveLib := sdk.GetContract(suave.RuntimeAddr, artifacts.SuaveAbi, fr.NewSDKClient())

	{
		targetBlock := uint64(16103213)
		creationTx := types.NewTx(&types.ConfidentialComputeRequest{ExecutionNode: fr.ExecutionNode()})

		bid1, err := fr.ConfidentialEngine().InitializeBid(types.Bid{
			Salt:                suave.RandomBidId(),
			DecryptionCondition: targetBlock,
			AllowedPeekers:      []common.Address{common.HexToAddress("0x424344")},
			Version:             "default:v0:ethBundles",
		}, creationTx)

		require.NoError(t, err)

		bid2, err := fr.ConfidentialEngine().InitializeBid(types.Bid{
			Salt:                suave.RandomBidId(),
			DecryptionCondition: targetBlock,
			AllowedPeekers:      []common.Address{common.HexToAddress("0x424344")},
			Version:             "default:v0:ethBundles",
		}, creationTx)
		require.NoError(t, err)

		require.NoError(t, fr.ConfidentialStoreBackend().InitializeBid(bid1))
		require.NoError(t, fr.ConfidentialStoreBackend().InitializeBid(bid2))

		out, err := suaveLib.Call("fetchBids", []interface{}{targetBlock, "default:v0:ethBundles"})
		require.NoError(t, err)

		var bids []suave.Bid
		require.NoError(t, mapstructure.Decode(out[0], &bids))

		require.Equal(t, bid1.Id, bids[0].Id)
		require.Equal(t, bid1.Salt, bids[0].Salt)
		require.Equal(t, bid1.DecryptionCondition, bids[0].DecryptionCondition)
		require.Equal(t, bid1.AllowedPeekers, bids[0].AllowedPeekers)
		require.Equal(t, bid1.Version, bids[0].Version)

		require.Equal(t, bid2.Id, bids[1].Id)
		require.Equal(t, bid2.Salt, bids[1].Salt)
		require.Equal(t, bid2.DecryptionCondition, bids[1].DecryptionCondition)
		require.Equal(t, bid2.AllowedPeekers, bids[1].AllowedPeekers)
		require.Equal(t, bid2.Version, bids[1].Version)

		// Verify via transaction
		_, err = suaveLib.SendTransaction("fetchBids", []interface{}{targetBlock, "default:v0:ethBundles"}, nil)
		require.NoError(t, err)

		block := fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))

		receipts := block.Receipts
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.SuaveTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 1, len(block.Transactions()))
	}
}

func TestBundleBid(t *testing.T) {
	// t.Fatal("not implemented")

	fr := newFramework(t)
	defer fr.Close()

	clt := fr.NewSDKClient()

	// rpc := fr.suethSrv.RPCNode()

	{
		targetBlock := uint64(16103213)
		allowedPeekers := []common.Address{{0x41, 0x42, 0x43}, newBundleBidAddress}

		bundle := struct {
			Txs             types.Transactions `json:"txs"`
			RevertingHashes []common.Hash      `json:"revertingHashes"`
		}{
			Txs:             types.Transactions{types.NewTx(&types.LegacyTx{})},
			RevertingHashes: []common.Hash{},
		}
		bundleBytes, err := json.Marshal(bundle)
		require.NoError(t, err)

		confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		bundleBidContractI := sdk.GetContract(newBundleBidAddress, BundleBidContract.Abi, clt)
		_, err = bundleBidContractI.SendTransaction("newBid", []interface{}{targetBlock, allowedPeekers, []common.Address{}}, confidentialDataBytes)
		requireNoRpcError(t, err)

		block := fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))

		receipts := block.Receipts
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.SuaveTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 1, len(block.Transactions()))
		unpacked, err := BundleBidContract.Abi.Methods["emitBid"].Inputs.Unpack(block.Transactions()[0].Data()[4:])
		require.NoError(t, err)
		bid := unpacked[0].(struct {
			Id                  [16]uint8        "json:\"id\""
			Salt                [16]uint8        "json:\"salt\""
			DecryptionCondition uint64           "json:\"decryptionCondition\""
			AllowedPeekers      []common.Address "json:\"allowedPeekers\""
			AllowedStores       []common.Address "json:\"allowedStores\""
			Version             string           "json:\"version\""
		})
		require.Equal(t, targetBlock, bid.DecryptionCondition)
		require.Equal(t, allowedPeekers, bid.AllowedPeekers)

		require.NotNil(t, receipts[0].Logs[0])
		require.Equal(t, newBundleBidAddress, receipts[0].Logs[0].Address)

		unpacked, err = BundleBidContract.Abi.Events["BidEvent"].Inputs.Unpack(receipts[0].Logs[0].Data)
		require.NoError(t, err)

		require.Equal(t, bid.Id, unpacked[0].([16]byte))
		require.Equal(t, bid.DecryptionCondition, unpacked[1].(uint64))
		require.Equal(t, bid.AllowedPeekers, unpacked[2].([]common.Address))

		_, err = fr.ConfidentialEngine().Retrieve(bid.Id, common.Address{0x41, 0x42, 0x43}, "default:v0:ethBundleSimResults")
		require.NoError(t, err)
	}
}

func TestMevShare(t *testing.T) {
	// 1. craft mevshare transaction
	//   1a. confirm submission
	// 2. send backrun txn
	//	 2a. confirm submission
	// 3. build share block
	//   3a. confirm share bundle

	fr := newFramework(t, WithExecutionNode())
	defer fr.Close()

	rpc := fr.suethSrv.RPCNode()
	clt := fr.NewSDKClient()

	// ************ 1. Initial mevshare transaction Portion ************

	ethTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21000,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	}), signer, testKey)
	require.NoError(t, err)

	bundle := types.SBundle{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
		RefundPercent:   10,
	}
	bundleBytes, err := json.Marshal(bundle)
	t.Log("extractHint", "bundleBytes", bundleBytes)

	require.NoError(t, err)

	targetBlock := uint64(1)

	// Send a bundle bid
	allowedPeekers := []common.Address{{0x41, 0x42, 0x43}, newBlockBidAddress, mevShareAddress}

	// TODO : reusing this function selector from bid contract to avoid creating another ABI
	confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
	require.NoError(t, err)

	bundleBidContractI := sdk.GetContract(mevShareAddress, BundleBidContract.Abi, clt)
	_, err = bundleBidContractI.SendTransaction("newBid", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{fr.ExecutionNode()}}, confidentialDataBytes)
	requireNoRpcError(t, err)

	//   1a. confirm submission
	block := fr.suethSrv.ProgressChain()
	require.Equal(t, 1, len(block.Transactions()))
	// check txn in block went to mev share
	require.Equal(t, block.Transactions()[0].To(), &mevShareAddress)

	// 2b. check logs emitted in the transaction
	var r *types.Receipt
	rpc.Call(&r, "eth_getTransactionReceipt", block.Transactions()[0].Hash())
	require.NotEmpty(t, r)

	t.Log("logs", r.Logs)
	require.NoError(t, err)
	require.NotEmpty(t, r.Logs)

	// extract share BidId
	unpacked, err := MevShareBidContract.Abi.Events["HintEvent"].Inputs.Unpack(r.Logs[1].Data)
	require.NoError(t, err)
	shareBidId := unpacked[0].([16]byte)

	// ************ 2. Match Portion ************
	backrunTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21420,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	}), signer, testKey2)
	require.NoError(t, err)

	backRunBundle := types.SBundle{
		Txs:             types.Transactions{backrunTx},
		RevertingHashes: []common.Hash{},
		MatchId:         shareBidId,
	}
	backRunBundleBytes, err := json.Marshal(backRunBundle)
	require.NoError(t, err)

	// decryption conditions are assumed to be eth blocks right now

	// TODO : reusing this function selector from bid contract to avoid creating another ABI
	confidentialDataMatchBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(backRunBundleBytes)
	require.NoError(t, err)

	cc := sdk.GetContract(mevShareAddress, MevShareBidContract.Abi, clt)
	_, err = cc.SendTransaction("newMatch", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{fr.ExecutionNode()}, shareBidId}, confidentialDataMatchBytes)
	requireNoRpcError(t, err)

	block = fr.suethSrv.ProgressChain()
	require.Equal(t, 1, len(block.Transactions()))
	// check txn in block went to mev share
	require.Equal(t, block.Transactions()[0].To(), &mevShareAddress)

	var r2 *types.Receipt
	rpc.Call(&r2, "eth_getTransactionReceipt", block.Transactions()[0].Hash())
	require.NotEmpty(t, r2)
	require.NotEmpty(t, r.Logs)

	t.Log("logs", r2.Logs)

	// ************ 3. Build Share Block ************

	ethHead := fr.ethSrv.CurrentBlock()

	payloadArgsTuple := types.BuildBlockArgs{
		Timestamp:    ethHead.Time + uint64(12),
		FeeRecipient: common.Address{0x42},
	}

	cc = sdk.GetContract(newBlockBidAddress, buildEthBlockContract.Abi, clt)
	_, err = cc.SendTransaction("buildMevShare", []interface{}{payloadArgsTuple, targetBlock + 1}, nil)
	requireNoRpcError(t, err)

	block = fr.suethSrv.ProgressChain() // block = progressChain(t, ethservice, block.Header())
	require.Equal(t, 1, len(block.Transactions()))

	var r3 *types.Receipt
	requireNoRpcError(t, rpc.Call(&r3, "eth_getTransactionReceipt", block.Transactions()[0].Hash()))
	require.NotEmpty(t, r3.Logs)

	{ // Fetch the built block id and check that the payload contains mev share trasnactions!
		receipts := block.Receipts
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.SuaveTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 2, len(receipts[0].Logs))
		require.NotNil(t, receipts[0].Logs[1])
		unpacked, err := BundleBidContract.Abi.Events["BidEvent"].Inputs.Unpack(receipts[0].Logs[1].Data)
		require.NoError(t, err)

		bidId := unpacked[0].([16]byte)
		payloadData, err := fr.ConfidentialEngine().Retrieve(bidId, newBlockBidAddress, "default:v0:builderPayload")
		require.NoError(t, err)

		var payloadEnvelope engine.ExecutionPayloadEnvelope
		require.NoError(t, json.Unmarshal(payloadData, &payloadEnvelope))
		require.Equal(t, 4, len(payloadEnvelope.ExecutionPayload.Transactions)) // users tx, backrun, user refund, proposer payment

		ethBlock, err := engine.ExecutableDataToBlock(*payloadEnvelope.ExecutionPayload)
		require.NoError(t, err)

		require.Equal(t, ethTx.Hash(), ethBlock.Transactions()[0].Hash())
		require.Equal(t, backrunTx.Hash(), ethBlock.Transactions()[1].Hash())

		userAddr, err := signer.Sender(ethTx)
		require.NoError(t, err)
		require.Equal(t, userAddr, *ethBlock.Transactions()[2].To())
		require.Equal(t, common.Address{0x42}, *ethBlock.Transactions()[3].To())
	}
}

func TestBlockBuildingPrecompiles(t *testing.T) {
	fr := newFramework(t, WithExecutionNode())
	defer fr.Close()

	suaveLib := sdk.GetContract(suave.RuntimeAddr, artifacts.SuaveAbi, fr.NewSDKClient())

	ethTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21000,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	}), signer, testKey)
	require.NoError(t, err)

	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
	}{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	{ // Test the bundle simulation precompile through eth_call
		out, err := suaveLib.Call("simulateBundle", []interface{}{bundleBytes})
		require.NoError(t, err)
		require.Equal(t, 13, int(out[0].(uint64)))
	}

	{ // Test the block building precompile through eth_call
		// function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bid) internal view returns (bytes memory, bytes memory) {

		dummyCreationTx, err := types.SignNewTx(testKey, signer, &types.ConfidentialComputeRequest{
			ExecutionNode: fr.ExecutionNode(),
		})
		require.NoError(t, err)

		bid, err := fr.ConfidentialEngine().InitializeBid(types.Bid{
			DecryptionCondition: uint64(1),
			AllowedPeekers:      []common.Address{{0x41, 0x42, 0x43}, {}},
			AllowedStores:       []common.Address{fr.ExecutionNode()},
			Version:             "default:v0:ethBundles",
		}, dummyCreationTx)
		require.NoError(t, err)

		err = fr.ConfidentialEngine().Finalize(dummyCreationTx, map[suave.BidId]suave.Bid{bid.Id: bid}, []suave.StoreWrite{{
			Bid:    bid,
			Caller: common.Address{0x41, 0x42, 0x43},
			Key:    "default:v0:ethBundles",
			Value:  bundleBytes,
		}})
		require.NoError(t, err)

		ethHead := fr.ethSrv.CurrentBlock()

		payloadArgsTuple := types.BuildBlockArgs{
			Timestamp:    ethHead.Time + uint64(12),
			FeeRecipient: common.Address{0x42},
		}

		out, err := suaveLib.Call("buildEthBlock", []interface{}{payloadArgsTuple, bid.Id, ""})
		require.NoError(t, err)

		var envelope *engine.ExecutionPayloadEnvelope
		require.NoError(t, json.Unmarshal(out[1].([]byte), &envelope))
		require.Equal(t, 2, len(envelope.ExecutionPayload.Transactions))

		var tx types.Transaction
		require.NoError(t, tx.UnmarshalBinary(envelope.ExecutionPayload.Transactions[0]))

		require.Equal(t, ethTx.Data(), tx.Data())
		require.Equal(t, ethTx.Hash(), tx.Hash())
	}
}

func TestBlockBuildingContract(t *testing.T) {
	fr := newFramework(t, WithExecutionNode())
	defer fr.Close()

	clt := fr.NewSDKClient()

	ethTx, err := clt.SignTxn(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21000,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	})
	require.NoError(t, err)

	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
	}{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	targetBlock := uint64(1)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{newBlockBidAddress, newBundleBidAddress}

		confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		bundleBidContractI := sdk.GetContract(newBundleBidAddress, BundleBidContract.Abi, clt)

		_, err = bundleBidContractI.SendTransaction("newBid", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{}}, confidentialDataBytes)
		require.NoError(t, err)
	}

	block := fr.suethSrv.ProgressChain()
	require.Equal(t, 1, len(block.Transactions()))

	{
		ethHead := fr.ethSrv.CurrentBlock()

		payloadArgsTuple := types.BuildBlockArgs{
			ProposerPubkey: []byte{0x42},
			Timestamp:      ethHead.Time + uint64(12),
			FeeRecipient:   common.Address{0x42},
		}

		buildEthBlockContractI := sdk.GetContract(newBlockBidAddress, buildEthBlockContract.Abi, clt)

		_, err = buildEthBlockContractI.SendTransaction("buildFromPool", []interface{}{payloadArgsTuple, targetBlock + 1}, nil)
		require.NoError(t, err)

		block = fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))
	}
}

func TestRelayBlockSubmissionContract(t *testing.T) {
	fr := newFramework(t, WithExecutionNode())
	defer fr.Close()

	rpc := fr.suethSrv.RPCNode()
	clt := fr.NewSDKClient()

	var block *block

	var ethBlockBidSenderAddr common.Address

	var blockPayloadSentToRelay *builderCapella.SubmitBlockRequest = &builderCapella.SubmitBlockRequest{}
	serveHttp := func(t *testing.T, w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		err = json.Unmarshal(bodyBytes, blockPayloadSentToRelay)
		if err != nil {
			blockPayloadSentToRelay = nil
			require.NoError(t, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
	testHandler := &fakeRelayHandler{t, serveHttp}

	fakeRelayServer := httptest.NewServer(testHandler)
	defer fakeRelayServer.Close()

	{ // Deploy the contract
		abiEncodedRelayUrl, err := ethBlockBidSenderContract.Abi.Pack("", fakeRelayServer.URL)
		require.NoError(t, err)

		calldata := append(ethBlockBidSenderContract.Code, abiEncodedRelayUrl...)
		tx, err := clt.SignTxn(&types.LegacyTx{
			Nonce:    0,
			To:       nil, // contract creation
			Value:    big.NewInt(0),
			Gas:      10000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		})
		require.NoError(t, err)

		from, _ := types.Sender(signer, tx)
		ethBlockBidSenderAddr = crypto.CreateAddress(from, tx.Nonce())

		txBytes, err := tx.MarshalBinary()
		require.NoError(t, err)

		var txHash common.Hash
		requireNoRpcError(t, rpc.Call(&txHash, "eth_sendRawTransaction", hexutil.Encode(txBytes)))

		block = fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))
		require.Equal(t, uint64(1), block.Receipts[0].Status)
	}

	ethTx, err := clt.SignTxn(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21000,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	})
	require.NoError(t, err)

	bundle := struct {
		Txs             types.Transactions `json:"txs"`
		RevertingHashes []common.Hash      `json:"revertingHashes"`
	}{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	targetBlock := uint64(1)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{ethBlockBidSenderAddr, newBundleBidAddress}

		confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		bundleBidContractI := sdk.GetContract(newBundleBidAddress, BundleBidContract.Abi, clt)
		_, err = bundleBidContractI.SendTransaction("newBid", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{}}, confidentialDataBytes)
		requireNoRpcError(t, err)
	}

	block = fr.suethSrv.ProgressChain()
	require.Equal(t, 1, len(block.Transactions()))

	{
		ethHead := fr.ethSrv.CurrentBlock()

		payloadArgsTuple := types.BuildBlockArgs{
			ProposerPubkey: []byte{0x42},
			Timestamp:      ethHead.Time + uint64(12),
			FeeRecipient:   common.Address{0x42},
		}

		ethBlockBidSenderContractI := sdk.GetContract(ethBlockBidSenderAddr, ethBlockBidSenderContract.Abi, clt)
		_, err = ethBlockBidSenderContractI.SendTransaction("buildFromPool", []interface{}{payloadArgsTuple, targetBlock + 1}, nil)
		require.NoError(t, err)

		block = fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))
	}

	require.NotNil(t, blockPayloadSentToRelay)
	require.NotNil(t, blockPayloadSentToRelay.ExecutionPayload)
	require.Equal(t, 2, len(blockPayloadSentToRelay.ExecutionPayload.Transactions)) // Should be 2, including the proposer payment tx - todo
	ethTxBytes, _ := ethTx.MarshalBinary()
	require.Equal(t, bellatrixSpec.Transaction(ethTxBytes), blockPayloadSentToRelay.ExecutionPayload.Transactions[0])

	require.Equal(t, bellatrixSpec.ExecutionAddress(common.Address{0x42}), blockPayloadSentToRelay.Message.ProposerFeeRecipient)
	require.Equal(t, phase0.BLSPubKey{0x42}, blockPayloadSentToRelay.Message.ProposerPubkey)

	builderPubkey := blockPayloadSentToRelay.Message.BuilderPubkey
	signature := blockPayloadSentToRelay.Signature
	builderSigningDomain := ssz.ComputeDomain(ssz.DomainTypeAppBuilder, phase0.Version{0x00, 0x00, 0x10, 0x20}, phase0.Root{})
	ok, err := ssz.VerifySignature(blockPayloadSentToRelay.Message, builderSigningDomain, builderPubkey[:], signature[:])
	require.NoError(t, err)
	require.True(t, ok)
}

type clientWrapper struct {
	t *testing.T

	node    *node.Node
	service *eth.Ethereum
}

func (c *clientWrapper) Close() {
	c.node.Close()
}

func (c *clientWrapper) RPCNode() *rpc.Client {
	rpc, err := c.node.Attach()
	if err != nil {
		c.t.Fatal(err)
	}
	return rpc
}

func (c *clientWrapper) CurrentBlock() *types.Header {
	return c.service.BlockChain().CurrentBlock()
}

type block struct {
	*types.Block

	Receipts []*types.Receipt
}

func (c *clientWrapper) ProgressChain() *block {
	tBlock := progressChain(c.t, c.service, c.service.BlockChain().CurrentBlock())
	receipts := c.service.BlockChain().GetReceiptsByHash(tBlock.Hash())

	return &block{
		Block:    tBlock,
		Receipts: receipts,
	}
}

type framework struct {
	t *testing.T

	ethSrv   *clientWrapper
	suethSrv *clientWrapper
}

type frameworkConfig struct {
	executionNode     bool
	redisStoreBackend bool
	suaveConfig       suave.Config
}

var defaultFrameworkConfig = frameworkConfig{
	executionNode:     false,
	redisStoreBackend: false,
	suaveConfig:       suave.Config{},
}

type frameworkOpt func(*frameworkConfig)

func WithExecutionNode() frameworkOpt {
	return func(c *frameworkConfig) {
		c.executionNode = true
	}
}

func WithRedisStoreBackend() frameworkOpt {
	return func(c *frameworkConfig) {
		c.redisStoreBackend = true
	}
}

func WithRedisTransportOpt(t *testing.T) frameworkOpt {
	mr := miniredis.RunT(t)
	return func(c *frameworkConfig) {
		c.suaveConfig.RedisStorePubsubUri = mr.Addr()
	}
}

func newFramework(t *testing.T, opts ...frameworkOpt) *framework {
	cfg := defaultFrameworkConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var ethSrv *clientWrapper

	if cfg.executionNode {
		ethNode, ethEthService := startEthService(t, testEthGenesis, nil)
		ethEthService.APIs()

		ethSrv = &clientWrapper{t, ethNode, ethEthService}

		cfg.suaveConfig.SuaveEthRemoteBackendEndpoint = ethNode.HTTPEndpoint()
	}

	if cfg.redisStoreBackend {
		mr := miniredis.RunT(t)
		cfg.suaveConfig.RedisStoreUri = mr.Addr()
	}

	node, ethservice := startSuethService(t, testSuaveGenesis, nil, cfg.suaveConfig)

	f := &framework{
		t:        t,
		ethSrv:   ethSrv,
		suethSrv: &clientWrapper{t, node, ethservice},
	}

	return f
}

func (f *framework) NewSDKClient() *sdk.Client {
	return sdk.NewClient(f.suethSrv.RPCNode(), testKey, f.ExecutionNode())
}

func (f *framework) ConfidentialStoreBackend() suave.ConfidentialStoreBackend {
	return f.suethSrv.service.APIBackend.SuaveEngine().Backend()
}

func (f *framework) ConfidentialEngine() *suave.ConfidentialStoreEngine {
	return f.suethSrv.service.APIBackend.SuaveEngine()
}

func (f *framework) ExecutionNode() common.Address {
	return f.suethSrv.service.AccountManager().Accounts()[0]
}

func (f *framework) Close() {
	if f.ethSrv != nil {
		f.ethSrv.Close()
	}
	f.suethSrv.Close()
}

// Utilities

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)

	testKey2, _ = crypto.HexToECDSA("a71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr2 = crypto.PubkeyToAddress(testKey2.PublicKey)

	testBalance = big.NewInt(2e18)

	isConfidentialAddress = common.HexToAddress("0x42010000")

	newBundleBidAddress   = common.HexToAddress("0x42300000")
	newBlockBidAddress    = common.HexToAddress("0x42300001")
	blockBidSenderAddress = common.HexToAddress("0x42300002")
	mevShareAddress       = common.HexToAddress("0x42100073")

	testSuaveGenesis *core.Genesis = &core.Genesis{
		Timestamp:  1680000000,
		ExtraData:  nil,
		GasLimit:   30000000,
		BaseFee:    big.NewInt(0),
		Difficulty: big.NewInt(0),
		Alloc: core.GenesisAlloc{
			testAddr:              {Balance: testBalance},
			testAddr2:             {Balance: testBalance},
			newBundleBidAddress:   {Balance: big.NewInt(0), Code: BundleBidContract.DeployedCode},
			newBlockBidAddress:    {Balance: big.NewInt(0), Code: buildEthBlockContract.DeployedCode},
			blockBidSenderAddress: {Balance: big.NewInt(0), Code: ethBlockBidSenderContract.DeployedCode},
			mevShareAddress:       {Balance: big.NewInt(0), Code: MevShareBidContract.DeployedCode},
		},
	}

	testEthGenesis *core.Genesis = &core.Genesis{
		Timestamp:  1680000000,
		ExtraData:  nil,
		GasLimit:   30000000,
		BaseFee:    big.NewInt(0),
		Difficulty: big.NewInt(0),
		Alloc: core.GenesisAlloc{
			testAddr:  {Balance: testBalance},
			testAddr2: {Balance: testBalance},
		},
	}

	signer = types.NewSuaveSigner(params.AllEthashProtocolChanges.ChainID)
)

func init() {
	suaveConfig := *params.AllEthashProtocolChanges
	suaveConfig.TerminalTotalDifficulty = new(big.Int)
	testSuaveGenesis.Config = &suaveConfig

	ethConfig := *params.AllEthashProtocolChanges
	ethConfig.TerminalTotalDifficulty = new(big.Int)
	ethConfig.SuaveBlock = nil
	testEthGenesis.Config = &ethConfig
}

// startSuethService creates a full node instance for testing.
func startSuethService(t *testing.T, genesis *core.Genesis, blocks []*types.Block, suaveConfig suave.Config) (*node.Node, *eth.Ethereum) {
	t.Helper()

	n, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		}})
	if err != nil {
		t.Fatal("can't create node:", err)
	}

	ethcfg := &ethconfig.Config{Genesis: genesis, SyncMode: downloader.FullSync, TrieTimeout: time.Minute, TrieDirtyCache: 256, TrieCleanCache: 256, Suave: suaveConfig}
	ethservice, err := eth.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}
	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks); err != nil {
		n.Close()
		t.Fatal("can't import test blocks:", err)
	}

	ethservice.SetEtherbase(testAddr)
	ethservice.SetSynced()

	keydir := t.TempDir()
	keystore := keystore.NewPlaintextKeyStore(keydir)
	acc, err := keystore.NewAccount("")
	require.NoError(t, err)
	require.NoError(t, keystore.TimedUnlock(acc, "", time.Hour))

	ethservice.AccountManager().AddBackend(keystore)
	return n, ethservice
}

func startEthService(t *testing.T, genesis *core.Genesis, blocks []*types.Block) (*node.Node, *eth.Ethereum) {
	t.Helper()

	n, err := node.New(&node.Config{
		HTTPHost: "127.0.0.1",
	})
	if err != nil {
		t.Fatal("can't create node:", err)
	}

	ethcfg := &ethconfig.Config{Genesis: genesis, SyncMode: downloader.FullSync, TrieTimeout: time.Minute, TrieDirtyCache: 256, TrieCleanCache: 256}
	ethservice, err := eth.New(n, ethcfg)
	if err != nil {
		t.Fatal("can't create eth service:", err)
	}
	if err := n.Start(); err != nil {
		t.Fatal("can't start node:", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks); err != nil {
		n.Close()
		t.Fatal("can't import test blocks:", err)
	}

	ethservice.SetEtherbase(testAddr)
	ethservice.SetSynced()
	return n, ethservice
}

func progressChain(t *testing.T, ethservice *eth.Ethereum, parent *types.Header) *types.Block {
	payload, err := ethservice.Miner().BuildPayload(&miner.BuildPayloadArgs{
		Parent:       parent.Hash(),
		Timestamp:    parent.Time + 12,
		FeeRecipient: testAddr,
	})
	require.NoError(t, err)
	envelope := payload.ResolveFull()
	block, err := engine.ExecutableDataToBlock(*envelope.ExecutionPayload)
	require.NoError(t, err)

	n, err := ethservice.BlockChain().InsertChain(types.Blocks{block})
	require.NoError(t, err)
	require.Equal(t, 1, n)

	return block
}

func requireNoRpcError(t *testing.T, rpcErr error) {
	if rpcErr != nil {
		if len(rpcErr.Error()) < len("execution reverted: 0x") {
			require.NoError(t, rpcErr)
		}
		decodedError, err := hexutil.Decode(rpcErr.Error()[len("execution reverted: "):])
		if err != nil {
			require.NoError(t, rpcErr, err.Error())
		}

		if len(decodedError) < 4 {
			require.NoError(t, rpcErr, decodedError)
		}

		unpacked, err := artifacts.SuaveAbi.Errors["PeekerReverted"].Inputs.Unpack(decodedError[4:])
		if err != nil {
			require.NoError(t, err, rpcErr.Error())
		} else {
			require.NoError(t, rpcErr, fmt.Sprintf("peeker 0x%x reverted: %s", unpacked[0].(common.Address), unpacked[1].([]byte)))
		}
	}
}

type fakeRelayHandler struct {
	t         *testing.T
	serveHttp func(t *testing.T, w http.ResponseWriter, r *http.Request)
}

func (h *fakeRelayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.serveHttp(h.t, w, r)
}

func mustParseMethodAbi(data string, method string) abi.Method {
	inoutAbi, err := abi.JSON(strings.NewReader(data))
	if err != nil {
		panic(err.Error())
	}

	return inoutAbi.Methods[method]
}

func setTxArgsDefaults(args ethapi.TransactionArgs) ethapi.TransactionArgs {
	if args.Gas == nil {
		gas := hexutil.Uint64(1000000)
		args.Gas = &gas
	}

	if args.Nonce == nil {
		nonce := hexutil.Uint64(0)
		args.Nonce = &nonce
	}

	if args.GasPrice == nil {
		value := big.NewInt(0)
		args.GasPrice = (*hexutil.Big)(value)
	}

	if args.Value == nil {
		value := big.NewInt(0)
		args.Value = (*hexutil.Big)(value)
	}

	return args
}
