package vm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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

type suaveRuntime struct {
	suaveContext *SuaveContext
}

var _ SuaveRuntime = &suaveRuntime{}

func (s *suaveRuntime) doHTTPRequest(request types.HttpRequest) ([]byte, error) {
	if request.Method != "GET" && request.Method != "POST" {
		return nil, fmt.Errorf("only GET and POST methods are supported")
	}
	if request.Url == "" {
		return nil, fmt.Errorf("url is empty")
	}

	var body io.Reader
	if request.Body != nil {
		body = bytes.NewReader(request.Body)
	}

	// decode the url and check if the domain is allowed
	parsedURL, err := url.Parse(request.Url)
	if err != nil {
		panic(err)
	}

	var allowed bool
	for _, allowedDomain := range s.suaveContext.Backend.ExternalWhitelist {
		if allowedDomain == "*" || allowedDomain == parsedURL.Hostname() {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("domain %s is not allowed", parsedURL.Hostname())
	}

	req, err := http.NewRequest(request.Method, request.Url, body)
	if err != nil {
		return nil, err
	}

	for _, header := range request.Headers {
		indx := strings.Index(header, ":")
		if indx == -1 {
			return nil, fmt.Errorf("incorrect header format '%s', no ':' present", header)
		}
		req.Header.Add(header[:indx], header[indx+1:])
	}

	client := &http.Client{
		Timeout: 5 * time.Second, // TODO: test
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("http error: %s: %v", resp.Status, data)
	}
	return data, nil
}

func (s *suaveRuntime) newBuilder() (string, error) {
	fmt.Println("-- new builder --")
	return s.suaveContext.Backend.ConfidentialEthBackend.NewSession(context.Background())
}

func (s *suaveRuntime) simulateTransaction(session string, txnBytes []byte) (types.SimulateTransactionResult, error) {
	txn := new(types.Transaction)
	if err := txn.UnmarshalBinary(txnBytes); err != nil {
		return types.SimulateTransactionResult{}, err
	}

	receipt, err := s.suaveContext.Backend.ConfidentialEthBackend.AddTransaction(context.Background(), session, txn)
	if err != nil {
		return types.SimulateTransactionResult{}, err
	}

	res := types.SimulateTransactionResult{
		Logs: []*types.SimulatedLog{},
	}
	for _, log := range receipt.Logs {
		res.Logs = append(res.Logs, &types.SimulatedLog{
			Addr:   log.Address,
			Topics: log.Topics,
			Data:   log.Data,
		})
	}

	return res, nil
}
