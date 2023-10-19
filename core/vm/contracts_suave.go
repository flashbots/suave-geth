package vm

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/mitchellh/mapstructure"
)

var (
	confStorePrecompileStoreMeter    = metrics.NewRegisteredMeter("suave/confstore/store", nil)
	confStorePrecompileRetrieveMeter = metrics.NewRegisteredMeter("suave/confstore/retrieve", nil)
)

type suaveRuntimePrecompile struct {
	suaveContext *SuaveContext
}

func (y *suaveRuntimePrecompile) RequiredGas(input []byte) uint64 {
	return suaveRuntime.RequiredGas(input)
}

func (y *suaveRuntimePrecompile) Run(input []byte) ([]byte, error) {
	return suaveRuntime.Run(y.suaveContext, input)
}

var suaveRuntime *Runtime

func GetRuntime() *Runtime {
	return suaveRuntime
}

func init() {
	suaveRuntime = &Runtime{}
	suaveRuntime.MustRegister(&isConfidentialPrecompile2{})
	suaveRuntime.MustRegister(&confidentialInputsPrecompile{})
	suaveRuntime.MustRegister(&confStoreRetrieve{})
	suaveRuntime.MustRegister(&confStoreStore{})
	suaveRuntime.MustRegister(&extractHint{})
	suaveRuntime.MustRegister(&simulateBundle{})
	suaveRuntime.MustRegister(&newBid{})
	suaveRuntime.MustRegister(&fetchBids{})
	suaveRuntime.MustRegister(&buildEthBlock{})
	suaveRuntime.MustRegister(&submitEthBlockBidToRelay{})
}

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

type confidentialInputsPrecompile struct{}

func (c *confidentialInputsPrecompile) Do(suaveContext *SuaveContext) ([]byte, error) {
	return suaveContext.ConfidentialInputs, nil
}

func (c *confidentialInputsPrecompile) Name() string {
	return "confidentialInputs"
}

func (c *confidentialInputsPrecompile) RequiredGas(input []byte) uint64 {
	return 0 // incurs only the call cost (100)
}

/* Confidential store precompiles */

type confStoreStore struct {
}

func (c *confStoreStore) RequiredGas(input []byte) uint64 {
	return uint64(100 * len(input))
}

func (c *confStoreStore) Name() string {
	return "confidentialStoreStore"
}

func (c *confStoreStore) Do(suaveContext *SuaveContext, bidId suave.BidId, key string, data []byte) error {
	if len(suaveContext.CallerStack) == 0 {
		return errors.New("not allowed in this suaveContext")
	}

	caller := suaveContext.getCaller()

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
}

func (c *confStoreRetrieve) RequiredGas(input []byte) uint64 {
	return 100
}

func (c *confStoreRetrieve) Name() string {
	return "confidentialStoreRetrieve"
}

func (c *confStoreRetrieve) Do(suaveContext *SuaveContext, bidId suave.BidId, key string) ([]byte, error) {
	if len(suaveContext.CallerStack) == 0 {
		return nil, errors.New("not allowed in this suaveContext")
	}

	log.Info("confStoreRetrieve", "bidId", bidId, "key", key)

	caller := suaveContext.getCaller()

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
}

func (c *newBid) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *newBid) Do(suaveContext *SuaveContext, decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, version string) (*types.Bid, error) {
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
}

func (c *fetchBids) RequiredGas(input []byte) uint64 {
	return 1000
}

func (c *fetchBids) Do(suaveContext *SuaveContext, targetBlock uint64, namespace string) ([]types.Bid, error) {
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

func formatPeekerError(format string, args ...any) ([]byte, error) {
	err := fmt.Errorf(format, args...)
	return []byte(err.Error()), err
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

type Runtime struct {
	methods map[string]runtimeMethod
}

func (r *Runtime) GetMethods() []*abi.Method {
	res := make([]*abi.Method, 0, len(r.methods))
	for _, method := range r.methods {
		res = append(res, method.method)
	}
	return res
}

func (r *Runtime) RequiredGas(input []byte) uint64 {
	sig := hex.EncodeToString(input[:4])
	input = input[4:]

	method, ok := r.methods[sig]
	if !ok {
		return 0
	}

	return method.logic.RequiredGas(input)
}

func (r *Runtime) Run(suaveContext *SuaveContext, input []byte) ([]byte, error) {
	sig := hex.EncodeToString(input[:4])
	input = input[4:]

	method, ok := r.methods[sig]
	if !ok {
		return nil, fmt.Errorf("runtime method %s not found", sig)
	}

	log.Info("runtime.Handle", "sig", sig, "name", method.name)

	if metrics.EnabledExpensive {
		metrics.GetOrRegisterMeter("suave/runtime/"+method.name, nil).Mark(1)

		now := time.Now()
		defer func() {
			metrics.GetOrRegisterTimer("suave/runtime/"+method.name+"/duration", nil).Update(time.Since(now))
		}()
	}

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
			if typ := method.reqT[i+2]; typ.Kind() == reflect.Struct {
				val := reflect.New(typ)
				if err = mapstructure.Decode(inputs[i], val.Interface()); err != nil {
					return nil, err
				}
				inArgs[i+2] = val.Elem()
			} else {
				inArgs[i+2] = reflect.ValueOf(inputs[i])
			}
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

func (r *Runtime) MustRegister(fn SuavePrecompiledContract) {
	if err := r.Register(fn); err != nil {
		panic(err)
	}
}

// SuavePrecompiledContract is an optional interface for precompiled Suave contracts.
// During confidential execution the contract will be called with their RunConfidential method.
type SuavePrecompiledContract interface {
	RequiredGas(input []byte) uint64
}

func (r *Runtime) Register(fn SuavePrecompiledContract) error {
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

	log.Debug("runtime registered", "name", funcName, "sig", method.Sig, "id", hex.EncodeToString(method.ID))

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

		nameTyp := subType.Name()
		if nameTyp != "" && subType.Name() != subType.Kind().String() {
			// skip basic types for Address and Hash since those are native
			if nameTyp != "Address" && nameTyp != "Hash" {
				fields[i].InternalType = nameTyp
			}
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
