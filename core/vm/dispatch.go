package vm

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/mitchellh/mapstructure"
)

type DispatchTable struct {
	methods map[common.Address]*runtimeMethod

	// addrs stores an index of the addresses of the precompiled contracts
	addrs []common.Address
}

type runtimeMethod struct {
	name string
	addr common.Address

	method *abi.Method

	// sv is a reflect reference to the struct
	sv reflect.Value

	// fv is the reference to the run function
	fv reflect.Value

	// reqT is a list of input types for the run function
	// The first parameter is the pointer receiver type for the struct
	reqT []reflect.Type
}

func NewDispatchTable() *DispatchTable {
	return &DispatchTable{
		methods: make(map[common.Address]*runtimeMethod),
		addrs:   []common.Address{},
	}
}

// SuavePrecompiledContract is an optional interface for precompiled Suave contracts.
// During confidential execution the contract will be called with their RunConfidential method.
type SuavePrecompiledContract interface {
	RequiredGas(input []byte) uint64
	Address() common.Address
	Name() string
}

type SuavePrecompiledContractWrapper2 struct {
	ctx        *SuaveContext
	addr       common.Address
	dispatcher *DispatchTable
}

func (s *SuavePrecompiledContractWrapper2) RequiredGas(input []byte) uint64 {
	return 0
}

func (s *SuavePrecompiledContractWrapper2) Run(input []byte) ([]byte, error) {
	return s.dispatcher.Run(s.ctx, s.addr, input)
}

func (d *DispatchTable) Wrap(ctx *SuaveContext, addr common.Address) *SuavePrecompiledContractWrapper2 {
	return &SuavePrecompiledContractWrapper2{
		ctx:        ctx,
		addr:       addr,
		dispatcher: d,
	}
}

type PrecompileMethod struct {
	*abi.Method
	Addr common.Address
}

func (d *DispatchTable) GetAddrFromName(name string) (common.Address, bool) {
	for _, m := range d.methods {
		if m.name == name {
			return m.addr, true
		}
	}
	return common.Address{}, false
}

func (d *DispatchTable) GetMethods() []*PrecompileMethod {
	res := make([]*PrecompileMethod, 0, len(d.methods))
	for _, method := range d.methods {
		res = append(res, &PrecompileMethod{Method: method.method, Addr: method.addr})
	}
	return res
}

func (d *DispatchTable) IsPrecompile(addr common.Address) bool {
	for _, a := range d.addrs {
		if a == addr {
			return true
		}
	}
	return false
}

func (d *DispatchTable) packAndRun(suaveContext *SuaveContext, methodName string, args ...interface{}) ([]interface{}, error) {
	// find the method by name
	var method *runtimeMethod
	for _, m := range d.methods {
		if m.name == methodName {
			method = m
			break
		}
	}
	if method == nil {
		return nil, fmt.Errorf("runtime method %s not found", methodName)
	}

	// pack the input
	input, err := method.method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}

	// run the method
	output, err := d.Run(suaveContext, method.addr, input)
	if err != nil {
		return nil, err
	}

	// unpack the output
	outputs, err := method.method.Outputs.Unpack(output)
	if err != nil {
		return nil, err
	}

	return outputs, nil
}

func (d *DispatchTable) Run(suaveContext *SuaveContext, addr common.Address, input []byte) ([]byte, error) {
	method, ok := d.methods[addr]
	if !ok {
		return nil, fmt.Errorf("runtime method %s not found", addr)
	}

	log.Info("runtime.Handle", "name", method.name)

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

func (d *DispatchTable) Register(fn SuavePrecompiledContract) error {
	// reflect and generate the type of the 'Do' function
	typ := reflect.TypeOf(fn)

	var funcName string
	if fn, ok := fn.(PrecompileWithName); ok {
		funcName = fn.Name()
	} else {
		funcName = typ.Elem().Name()
	}

	if metrics.EnabledExpensive {
		metrics.GetOrRegisterMeter("suave/runtime/"+funcName, nil).Mark(1)

		now := time.Now()
		defer func() {
			metrics.GetOrRegisterTimer("suave/runtime/"+funcName+"/duration", nil).Update(time.Since(now))
		}()
	}

	methodName := "Do"
	methodTyp, found := typ.MethodByName(methodName)
	if !found {
		return fmt.Errorf("Method %s not found on the interface ('%s')", methodName, funcName)
	}

	// It needs at least one input parameter, the suave context.
	numIns := methodTyp.Type.NumIn()
	if numIns == 1 { // 1 parameter is the receiver
		return fmt.Errorf("Method %s must have at least one input parameter ('%s')", methodName, funcName)
	}
	if methodTyp.Type.In(1) != reflect.TypeOf(&SuaveContext{}) {
		return fmt.Errorf("First input parameter of method %s must be a *SuaveContext ('%s')", methodName, funcName)
	}

	// It needs at least one output parameter (the internal error) and must
	// be the last parameter
	numOuts := methodTyp.Type.NumOut()
	if numOuts == 0 {
		return fmt.Errorf("Method %s must have at least one output parameter ('%s')", methodName, funcName)
	}
	if !isErrorType(methodTyp.Type.Out(numOuts - 1)) {
		return fmt.Errorf("Last output parameter of method %s must be an error ('%s')", methodName, funcName)
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

	d.methods[fn.Address()] = &runtimeMethod{
		name:   funcName,
		method: &method,
		reqT:   inTypes,
		addr:   fn.Address(),
		sv:     reflect.ValueOf(fn),
		fv:     methodTyp.Func,
	}
	d.addrs = append(d.addrs, fn.Address())
	return nil
}

func (d *DispatchTable) MustRegister(fn SuavePrecompiledContract) {
	if err := d.Register(fn); err != nil {
		panic(err)
	}
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
