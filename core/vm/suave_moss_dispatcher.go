package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mitchellh/mapstructure"
)

func (e *EVM) AddDispatchTable(r ...DispatchRuntime) {
	e.DispatchTable = NewDispatchTable()
	for _, rt := range r {
		e.DispatchTable.MustRegister(rt)
	}
}

type DispatchRuntimeFactory func() DispatchRuntime

type DispatchRuntime interface {
	Address() common.Address
}

type DispatchTable struct {
	sv reflect.Value

	methods map[common.Address]map[string]*runtimeMethod
}

type runtimeMethod struct {
	method *abi.Method

	// fv is the reference to the run function
	fv reflect.Value

	// reqT is a list of input types for the run function
	// The first parameter is the pointer receiver type for the struct
	reqT []reflect.Type
}

func NewDispatchTable() *DispatchTable {
	return &DispatchTable{
		methods: map[common.Address]map[string]*runtimeMethod{},
	}
}

type dispatchPrecompile struct {
	addr common.Address
	d    *DispatchTable
}

func (p *dispatchPrecompile) Run(input []byte) ([]byte, error) {
	return p.d.Run(p.addr, input)
}

func (p *dispatchPrecompile) RequiredGas([]byte) uint64 {
	return 0
}

func (d *DispatchTable) Contains(addr common.Address) bool {
	_, ok := d.methods[addr]
	return ok
}

func (d *DispatchTable) GetPrecompiled(addr common.Address) (PrecompiledContract, bool) {
	_, ok := d.methods[addr]
	if !ok {
		return nil, false
	}

	return &dispatchPrecompile{
		addr: addr,
		d:    d,
	}, true
}

func (d *DispatchTable) packAndRun(addr common.Address, methodName string, args ...interface{}) ([]interface{}, error) {
	methods, ok := d.methods[addr]
	if !ok {
		return nil, fmt.Errorf("runtime method %s not found", addr)
	}

	// find the method by name
	method, ok := methods[methodName]
	if !ok {
		return nil, fmt.Errorf("runtime method %s not found", methodName)
	}

	// pack the input
	input, err := method.method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	input = append(method.method.ID, input...)

	// run the method
	output, err := d.Run(addr, input)
	if err != nil {
		return nil, fmt.Errorf("runtime method %s failed: %v", methodName, err)
	}

	// unpack the output
	outputs, err := method.method.Outputs.Unpack(output)
	if err != nil {
		return nil, fmt.Errorf("runtime output decode %s failed: %v", methodName, err)
	}

	return outputs, nil
}

func (d *DispatchTable) Run(addr common.Address, input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, fmt.Errorf("input data too short")
	}

	methods, ok := d.methods[addr]
	if !ok {
		return nil, fmt.Errorf("runtime method not found")
	}

	// find the method by signature
	var method *runtimeMethod

	sig := input[:4]
	for _, m := range methods {
		if bytes.Equal(m.method.ID, sig) {
			method = m
			break
		}
	}

	input = input[4:]

	if method == nil {
		return nil, fmt.Errorf("runtime method not found")
	}

	log.Info("runtime.Handle", "name", method.method.Name)

	inNum := len(method.reqT)

	inArgs := make([]reflect.Value, inNum)
	inArgs[0] = d.sv

	if inNum != 1 {
		// decode the input parameters
		inputs, err := method.method.Inputs.Unpack(input)
		if err != nil {
			return nil, err
		}

		for i := 0; i < inNum-1; i++ {
			if typ := method.reqT[i+1]; typ.Kind() == reflect.Struct {
				val := reflect.New(typ)
				if err = mapstructure.Decode(inputs[i], val.Interface()); err != nil {
					return nil, err
				}
				inArgs[i+1] = val.Elem()
			} else {
				inArgs[i+1] = reflect.ValueOf(inputs[i])
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

func (d *DispatchTable) Register(service DispatchRuntime) error {
	methods := map[string]*runtimeMethod{}

	st := reflect.TypeOf(service)
	if st.Kind() == reflect.Struct {
		return errors.New("jsonrpc: service must be a pointer to struct")
	}

	d.sv = reflect.ValueOf(service)
	for i := 0; i < st.NumMethod(); i++ {
		methodTyp := st.Method(i)
		if methodTyp.PkgPath != "" {
			// skip unexported methods
			continue
		}
		if methodTyp.Name == "Address" {
			continue
		}

		methodName := firstToLower(methodTyp.Name)

		// It needs at least one input parameter, the suave context.
		numIns := methodTyp.Type.NumIn()

		// It needs at least one output parameter (the internal error) and must
		// be the last parameter
		numOuts := methodTyp.Type.NumOut()
		if numOuts == 0 {
			return fmt.Errorf("method %s must have at least one output parameter", methodName)
		}
		if !isErrorType(methodTyp.Type.Out(numOuts - 1)) {
			return fmt.Errorf("last output parameter of method %s must be an error", methodName)
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
			Name:    methodName,
			Inputs:  convertStructToABITypes(reflectStructFromTypes(inTypes[1:])),
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

		methodABI, ok := abi.Methods[methodName]
		if !ok {
			return fmt.Errorf("method %s not found on the abi", methodName)
		}

		fmt.Println("registering method, name " + methodName + " sig " + methodABI.Sig)

		methods[methodName] = &runtimeMethod{
			method: &methodABI,
			reqT:   inTypes,
			fv:     methodTyp.Func,
		}
	}
	d.methods[service.Address()] = methods

	return nil
}

func firstToLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:]
}

func (d *DispatchTable) MustRegister(fn DispatchRuntime) {
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
