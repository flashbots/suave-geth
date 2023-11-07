package e2e

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
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
	"github.com/ethereum/go-ethereum/accounts"
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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/ethereum/go-ethereum/suave/cstore"
	"github.com/ethereum/go-ethereum/suave/sdk"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/ssz"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

func TestIsConfidential(t *testing.T) {
	// t.Fatal("not implemented")

	fr := newFramework(t)
	defer fr.Close()

	rpc := fr.suethSrv.RPCNode()

	chainId := hexutil.Big(*testSuaveGenesis.Config.ChainID)

	{
		// Verify eth_call of isConfidentialAddress returns 1/0 depending on confidential compute setting
		var result string
		requireNoRpcError(t, rpc.Call(&result, "eth_call", setTxArgsDefaults(ethapi.TransactionArgs{
			To:             &isConfidentialAddress,
			IsConfidential: true,
			ChainID:        &chainId,
		}), "latest"))
		require.Equal(t, []byte{1}, hexutil.MustDecode(result))
	}

	{
		// Verify sending computation requests and onchain transactions to isConfidentialAddress
		confidentialRequestTx, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
			ConfidentialComputeRecord: types.ConfidentialComputeRecord{
				KettleAddress: fr.KettleAddress(),
				Nonce:         0,
				To:            &isConfidentialAddress,
				Value:         nil,
				Gas:           1000000,
				GasPrice:      big.NewInt(10),
				Data:          []byte{},
			},
		}), signer, testKey)
		require.NoError(t, err)

		confidentialRequestTxBytes, err := confidentialRequestTx.MarshalBinary()
		require.NoError(t, err)

		var confidentialRequestTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&confidentialRequestTxHash, "eth_sendRawTransaction", hexutil.Encode(confidentialRequestTxBytes)))

		onchainTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
			Nonce:    1,
			To:       &isConfidentialAddress,
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

		block := fr.suethSrv.ProgressChain()
		require.Equal(t, 2, len(block.Transactions()))

		receipts := block.Receipts
		require.Equal(t, 2, len(receipts))
		require.Equal(t, uint8(types.SuaveTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)
		require.Equal(t, uint8(types.LegacyTxType), receipts[1].Type)
		require.Equal(t, uint64(1), receipts[1].Status)

		require.Equal(t, 2, len(block.Transactions()))
		require.Equal(t, []byte{1}, block.Transactions()[0].Data())
		require.Equal(t, []byte{}, block.Transactions()[1].Data())
	}
}

func TestMempool(t *testing.T) {
	// t.Fatal("not implemented")
	fr := newFramework(t)
	defer fr.Close()

	rpc := fr.suethSrv.RPCNode()

	gas := hexutil.Uint64(1000000)
	chainId := hexutil.Big(*testSuaveGenesis.Config.ChainID)

	{
		targetBlock := uint64(16103213)
		creationTx := types.NewTx(&types.ConfidentialComputeRequest{
			ConfidentialComputeRecord: types.ConfidentialComputeRecord{
				KettleAddress: fr.KettleAddress(),
			},
		})

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

		inoutAbi := mustParseMethodAbi(`[ { "inputs": [ { "internalType": "uint64", "name": "cond", "type": "uint64" }, { "internalType": "string", "name": "namespace", "type": "string" } ], "name": "fetchBids", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "Suave.BidId", "name": "salt", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "address[]", "name": "allowedStores", "type": "address[]" }, { "internalType": "string", "name": "version", "type": "string" } ], "internalType": "struct Suave.Bid[]", "name": "", "type": "tuple[]" } ], "stateMutability": "view", "type": "function" } ]`, "fetchBids")

		calldata, err := inoutAbi.Inputs.Pack(targetBlock, "default:v0:ethBundles")
		require.NoError(t, err)

		var simResult hexutil.Bytes
		requireNoRpcError(t, rpc.Call(&simResult, "eth_call", setTxArgsDefaults(ethapi.TransactionArgs{
			To:             &fetchBidsAddress,
			Gas:            &gas,
			IsConfidential: true,
			ChainID:        &chainId,
			Data:           (*hexutil.Bytes)(&calldata),
		}), "latest"))

		unpacked, err := inoutAbi.Outputs.Unpack(simResult)
		require.NoError(t, err)

		var bids []suave.Bid
		require.NoError(t, mapstructure.Decode(unpacked[0], &bids))

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
		confidentialRequestTx, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
			ConfidentialComputeRecord: types.ConfidentialComputeRecord{
				KettleAddress: fr.KettleAddress(),
				Nonce:         0,
				To:            &fetchBidsAddress,
				Value:         nil,
				Gas:           1000000,
				GasPrice:      big.NewInt(10),
				Data:          calldata,
			},
		}), signer, testKey)
		require.NoError(t, err)

		confidentialRequestTxBytes, err := confidentialRequestTx.MarshalBinary()
		require.NoError(t, err)

		var confidentialRequestTxHash common.Hash
		requireNoRpcError(t, rpc.Call(&confidentialRequestTxHash, "eth_sendRawTransaction", hexutil.Encode(confidentialRequestTxBytes)))

		block := fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))

		receipts := block.Receipts
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.SuaveTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 1, len(block.Transactions()))
		require.Equal(t, []byte(simResult), block.Transactions()[0].Data())
	}
}

func TestTxSigningPrecompile(t *testing.T) {
	fr := newFramework(t)
	defer fr.Close()

	tx := types.NewTransaction(15, common.Address{0x14}, big.NewInt(50), 1000000, big.NewInt(42313), []byte{0x42})
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err)

	sk, err := crypto.GenerateKey()
	require.NoError(t, err)
	skHex := hex.EncodeToString(crypto.FromECDSA(sk))

	txChainId := big.NewInt(13)
	chainIdHex := hexutil.EncodeBig(txChainId)

	// function signEthTransaction(bytes memory txn, string memory chainId, string memory signingKey)
	args, err := artifacts.SuaveAbi.Methods["signEthTransaction"].Inputs.Pack(txBytes, chainIdHex, skHex)
	require.NoError(t, err)

	gas := hexutil.Uint64(1000000)
	chainId := hexutil.Big(*testSuaveGenesis.Config.ChainID)

	var callResult hexutil.Bytes
	err = fr.suethSrv.RPCNode().Call(&callResult, "eth_call", setTxArgsDefaults(ethapi.TransactionArgs{
		To:             &signEthTransaction,
		Gas:            &gas,
		IsConfidential: true,
		ChainID:        &chainId,
		Data:           (*hexutil.Bytes)(&args),
	}), "latest")
	requireNoRpcError(t, err)

	unpackedCallResult, err := artifacts.SuaveAbi.Methods["signEthTransaction"].Outputs.Unpack(callResult)
	require.NoError(t, err)

	var signedTx types.Transaction
	require.NoError(t, signedTx.UnmarshalBinary(unpackedCallResult[0].([]byte)))

	require.Equal(t, tx.Nonce(), signedTx.Nonce())
	require.Equal(t, *tx.To(), *signedTx.To())
	require.Equal(t, 0, tx.Value().Cmp(signedTx.Value()))
	require.Equal(t, tx.Gas(), signedTx.Gas())
	require.Equal(t, tx.GasPrice(), signedTx.GasPrice())
	require.Equal(t, tx.Data(), signedTx.Data())

	sender, err := types.Sender(types.LatestSignerForChainID(txChainId), &signedTx)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(sk.PublicKey), sender)
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

		bundle := &types.SBundle{
			Txs: types.Transactions{types.NewTx(&types.LegacyTx{})},
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

func TestBundleSenderContract(t *testing.T) {
	skOpt, bundleSigningKeyPub := WithBundleSigningKeyOpt(t)
	fr := newFramework(t, skOpt)
	defer fr.Close()

	clt := fr.NewSDKClient()

	bundleSentToBuilder := &struct {
		Id     json.RawMessage
		Params []types.RpcSBundle
	}{}
	serveHttp := func(t *testing.T, w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		err = json.Unmarshal(bodyBytes, bundleSentToBuilder)
		if err != nil {
			require.NoError(t, err, string(bodyBytes))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		splitSig := strings.Split(r.Header.Get("x-flashbots-signature"), ":")
		require.Equal(t, 2, len(splitSig))
		require.Equal(t, splitSig[0], crypto.PubkeyToAddress(*bundleSigningKeyPub).Hex())

		signature := hexutil.MustDecode(splitSig[1])
		hashedBody := accounts.TextHash([]byte(crypto.Keccak256Hash(bodyBytes).Hex()))
		pk, err := crypto.SigToPub(hashedBody, signature)
		require.NoError(t, err)
		require.True(t, pk.Equal(bundleSigningKeyPub))
		require.True(t, crypto.VerifySignature(crypto.CompressPubkey(bundleSigningKeyPub), hashedBody, signature[:64]))

		w.WriteHeader(http.StatusOK)
	}

	fakeRelayServer := httptest.NewServer(&fakeRelayHandler{t, serveHttp})
	defer fakeRelayServer.Close()

	{
		targetBlock := uint64(16103213)

		signedTx, err := types.SignNewTx(testKey, signer, &types.LegacyTx{
			Nonce:    15,
			GasPrice: big.NewInt(1000),
			Gas:      100000,
		})
		require.NoError(t, err)

		bundle := &types.SBundle{
			BlockNumber: big.NewInt(int64(targetBlock)),
			Txs:         types.Transactions{signedTx},
		}
		bundleBytes, err := json.Marshal(bundle)
		require.NoError(t, err)

		confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
		require.NoError(t, err)

		constructorArgs, err := EthBundleSenderContract.Abi.Constructor.Inputs.Pack([]string{fakeRelayServer.URL})
		require.NoError(t, err)

		deployCode := EthBundleSenderContract.Code
		deployCode = append(deployCode, constructorArgs...)
		txRes, err := sdk.DeployContract(deployCode, clt)
		require.NoError(t, err)

		fr.suethSrv.ProgressChain()

		receipt, err := txRes.Wait()
		require.NoError(t, err)
		bundleSenderContract := sdk.GetContract(receipt.ContractAddress, EthBundleSenderContract.Abi, clt)

		allowedPeekers := []common.Address{bundleSenderContract.Address()}

		_, err = bundleSenderContract.SendTransaction("newBid", []interface{}{targetBlock, allowedPeekers, []common.Address{}}, confidentialDataBytes)
		requireNoRpcError(t, err)

		block := fr.suethSrv.ProgressChain()
		require.Equal(t, 1, len(block.Transactions()))

		receipts := block.Receipts
		require.Equal(t, 1, len(receipts))
		require.Equal(t, uint8(types.SuaveTxType), receipts[0].Type)
		require.Equal(t, uint64(1), receipts[0].Status)

		require.Equal(t, 1, len(bundleSentToBuilder.Params))
		require.Equal(t, 1, len(bundleSentToBuilder.Params[0].Txs))

		var recoveredTx types.Transaction
		require.NoError(t, recoveredTx.UnmarshalBinary(bundleSentToBuilder.Params[0].Txs[0]))
		expectedTxJson, err := signedTx.MarshalJSON()
		require.NoError(t, err)
		recoveredTxJson, err := recoveredTx.MarshalJSON()
		require.NoError(t, err)
		require.Equal(t, expectedTxJson, recoveredTxJson)

		require.Equal(t, bundleSentToBuilder.Params[0].BlockNumber.ToInt().Uint64(), targetBlock)
	}
}

func prepareMevShareBundle(t *testing.T) (*types.Transaction, types.SBundle, []byte) {
	ethTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21000,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	}), signer, testKey)
	require.NoError(t, err)

	refundPercent := 10
	bundle := &types.SBundle{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
		RefundPercent:   &refundPercent,
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	confidentialDataBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)
	require.NoError(t, err)

	return ethTx, *bundle, confidentialDataBytes
}

func prepareMevShareBackrun(t *testing.T, shareBidId types.BidId) (*types.Transaction, types.SBundle, []byte) {
	backrunTx, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    0,
		To:       &testAddr,
		Value:    big.NewInt(1000),
		Gas:      21420,
		GasPrice: big.NewInt(13),
		Data:     []byte{},
	}), signer, testKey2)
	require.NoError(t, err)

	backRunBundle := &types.SBundle{
		Txs:             types.Transactions{backrunTx},
		RevertingHashes: []common.Hash{},
	}
	backRunBundleBytes, err := json.Marshal(backRunBundle)
	require.NoError(t, err)

	confidentialDataMatchBytes, err := BundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(backRunBundleBytes)
	require.NoError(t, err)

	return backrunTx, *backRunBundle, confidentialDataMatchBytes
}

func TestMevShare(t *testing.T) {
	// 1. craft mevshare transaction
	//   1a. confirm submission
	// 2. send backrun txn
	//	 2a. confirm submission
	// 3. build share block
	//   3a. confirm share bundle

	fr := newFramework(t, WithKettleAddress())
	defer fr.Close()

	rpc := fr.suethSrv.RPCNode()
	clt := fr.NewSDKClient()

	// ************ 1. Initial mevshare transaction Portion ************

	ethTx, _, confidentialDataBytes := prepareMevShareBundle(t)
	targetBlock := uint64(1)

	// Send a bundle bid
	allowedPeekers := []common.Address{{0x41, 0x42, 0x43}, newBlockBidAddress, buildEthBlockAddress, mevShareAddress}

	bundleBidContractI := sdk.GetContract(mevShareAddress, BundleBidContract.Abi, clt)
	_, err := bundleBidContractI.SendTransaction("newBid", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{fr.KettleAddress()}}, confidentialDataBytes)
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

	backrunTx, _, confidentialDataMatchBytes := prepareMevShareBackrun(t, shareBidId)

	cc := sdk.GetContract(mevShareAddress, MevShareBidContract.Abi, clt)
	_, err = cc.SendTransaction("newMatch", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{fr.KettleAddress()}, shareBidId}, confidentialDataMatchBytes)
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

func TestMevShareBundleSenderContract(t *testing.T) {
	fr := newFramework(t)
	defer fr.Close()

	clt := fr.NewSDKClient()

	bundleSentToBuilder := &struct {
		Params []types.RPCMevShareBundle
	}{}
	serveHttp := func(t *testing.T, w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		err = json.Unmarshal(bodyBytes, bundleSentToBuilder)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}

	fakeRelayServer := httptest.NewServer(&fakeRelayHandler{t, serveHttp})
	defer fakeRelayServer.Close()

	constructorArgs, err := MevShareBundleSenderContract.Abi.Constructor.Inputs.Pack([]string{fakeRelayServer.URL})
	require.NoError(t, err)

	deployCode := MevShareBundleSenderContract.Code
	deployCode = append(deployCode, constructorArgs...)
	txRes, err := sdk.DeployContract(deployCode, clt)
	require.NoError(t, err)

	fr.suethSrv.ProgressChain()

	receipt, err := txRes.Wait()
	require.NoError(t, err)
	bundleSenderContract := sdk.GetContract(receipt.ContractAddress, MevShareBundleSenderContract.Abi, clt)

	{
		// ************ 1. Initial mevshare transaction Portion ************

		userTx, _, confidentialDataBytes := prepareMevShareBundle(t)
		targetBlock := uint64(1)

		// Send a bundle bid
		allowedPeekers := []common.Address{fillMevShareBundleAddress, bundleSenderContract.Address()}

		txRes, err := bundleSenderContract.SendTransaction("newBid", []interface{}{targetBlock, allowedPeekers, []common.Address{}}, confidentialDataBytes)
		requireNoRpcError(t, err)

		fr.suethSrv.ProgressChain()

		receipt, err := txRes.Wait()
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status)

		require.NotEmpty(t, receipt.Logs)

		// extract share BidId
		unpacked, err := MevShareBidContract.Abi.Events["HintEvent"].Inputs.Unpack(receipt.Logs[1].Data)
		require.NoError(t, err)
		shareBidId := unpacked[0].([16]byte)

		// ************ 2. Match Portion ************

		matchTx, _, confidentialDataMatchBytes := prepareMevShareBackrun(t, shareBidId)

		txRes, err = bundleSenderContract.SendTransaction("newMatch", []interface{}{targetBlock, allowedPeekers, []common.Address{fr.KettleAddress()}, shareBidId}, confidentialDataMatchBytes)
		requireNoRpcError(t, err)

		fr.suethSrv.ProgressChain()

		receipt, err = txRes.Wait()
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status)

		retrievedBlockNumber, err := hexutil.DecodeUint64(bundleSentToBuilder.Params[0].Inclusion.Block)
		require.NoError(t, err)
		require.Equal(t, targetBlock, retrievedBlockNumber)

		encodedUserTxBytes, err := userTx.MarshalBinary()
		require.NoError(t, err)

		encodedmatchTxBytes, err := matchTx.MarshalBinary()
		require.NoError(t, err)

		retrievedTxs := []string{}
		for _, be := range bundleSentToBuilder.Params[0].Body {
			retrievedTxs = append(retrievedTxs, be.Tx)
		}

		expectedTxs := []string{hexutil.Encode(encodedUserTxBytes), hexutil.Encode(encodedmatchTxBytes)}
		require.Equal(t, expectedTxs, retrievedTxs)
	}
}

func TestBlockBuildingPrecompiles(t *testing.T) {
	fr := newFramework(t, WithKettleAddress())
	defer fr.Close()

	rpc := fr.suethSrv.RPCNode()

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

	bundle := &types.SBundle{
		Txs: types.Transactions{ethTx},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	{ // Test the bundle simulation precompile through eth_call
		calldata, err := suaveLibContract.Abi.Methods["simulateBundle"].Inputs.Pack(bundleBytes)
		require.NoError(t, err)

		var simResult hexutil.Bytes
		requireNoRpcError(t, rpc.CallContext(ctx, &simResult, "eth_call", setTxArgsDefaults(ethapi.TransactionArgs{
			To:             &simulateBundleAddress,
			Gas:            &gas,
			IsConfidential: true,
			ChainID:        &chainId,
			Data:           (*hexutil.Bytes)(&calldata),
		}), "latest"))

		require.Equal(t, 32, len(simResult))
		require.Equal(t, 13, int(simResult[31]))
	}

	{ // Test the block building precompile through eth_call
		// function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bid) internal view returns (bytes memory, bytes memory) {

		dummyCreationTx, err := types.SignNewTx(testKey, signer, &types.ConfidentialComputeRequest{
			ConfidentialComputeRecord: types.ConfidentialComputeRecord{
				KettleAddress: fr.KettleAddress(),
			},
		})
		require.NoError(t, err)

		bid, err := fr.ConfidentialEngine().InitializeBid(types.Bid{
			DecryptionCondition: uint64(1),
			AllowedPeekers:      []common.Address{suave.AllowedPeekerAny},
			AllowedStores:       []common.Address{fr.KettleAddress()},
			Version:             "default:v0:ethBundles",
		}, dummyCreationTx)
		require.NoError(t, err)

		err = fr.ConfidentialEngine().Finalize(dummyCreationTx, map[suave.BidId]suave.Bid{bid.Id: bid}, []cstore.StoreWrite{{

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

		packed, err := suaveLibContract.Abi.Methods["buildEthBlock"].Inputs.Pack(payloadArgsTuple, bid.Id, "")
		require.NoError(t, err)

		var simResult hexutil.Bytes
		requireNoRpcError(t, rpc.CallContext(ctx, &simResult, "eth_call", setTxArgsDefaults(ethapi.TransactionArgs{
			To:             &buildEthBlockAddress,
			Gas:            &gas,
			IsConfidential: true,
			ChainID:        &chainId,
			Data:           (*hexutil.Bytes)(&packed),
		}), "latest"))

		require.NotNil(t, simResult)

		unpacked, err := suaveLibContract.Abi.Methods["buildEthBlock"].Outputs.Unpack(simResult)
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
	fr := newFramework(t, WithKettleAddress())
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

	bundle := &types.SBundle{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	targetBlock := uint64(1)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{newBlockBidAddress, newBundleBidAddress, buildEthBlockAddress}

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
	skOpt, signingPubkey := WithBlockSigningKeyOpt(t)
	fr := newFramework(t, WithKettleAddress(), skOpt)
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

		genesisForkVersion := phase0.Version{0x00, 0x00, 0x10, 0x20}
		builderSigningDomain := ssz.ComputeDomain(ssz.DomainTypeAppBuilder, genesisForkVersion, phase0.Root{})
		ok, err := ssz.VerifySignature(blockPayloadSentToRelay.Message, builderSigningDomain, bls.PublicKeyToBytes(signingPubkey), blockPayloadSentToRelay.Signature[:])
		require.NoError(t, err)
		require.True(t, ok)

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

	bundle := &types.SBundle{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	targetBlock := uint64(1)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{ethBlockBidSenderAddr, newBundleBidAddress, buildEthBlockAddress}

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

func TestE2E_ForgeIntegration(t *testing.T) {
	// This end-to-end test ensures that the precompile lifecycle expected in Forge works
	fr := newFramework(t, WithKettleAddress())
	defer fr.Close()

	rpcClient := fr.suethSrv.RPCNode()
	ethClient := ethclient.NewClient(rpcClient)

	chainIdRaw, err := ethClient.ChainID(context.Background())
	require.NoError(t, err)

	doCall := func(methodName string, args ...interface{}) []interface{} {
		toAddr, ok := artifacts.SuaveMethods[methodName]
		require.True(t, ok, fmt.Sprintf("suave method %s not found", methodName))

		method := artifacts.SuaveAbi.Methods[methodName]

		input, err := method.Inputs.Pack(args...)
		require.NoError(t, err)

		chainId := hexutil.Big(*chainIdRaw)

		callArgs := ethapi.TransactionArgs{
			To:             &toAddr,
			IsConfidential: true,
			ChainID:        &chainId,
			Data:           (*hexutil.Bytes)(&input),
		}
		var simResult hexutil.Bytes
		err = rpcClient.Call(&simResult, "eth_call", setTxArgsDefaults(callArgs), "latest")
		require.NoError(t, err)

		if methodName == "confidentialRetrieve" {
			// this method does not abi pack the output
			return []interface{}{[]byte(simResult)}
		}

		result, err := method.Outputs.Unpack(simResult)
		require.NoError(t, err)
		return result
	}

	addrList := []common.Address{suave.AllowedPeekerAny}
	bidRaw := doCall("newBid", uint64(0), addrList, addrList, "default:v0:ethBundles")

	var bid types.Bid
	require.NoError(t, mapstructure.Decode(bidRaw[0], &bid))

	bidsRaw := doCall("fetchBids", uint64(0), "default:v0:ethBundles")
	var bids []types.Bid
	require.NoError(t, mapstructure.Decode(bidsRaw[0], &bids))
	require.Len(t, bids, 1)
	require.Equal(t, bids[0].Id, bid.Id)

	val := []byte{0x1, 0x2, 0x3}
	doCall("confidentialStore", bid.Id, "a", val)

	valRaw := doCall("confidentialRetrieve", bid.Id, "a")
	require.Equal(t, val, valRaw[0])
}

func TestE2EPrecompile_Call(t *testing.T) {
	// This end-to-end tests that the callx precompile gets called from a confidential request
	fr := newFramework(t, WithKettleAddress())
	defer fr.Close()

	clt := fr.NewSDKClient()

	// We reuse the same address for both the source and target contract
	contractAddr := common.Address{0x3}
	sourceContract := sdk.GetContract(contractAddr, exampleCallSourceContract.Abi, clt)

	expectedNum := big.NewInt(101)
	_, err := sourceContract.SendTransaction("callTarget", []interface{}{contractAddr, expectedNum}, nil)
	require.NoError(t, err)

	incorrectNum := big.NewInt(102)
	_, err = sourceContract.SendTransaction("callTarget", []interface{}{contractAddr, incorrectNum}, nil)
	require.Error(t, err)
}

func TestE2EKettleAddressEndpoint(t *testing.T) {
	// this end-to-end tests ensures that we can call eth_kettleAddress endpoint in a MEVM node
	// and return the correct execution address list
	fr := newFramework(t, WithKettleAddress())
	defer fr.Close()

	var addrs []common.Address
	require.NoError(t, fr.suethSrv.RPCNode().Call(&addrs, "eth_kettleAddress"))
	require.NotEmpty(t, addrs)
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
	kettleAddress     bool
	redisStoreBackend bool
	suaveConfig       suave.Config
}

var defaultFrameworkConfig = frameworkConfig{
	kettleAddress:     false,
	redisStoreBackend: false,
	suaveConfig:       suave.Config{},
}

type frameworkOpt func(*frameworkConfig)

func WithKettleAddress() frameworkOpt {
	return func(c *frameworkConfig) {
		c.kettleAddress = true
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

func WithBundleSigningKeyOpt(t *testing.T) (frameworkOpt, *ecdsa.PublicKey) {
	sk, err := crypto.GenerateKey()
	require.NoError(t, err)
	return func(c *frameworkConfig) {
		c.suaveConfig.EthBundleSigningKeyHex = hex.EncodeToString(crypto.FromECDSA(sk))
	}, &sk.PublicKey
}

func WithBlockSigningKeyOpt(t *testing.T) (frameworkOpt, *bls.PublicKey) {
	sk, pk, err := bls.GenerateNewKeypair()
	require.NoError(t, err)
	return func(c *frameworkConfig) {
		c.suaveConfig.EthBlockSigningKeyHex = hexutil.Encode(bls.SecretKeyToBytes(sk))
	}, pk
}

func newFramework(t *testing.T, opts ...frameworkOpt) *framework {
	cfg := defaultFrameworkConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var ethSrv *clientWrapper

	if cfg.kettleAddress {
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
	return sdk.NewClient(f.suethSrv.RPCNode(), testKey, f.KettleAddress())
}

func (f *framework) ConfidentialStoreBackend() cstore.ConfidentialStorageBackend {
	return f.suethSrv.service.APIBackend.SuaveEngine().Backend()
}

func (f *framework) ConfidentialEngine() *cstore.ConfidentialStoreEngine {
	return f.suethSrv.service.APIBackend.SuaveEngine()
}

func (f *framework) KettleAddress() common.Address {
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

	testAddr3 = common.Address{0x3}

	testBalance = big.NewInt(2e18)

	/* precompiles */
	isConfidentialAddress     = common.HexToAddress("0x42010000")
	fetchBidsAddress          = common.HexToAddress("0x42030001")
	fillMevShareBundleAddress = common.HexToAddress("0x43200001")

	signEthTransaction    = common.HexToAddress("0x40100001")
	simulateBundleAddress = common.HexToAddress("0x42100000")
	buildEthBlockAddress  = common.HexToAddress("0x42100001")

	/* contracts */
	newBundleBidAddress = common.HexToAddress("0x642300000")
	newBlockBidAddress  = common.HexToAddress("0x642310000")
	mevShareAddress     = common.HexToAddress("0x642100073")

	testSuaveGenesis *core.Genesis = &core.Genesis{
		Timestamp:  1680000000,
		ExtraData:  nil,
		GasLimit:   30000000,
		BaseFee:    big.NewInt(0),
		Difficulty: big.NewInt(0),
		Alloc: core.GenesisAlloc{
			testAddr:            {Balance: testBalance},
			testAddr2:           {Balance: testBalance},
			newBundleBidAddress: {Balance: big.NewInt(0), Code: BundleBidContract.DeployedCode},
			newBlockBidAddress:  {Balance: big.NewInt(0), Code: buildEthBlockContract.DeployedCode},
			mevShareAddress:     {Balance: big.NewInt(0), Code: MevShareBidContract.DeployedCode},
			testAddr3:           {Balance: big.NewInt(0), Code: exampleCallSourceContract.DeployedCode},
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
			testAddr3: {Balance: big.NewInt(0), Code: exampleCallTargetContract.DeployedCode},
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

		unpacked, err := suaveLibContract.Abi.Errors["PeekerReverted"].Inputs.Unpack(decodedError[4:])
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
