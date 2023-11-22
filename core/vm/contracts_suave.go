package vm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	betterAbi "github.com/umbracle/ethgo/abi"

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

func formatPeekerError(format string, args ...any) ([]byte, error) {
	err := fmt.Errorf(format, args...)
	return []byte(err.Error()), err
}

type suaveRuntime struct {
	suaveContext *SuaveContext
}

var _ SuaveRuntime = &suaveRuntime{}

func (s *suaveRuntime) httpGet(url string, config types.HttpConfig) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// decode headers
	for _, header := range config.Headers {
		prts := strings.Split(header, ":")
		if len(prts) != 2 {
			return nil, fmt.Errorf("incorrect header format")
		}
		req.Header.Add(prts[0], prts[1])
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *suaveRuntime) httpPost(url string, body []byte, config types.HttpConfig) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	fmt.Println("__ POST __", url, string(body))

	// decode headers
	for _, header := range config.Headers {
		prts := strings.Split(header, ":")
		if len(prts) != 2 {
			return nil, fmt.Errorf("incorrect header format")
		}
		req.Header.Add(prts[0], prts[1])
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println("__ POST RESPONSE __", string(data))

	return data, nil
}

func convertStringJsonToAbiSpec(a string) string {
	var f map[string]interface{}
	if err := json.Unmarshal([]byte(a), &f); err != nil {
		panic(err)
	}
	return convertInterfaceToAbiSpec(f)
}

func convertInterfaceToAbiSpec(i interface{}) string {
	switch x := i.(type) {
	case map[string]interface{}:
		elems := []struct {
			name string
			typ  string
		}{}

		for k, v := range x {
			elems = append(elems, struct {
				name string
				typ  string
			}{
				name: k,
				typ:  convertInterfaceToAbiSpec(v),
			})
		}
		// sort
		sort.Slice(elems, func(i, j int) bool {
			return elems[i].name < elems[j].name
		})

		finElems := []string{}
		for _, elem := range elems {
			finElems = append(finElems, elem.typ+" "+elem.name)
		}
		return "tuple(" + strings.Join(finElems, ", ") + ")"

	case []interface{}:
		return convertInterfaceToAbiSpec(x[0]) + "[]"

	case string:
		return "string"

	case float64:
		return "uint256"

	default:
		panic(fmt.Sprintf("unknown type %T", x))
	}
}

func (s *suaveRuntime) jsonUnmarshal(a string) ([]byte, error) {
	// Parse the json file as an abstract map
	var f map[string]interface{}
	if err := json.Unmarshal([]byte(a), &f); err != nil {
		return nil, err
	}

	xx := convertInterfaceToAbiSpec(f)

	fmt.Println("- abi spec 1 -")
	fmt.Println(xx)

	typ, err := betterAbi.NewType(xx)
	if err != nil {
		return nil, err
	}

	result, err := typ.Encode(f)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *suaveRuntime) jsonMarshal(abispec string, obj []byte) ([]byte, error) {
	fmt.Println("-- obj --")
	fmt.Println(obj)
	fmt.Println("- abi spec 2 -")
	fmt.Println(abispec)

	typ, err := betterAbi.NewType(abispec)
	if err != nil {
		return nil, err
	}
	fmt.Println(typ)

	xx, err := typ.Decode(obj)
	if err != nil {
		return nil, err
	}

	fmt.Println("-- decoded --")
	fmt.Println(xx)

	yy, err := json.Marshal(xx)
	if err != nil {
		return nil, err
	}

	fmt.Println("-- final --")
	fmt.Println(string(yy))

	return yy, nil
}

func (s *suaveRuntime) simpleConsole(b []byte) error {
	fmt.Println("_ SIMPLE CONSOLE _")

	fmt.Println(b)
	fmt.Println(string(b))

	return nil
}
