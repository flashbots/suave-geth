package main

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

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
		}

		bid2 := suave.Bid{
			Id:                  suave.BidId(uuid.New()),
			DecryptionCondition: targetBlock,
			AllowedPeekers:      []common.Address{common.HexToAddress("0x424344")},
		}

		ethservice.APIBackend.OffchainBackend().MempoolBackned.SubmitBid(bid1)
		ethservice.APIBackend.OffchainBackend().MempoolBackned.SubmitBid(bid2)

		inoutAbi := mustParseMethodAbi(`[{"inputs":[{"internalType":"uint64","name":"cond","type":"uint64"}],"name":"fetchBids","outputs":[{"components":[{"internalType":"Suave.BidId","name":"id","type":"bytes16"},{"internalType":"uint64","name":"decryptionCondition","type":"uint64"},{"internalType":"address[]","name":"allowedPeekers","type":"address[]"}],"internalType":"struct Suave.Bid[]","name":"","type":"tuple[]"}],"stateMutability":"view","type":"function"}]`, "fetchBids")

		calldata, err := inoutAbi.Inputs.Pack(targetBlock)
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
		})

		require.Equal(t, bid1, suave.Bid{
			Id:                  bids[0].Id,
			DecryptionCondition: bids[0].DecryptionCondition,
			AllowedPeekers:      bids[0].AllowedPeekers,
		})
		require.Equal(t, bid2, suave.Bid{
			Id:                  bids[1].Id,
			DecryptionCondition: bids[1].DecryptionCondition,
			AllowedPeekers:      bids[1].AllowedPeekers,
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

		_, err = ethservice.APIBackend.OffchainBackend().ConfiendialStoreBackend.Retrieve(bid.Id, common.Address{0x41, 0x42, 0x43}, "ethBundleSimResults")
		require.NoError(t, err)
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
		}

		ethservice.APIBackend.OffchainBackend().MempoolBackned.SubmitBid(bid)
		ethservice.APIBackend.OffchainBackend().ConfiendialStoreBackend.Initialize(bid, "ethBundle", bundleBytes)

		ethHead := ethEthService.BlockChain().CurrentBlock()
		payloadArgsTuple := struct {
			Parent       common.Hash
			Timestamp    uint64
			FeeRecipient common.Address
			GasLimit     uint64
			Random       common.Hash
			Withdrawals  []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			Timestamp:    ethHead.Time + uint64(12),
			FeeRecipient: common.Address{0x42},
		}

		packed, err := suaveLibAbi.Methods["buildEthBlock"].Inputs.Pack(payloadArgsTuple, bid.Id)
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
		require.Equal(t, 1, len(envelope.ExecutionPayload.Transactions))

		var tx types.Transaction
		require.NoError(t, tx.UnmarshalBinary(envelope.ExecutionPayload.Transactions[0]))

		require.Equal(t, ethTx.Hash(), tx.Hash()) // Make ethTx cache its hash
		require.Equal(t, ethTx.Data(), tx.Data())
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
	}{
		Txs:             types.Transactions{ethTx},
		RevertingHashes: []common.Hash{},
	}
	bundleBytes, err := json.Marshal(bundle)
	require.NoError(t, err)

	targetBlock := uint64(1)

	{ // Send a bundle bid
		allowedPeekers := []common.Address{common.Address{0x41, 0x42, 0x43}, newBlockBidAddress, newBundleBidAddress, buildEthBlockAddress}
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
			Parent       common.Hash
			Timestamp    uint64
			FeeRecipient common.Address
			GasLimit     uint64
			Random       common.Hash
			Withdrawals  []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			Timestamp:    ethHead.Time + uint64(12),
			FeeRecipient: common.Address{0x42},
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

// Utilities

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)

	testBalance = big.NewInt(2e18)

	isOffchainAddress = common.HexToAddress("0x42010000")

	// confidentialStoreAddress = common.HexToAddress("0x42020000")
	// confStoreRetrieveAddress = common.HexToAddress("0x42020001")

	fetchBidsAddress    = common.HexToAddress("0x42030001")
	newBundleBidAddress = common.HexToAddress("0x42300000")
	newBlockBidAddress  = common.HexToAddress("0x42300001")

	simulateBundleAddress = common.HexToAddress("0x42100000")
	buildEthBlockAddress  = common.HexToAddress("0x42100001")

	testSuaveGenesis *core.Genesis = &core.Genesis{
		Timestamp:  1680000000,
		ExtraData:  nil,
		GasLimit:   30000000,
		BaseFee:    big.NewInt(0),
		Difficulty: big.NewInt(0),
		Alloc: core.GenesisAlloc{
			testAddr:            {Balance: testBalance},
			newBundleBidAddress: {Balance: big.NewInt(0), Code: hexutil.MustDecode(core.BidsContractCode)},
			newBlockBidAddress:  {Balance: big.NewInt(0), Code: hexutil.MustDecode(core.BlockBidContractCode)},
		},
	}

	testEthGenesis *core.Genesis = &core.Genesis{
		Timestamp:  1680000000,
		ExtraData:  nil,
		GasLimit:   30000000,
		BaseFee:    big.NewInt(0),
		Difficulty: big.NewInt(0),
		Alloc:      core.GenesisAlloc{testAddr: {Balance: testBalance}},
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
		if len(rpcErr.Error()) < 26 {
			require.NoError(t, rpcErr)
		}
		decodedError, err := hexutil.Decode(rpcErr.Error()[20:])
		if err != nil {
			require.NoError(t, rpcErr, err.Error())
		}

		unpacked, err := suaveLibAbi.Errors["PeekerReverted"].Inputs.Unpack(decodedError[4:])
		require.NoError(t, err, rpcErr.Error())

		require.NoError(t, rpcErr, string(unpacked[1].([]byte)))
	}
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

var (
	suaveLibAbi = mustParseAbi(`[{ "inputs": [ { "components": [ { "internalType": "bytes32", "name": "parent", "type": "bytes32" }, { "internalType": "uint64", "name": "timestamp", "type": "uint64" }, { "internalType": "address", "name": "feeRecipient", "type": "address" }, { "internalType": "uint64", "name": "gasLimit", "type": "uint64" }, { "internalType": "bytes32", "name": "random", "type": "bytes32" }, { "components": [ { "internalType": "uint64", "name": "index", "type": "uint64" }, { "internalType": "uint64", "name": "validator", "type": "uint64" }, { "internalType": "address", "name": "Address", "type": "address" }, { "internalType": "uint64", "name": "amount", "type": "uint64" } ], "internalType": "struct Suave.Withdrawal[]", "name": "withdrawals", "type": "tuple[]" } ], "internalType": "struct Suave.BuildBlockArgs", "name": "blockArgs", "type": "tuple" }, { "internalType": "Suave.BidId", "name": "bid", "type": "bytes16" } ], "name": "buildEthBlock", "outputs": [ { "internalType": "bytes", "name": "", "type": "bytes" }, { "internalType": "bytes", "name": "", "type": "bytes" } ], "stateMutability": "view", "type": "function" }, { "inputs": [ { "internalType": "address", "name": "", "type": "address" }, { "internalType": "bytes", "name": "", "type": "bytes" } ], "name": "PeekerReverted", "type": "error" }]`)

	bundleBidAbi = mustParseAbi(`[ { "anonymous": false, "inputs": [ { "indexed": false, "internalType": "Suave.BidId", "name": "bidId", "type": "bytes16" }, { "indexed": false, "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "indexed": false, "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "name": "BidEvent", "type": "event" }, { "inputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid", "name": "bid", "type": "tuple" } ], "name": "emitBid", "outputs": [], "stateMutability": "nonpayable", "type": "function" }, { "inputs": [], "name": "fetchBidConfidentialBundleData", "outputs": [ { "internalType": "bytes", "name": "", "type": "bytes" } ], "stateMutability": "nonpayable", "type": "function" }, { "inputs": [ { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "bidAllowedPeekers", "type": "address[]" } ], "name": "newBid", "outputs": [ { "internalType": "bytes", "name": "", "type": "bytes" } ], "stateMutability": "payable", "type": "function" } ]`)

	buildEthBlockAbi = mustParseAbi(`[ { "anonymous": false, "inputs": [ { "indexed": false, "internalType": "Suave.BidId", "name": "bidId", "type": "bytes16" }, { "indexed": false, "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "indexed": false, "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "name": "BidEvent", "type": "event" }, { "anonymous": false, "inputs": [ { "indexed": false, "internalType": "Suave.BidId", "name": "bidId", "type": "bytes16" }, { "indexed": false, "internalType": "bytes", "name": "builderBid", "type": "bytes" } ], "name": "BuilderBoostBidEvent", "type": "event" }, { "inputs": [ { "components": [ { "internalType": "bytes32", "name": "parent", "type": "bytes32" }, { "internalType": "uint64", "name": "timestamp", "type": "uint64" }, { "internalType": "address", "name": "feeRecipient", "type": "address" }, { "internalType": "uint64", "name": "gasLimit", "type": "uint64" }, { "internalType": "bytes32", "name": "random", "type": "bytes32" }, { "components": [ { "internalType": "uint64", "name": "index", "type": "uint64" }, { "internalType": "uint64", "name": "validator", "type": "uint64" }, { "internalType": "address", "name": "Address", "type": "address" }, { "internalType": "uint64", "name": "amount", "type": "uint64" } ], "internalType": "struct Suave.Withdrawal[]", "name": "withdrawals", "type": "tuple[]" } ], "internalType": "struct Suave.BuildBlockArgs", "name": "blockArgs", "type": "tuple" }, { "internalType": "uint64", "name": "blockHeight", "type": "uint64" }, { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid[]", "name": "bids", "type": "tuple[]" } ], "name": "build", "outputs": [ { "internalType": "bytes", "name": "", "type": "bytes" } ], "stateMutability": "nonpayable", "type": "function" }, { "inputs": [ { "components": [ { "internalType": "bytes32", "name": "parent", "type": "bytes32" }, { "internalType": "uint64", "name": "timestamp", "type": "uint64" }, { "internalType": "address", "name": "feeRecipient", "type": "address" }, { "internalType": "uint64", "name": "gasLimit", "type": "uint64" }, { "internalType": "bytes32", "name": "random", "type": "bytes32" }, { "components": [ { "internalType": "uint64", "name": "index", "type": "uint64" }, { "internalType": "uint64", "name": "validator", "type": "uint64" }, { "internalType": "address", "name": "Address", "type": "address" }, { "internalType": "uint64", "name": "amount", "type": "uint64" } ], "internalType": "struct Suave.Withdrawal[]", "name": "withdrawals", "type": "tuple[]" } ], "internalType": "struct Suave.BuildBlockArgs", "name": "blockArgs", "type": "tuple" }, { "internalType": "uint64", "name": "blockHeight", "type": "uint64" } ], "name": "buildFromPool", "outputs": [ { "internalType": "bytes", "name": "", "type": "bytes" } ], "stateMutability": "nonpayable", "type": "function" }, { "inputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid", "name": "bid", "type": "tuple" } ], "name": "emitBid", "outputs": [], "stateMutability": "nonpayable", "type": "function" }, { "inputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid", "name": "bid", "type": "tuple" }, { "internalType": "bytes", "name": "builderBid", "type": "bytes" } ], "name": "emitBuilderBidAndBid", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid", "name": "", "type": "tuple" }, { "internalType": "bytes", "name": "", "type": "bytes" } ], "stateMutability": "nonpayable", "type": "function" }, { "inputs": [ { "internalType": "Suave.BidId", "name": "bidId", "type": "bytes16" }, { "internalType": "bytes", "name": "signedBlindedHeader", "type": "bytes" } ], "name": "unlock", "outputs": [ { "internalType": "bytes", "name": "", "type": "bytes" } ], "stateMutability": "nonpayable", "type": "function" } ]`)
)
