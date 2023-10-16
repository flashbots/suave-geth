package vm

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

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
	isConfidentialAddress               = common.HexToAddress("0x42010000")
	errIsConfidentialInvalidInputLength = errors.New("invalid input length")

	confidentialInputsAddress = common.HexToAddress("0x42010001")

	confStoreStoreAddress    = common.HexToAddress("0x42020000")
	confStoreRetrieveAddress = common.HexToAddress("0x42020001")

	newBidAddress    = common.HexToAddress("0x42030000")
	fetchBidsAddress = common.HexToAddress("0x42030001")
)

/* General utility precompiles */

type yyyyyy struct {
	isConfidential bool
	suaveContext   *SuaveContext
}

func (y *yyyyyy) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (y *yyyyyy) Run(input []byte) ([]byte, error) {
	res, err := rrr.Handle(y.suaveContext, input)
	return res, err
}

var rrr *runtime

func init() {
	rrr = &runtime{}
	rrr.MustRegister(&isConfidentialPrecompile2{})
	rrr.MustRegister(&confidentialInputsPrecompile2{})
	rrr.MustRegister(&confStoreRetrieve{})
	rrr.MustRegister(&confStoreStore{})
	rrr.MustRegister(&extractHint{})
	rrr.MustRegister(&simulateBundle{})
	rrr.MustRegister(&newBid{})
	rrr.MustRegister(&fetchBids{})
	rrr.MustRegister(&buildEthBlock{})
}

var (
	trueB  = []byte{}
	falseB = []byte{}
)

type isConfidentialPrecompile2 struct{}

func (c *isConfidentialPrecompile2) Do(suaveContext *SuaveContext) (bool, error) {
	return true, nil
}

func (c *isConfidentialPrecompile2) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *isConfidentialPrecompile2) Name() string {
	return "isConfidential"
}

func (c *isConfidentialPrecompile2) Run(input []byte) ([]byte, error) {
	if len(input) == 1 {
		// The precompile was called *directly* confidentially, and the result was cached - return 1
		if input[0] == 0x01 {
			return trueB, nil
		} else {
			return nil, errors.New("incorrect value passed in")
		}
	}

	if len(input) > 1 {
		return nil, errIsConfidentialInvalidInputLength
	}

	return falseB, nil
}

func (c *isConfidentialPrecompile2) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	fmt.Println("ddd1", input)

	if len(input) != 0 {
		return nil, errIsConfidentialInvalidInputLength
	}
	return trueB, nil
}

type confidentialInputsPrecompile2 struct{}

func (c *confidentialInputsPrecompile2) Do(suaveContext *SuaveContext) ([]byte, error) {
	fmt.Println("__ TWO __")
	return suaveContext.ConfidentialInputs, nil
}

func (c *confidentialInputsPrecompile2) Name() string {
	return "confidentialInputs"
}

func (c *confidentialInputsPrecompile2) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *confidentialInputsPrecompile2) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confidentialInputsPrecompile2) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	inoutAbi := mustParseMethodAbi(`[{"outputs":[{"type":"bytes"}],"name":"store","inputs":[],"stateMutability":"nonpayable","type":"function"}]`, "store")

	res, err := inoutAbi.Outputs.Pack(suaveContext.ConfidentialInputs)
	if err != nil {
		panic(err)
	}

	fmt.Println("-- return byes in confidential ")
	fmt.Println(res, len(res), suaveContext.ConfidentialInputs)

	return res, nil
}

type isConfidentialPrecompile struct{}

func (c *isConfidentialPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *isConfidentialPrecompile) Run(input []byte) ([]byte, error) {
	if len(input) == 1 {
		// The precompile was called *directly* confidentially, and the result was cached - return 1
		if input[0] == 0x01 {
			return []byte{0x01}, nil
		} else {
			return nil, errors.New("incorrect value passed in")
		}
	}

	if len(input) > 1 {
		return nil, errIsConfidentialInvalidInputLength
	}

	return []byte{0x00}, nil
}

func (c *isConfidentialPrecompile) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	if len(input) != 0 {
		return nil, errIsConfidentialInvalidInputLength
	}
	return []byte{0x01}, nil
}

type confidentialInputsPrecompile struct{}

func (c *confidentialInputsPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

func (c *confidentialInputsPrecompile) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confidentialInputsPrecompile) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	return suaveContext.ConfidentialInputs, nil
}

/* Confidential store precompiles */

type confStoreStore struct {
	inoutAbi abi.Method
}

func newConfStoreStore() *confStoreStore {
	inoutAbi := mustParseMethodAbi(`[{"inputs":[{"type":"bytes16"}, {"type":"bytes16"}, {"type":"string"}, {"type":"bytes"}],"name":"store","outputs":[],"stateMutability":"nonpayable","type":"function"}]`, "store")

	return &confStoreStore{inoutAbi}
}

func (c *confStoreStore) RequiredGas(input []byte) uint64 {
	return uint64(100 * len(input))
}

func (c *confStoreStore) Name() string {
	return "confidentialStoreStore"
}

func (c *confStoreStore) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confStoreStore) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	if len(suaveContext.CallerStack) == 0 {
		return []byte("not allowed"), errors.New("not allowed in this suaveContext")
	}

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bidId := unpacked[0].(types.BidId)
	key := unpacked[1].(string)
	data := unpacked[2].([]byte)

	if err := c.runImpl(suaveContext, bidId, key, data); err != nil {
		return []byte(err.Error()), err
	}
	return nil, nil
}

func (c *confStoreStore) Do(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
	return c.runImpl(suaveContext, bidId, key, data)
}

func (c *confStoreStore) runImpl(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
	if len(suaveContext.CallerStack) == 0 {
		return errors.New("not allowed in this suaveContext")
	}

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(suaveContext.CallerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := suaveContext.CallerStack[i]; _c != nil && *_c != confStoreStoreAddress {
			caller = *_c
			break
		}
	}

	if metrics.Enabled {
		confStorePrecompileStoreMeter.Mark(int64(len(data)))
	}

	_, err := suaveContext.Backend.ConfidentialStore.Store(bidId, caller, key, data)
	if err != nil {
		return err
	}

	return nil
}

type confStoreRetrieve struct {
	inoutAbi abi.Method
}

func newConfStoreRetrieve() *confStoreRetrieve {
	inoutAbi := mustParseMethodAbi(`[{"inputs":[{"type":"bytes16"}, {"type":"bytes16"}, {"type":"string"}],"name":"retrieve","outputs":[{"type":"bytes"}],"stateMutability":"nonpayable","type":"function"}]`, "retrieve")

	return &confStoreRetrieve{inoutAbi}
}

func (c *confStoreRetrieve) RequiredGas(input []byte) uint64 {
	return 100
}

func (c *confStoreRetrieve) Run(input []byte) ([]byte, error) {
	return nil, errors.New("not available in this suaveContext")
}

func (c *confStoreRetrieve) Name() string {
	return "confidentialStoreRetrieve"
}

func (c *confStoreRetrieve) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	if len(suaveContext.CallerStack) == 0 {
		return []byte("not allowed"), errors.New("not allowed in this suaveContext")
	}

	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	bidId := unpacked[0].(suave.BidId)
	key := unpacked[1].(string)

	return c.runImpl(suaveContext, bidId, key)
}

func (c *confStoreRetrieve) Do(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
	return c.runImpl(suaveContext, bidId, key)
}

func (c *confStoreRetrieve) runImpl(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
	if len(suaveContext.CallerStack) == 0 {
		return nil, errors.New("not allowed in this suaveContext")
	}

	log.Info("confStoreRetrieve", "bidId", bidId, "key", key)

	// Can be zeroes in some fringe cases!
	var caller common.Address
	for i := len(suaveContext.CallerStack) - 1; i >= 0; i-- {
		// Most recent non-nil non-this caller
		if _c := suaveContext.CallerStack[i]; _c != nil && *_c != confStoreRetrieveAddress {
			caller = *_c
			break
		}
	}

	data, err := suaveContext.Backend.ConfidentialStore.Retrieve(bidId, caller, key)
	if err != nil {
		return []byte(err.Error()), err
	}

	if metrics.Enabled {
		confStorePrecompileRetrieveMeter.Mark(int64(len(data)))
	}

	return data, nil
}

/* Bid precompiles */

type newBid struct {
	inoutAbi abi.Method
}

func newNewBid() *newBid {
	inoutAbi := mustParseMethodAbi(`[{ "inputs": [ { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "string", "name": "BidType", "type": "string" } ], "name": "newBid", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "Suave.BidId", "name": "salt", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" } ], "internalType": "struct Suave.Bid", "name": "", "type": "tuple" } ], "stateMutability": "view", "type": "function" }]`, "newBid")

	return &newBid{inoutAbi}
}

func (c *newBid) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *newBid) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *newBid) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}
	version := unpacked[2].(string)

	decryptionCondition := unpacked[0].(uint64)
	allowedPeekers := unpacked[1].([]common.Address)

	bid, err := c.runImpl(suaveContext, version, decryptionCondition, allowedPeekers, []common.Address{})
	if err != nil {
		return []byte(err.Error()), err
	}

	return c.inoutAbi.Outputs.Pack(bid)
}

// TODO: The order of this one is changed.
func (c *newBid) Do(suaveContext *SuaveContext, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, version string) (*types.Bid, error) {
	return c.runImpl(suaveContext, version, decryptionCondition, allowedPeekers, allowedStores)
}

func (c *newBid) runImpl(suaveContext *SuaveContext, version string, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address) (*types.Bid, error) {
	if suaveContext.ConfidentialComputeRequestTx == nil {
		panic("newBid: source transaction not present")
	}

	bid, err := suaveContext.Backend.ConfidentialStore.InitializeBid(types.Bid{
		Salt:                suave.RandomBidId(),
		DecryptionCondition: decryptionCondition,
		AllowedPeekers:      allowedPeekers,
		AllowedStores:       allowedStores,
		Version:             version, // TODO : make generic
	})
	if err != nil {
		return nil, err
	}

	return &bid, nil
}

type fetchBids struct {
	inoutAbi abi.Method
}

func newFetchBids() *fetchBids {
	inoutAbi := mustParseMethodAbi(`[ { "inputs": [ { "internalType": "uint64", "name": "cond", "type": "uint64" }, { "internalType": "string", "name": "namespace", "type": "string" } ], "name": "fetchBids", "outputs": [ { "components": [ { "internalType": "Suave.BidId", "name": "id", "type": "bytes16" }, { "internalType": "Suave.BidId", "name": "salt", "type": "bytes16" }, { "internalType": "uint64", "name": "decryptionCondition", "type": "uint64" }, { "internalType": "address[]", "name": "allowedPeekers", "type": "address[]" }, { "internalType": "address[]", "name": "allowedStores", "type": "address[]" }, { "internalType": "string", "name": "version", "type": "string" } ], "internalType": "struct Suave.Bid[]", "name": "", "type": "tuple[]" } ], "stateMutability": "view", "type": "function" } ]`, "fetchBids")

	return &fetchBids{inoutAbi}
}

func (c *fetchBids) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *fetchBids) Run(input []byte) ([]byte, error) {
	return input, nil
}

func (c *fetchBids) Do(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
	return c.runImpl(suaveContext, targetBlock, namespace)
}

func (c *fetchBids) RunConfidential(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	unpacked, err := c.inoutAbi.Inputs.Unpack(input)
	if err != nil {
		return []byte(err.Error()), err
	}

	targetBlock := unpacked[0].(uint64)
	namespace := unpacked[1].(string)

	bids, err := c.runImpl(suaveContext, targetBlock, namespace)
	if err != nil {
		return []byte(err.Error()), err
	}

	return c.inoutAbi.Outputs.Pack(bids)
}

func (c *fetchBids) runImpl(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
	bids1 := suaveContext.Backend.ConfidentialStore.FetchBidsByProtocolAndBlock(targetBlock, namespace)

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

func (b *suaveRuntime) buildEthBlock(blockArgs types.BuildBlockArgs, bid types.BidId, namespace string) ([]byte, []byte, error) {
	return (&buildEthBlock{}).runImpl(b.suaveContext, blockArgs, bid, namespace)
}

func (b *suaveRuntime) confidentialInputs() ([]byte, error) {
	return nil, nil
}

func (b *suaveRuntime) confidentialStoreRetrieve(bidId types.BidId, key string) ([]byte, error) {
	return (&confStoreRetrieve{}).runImpl(b.suaveContext, bidId, key)
}

func (b *suaveRuntime) confidentialStoreStore(bidId types.BidId, key string, data []byte) error {
	return (&confStoreStore{}).runImpl(b.suaveContext, bidId, key, data)
}

func (b *suaveRuntime) extractHint(bundleData []byte) ([]byte, error) {
	return (&extractHint{}).runImpl(b.suaveContext, bundleData)
}

func (b *suaveRuntime) fetchBids(cond uint64, namespace string) ([]types.Bid, error) {
	bids, err := (&fetchBids{}).runImpl(b.suaveContext, cond, namespace)
	if err != nil {
		return nil, err
	}
	return bids, nil
}

func (b *suaveRuntime) newBid(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, BidType string) (types.Bid, error) {
	bid, err := (&newBid{}).runImpl(b.suaveContext, BidType, decryptionCondition, allowedPeekers, allowedStores)
	if err != nil {
		return types.Bid{}, err
	}
	return *bid, nil
}

func (b *suaveRuntime) simulateBundle(bundleData []byte) (uint64, error) {
	num, err := (&simulateBundle{}).runImpl(b.suaveContext, bundleData)
	if err != nil {
		return 0, err
	}
	return num.Uint64(), nil
}

func (b *suaveRuntime) submitEthBlockBidToRelay(relayUrl string, builderBid []byte) ([]byte, error) {
	return (&submitEthBlockBidToRelay{}).runImpl(b.suaveContext, relayUrl, builderBid)
}

// *-----

type runtimeMethod struct {
	name string

	method *abi.Method
	logic  SuavePrecompiledContract

	// sv is a reflect reference to the struct
	sv reflect.Value

	// fv is the reference to the run function
	fv reflect.Value

	// reqT is a list of input types for the run function
	// The first parameter is the pointer receiver type for the struct
	reqT []reflect.Type
}

type runtime struct {
	methods map[string]runtimeMethod
}

func (r *runtime) Handle(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	sig := hex.EncodeToString(input[:4])
	input = input[4:]

	method, ok := r.methods[sig]
	if !ok {
		log.Info("runtime.Handle", "sig", sig, "input", input)
		panic("failed to load method")
	}

	log.Info("runtime.Handle", "sig", sig, "name", method.name, "input", input)

	inNum := len(method.reqT)

	inArgs := make([]reflect.Value, inNum)
	inArgs[0] = method.sv
	inArgs[1] = reflect.ValueOf(suaveContext)

	if inNum != 2 {
		// decode the input parameters
		inputs, err := method.method.Inputs.Unpack(input)
		if err != nil {
			return nil, err
		}
		for i := 0; i < inNum-2; i++ {
			inArgs[i+2] = reflect.ValueOf(inputs[i])
		}
	}

	// make the execution call
	output := method.fv.Call(inArgs)
	if err := getError(output[len(output)-1]); err != nil {
		return nil, err
	}

	// encode the output as ABI
	paramOutput := make([]interface{}, len(output)-1)
	for i := 0; i < len(output)-1; i++ {
		paramOutput[i] = output[i].Interface()
	}

	outputBytes, err := method.method.Outputs.Pack(paramOutput...)
	if err != nil {
		return nil, err
	}
	return outputBytes, nil
}

type PrecompileWithName interface {
	Name() string
}

func (r *runtime) getByName(name string) runtimeMethod {
	for _, method := range r.methods {
		if method.name == name {
			return method
		}
	}
	panic("not found")
}

func (r *runtime) MustRegister(fn SuavePrecompiledContract) {
	if err := r.Register(fn); err != nil {
		panic(err)
	}
}

func (r *runtime) Register(fn SuavePrecompiledContract) error {
	// reflect and generate the type of the 'Do' function
	typ := reflect.TypeOf(fn)

	methodName := "Do"
	methodTyp, found := typ.MethodByName(methodName)
	if !found {
		return fmt.Errorf("Method %s not found on the interface\n", methodName)
	}

	var funcName string
	if fn, ok := fn.(PrecompileWithName); ok {
		funcName = fn.Name()
	} else {
		funcName = typ.Elem().Name()
	}

	// It needs at least one input parameter, the suave context.
	numIns := methodTyp.Type.NumIn()
	if numIns == 1 { // 1 parameter is the receiver
		return fmt.Errorf("Method %s must have at least one input parameter\n", methodName)
	}
	if methodTyp.Type.In(1) != reflect.TypeOf(&SuaveContext{}) {
		return fmt.Errorf("First input parameter of method %s must be a *SuaveContext\n", methodName)
	}

	// It needs at least one output parameter (the internal error) and must
	// be the last parameter
	numOuts := methodTyp.Type.NumOut()
	if numOuts == 0 {
		return fmt.Errorf("Method %s must have at least one output parameter\n", methodName)
	}
	if !isErrorType(methodTyp.Type.Out(numOuts - 1)) {
		return fmt.Errorf("Last output parameter of method %s must be an error\n", methodName)
	}

	// Get the input arguments of the function. The first parameter
	// is the pointer receiver for the struct
	inTypes := []reflect.Type{}
	for i := 0; i < numIns; i++ {
		inTypes = append(inTypes, methodTyp.Func.Type().In(i))
	}

	// Get the out arguments expect for the last error type
	outTypes := []reflect.Type{}
	for i := 0; i < numOuts-1; i++ {
		outTypes = append(outTypes, methodTyp.Func.Type().Out(i))
	}

	abiM := &abiField{
		Type:    "function",
		Name:    funcName,
		Inputs:  convertStructToABITypes(reflectStructFromTypes(inTypes[2:])),
		Outputs: convertStructToABITypes(reflectStructFromTypes(outTypes)),
	}

	raw, err := json.Marshal([]*abiField{abiM})
	if err != nil {
		return err
	}

	var abi abi.ABI
	if err := json.Unmarshal(raw, &abi); err != nil {
		return err
	}
	method, ok := abi.Methods[funcName]
	if !ok {
		return fmt.Errorf("Method %s not found on the abi", funcName)
	}

	log.Info("runtime registered", "name", funcName, "sig", method.Sig, "id", hex.EncodeToString(method.ID))

	if r.methods == nil {
		r.methods = make(map[string]runtimeMethod)
	}
	r.methods[hex.EncodeToString(method.ID)] = runtimeMethod{
		name:   funcName,
		method: &method,
		logic:  fn,
		reqT:   inTypes,
		sv:     reflect.ValueOf(fn),
		fv:     methodTyp.Func,
	}
	return nil
}

var errt = reflect.TypeOf((*error)(nil)).Elem()

func isErrorType(t reflect.Type) bool {
	return t.Implements(errt)
}

func isBytesTyp(t reflect.Type) bool {
	return (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) && t.Elem().Kind() == reflect.Uint8
}

type abiField struct {
	Type    string      `json:"type"`
	Name    string      `json:"name"`
	Inputs  []arguments `json:"inputs,omitempty"`
	Outputs []arguments `json:"outputs,omitempty"`
}

type arguments struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	InternalType string      `json:"internalType,omitempty"`
	Components   []arguments `json:"components,omitempty"`
	Indexed      bool        `json:"indexed,omitempty"`
}

func convertStructToABITypes(typ reflect.Type) []arguments {
	if typ.Kind() != reflect.Struct {
		panic("not a struct")
	}

	numFields := typ.NumField()
	fields := make([]arguments, numFields)

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)

		fields[i] = arguments{
			Name: field.Name,
		}

		var typeSuffix string
		subType := field.Type

	INFER:
		for {
			if isBytesTyp(subType) {
				// type []byte or [n]byte, it is decoded
				// as a simple type
				break INFER
			}

			switch subType.Kind() {
			case reflect.Slice:
				typeSuffix += "[]"
			case reflect.Array:
				typeSuffix += fmt.Sprintf("[%d]", subType.Len())
			case reflect.Ptr:
			default:
				break INFER
			}

			subType = subType.Elem()
		}

		if subType.Kind() == reflect.Struct {
			fields[i].Components = convertStructToABITypes(subType)
			fields[i].Type = "tuple" + typeSuffix
		} else {
			// parse basic type
			var basicType string
			switch subType.Kind() {
			case reflect.Bool:
				basicType = "bool"
			case reflect.Slice: // []byte
				basicType = "bytes"
			case reflect.Array: // [n]byte
				if subType.Len() == 20 {
					// TODO: we could improve this by checking if the type
					// is common.Address{}
					basicType = "address"
				} else {
					basicType = fmt.Sprintf("bytes%d", subType.Len())
				}
			case reflect.String:
				basicType = "string"
			case reflect.Uint64:
				basicType = "uint64"
			default:
				panic(fmt.Errorf("unknown type: %s", subType.Kind()))
			}
			fields[i].Type = basicType + typeSuffix
		}
	}

	return fields
}

func reflectStructFromTypes(argTypes []reflect.Type) reflect.Type {
	structFields := make([]reflect.StructField, len(argTypes))
	for i, argType := range argTypes {
		structFields[i] = reflect.StructField{
			Name: fmt.Sprintf("Param%d", i+1),
			Type: argType,
		}
	}

	return reflect.StructOf(structFields)
}

func getError(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}

	extractedErr, ok := v.Interface().(error)
	if !ok {
		return errors.New("invalid type assertion, unable to extract error")
	}

	return extractedErr
}
