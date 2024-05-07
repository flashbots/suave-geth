package e2e

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/sdk"
	"github.com/stretchr/testify/require"
)

var (
	kettleAddr = common.HexToAddress("b5feafbdd752ad52afb7e1bd2e40432a485bbb7f")
)

func TestOp_FundKettle(t *testing.T) {
	// funded private key in optimism, we get this from the make devnet-up command in optimism
	// does it change? please make sure it is the correct one in your setup in case it changes.
	privateKey, _ := crypto.HexToECDSA("8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba")
	privateKeyAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	fmt.Printf("Using funding account: %s\n", privateKeyAddr)

	// deploy the contrac
	rpcConn, err := rpc.Dial("http://localhost:9546")
	if err != nil {
		t.Fatal(err)
	}

	fundAddresses := []common.Address{
		kettleAddr,
		testAddr,
	}
	for _, addr := range fundAddresses {
		val, _ := new(big.Int).SetString("1119998885232998885233", 10)

		clt := sdk.NewClient(rpcConn, privateKey, common.Address{})
		txn, err := clt.SendTransaction(&types.LegacyTx{
			To:    &addr,
			Value: val,
		})
		if err != nil {
			t.Fatal(err)
		}

		receipt, err := txn.Wait()
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("Funded %s (%d)\n", addr.Hex(), receipt.BlockNumber)
	}
}

func TestOP_DeployContract(t *testing.T) {
	rpcConn, err := rpc.Dial("http://localhost:9546")
	if err != nil {
		t.Fatal(err)
	}
	clt := sdk.NewClient(rpcConn, testKey, common.Address{})
	txn, err := sdk.DeployContract(mossBundle1.Code, clt)
	if err != nil {
		t.Fatal(err)
	}

	receipt, err := txn.Wait()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Suapp deployed: %s\n", receipt.ContractAddress.Hex())
}

var suappAddr = common.HexToAddress("0x3A220f351252089D385b29beca14e27F204c296A")

func TestOp_SuappCCRs(t *testing.T) {
	rpcConn, err := rpc.Dial("http://localhost:9546")
	if err != nil {
		t.Fatal(err)
	}
	clt := sdk.NewClient(rpcConn, testKey, kettleAddr)
	contract := sdk.GetContract(suappAddr, mossBundle1.Abi, clt)

	txn, err := contract.SendTransaction("incr", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("CCR done (%d)\n", txn.Hash())
}

func TestOp_Moss(t *testing.T) {
	rpcConn, err := rpc.Dial("http://localhost:9546")
	if err != nil {
		t.Fatal(err)
	}

	ethClt := ethclient.NewClient(rpcConn)
	clt := sdk.NewClient(rpcConn, testKey, common.Address{})

	// use a new account to make the transaction
	newAcct, _ := crypto.GenerateKey()
	newAcctAddr := crypto.PubkeyToAddress(newAcct.PublicKey)

	// fund the new account
	{
		txn, err := clt.SendTransaction(&types.LegacyTx{
			To:    &newAcctAddr,
			Value: big.NewInt(100000000),
		})
		if err != nil {
			t.Fatal(err)
		}

		receipt, err := txn.Wait()
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("Funded %s (%d)\n", newAcctAddr.Hex(), receipt.BlockNumber)
	}

	// build the internal transaction, a simple transfer
	txn1, err := types.SignTx(types.NewTx(&types.LegacyTx{
		Nonce:    0, // because the account is new
		To:       &testAddr2,
		Value:    big.NewInt(1111),
		Gas:      1000000,
		GasPrice: big.NewInt(10),
	}), types.NewLondonSigner(big.NewInt(901)), newAcct)
	require.NoError(t, err)

	txn1Marshal, err := txn1.MarshalBinary()
	require.NoError(t, err)

	bundle := &Moss1Bundle{
		Txns: []Moss1BundleTxn{
			{Txn: txn1Marshal, CanRevert: true},
		},
	}
	data, err := mossBundle1.Abi.Pack("applyFn", bundle)
	require.NoError(t, err)

	blockNumber, err := ethClt.BlockNumber(context.Background())
	require.NoError(t, err)

	// send the bundle
	ethClt.SendBundle(context.Background(), &types.MossBundle{
		To:             suappAddr,
		Data:           data,
		BlockNumber:    blockNumber + 1,
		MaxBlockNumber: blockNumber + 10,
	})
}
