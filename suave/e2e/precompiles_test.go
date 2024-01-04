package e2e

import (
	"context"
	"encoding/hex"
	"io"
	"math/big"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/ethereum/go-ethereum/suave/sdk"
	"github.com/stretchr/testify/require"
)

func TestE2E_Precompiles_IsConfidential(t *testing.T) {
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

func TestE2E_Precompiles_SignEthTransaction(t *testing.T) {
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

func TestE2E_Precompiles_SignMessage(t *testing.T) {
	fr := newFramework(t)
	defer fr.Close()

	// Prepare the message digest and generate a signing key
	message := "Hello, world!"
	digest := crypto.Keccak256([]byte(message))

	sk, err := crypto.GenerateKey()
	require.NoError(t, err)
	skHex := hex.EncodeToString(crypto.FromECDSA(sk))

	// function signMessage(bytes memory digest, string memory signingKey)
	args, err := artifacts.SuaveAbi.Methods["signMessage"].Inputs.Pack(digest, skHex)
	require.NoError(t, err)

	gas := hexutil.Uint64(1000000)

	var callResult hexutil.Bytes
	err = fr.suethSrv.RPCNode().Call(&callResult, "eth_call", setTxArgsDefaults(ethapi.TransactionArgs{
		To:             &signMessage,
		Gas:            &gas,
		IsConfidential: true,
		Data:           (*hexutil.Bytes)(&args),
	}), "latest")
	requireNoRpcError(t, err)

	// Unpack the call result to get the signed message
	unpackedCallResult, err := artifacts.SuaveAbi.Methods["signMessage"].Outputs.Unpack(callResult)
	require.NoError(t, err)

	// Assert that recovered key is correct
	signature := unpackedCallResult[0].([]byte)
	pubKeyRecovered, err := crypto.SigToPub(digest, signature)
	require.NoError(t, err)

	require.Equal(t, crypto.PubkeyToAddress(sk.PublicKey), crypto.PubkeyToAddress(*pubKeyRecovered))
}

func TestE2E_Precompiles_HttpRemoteCalls(t *testing.T) {
	fr := newFramework(t, WithWhitelist([]string{"127.0.0.1"}))
	defer fr.Close()

	clt := fr.NewSDKClient()

	contractAddr := common.Address{0x3}
	contract := sdk.GetContract(contractAddr, exampleCallSourceContract.Abi, clt)

	t.Run("Get", func(t *testing.T) {
		srvAddr := fr.testHttpRelayer(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, r.Method, "GET")
			require.Equal(t, r.Header.Get("a"), "b")
			w.Write([]byte{0x1, 0x2, 0x3})
		})

		req := &types.HttpRequest{
			Method:  "GET",
			Url:     srvAddr,
			Headers: []string{"a:b"},
		}
		contract.SendTransaction("remoteCall", []interface{}{req}, nil)
	})

	t.Run("Post", func(t *testing.T) {
		body := []byte{0x1, 0x2, 0x3}

		srvAddr := fr.testHttpRelayer(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, r.Method, "POST")
			require.Equal(t, r.Header.Get("b"), "c")

			bodyRes, _ := io.ReadAll(r.Body)
			require.Equal(t, body, bodyRes)

			w.Write([]byte{0x1, 0x2, 0x3})
		})

		req := &types.HttpRequest{
			Method:  "POST",
			Url:     srvAddr,
			Headers: []string{"b:c"},
			Body:    body,
		}
		contract.SendTransaction("remoteCall", []interface{}{req}, nil)
	})

	t.Run("Not whitelisted", func(t *testing.T) {
		req := &types.HttpRequest{
			Method:  "POST",
			Url:     "http://example.com",
			Headers: []string{"b:c"},
		}
		_, err := contract.SendTransaction("remoteCall", []interface{}{req}, nil)
		require.Error(t, err)
	})
}

func TestE2E_Precompiles_Call(t *testing.T) {
	// This end-to-end tests that the callx precompile gets called from a confidential request
	fr := newFramework(t, WithKettleAddress())
	defer fr.Close()

	clt := fr.NewSDKClient()

	// We reuse the same address for both the source and target contract
	contractAddr := common.Address{0x3}
	sourceContract := sdk.GetContract(contractAddr, exampleCallSourceContract.Abi, clt)

	expectedNum := big.NewInt(101)
	res, err := sourceContract.SendTransaction("callTarget", []interface{}{contractAddr, expectedNum}, nil)
	require.NoError(t, err)

	// make sure we can retrieve the transaction
	tx, _, err := ethclient.NewClient(fr.suethSrv.RPCNode()).TransactionByHash(context.Background(), res.Hash())
	require.NoError(t, err)
	require.Equal(t, tx.Type(), uint8(types.SuaveTxType))

	incorrectNum := big.NewInt(102)
	_, err = sourceContract.SendTransaction("callTarget", []interface{}{contractAddr, incorrectNum}, nil)
	require.Error(t, err)
}

func TestE2E_Precompiles_Builder(t *testing.T) {
	fr := newFramework(t, WithKettleAddress())
	defer fr.Close()

	clt := fr.NewSDKClient()

	// TODO: We do this all the time, unify in a single function?
	contractAddr := common.Address{0x3}
	sourceContract := sdk.GetContract(contractAddr, exampleCallSourceContract.Abi, clt)

	// build a txn that calls the contract 'func1' in 'ExampleEthCallTarget'
	var subTxns []*types.Transaction
	for i := 0; i < 2; i++ {
		subTxn, _ := types.SignTx(types.NewTx(&types.LegacyTx{
			To:       &testAddr3,
			Gas:      220000,
			GasPrice: big.NewInt(13),
			Nonce:    uint64(i),
			Data:     exampleCallTargetContract.Abi.Methods["func1"].ID,
		}), signer, testKey)

		subTxns = append(subTxns, subTxn)
	}

	subTxnBytes1, _ := subTxns[0].MarshalBinary()
	subTxnBytes2, _ := subTxns[1].MarshalBinary()

	_, err := sourceContract.SendTransaction("sessionE2ETest", []interface{}{subTxnBytes1, subTxnBytes2}, nil)
	require.NoError(t, err)
}
