package sdk

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type Contract struct {
	addr common.Address
	abi  *abi.ABI
	rpc  *rpc.Client
}

func GetContract(addr common.Address, abi *abi.ABI, rpc *rpc.Client) *Contract {
	c := &Contract{
		addr: addr,
		abi:  abi,
		rpc:  rpc,
	}
	return c
}

func (c *Contract) Call(methodName string, args []interface{}) ([]interface{}, error) {
	return nil, nil
}

func (c *Contract) SendTransaction(method string, args []interface{}, confidentialDataBytes []byte, execNode common.Address, testKey *ecdsa.PrivateKey) error {
	clt := ethclient.NewClient(c.rpc)
	chainID, err := clt.ChainID(context.TODO())
	if err != nil {
		return err
	}

	signer := types.NewOffchainSigner(chainID)

	calldata, err := c.abi.Pack(method, args...)
	if err != nil {
		return err
	}

	wrappedTxData := &types.LegacyTx{
		Nonce:    0,
		To:       &c.addr,
		Value:    nil,
		Gas:      1000000,
		GasPrice: big.NewInt(10),
		Data:     calldata,
	}

	offchainTx, err := types.SignTx(types.NewTx(&types.OffchainTx{
		ExecutionNode: execNode,
		Wrapped:       *types.NewTx(wrappedTxData),
	}), signer, testKey)
	if err != nil {
		return err
	}

	offchainTxBytes, err := offchainTx.MarshalBinary()
	if err != nil {
		return err
	}

	var offchainTxHash common.Hash
	err = c.rpc.Call(&offchainTxHash, "eth_sendRawTransaction", hexutil.Encode(offchainTxBytes), hexutil.Encode(confidentialDataBytes))
	if err != nil {
		return err
	}

	return nil
}

type Client interface {
}
