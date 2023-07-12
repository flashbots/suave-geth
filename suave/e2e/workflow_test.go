package main

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/flashbots/go-boost-utils/ssz"
	boostSsz "github.com/flashbots/go-boost-utils/ssz"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIsOffchain(t *testing.T) {
	// t.Fatal("not implemented")
	node, ethservice := startSuethService(t, testSuaveGenesis, nil, suave.Config{})
	defer node.Close()

	rpc, err := node.Attach()
	require.NoError(t, err)

	gas := hexutil.Uint64(1000000)
	chainId := hexutil.Big(*testSuaveGenesis.Config.ChainID)

	{
		// Verify eth_call of isOffchainAddress returns 1/0 depending on offchain setting
		var result string
		requireNoRpcError(t, rpc.Call(&result, "eth_call", ethapi.TransactionArgs{
			To:         &isOffchainAddress,
			Gas:        &gas,
			IsOffchain: true,
			ChainID:    &chainId,
		}, "latest"))
		require.Equal(t, []byte{1}, hexutil.MustDecode(result))

		requireNoRpcError(t, rpc.Call(&result, "eth_call", ethapi.TransactionArgs{
			To:         &isOffchainAddress,
			Gas:        &gas,
			IsOffchain: false,
			ChainID:    &chainId,
		}, "latest"))
		require.Equal(t, []byte{0}, hexutil.MustDecode(result))
	}

	{
		// Verify sending offchain and onchain transactions to isOffchainAddress
		wrappedTxData := &types.LegacyTx{
			Nonce:    0,
			To:       &isOffchainAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     []byte{},
		}

		offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
			ExecutionNode: ethservice.AccountManager().Accounts()[0],
			Wrapped:       *types.NewTx(wrappedTxData),
		}), signer, testKey)
		require.NoError(t, err)

		offchainTxBytes, err := offchainTx.MarshalBinary()
		require.NoError(t, err)

		var offchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes)))

		onchainTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    1,
			To:       &isOffchainAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     []byte{},
		}), signer, testKey)
		require.NoError(t, err)

		onchainTxBytes, err := onchainTx.MarshalBinary()
		require.NoError(t, err)

		var onchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&onchainTxHash, "eth_sendRawTransaction", hexutil.Encode(onchainTxBytes)))
		require.Equal(t, common.HexToHash("0x031415a9010d25f2a882758cf7b8dbb3750678828e9973f32f0c73ef49a038b4"), onchainTxHash)

		block := progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
		require.Equal(t, 2, len(block.Transactions()))

		receipts := ethservice.BlockChain().GetReceiptsByHash(block.Hash())
		require.Equal(t, 2, len(receipts))
		require.Equal(t, uint8(types.OffchainExecutedTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)
		require.Equal(t, uint8(types.LegacyTxType), receipts[1].Type)
		require.Equal(t, uint64(1), receipts[1].Status)

		require.Equal(t, 2, len(block.Transactions()))
		require.Equal(t, []byte{1}, block.Transactions()[0].Data()) // offchain execution relays the offchain result
		require.Equal(t, []byte{}, block.Transactions()[1].Data())
	}
}

func TestMempool(t *testing.T) {
	// t.Fatal("not implemented")
	node, ethservice := startSuethService(t, testSuaveGenesis, nil, suave.Config{})
	defer node.Close()

	rpc, err := node.Attach()
	require.NoError(t, err)

	gas := hexutil.Uint64(1000000)
	chainId := hexutil.Big(*testSuaveGenesis.Config.ChainID)

	{
		targetBlock := uint64(16103213)

		bid1 := suave.Bid{
			Id:                  suave.BidId(uuid.New()),
			DecryptionCondition: targetBlock,
			AllowedPeekers:      []common.Address{common.HexToAddress("0x424344")},
			Version:             "default:v0",
		}

		bid2 := suave.Bid{
			Id:                  suave.BidId(uuid.New()),
			DecryptionCondition: targetBlock,
			AllowedPeekers:      []common.Address{common.HexToAddress("0x424344")},
			Version:             "default:v0",
		}

		ethservice.APIBackend.OffchainBackend().MempoolBackned.SubmitBid(bid1)
		ethservice.APIBackend.OffchainBackend().MempoolBackned.SubmitBid(bid2)

		inoutAbi := mustParseMethodAbi(`[ { "inputs": [ { "internalType": "uint64", "name": "cond", "type": "uint64" }, { "internalType": "string", "name": "namespace", "type": "string" } ], "name": "fetchBids", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "string", "name": "version", "type": "string" } ], "internalType": "struct Suave.Bid[]", "name": "", "type": "tuple[]" } ], "stateMutability": "view", "type": "function" } ]`, "fetchBids")

		calldata, err := inoutAbi.Inputs.Pack(targetBlock, "default:v0")
		require.NoError(t, err)

		var simResult hexutil.Bytes
		requireNoRpcError(t, rpc.Call(&simResult, "eth_call", ethapi.TransactionArgs{
			To:         &fetchBidsAddress,
			Gas:        &gas,
			IsOffchain: true,
			ChainID:    &chainId,
			Data:       (*hexutil.Bytes)(&calldata),
		}, "latest"))

		unpacked, err := inoutAbi.Outputs.Unpack(simResult)
		require.NoError(t, err)

		bids := unpacked[0].([]struct {
			Id                  [16]uint8        "json:\"id\""
			DecryptionCondition uint64           "json:\"decryptionCondition\""
			AllowedPeekers      []common.Address "json:\"allowedPeekers\""
			Version             string           `json:"version"`
		})

		require.Equal(t, bid1, suave.Bid{
			Id:                  bids[0].Id,
			DecryptionCondition: bids[0].DecryptionCondition,
			AllowedPeekers:      bids[0].AllowedPeekers,
			Version:             bids[0].Version,
		})
		require.Equal(t, bid2, suave.Bid{
			Id:                  bids[1].Id,
			DecryptionCondition: bids[1].DecryptionCondition,
			AllowedPeekers:      bids[1].AllowedPeekers,
			Version:             bids[1].Version,
		})

		// Verify via transaction
		wrappedTxData := &types.LegacyTx{
			Nonce:    0,
			To:       &fetchBidsAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		}

		offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
			ExecutionNode: ethservice.AccountManager().Accounts()[0],
			Wrapped:       *types.NewTx(wrappedTxData),
		}), signer, testKey)
		require.NoError(t, err)

		offchainTxBytes, err := offchainTx.MarshalBinary()
		require.NoError(t, err)

		var offchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes)))

		block := progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
		require.Equal(t, 1, len(block.Transactions()))

		receipts := ethservice.BlockChain().GetReceiptsByHash(block.Hash())
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.OffchainExecutedTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 1, len(block.Transactions()))
		require.Equal(t, []byte(simResult), block.Transactions()[0].Data())
	}
}

func TestBundleBid(t *testing.T) {
	// t.Fatal("not implemented")
	node, ethservice := startSuethService(t, testSuaveGenesis, nil, suave.Config{})
	defer node.Close()

	rpc, err := node.Attach()
	require.NoError(t, err)

	{
		targetBlock := uint64(16103213)
		allowedPeekers := []common.Address{common.Address{0x41, 0x42, 0x43}, newBundleBidAddress}

		bundle := struct {
			Txs             types.Transactions `json:"txs"`
			RevertingHashes []common.Hash      `json:"revertingHashes"`
		}{
			Txs:             types.Transactions{types.NewTx(&types.LegacyTx{})},
			RevertingHashes: []common.Hash{},
		}
		bundleBytes, err := json.Marshal(bundle)
		require.NoError(t, err)

		calldata, err := bundleBidAbi.Pack("newBid", targetBlock, allowedPeekers)
		require.NoError(t, err)

		// Verify via transaction
		wrappedTxData := &types.LegacyTx{
			Nonce:    0,
			To:       &newBundleBidAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		}

		offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
			ExecutionNode: ethservice.AccountManager().Accounts()[0],
			Wrapped:       *types.NewTx(wrappedTxData),
		}), signer, testKey)
		require.NoError(t, err)

		offchainTxBytes, err := offchainTx.MarshalBinary()
		require.NoError(t, err)

		confidentialDataBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		var offchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes), hexutil.Encode(confidentialDataBytes)))

		block := progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
		require.Equal(t, 1, len(block.Transactions()))

		receipts := ethservice.BlockChain().GetReceiptsByHash(block.Hash())
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.OffchainExecutedTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 1, len(block.Transactions()))
		unpacked, err := bundleBidAbi.Methods["emitBid"].Inputs.Unpack(block.Transactions()[0].Data()[4:])
		require.NoError(t, err)
		bid := unpacked[0].(struct {
			Id                  [16]uint8        "json:\"id\""
			DecryptionCondition uint64           "json:\"decryptionCondition\""
			AllowedPeekers      []common.Address "json:\"allowedPeekers\""
		})
		require.Equal(t, targetBlock, bid.DecryptionCondition)
		require.Equal(t, allowedPeekers, bid.AllowedPeekers)

		require.NotNil(t, receipts[0].Logs[0])
		require.Equal(t, newBundleBidAddress, receipts[0].Logs[0].Address)

		unpacked, err = bundleBidAbi.Events["BidEvent"].Inputs.Unpack(receipts[0].Logs[0].Data)
		require.NoError(t, err)

		require.Equal(t, bid.Id, unpacked[0].([16]byte))
		require.Equal(t, bid.DecryptionCondition, unpacked[1].(uint64))
		require.Equal(t, bid.AllowedPeekers, unpacked[2].([]common.Address))

		_, err = ethservice.APIBackend.OffchainBackend().ConfiendialStoreBackend.Retrieve(bid.Id, common.Address{0x41, 0x42, 0x43}, "default:v0:ethBundleSimResults")
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

	ethNode, ethEthService := startEthService(t, testEthGenesis, nil)
	defer ethNode.Close()
	ethEthService.APIs()

	node, ethservice := startSuethService(t, testSuaveGenesis, nil, suave.Config{SuaveEthRemoteBackendEndpoint: "http://127.0.0.1:8596"})
	defer node.Close()
	ethservice.APIs()

	rpc, err := node.Attach()
	require.NoError(t, err)

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
	allowedPeekers := []common.Address{common.Address{0x41, 0x42, 0x43}, newBlockBidAddress, extractHintAddress, buildEthBlockAddress, mevShareAddress}
	calldata, err := bundleBidAbi.Pack("newBid", targetBlock+1, allowedPeekers)
	require.NoError(t, err)

	wrappedTxData := &types.LegacyTx{
		Nonce:    0,
		To:       &mevShareAddress,
		Value:    nil,
		Gas:      1000069,
		GasPrice: big.NewInt(10),
		Data:     calldata,
	}

	offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
		ExecutionNode: ethservice.AccountManager().Accounts()[0],
		Wrapped:       *types.NewTx(wrappedTxData),
	}), signer, testKey)
	require.NoError(t, err)

	offchainTxBytes, err := offchainTx.MarshalBinary()
	require.NoError(t, err)

	// TODO : reusing this function selector from bid contract to avoid creating another ABI
	confidentialDataBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
	require.NoError(t, err)

	var offchainTxHash common.Hash
	requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes), hexutil.Encode(confidentialDataBytes)))

	//   1a. confirm submission
	block := progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
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
	unpacked, err := matchBidAbi.Events["HintEvent"].Inputs.Unpack(r.Logs[1].Data)
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
	backRunCalldata, err := matchBidAbi.Pack("newMatch", targetBlock+1, allowedPeekers, shareBidId)
	require.NoError(t, err)

	wrappedMatchTxData := &types.LegacyTx{
		Nonce:    1,
		To:       &mevShareAddress,
		Value:    nil,
		Gas:      1000069,
		GasPrice: big.NewInt(10),
		Data:     backRunCalldata,
	}

	offchainMatchTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
		ExecutionNode: ethservice.AccountManager().Accounts()[0],
		Wrapped:       *types.NewTx(wrappedMatchTxData),
	}), signer, testKey)
	require.NoError(t, err)

	offchainMatchTxBytes, err := offchainMatchTx.MarshalBinary()
	require.NoError(t, err)

	// TODO : reusing this function selector from bid contract to avoid creating another ABI
	confidentialDataMatchBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(backRunBundleBytes)
	require.NoError(t, err)

	var offchainMatchTxHash common.Hash
	requireNoRpcError(t, rpc.Call(&offchainMatchTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainMatchTxBytes), hexutil.Encode(confidentialDataMatchBytes)))

	block = progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
	require.Equal(t, 1, len(block.Transactions()))
	// check txn in block went to mev share
	require.Equal(t, block.Transactions()[0].To(), &mevShareAddress)

	var r2 *types.Receipt
	rpc.Call(&r2, "eth_getTransactionReceipt", block.Transactions()[0].Hash())
	require.NotEmpty(t, r2)
	require.NotEmpty(t, r.Logs)

	t.Log("logs", r2.Logs)

	// ************ 3. Build Share Block ************

	ethHead := ethEthService.BlockChain().CurrentBlock()
	payloadArgsTuple := struct {
		Slot           uint64
		ProposerPubkey []byte
		Parent         common.Hash
		Timestamp      uint64
		FeeRecipient   common.Address
		GasLimit       uint64
		Random         common.Hash
		Withdrawals    []struct {
			Index     uint64
			Validator uint64
			Address   common.Address
			Amount    uint64
		}
	}{
		Timestamp:    ethHead.Time + uint64(12),
		FeeRecipient: common.Address{0x42},
	}

	calldata, err = buildEthBlockAbi.Pack("buildMevShare", payloadArgsTuple, targetBlock+1)
	require.NoError(t, err)

	wrappedTxDataBB := &types.LegacyTx{
		Nonce:    2,
		To:       &newBlockBidAddress,
		Value:    nil,
		Gas:      1000000,
		GasPrice: big.NewInt(10),
		Data:     calldata,
	}

	offchainTxBB, err := types.SignTx(types.NewTx(&types.OffchainTx{
		ExecutionNode: ethservice.AccountManager().Accounts()[0],
		Wrapped:       *types.NewTx(wrappedTxDataBB),
	}), signer, testKey)
	require.NoError(t, err)

	offchainTxBytesBB, err := offchainTxBB.MarshalBinary()
	require.NoError(t, err)

	var offchainTxHashBB common.Hash
	requireNoRpcError(t, rpc.Call(&offchainTxHashBB, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytesBB)))

	block = progressChain(t, ethservice, block.Header())
	require.Equal(t, 1, len(block.Transactions()))

	var r3 *types.Receipt
	requireNoRpcError(t, rpc.Call(&r3, "eth_getTransactionReceipt", block.Transactions()[0].Hash()))
	require.NotEmpty(t, r3.Logs)

	{ // Fetch the built block id and check that the payload contains mev share trasnactions!
		receipts := ethservice.BlockChain().GetReceiptsByHash(block.Hash())
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.OffchainExecutedTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 2, len(receipts[0].Logs))
		require.NotNil(t, receipts[0].Logs[1])
		unpacked, err := bundleBidAbi.Events["BidEvent"].Inputs.Unpack(receipts[0].Logs[1].Data)
		require.NoError(t, err)

		bidId := unpacked[0].([16]byte)
		payloadData, err := ethservice.APIBackend.OffchainBackend().ConfiendialStoreBackend.Retrieve(bidId, newBlockBidAddress, "default:v0:builderPayload")
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
	ethNode, ethEthService := startEthService(t, testEthGenesis, nil)
	defer ethNode.Close()
	ethEthService.APIs()

	node, ethservice := startSuethService(t, testSuaveGenesis, nil, suave.Config{SuaveEthRemoteBackendEndpoint: "http://127.0.0.1:8596"})
	defer node.Close()
	ethservice.APIs()

	rpc, err := node.Attach()
	require.NoError(t, err)

	gas := hexutil.Uint64(1000000)
	chainId := hexutil.Big(*testSuaveGenesis.Config.ChainID)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()

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
		var simResult hexutil.Bytes
		requireNoRpcError(t, rpc.CallContext(ctx, &simResult, "eth_call", ethapi.TransactionArgs{
			To:         &simulateBundleAddress,
			Gas:        &gas,
			IsOffchain: true,
			ChainID:    &chainId,
			Data:       (*hexutil.Bytes)(&bundleBytes),
		}, "latest"))

		require.Equal(t, 32, len(simResult))
		require.Equal(t, 13, int(simResult[31]))
	}

	{ // Test the block building precompile through eth_call
		// function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bid) internal view returns (bytes memory, bytes memory) {

		bid := suave.Bid{
			Id:                  suave.BidId(uuid.New()),
			DecryptionCondition: uint64(1),
			AllowedPeekers:      []common.Address{common.Address{0x41, 0x42, 0x43}, buildEthBlockAddress},
			Version:             "default:v0",
		}

		ethservice.APIBackend.OffchainBackend().MempoolBackned.SubmitBid(bid)
		ethservice.APIBackend.OffchainBackend().ConfiendialStoreBackend.Initialize(bid, "default:v0:ethBundles", bundleBytes)

		ethHead := ethEthService.BlockChain().CurrentBlock()
		payloadArgsTuple := struct {
			Slot           uint64
			ProposerPubkey []byte
			Parent         common.Hash
			Timestamp      uint64
			FeeRecipient   common.Address
			GasLimit       uint64
			Random         common.Hash
			Withdrawals    []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			Timestamp:    ethHead.Time + uint64(12),
			FeeRecipient: common.Address{0x42},
		}

		packed, err := suaveLibAbi.Methods["buildEthBlock"].Inputs.Pack(payloadArgsTuple, bid.Id, "default:v0")
		require.NoError(t, err)

		var simResult hexutil.Bytes
		requireNoRpcError(t, rpc.CallContext(ctx, &simResult, "eth_call", ethapi.TransactionArgs{
			To:         &buildEthBlockAddress,
			Gas:        &gas,
			IsOffchain: true,
			ChainID:    &chainId,
			Data:       (*hexutil.Bytes)(&packed),
		}, "latest"))

		require.NotNil(t, simResult)

		unpacked, err := suaveLibAbi.Methods["buildEthBlock"].Outputs.Unpack(simResult)
		require.NoError(t, err)

		// TODO: test builder bid
		var envelope *engine.ExecutionPayloadEnvelope
		require.NoError(t, json.Unmarshal(unpacked[1].([]byte), &envelope))
		require.Equal(t, 2, len(envelope.ExecutionPayload.Transactions))

		var tx types.Transaction
		require.NoError(t, tx.UnmarshalBinary(envelope.ExecutionPayload.Transactions[0]))

		require.Equal(t, ethTx.Data(), tx.Data())
		require.Equal(t, ethTx.Hash(), tx.Hash())
	}
}

func TestBlockBuildingContract(t *testing.T) {
	ethNode, ethEthService := startEthService(t, testEthGenesis, nil)
	defer ethNode.Close()
	ethEthService.APIs()

	node, ethservice := startSuethService(t, testSuaveGenesis, nil, suave.Config{SuaveEthRemoteBackendEndpoint: "http://127.0.0.1:8596"})
	defer node.Close()
	ethservice.APIs()

	rpc, err := node.Attach()
	require.NoError(t, err)

	// ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	// defer cancel()

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
		Version         string             `json:"version"`
	}{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
		Version:         "default:v0",
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	targetBlock := uint64(1)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{newBlockBidAddress, newBundleBidAddress, buildEthBlockAddress}
		calldata, err := bundleBidAbi.Pack("newBid", targetBlock+1, allowedPeekers)
		require.NoError(t, err)

		wrappedTxData := &types.LegacyTx{
			Nonce:    0,
			To:       &newBundleBidAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		}

		offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
			ExecutionNode: ethservice.AccountManager().Accounts()[0],
			Wrapped:       *types.NewTx(wrappedTxData),
		}), signer, testKey)
		require.NoError(t, err)

		offchainTxBytes, err := offchainTx.MarshalBinary()
		require.NoError(t, err)

		confidentialDataBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		var offchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes), hexutil.Encode(confidentialDataBytes)))
	}

	block := progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
	require.Equal(t, 1, len(block.Transactions()))

	{
		ethHead := ethEthService.BlockChain().CurrentBlock()
		payloadArgsTuple := struct {
			Slot           uint64
			ProposerPubkey []byte
			Parent         common.Hash
			Timestamp      uint64
			FeeRecipient   common.Address
			GasLimit       uint64
			Random         common.Hash
			Withdrawals    []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			ProposerPubkey: []byte{0x42},
			Timestamp:      ethHead.Time + uint64(12),
			FeeRecipient:   common.Address{0x42},
		}

		calldata, err := buildEthBlockAbi.Pack("buildFromPool", payloadArgsTuple, targetBlock+1)
		require.NoError(t, err)

		wrappedTxData := &types.LegacyTx{
			Nonce:    1,
			To:       &newBlockBidAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		}

		offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
			ExecutionNode: ethservice.AccountManager().Accounts()[0],
			Wrapped:       *types.NewTx(wrappedTxData),
		}), signer, testKey)
		require.NoError(t, err)

		offchainTxBytes, err := offchainTx.MarshalBinary()
		require.NoError(t, err)

		var offchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes)))

		block = progressChain(t, ethservice, block.Header())
		require.Equal(t, 1, len(block.Transactions()))
	}
}

func TestRelayBlockSubmissionContract(t *testing.T) {
	ethNode, ethEthService := startEthService(t, testEthGenesis, nil)
	defer ethNode.Close()
	ethEthService.APIs()

	node, ethservice := startSuethService(t, testSuaveGenesis, nil, suave.Config{SuaveEthRemoteBackendEndpoint: "http://127.0.0.1:8596"})
	defer node.Close()
	ethservice.APIs()

	rpc, err := node.Attach()
	require.NoError(t, err)
	var block *types.Block

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
		abiEncodedRelayUrl, err := ethBlockBidSenderAbi.Pack("", fakeRelayServer.URL)
		require.NoError(t, err)

		calldata := append(hexutil.MustDecode(blockBidSenderContractCode), abiEncodedRelayUrl...)
		ccTxData := &types.LegacyTx{
			Nonce:    0,
			To:       nil, // contract creation
			Value:    big.NewInt(0),
			Gas:      10000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		}

		tx, err := types.SignTx(types.NewTx(ccTxData), signer, testKey)
		require.NoError(t, err)

		from, _ := types.Sender(signer, tx)
		ethBlockBidSenderAddr = crypto.CreateAddress(from, tx.Nonce())

		txBytes, err := tx.MarshalBinary()
		require.NoError(t, err)

		var txHash common.Hash
		requireNoRpcError(t, rpc.Call(&txHash, "eth_sendRawTransaction", hexutil.Encode(txBytes)))

		block = progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
		require.Equal(t, 1, len(block.Transactions()))
		receipts := ethservice.BlockChain().GetReceiptsByHash(block.Hash())
		require.Equal(t, uint64(1), receipts[0].Status)
	}

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

	targetBlock := uint64(1)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{ethBlockBidSenderAddr, newBundleBidAddress, buildEthBlockAddress}
		calldata, err := bundleBidAbi.Pack("newBid", targetBlock+1, allowedPeekers)
		require.NoError(t, err)

		wrappedTxData := &types.LegacyTx{
			Nonce:    1,
			To:       &newBundleBidAddress,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		}

		offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
			ExecutionNode: ethservice.AccountManager().Accounts()[0],
			Wrapped:       *types.NewTx(wrappedTxData),
		}), signer, testKey)
		require.NoError(t, err)

		offchainTxBytes, err := offchainTx.MarshalBinary()
		require.NoError(t, err)

		confidentialDataBytes, err := bundleBidAbi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		var offchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes), hexutil.Encode(confidentialDataBytes)))
	}

	block = progressChain(t, ethservice, ethservice.BlockChain().CurrentBlock())
	require.Equal(t, 1, len(block.Transactions()))

	{
		ethHead := ethEthService.BlockChain().CurrentBlock()
		payloadArgsTuple := struct {
			Slot           uint64
			ProposerPubkey []byte
			Parent         common.Hash
			Timestamp      uint64
			FeeRecipient   common.Address
			GasLimit       uint64
			Random         common.Hash
			Withdrawals    []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			ProposerPubkey: []byte{0x42},
			Timestamp:      ethHead.Time + uint64(12),
			FeeRecipient:   common.Address{0x42},
		}

		calldata, err := ethBlockBidSenderAbi.Pack("buildFromPool", payloadArgsTuple, targetBlock+1)
		require.NoError(t, err)

		wrappedTxData := &types.LegacyTx{
			Nonce:    2,
			To:       &ethBlockBidSenderAddr,
			Value:    nil,
			Gas:      1000000,
			GasPrice: big.NewInt(10),
			Data:     calldata,
		}

		offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
			ExecutionNode: ethservice.AccountManager().Accounts()[0],
			Wrapped:       *types.NewTx(wrappedTxData),
		}), signer, testKey)
		require.NoError(t, err)

		offchainTxBytes, err := offchainTx.MarshalBinary()
		require.NoError(t, err)

		var offchainTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes)))

		block = progressChain(t, ethservice, block.Header())
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
	ok, err := boostSsz.VerifySignature(blockPayloadSentToRelay.Message, builderSigningDomain, builderPubkey[:], signature[:])
	require.NoError(t, err)
	require.True(t, ok)
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

	isOffchainAddress = common.HexToAddress("0x42010000")

	// confidentialStoreAddress = common.HexToAddress("0x42020000")
	// confStoreRetrieveAddress = common.HexToAddress("0x42020001")

	fetchBidsAddress    = common.HexToAddress("0x42030001")
	newBundleBidAddress = common.HexToAddress("0x42300000")
	newBlockBidAddress  = common.HexToAddress("0x42300001")

	simulateBundleAddress = common.HexToAddress("0x42100000")
	extractHintAddress    = common.HexToAddress("0x42100037")

	buildEthBlockAddress  = common.HexToAddress("0x42100001")
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
			newBundleBidAddress:   {Balance: big.NewInt(0), Code: hexutil.MustDecode(bidsContractCode)},
			newBlockBidAddress:    {Balance: big.NewInt(0), Code: hexutil.MustDecode(blockBidContractCode)},
			blockBidSenderAddress: {Balance: big.NewInt(0), Code: hexutil.MustDecode(blockBidSenderContractCode)},
			mevShareAddress:       {Balance: big.NewInt(0), Code: hexutil.MustDecode(mevShareBidContractCode)},
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

	signer = types.NewOffchainSigner(params.AllEthashProtocolChanges.ChainID)
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
		HTTPPort: 8596,
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		}})
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

		unpacked, err := suaveLibAbi.Errors["PeekerReverted"].Inputs.Unpack(decodedError[4:])
		if err != nil {
			require.NoError(t, err, rpcErr.Error())
		} else {
			require.NoError(t, rpcErr, string(unpacked[1].([]byte)))
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

func mustParseAbi(data string) abi.ABI {
	inoutAbi, err := abi.JSON(strings.NewReader(data))
	if err != nil {
		panic(err.Error())
	}

	return inoutAbi
}
