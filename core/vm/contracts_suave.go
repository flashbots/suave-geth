package vm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

var (
	confStorePrecompileStoreMeter    = metrics.NewRegisteredMeter("suave/confstore/store", nil)
	confStorePrecompileRetrieveMeter = metrics.NewRegisteredMeter("suave/confstore/retrieve", nil)
)

var (
	isConfidentialAddress = common.HexToAddress("0x42010000")
)

/* General utility precompiles */

func (b *suaveRuntime) confidentialInputs() ([]byte, error) {
	return b.suaveContext.ConfidentialInputs, nil
}

/* Confidential store precompiles */

func (b *suaveRuntime) confidentialStore(bidId types.BidId, key string, data []byte) error {
	bid, err := b.suaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
	if err != nil {
		return suave.ErrBidNotFound
	}

	log.Info("confStore", "bidId", bidId, "key", key)

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialStoreAddr, bid)
	if err != nil {
		return err
	}

	if metrics.Enabled {
		confStorePrecompileStoreMeter.Mark(int64(len(data)))
	}

	_, err = b.suaveContext.Backend.ConfidentialStore.Store(bidId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

func (b *suaveRuntime) confidentialRetrieve(bidId types.BidId, key string) ([]byte, error) {
	bid, err := b.suaveContext.Backend.ConfidentialStore.FetchBidById(bidId)
	if err != nil {
		return nil, suave.ErrBidNotFound
	}

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialRetrieveAddr, bid)
	if err != nil {
		return nil, err
	}

	data, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(bidId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
	}

	return data, nil
}

/* Bid precompiles */

func (b *suaveRuntime) newBid(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, BidType string) (types.Bid, error) {
	if b.suaveContext.ConfidentialComputeRequestTx == nil {
		panic("newBid: source transaction not present")
	}

	bid, err := b.suaveContext.Backend.ConfidentialStore.InitializeBid(types.Bid{
		Salt:                suave.RandomBidId(),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		AllowedStores:       allowedStores,
		Version:             BidType, // TODO : make generic
	})
	if err != nil {
		return types.Bid{}, err
	}

	return bid, nil
}

func (b *suaveRuntime) fetchBids(targetBlock uint64, namespace string) ([]types.Bid, error) {
	bids1 := b.suaveContext.Backend.ConfidentialStore.FetchBidsByProtocolAndBlock(targetBlock, namespace)

	bids := make([]types.Bid, 0, len(bids1))
	for _, bid := range bids1 {
		bids = append(bids, bid.ToInnerBid())
	}

	return bids, nil
}

func mustParseAbi(data string) abi.ABI {
	inoutAbi, err := abi.JSON(strings.NewReader(data))
	if err != nil {
		panic(err.Error())
	}

	return inoutAbi
}

func mustParseMethodAbi(data string, method string) abi.Method {
	inoutAbi := mustParseAbi(data)
	return inoutAbi.Methods[method]
}

func formatPeekerError(format string, args ...any) ([]byte, error) {
	err := fmt.Errorf(format, args...)
	return []byte(err.Error()), err
}

type suaveRuntime struct {
	suaveContext *SuaveContext
}

var _ SuaveRuntime = &suaveRuntime{}

// https://docs.flashbots.net/flashbots-auction/advanced/rpc-endpoint#eth_sendbundle
type sbundleJson struct {
	Txs         []string `json:"txs"`
	BlockNumber uint64   `json:"blockNumber"`
}

func (s *suaveRuntime) sendBundle(url string, bundle types.Bundle) error {
	input := sbundleJson{
		Txs:         []string{},
		BlockNumber: bundle.BlockNumber,
	}
	for _, txn := range bundle.Transactions {
		input.Txs = append(input.Txs, hex.EncodeToString(txn))
	}

	params, err := json.Marshal(input)
	if err != nil {
		return err
	}

	if _, err := s.submitBundleJsonRPC(url, "eth_sendBundle", params); err != nil {
		return err
	}
	return nil
}

func (s *suaveRuntime) sendMevShareBundle(url string, bundle types.MevShareBundle) error {
	shareBundle := &types.RPCMevShareBundle{
		Version: "v0.1",
	}

	for _, tx := range bundle.Transactions {
		shareBundle.Body = append(shareBundle.Body, struct {
			Tx        string `json:"tx"`
			CanRevert bool   `json:"canRevert"`
		}{Tx: hexutil.Encode(tx)})
	}
	for indx, refund := range bundle.RefundPercents {
		shareBundle.Validity.Refund = append(shareBundle.Validity.Refund, struct {
			BodyIdx int `json:"bodyIdx"`
			Percent int `json:"percent"`
		}{
			BodyIdx: indx,
			Percent: int(refund),
		})
	}

	params, err := json.Marshal(shareBundle)
	if err != nil {
		return err
	}

	if _, err := s.submitBundleJsonRPC(url, "mev_sendBundle", params); err != nil {
		return err
	}
	return nil
}

func (s *suaveRuntime) simulateTransactions(stxns []types.STransaction) (uint64, error) {
	txns := types.Transactions{}
	for _, stxn := range stxns {
		txn, err := STransactionToTransaction(&stxn)
		if err != nil {
			return 0, err
		}
		txns = append(txns, txn)
	}

	envelope, err := s.suaveContext.Backend.ConfidentialEthBackend.BuildEthBlock(context.Background(), nil, txns)
	if err != nil {
		return 0, err
	}
	if envelope.ExecutionPayload.GasUsed == 0 {
		return 0, err
	}

	egp := new(big.Int).Div(envelope.BlockValue, big.NewInt(int64(envelope.ExecutionPayload.GasUsed)))
	return egp.Uint64(), nil
}

func (s *suaveRuntime) encodeRLPTxn(stxn types.STransaction) ([]byte, error) {
	txn, err := STransactionToTransaction(&stxn)
	if err != nil {
		return nil, err
	}
	return txn.MarshalBinary()
}

var zeroAddress = common.Address{}

func STransactionToTransaction(stxn *types.STransaction) (*types.Transaction, error) {
	legacyTxn := &types.LegacyTx{
		Nonce:    stxn.Nonce,
		GasPrice: new(big.Int).SetUint64(stxn.GasPrice),
		Gas:      stxn.GasLimit,
		Value:    new(big.Int).SetUint64(stxn.Value),
		Data:     stxn.Data,
		V:        new(big.Int).SetBytes(stxn.V),
		R:        new(big.Int).SetBytes(stxn.R),
		S:        new(big.Int).SetBytes(stxn.S),
	}

	if stxn.To != zeroAddress {
		legacyTxn.To = &stxn.To
	}

	return types.NewTx(legacyTxn), nil
}

func TransactionToStransaction(ethTx *types.Transaction) *types.STransaction {
	to := ethTx.To()

	v, rr, s := ethTx.RawSignatureValues()

	return &types.STransaction{
		Nonce:    ethTx.Nonce(),
		To:       *to,
		Value:    ethTx.Value().Uint64(),
		GasPrice: ethTx.GasPrice().Uint64(),
		GasLimit: ethTx.Gas(),
		Data:     ethTx.Data(),
		V:        v.Bytes(),
		R:        rr.Bytes(),
		S:        s.Bytes(),
	}
}
