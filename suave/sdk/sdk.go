package sdk

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

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

func (c *Contract) SendTransaction(method string, args []interface{}, confidentialDataBytes []byte) (*TransactionResult, error) {
	clt := ethclient.NewClient(c.client.rpc)

	signer, err := c.client.getSigner()
	if err != nil {
		return nil, err
	}

	calldata, err := c.abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	senderAddr := crypto.PubkeyToAddress(c.client.key.PublicKey)
	nonce, err := clt.PendingNonceAt(context.Background(), senderAddr)
	if err != nil {
		return nil, err
	}

	wrappedTxData := &types.LegacyTx{
		Nonce:    nonce,
		To:       &c.addr,
		Value:    nil,
		Gas:      1000000,
		GasPrice: big.NewInt(10),
		Data:     calldata,
	}

	offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
		ExecutionNode: c.client.execNode,
		Wrapped:       *types.NewTx(wrappedTxData),
	}), signer, c.client.key)
	if err != nil {
		return nil, err
	}

	offchainTxBytes, err := offchainTx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	var hash common.Hash
	if err = c.client.rpc.Call(&hash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes), hexutil.Encode(confidentialDataBytes)); err != nil {
		return nil, err
	}

	res := &TransactionResult{
		hash: hash,
	}
	return res, nil
}

type TransactionResult struct {
	hash common.Hash
}

func (t *TransactionResult) Hash() common.Hash {
	return t.hash
}

type Client struct {
	rpc      *rpc.Client
	key      *ecdsa.PrivateKey
	execNode common.Address
}

func NewClient(rpc *rpc.Client, key *ecdsa.PrivateKey, execNode common.Address) *Client {
	c := &Client{
		rpc:      rpc,
		key:      key,
		execNode: execNode,
	}
	return c
}

func (c *Client) getSigner() (types.Signer, error) {
	clt := ethclient.NewClient(c.rpc)
	chainID, err := clt.ChainID(context.TODO())
	if err != nil {
		return nil, err
	}

	signer := types.NewOffchainSigner(chainID)
	return signer, nil
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
