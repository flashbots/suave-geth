package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	abi "github.com/ethereum/go-ethereum/accounts/abi2"
)

type router struct {
	funcMap *funcData
}

type funcData struct {
	name string

	method *abi.Method

	// fv is the reference to the run function
	fv reflect.Value

	// reqT is a list of input types for the run function
	// The first parameter is the pointer receiver type for the struct
	reqT []reflect.Type
}

// Register registers a function to be exported in the precompile.
func (r *router) Register(fn interface{}) error {
	methodTyp := reflect.TypeOf(fn)
	if kind := methodTyp.Kind(); kind != reflect.Func {
		return fmt.Errorf("expected func, got %v", kind)
	}

	fv := reflect.ValueOf(fn)

	// decode the name of the function
	funcName := ""
	//funcName := runtime.FuncForPC(fv.Pointer()).Name()
	//funcName = funcName[strings.LastIndex(funcName, ".")+1:]
	//return nil

	numIns := methodTyp.NumIn()

	// It needs at least one output parameter (the internal error) and must
	// be the last parameter
	numOuts := methodTyp.NumOut()
	if numOuts == 0 {
		return fmt.Errorf("Method %s must have at least one output parameter", funcName)
	}
	if !isErrorType(methodTyp.Out(numOuts - 1)) {
		return fmt.Errorf("Last output parameter of method %s must be an error", funcName)
	}

	// Get the input arguments of the function. The first parameter
	// is the pointer receiver for the struct
	inTypes := []reflect.Type{}
	for i := 0; i < numIns; i++ {
		inTypes = append(inTypes, methodTyp.In(i))
	}

	// Get the out arguments expect for the last error type
	outTypes := []reflect.Type{}
	for i := 0; i < numOuts-1; i++ {
		outTypes = append(outTypes, methodTyp.Out(i))
	}

	abiM := &abiField{
		Type:    "function",
		Name:    funcName,
		Inputs:  convertStructToABITypes(reflectStructFromTypes(inTypes)),
		Outputs: convertStructToABITypes(reflectStructFromTypes(outTypes)),
	}

	raw, err := json.Marshal([]*abiField{abiM})
	if err != nil {
		return err
	}
	fmt.Println(raw)

	var abi abi.ABI
	if err := json.Unmarshal(raw, &abi); err != nil {
		return err
	}
	method, ok := abi.Methods[funcName]
	if !ok {
		return fmt.Errorf("Method %s not found on the abi", funcName)
	}

	fd := &funcData{
		name:   funcName,
		method: &method,
		reqT:   inTypes,
		fv:     fv,
	}

	r.funcMap = fd
	return nil
}

func (r *router) Run(input []byte) ([]byte, error) {
	method := r.funcMap

	inNum := len(method.reqT)

	inArgs := make([]reflect.Value, inNum)
	if inNum != 0 {
		// decode the input parameters
		inputs, err := method.method.Inputs.Unpack(input)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack input: %v", err)
		}

		for i := 0; i < inNum; i++ {
			inArgs[i] = reflect.ValueOf(inputs[i])
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
