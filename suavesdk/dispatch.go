package suavesdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mitchellh/mapstructure"
)

type DispatchTable struct {
	sv reflect.Value

	methods map[string]*runtimeMethod
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
		methods: map[string]*runtimeMethod{},
	}
}

// SuavePrecompiledContract is an optional interface for precompiled Suave contracts.
// During confidential execution the contract will be called with their RunConfidential method.
type SuavePrecompiledContract interface {
	RequiredGas(input []byte) uint64
	Address() common.Address
	Name() string
}

func (d *DispatchTable) GetMethods() []*abi.Method {
	methods := []*abi.Method{}
	for _, m := range d.methods {
		methods = append(methods, m.method)
	}
	return methods
}

func (d *DispatchTable) packAndRun(methodName string, args ...interface{}) ([]interface{}, error) {
	// find the method by name
	method, ok := d.methods[methodName]
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
	output, err := d.Run(input)
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

func (d *DispatchTable) Run(input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, fmt.Errorf("input data too short")
	}

	// find the method by signature
	var method *runtimeMethod

	sig := input[:4]
	for _, m := range d.methods {
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

func (d *DispatchTable) Register(service interface{}) error {
	if d.methods == nil {
		d.methods = map[string]*runtimeMethod{}
	}

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

		methodName := strings.ToLower(methodTyp.Name)

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

		d.methods[methodName] = &runtimeMethod{
			method: &methodABI,
			reqT:   inTypes,
			fv:     methodTyp.Func,
		}
	}

	return nil
}

func (d *DispatchTable) MustRegister(fn interface{}) {
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
