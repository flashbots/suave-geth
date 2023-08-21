// Hash: 0479cfd5872468c477531906bed5fc154c66974e33d2e7b38e91f526ec8d4182
package examples

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/mitchellh/mapstructure"
)

type Client struct {
	rpc *rpc.Client
}

func NewClient(rpc *rpc.Client) *Client {
	return &Client{rpc: rpc}
}

func (c *Client) call(addr string, methodName string, args []interface{}) ([]interface{}, error) {
	method := artifacts.SuaveAbi.Methods[methodName]

	input, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}

	addrD := common.HexToAddress(addr)
	msg := ethapi.TransactionArgs{
		To:         &addrD,
		IsOffchain: true,
		Data:       (*hexutil.Bytes)(&input),
	}

	var respBytes hexutil.Bytes
	if err := c.rpc.Call(&respBytes, "eth_call", msg, "latest"); err != nil {
		return nil, err
	}

	resp, err := method.Outputs.Unpack(respBytes)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) BuildEthBlock(blockArgs types.BuildBlockArgs, bid [16]byte, namespace string) (resp0 []byte, resp1 []byte, err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042100001", "buildEthBlock", []interface{}{blockArgs, bid, namespace}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	resp0, ok = resp[0].([]byte)
	if !ok {
		err = fmt.Errorf("failed to decode argument 0")
		return
	}

	resp1, ok = resp[1].([]byte)
	if !ok {
		err = fmt.Errorf("failed to decode argument 1")
		return
	}

	return
}

func (c *Client) ConfidentialStoreRetrieve(bidId [16]byte, key string) (resp0 []byte, err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042020001", "confidentialStoreRetrieve", []interface{}{bidId, key}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	resp0, ok = resp[0].([]byte)
	if !ok {
		err = fmt.Errorf("failed to decode argument 0")
		return
	}

	return
}

func (c *Client) ConfidentialStoreStore(bidId [16]byte, key string, data []byte) (err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042020000", "confidentialStoreStore", []interface{}{bidId, key, data}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	return
}

func (c *Client) ExtractHint(bundleData []byte) (resp0 []byte, err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042100037", "extractHint", []interface{}{bundleData}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	resp0, ok = resp[0].([]byte)
	if !ok {
		err = fmt.Errorf("failed to decode argument 0")
		return
	}

	return
}

func (c *Client) FetchBids(cond uint64, namespace string) (resp0 []types.Bid, err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042030001", "fetchBids", []interface{}{cond, namespace}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	if err = mapstructure.Decode(resp[0], &resp0); err != nil {
		return
	}

	return
}

func (c *Client) NewBid(decryptionCondition uint64, allowedPeekers []common.Address, BidType string) (resp0 types.Bid, err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042030000", "newBid", []interface{}{decryptionCondition, allowedPeekers, BidType}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	if err = mapstructure.Decode(resp[0], &resp0); err != nil {
		return
	}

	return
}

func (c *Client) SimulateBundle(bundleData []byte) (resp0 uint64, err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042100000", "simulateBundle", []interface{}{bundleData}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	resp0, ok = resp[0].(uint64)
	if !ok {
		err = fmt.Errorf("failed to decode argument 0")
		return
	}

	return
}

func (c *Client) SubmitEthBlockBidToRelay(relayUrl string, builderBid []byte) (resp0 []byte, err error) {
	var resp []interface{}
	var ok bool

	if resp, err = c.call("0x0000000000000000000000000000000042100002", "submitEthBlockBidToRelay", []interface{}{relayUrl, builderBid}); err != nil {
		err = fmt.Errorf("failed to make rpc request: %v", err)
		return
	}

	_ = resp
	_ = ok

	resp0, ok = resp[0].([]byte)
	if !ok {
		err = fmt.Errorf("failed to decode argument 0")
		return
	}

	return
}
