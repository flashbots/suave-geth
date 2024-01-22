package sdk

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

func DeployContract(bytecode []byte, client *Client) (*TransactionResult, error) {
	txn := &types.LegacyTx{
		Data: bytecode,
	}
	return client.SendTransaction(txn)
}

type Contract struct {
	addr   common.Address
	abi    *abi.ABI
	client *Client
}

func GetContract(addr common.Address, abi *abi.ABI, client *Client) *Contract {
	c := &Contract{
		addr:   addr,
		abi:    abi,
		client: client,
	}
	return c
}

func (c *Contract) Address() common.Address {
	return c.addr
}

func (c *Contract) SendTransaction(method string, args []interface{}, confidentialDataBytes []byte) (*TransactionResult, error) {
	/*
		signer, err := c.client.getSigner()
		if err != nil {
			return nil, err
		}
	*/

	calldata, err := c.abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	senderAddr := crypto.PubkeyToAddress(c.client.key.PublicKey)
	nonce, err := c.client.rpc.PendingNonceAt(context.Background(), senderAddr)
	if err != nil {
		return nil, err
	}

	gasPrice, err := c.client.rpc.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}

	record := &types.ConfidentialComputeRecord{
		KettleAddress:          c.client.kettleAddress,
		Nonce:                  nonce,
		To:                     &c.addr,
		Value:                  big.NewInt(0),
		GasPrice:               gasPrice,
		Gas:                    1000000,
		Data:                   calldata,
		ConfidentialInputsHash: crypto.Keccak256Hash(confidentialDataBytes), // ?

		// initialize empty fields
		ChainID: big.NewInt(1337), // suave chain id?
		V:       big.NewInt(0),
		R:       big.NewInt(0),
		S:       big.NewInt(0),
	}

	rawRecord, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	eipTypedData := suave.BuildConfidentialRecordEIP712Envelope(record)
	typedDataHashed, _, err := apitypes.TypedDataAndHash(eipTypedData)
	if err != nil {
		panic("?")
		return nil, err
	}

	signedMsg, err := c.client.Sign(typedDataHashed[:])
	if err != nil {
		panic(err)
	}

	fmt.Println("-- sender of the txn --", c.client.Addr())
	fmt.Println(c.client.rpc.BalanceAt(context.Background(), c.client.Addr(), nil))

	/*
		fmt.Println(string(raw))

		panic("xxx")

		computeRequestBytes, err := computeRequest.MarshalBinary()
		if err != nil {
			return nil, err
		}

		var hash common.Hash
		if err = c.client.rpc.Client().Call(&hash, "eth_sendRawTransaction", hexutil.Encode(computeRequestBytes)); err != nil {
			return nil, err
		}
	*/

	envelope := &types.ConfidentialComputeRequest2{
		Message:   rawRecord,
		Signature: signedMsg,
		Sender:    c.client.Addr(),
	}

	var hash common.Hash
	if err = c.client.rpc.Client().Call(&hash, "eth_sendRawTransaction2", envelope, hexutil.Encode(confidentialDataBytes)); err != nil {
		return nil, err
	}

	res := &TransactionResult{
		clt:  c.client,
		hash: hash,
	}
	return res, nil
}

type TransactionResult struct {
	clt     *Client
	hash    common.Hash
	receipt *types.Receipt
}

func (t *TransactionResult) Wait() (*types.Receipt, error) {
	return t.WaitWithTimeout(10 * time.Second)
}

func (t *TransactionResult) WaitWithTimeout(timeout time.Duration) (*types.Receipt, error) {
	if t.receipt != nil {
		return t.receipt, nil
	}

	timer := time.NewTimer(timeout)

	var receipt *types.Receipt
	var err error

	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout")
		case <-time.After(100 * time.Millisecond):
			receipt, err = t.clt.rpc.TransactionReceipt(context.Background(), t.hash)
			if err != nil && err != ethereum.NotFound {
				return nil, err
			}
			if receipt != nil {
				t.receipt = receipt
				return t.receipt, nil
			}
		}
	}
}

func (t *TransactionResult) Hash() common.Hash {
	return t.hash
}

type Client struct {
	rpc           *ethclient.Client
	key           *ecdsa.PrivateKey
	kettleAddress common.Address
}

func NewClient(rpc *rpc.Client, key *ecdsa.PrivateKey, kettleAddress common.Address) *Client {
	c := &Client{
		rpc:           ethclient.NewClient(rpc),
		key:           key,
		kettleAddress: kettleAddress,
	}
	return c
}

func (c *Client) RPC() *ethclient.Client {
	return c.rpc
}

func (c *Client) getSigner() (types.Signer, error) {
	chainID, err := c.rpc.ChainID(context.TODO())
	if err != nil {
		return nil, err
	}

	signer := types.NewSuaveSigner(chainID)
	return signer, nil
}

func (c *Client) Addr() common.Address {
	return crypto.PubkeyToAddress(c.key.PublicKey)
}

func (c *Client) Sign(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, c.key)
}

func (c *Client) SignTxn(txn *types.LegacyTx) (*types.Transaction, error) {
	signer, err := c.getSigner()
	if err != nil {
		return nil, err
	}
	ethTx, err := types.SignTx(types.NewTx(txn), signer, c.key)
	if err != nil {
		return nil, err
	}
	return ethTx, nil
}

func (c *Client) SendTransaction(wrappedTxData *types.LegacyTx) (*TransactionResult, error) {
	senderAddr := crypto.PubkeyToAddress(c.key.PublicKey)

	if wrappedTxData.Nonce == 0 {
		nonce, err := c.rpc.PendingNonceAt(context.Background(), senderAddr)
		if err != nil {
			return nil, err
		}
		wrappedTxData.Nonce = nonce
	}

	if wrappedTxData.GasPrice == nil {
		gasPrice, err := c.rpc.SuggestGasPrice(context.Background())
		if err != nil {
			return nil, err
		}
		wrappedTxData.GasPrice = gasPrice
	}

	if wrappedTxData.Gas == 0 {
		estimateMsg := ethereum.CallMsg{
			From:     senderAddr,
			To:       wrappedTxData.To,
			GasPrice: wrappedTxData.GasPrice,
			Value:    wrappedTxData.Value,
			Data:     wrappedTxData.Data,
		}
		gasLimit, err := c.rpc.EstimateGas(context.Background(), estimateMsg)
		if err != nil {
			return nil, err
		}
		wrappedTxData.Gas = gasLimit
	}

	txn, err := c.SignTxn(wrappedTxData)
	if err != nil {
		return nil, err
	}

	txnBytes, err := txn.MarshalBinary()
	if err != nil {
		return nil, err
	}

	var hash common.Hash
	if err = c.rpc.Client().Call(&hash, "eth_sendRawTransaction", hexutil.Encode(txnBytes)); err != nil {
		return nil, err
	}

	res := &TransactionResult{
		clt:  c,
		hash: hash,
	}
	return res, nil
}
