package vm

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
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
	"github.com/flashbots/go-boost-utils/bls"
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
	return b.contextGet("confidentialInputs")
}

func (b *suaveRuntime) randomBytes(numBytes uint8) ([]byte, error) {
	buf := make([]byte, numBytes)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
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

func (s *suaveRuntime) signMessage(digest []byte, cryptoType types.CryptoSignature, signingKey string) ([]byte, error) {
	if !strings.HasPrefix(signingKey, "0x") {
		// we need to prefix with 0x if not present because the 'hexutil.Decode' fails to decode if there is no '0x' prefix
		signingKey = "0x" + signingKey
	}
	signingKeyBuf, err := hexutil.Decode(signingKey)
	if err != nil {
		return nil, fmt.Errorf("key not formatted properly: %w", err)
	}

	if cryptoType == types.CryptoSignature_SECP256 {
		key, err := crypto.ToECDSA(signingKeyBuf)
		if err != nil {
			return nil, fmt.Errorf("key not formatted properly: %w", err)
		}

		signature, err := crypto.Sign(digest, key)
		if err != nil {
			return nil, fmt.Errorf("failed to sign message: %v", err)
		}
		return signature, nil
	} else if cryptoType == types.CryptoSignature_BLS {
		suaveEthBlockSigningKey, err := bls.SecretKeyFromBytes(signingKeyBuf)
		if err != nil {
			fmt.Println("_B!", err)
			return nil, fmt.Errorf("failed to sign message: %v", err)
		}
		signature := bls.Sign(suaveEthBlockSigningKey, digest).Bytes()
		return signature[:], nil
	}

	return nil, fmt.Errorf("unsupported crypto type")
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

var contextCookieKeyPrefix = "__cookie_"

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

	url, err := s.resolveURL(request.Url)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(request.Method, url, body)
	if err != nil {
		return nil, err
	}

	// add any cookies stored in the context
	for key, val := range s.suaveContext.Context {
		if strings.HasPrefix(key, contextCookieKeyPrefix) {
			req.Header.Add("Cookie", string(val))
		}
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

	var timeout time.Duration
	if request.Timeout == 0 {
		timeout = 5 * time.Second
	} else {
		timeout = time.Duration(request.Timeout) * time.Millisecond
	}

	client := &http.Client{
		Timeout: timeout,
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

	// parse the LB cookies (AWSALB, AWSALBCORS) and set them in the context
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "AWSALB" || cookie.Name == "AWSALBCORS" {
			s.suaveContext.Context[contextCookieKeyPrefix+cookie.Name] = []byte(cookie.String())
		}
	}

	return data, nil
}

func (s *suaveRuntime) resolveURL(urlOrServiceName string) (string, error) {
	var allowed bool
	// resolve dns if possible
	if endpoint, ok := s.suaveContext.Backend.ServiceAliasRegistry[urlOrServiceName]; ok {
		return endpoint, nil
	}

	// decode the url and check if the domain is allowed
	parsedURL, err := url.Parse(urlOrServiceName)
	if err != nil {
		return "", err
	}

	// check if the domain is allowed
	for _, allowedDomain := range s.suaveContext.Backend.ExternalWhitelist {
		if allowedDomain == "*" || allowedDomain == parsedURL.Hostname() {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("domain %s is not allowed", parsedURL.Hostname())
	}

	return urlOrServiceName, nil
}

func (s *suaveRuntime) newBuilder() (string, error) {
	return s.suaveContext.Backend.ConfidentialEthBackend.NewSession(context.Background())
}

func (s *suaveRuntime) simulateTransaction(session string, txnBytes []byte) (types.SimulateTransactionResult, error) {
	txn := new(types.Transaction)
	if err := txn.UnmarshalBinary(txnBytes); err != nil {
		return types.SimulateTransactionResult{}, err
	}

	result, err := s.suaveContext.Backend.ConfidentialEthBackend.AddTransaction(context.Background(), session, txn)
	if err != nil {
		return types.SimulateTransactionResult{}, err
	}
	return *result, nil
}

func (s *suaveRuntime) contextGet(key string) ([]byte, error) {
	val, ok := s.suaveContext.Context[key]
	if !ok {
		return nil, fmt.Errorf("value not found")
	}
	return val, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *suaveRuntime) secp256k1Decrypt(privkey common.Hash, ciphertext []byte) ([]byte, error) {
	privKey := secp256k1.PrivKeyFromBytes(privkey.Bytes())

	// Read the sender's ephemeral public key from the start of the message.
	pubKeyLen := binary.LittleEndian.Uint32(ciphertext[:4])
	senderPubKeyBytes := ciphertext[4 : 4+pubKeyLen]
	senderPubKey, err := secp256k1.ParsePubKey(senderPubKeyBytes)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Derive the key used to seal the message, this time from the
	// recipient's private key and the sender's public key.
	recoveredCipherKey := sha256.Sum256(secp256k1.GenerateSharedSecret(privKey, senderPubKey))

	// Open the sealed message.
	aead, err := newAEAD(recoveredCipherKey[:])
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	recoveredPlaintext, err := aead.Open(nil, nonce, ciphertext[4+pubKeyLen:], senderPubKeyBytes)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return recoveredPlaintext, nil
}

func (s *suaveRuntime) secp256k1Encrypt(pubkey []byte, message []byte) ([]byte, error) {
	pubKey, err := secp256k1.ParsePubKey(pubkey)
	if err != nil {
		return nil, err
	}

	// Derive an ephemeral public/private keypair for performing ECDHE with
	// the recipient.
	ephemeralPrivKey, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	ephemeralPubKey := ephemeralPrivKey.PubKey().SerializeCompressed()

	// Using ECDHE, derive a shared symmetric key for encryption of the plaintext.
	cipherKey := sha256.Sum256(secp256k1.GenerateSharedSecret(ephemeralPrivKey, pubKey))

	// Seal the message using an AEAD.  Here we use AES-256-GCM.
	// The ephemeral public key must be included in this message, and becomes
	// the authenticated data for the AEAD.
	//
	// Note that unless a unique nonce can be guaranteed, the ephemeral
	// and/or shared keys must not be reused to encrypt different messages.
	// Doing so destroys the security of the scheme.  Random nonces may be
	// used if XChaCha20-Poly1305 is used instead, but the message must then
	// also encode the nonce (which we don't do here).
	//
	// Since a new ephemeral key is generated for every message ensuring there
	// is no key reuse and AES-GCM permits the nonce to be used as a counter,
	// the nonce is intentionally initialized to all zeros so it acts like the
	// first (and only) use of a counter.

	aead, err := newAEAD(cipherKey[:])
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	ciphertext := make([]byte, 4+len(ephemeralPubKey))
	binary.LittleEndian.PutUint32(ciphertext, uint32(len(ephemeralPubKey)))
	copy(ciphertext[4:], ephemeralPubKey)
	ciphertext = aead.Seal(ciphertext, nonce, message, ephemeralPubKey)

	return ciphertext, nil
}
