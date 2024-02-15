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

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/suave/consolelog"
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

func (b *suaveRuntime) confidentialStore(dataId types.DataId, key string, data []byte) error {
	if b.suaveContext.Backend.ConfidentialStore == nil {
		return fmt.Errorf("confidential store is not enabled")
	}

	record, err := b.suaveContext.Backend.ConfidentialStore.FetchRecordByID(dataId)
	if err != nil {
		return suave.ErrRecordNotFound
	}

	log.Debug("confStore", "dataId", dataId, "key", key)

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialStoreAddr, record)
	if err != nil {
		return err
	}

	if metrics.Enabled {
		confStorePrecompileStoreMeter.Mark(int64(len(data)))
	}

	_, err = b.suaveContext.Backend.ConfidentialStore.Store(dataId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

func (b *suaveRuntime) confidentialRetrieve(dataId types.DataId, key string) ([]byte, error) {
	if b.suaveContext.Backend.ConfidentialStore == nil {
		return nil, fmt.Errorf("confidential store is not enabled")
	}

	record, err := b.suaveContext.Backend.ConfidentialStore.FetchRecordByID(dataId)
	if err != nil {
		return nil, suave.ErrRecordNotFound
	}

	caller, err := checkIsPrecompileCallAllowed(b.suaveContext, confidentialRetrieveAddr, record)
	if err != nil {
		return nil, err
	}

	data, err := b.suaveContext.Backend.ConfidentialStore.Retrieve(dataId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
	}

	return data, nil
}

/* Data Record precompiles */

func (b *suaveRuntime) newDataRecord(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, RecordType string) (types.DataRecord, error) {
	if b.suaveContext.Backend.ConfidentialStore == nil {
		return types.DataRecord{}, fmt.Errorf("confidential store is not enabled")
	}

	record, err := b.suaveContext.Backend.ConfidentialStore.InitRecord(types.DataRecord{
		Salt:                suave.RandomDataRecordId(),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		AllowedStores:       allowedStores,
		Version:             RecordType, // TODO : make generic
	})
	if err != nil {
		return types.DataRecord{}, err
	}

	return record, nil
}

func (b *suaveRuntime) fetchDataRecords(targetBlock uint64, namespace string) ([]types.DataRecord, error) {
	if b.suaveContext.Backend.ConfidentialStore == nil {
		return nil, fmt.Errorf("confidential store is not enabled")
	}

	records1 := b.suaveContext.Backend.ConfidentialStore.FetchRecordsByProtocolAndBlock(targetBlock, namespace)

	records := make([]types.DataRecord, 0, len(records1))
	for _, record := range records1 {
		records = append(records, record.ToInnerRecord())
	}

	return records, nil
}

func (s *suaveRuntime) signMessage(digest []byte, signingKey string) ([]byte, error) {
	key, err := crypto.HexToECDSA(signingKey)
	if err != nil {
		return nil, fmt.Errorf("key not formatted properly: %w", err)
	}

	signature, err := crypto.Sign(digest, key)
	if err != nil {
		return nil, fmt.Errorf("Failed to sign message: %v", err)
	}

	return signature, nil
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

type consoleLogPrecompile struct {
}

func (c *consoleLogPrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *consoleLogPrecompile) Run(input []byte) ([]byte, error) {
	if err := consolelog.Print(input); err != nil {
		log.Error("failed to console2 print", "err", err)
	}
	return nil, nil
}

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

	if request.WithFlashbotsSignature {
		// hash the body and sign it with the kettle signing key
		hashedBody := crypto.Keccak256Hash(request.Body).Hex()
		sig, err := crypto.Sign(accounts.TextHash([]byte(hashedBody)), s.suaveContext.Backend.EthBundleSigningKey)
		if err != nil {
			return nil, err
		}

		signature := crypto.PubkeyToAddress(s.suaveContext.Backend.EthBundleSigningKey.PublicKey).Hex() + ":" + hexutil.Encode(sig)
		req.Header.Add("X-Flashbots-Signature", signature)
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
	return s.suaveContext.Backend.ConfidentialEthBackend.NewSession(context.Background())
}

func (s *suaveRuntime) simulateTransaction(session string, txnBytes []byte, chainId string) (types.SimulateTransactionResult, error) {
	txn := new(types.Transaction)
	if err := txn.UnmarshalBinary(txnBytes); err != nil {
		return types.SimulateTransactionResult{}, err
	}

	backend, err := s.suaveContext.Backend.GetConfidentialEthBackend(chainId)
	if err != nil {
		return types.SimulateTransactionResult{}, fmt.Errorf("failed to get backend: %w", err)
	}
	result, err := backend.AddTransaction(context.Background(), session, txn)
	if err != nil {
		return types.SimulateTransactionResult{}, err
	}
	return *result, nil
}
