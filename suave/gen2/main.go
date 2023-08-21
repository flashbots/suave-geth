package main

import (
	"bytes"
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
)

type function struct {
	name    string
	address common.Address
	input   []input
}

type input struct {
	name string
	typ  interface{}
}

func toAddressName(input string) string {
	var result strings.Builder
	upperPrev := true

	for _, r := range input {
		if unicode.IsUpper(r) && !upperPrev {
			result.WriteString("_")
		}
		result.WriteRune(unicode.ToUpper(r))
		upperPrev = unicode.IsUpper(r)
	}

	return result.String()
}

func generateSolidityStructFromInterface(iface interface{}, structName string) string {
	ifaceType := reflect.TypeOf(iface).Elem()
	structCode := fmt.Sprintf("struct %s {\n", structName)

	for i := 0; i < ifaceType.NumField(); i++ {
		field := ifaceType.Field(i)
		structCode += fmt.Sprintf("    %s %s;\n", strings.Title(field.Name), goTypeToSolidityType(field.Type))
	}

	structCode += "}\n"
	return structCode
}

func goTypeToSolidityType(goType reflect.Type) string {
	switch goType.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return "uint256"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	default:
		return ""
	}
}

type Bid struct {
	Amount uint64
	Price  uint64
}

func main() {
	ff := []function{
		{
			name:    "confidentialInputs",
			address: common.HexToAddress("0x0000000000000000000000000000000042010001"),
			input: []input{
				{
					name: "bid",
					typ:  &Bid{},
				},
				{
					name: "bidder",
					typ:  true,
				},
			},
		},
	}

	var (
		addresses []string
		functions []string
		structs   []string
	)

	for _, f := range ff {
		// list of addresses
		addresses = append(addresses, fmt.Sprintf(`    address public constant %s =
    	%s;
		`, toAddressName(f.name), f.address.Hex()))

		// struct types
		inputs := []string{}
		inputNames := []string{}

		for _, input := range f.input {
			inputNames = append(inputNames, input.name)

			// if it's a struct, generate a struct
			if reflect.TypeOf(input.typ).Kind() == reflect.Ptr {
				name := reflect.TypeOf(input.typ).Elem().Name()
				structs = append(structs, generateSolidityStructFromInterface(input.typ, name))
				inputs = append(inputs, fmt.Sprintf(`%s %s`, input.name, name))
			} else {
				inputs = append(inputs, fmt.Sprintf(`%s %s`, input.name, goTypeToSolidityType(reflect.TypeOf(input.typ))))
			}
		}

		// body of the function. It has three stages:
		// 1. encode input and call the contract
		encode := fmt.Sprintf(`(bool success, bytes memory data) = %s.staticcall(
            abi.encode(%s)
        );`, toAddressName(f.name), strings.Join(inputNames, ", "))

		// 2. handle error
		handlError := fmt.Sprintf(`if (!success) {
            revert PeekerReverted(%s, data);
        }`, toAddressName(f.name))

		// 3. decode output (if output type is defined) and return

		// list of functions
		functions = append(functions, fmt.Sprintf(`function %s(%s) internal view {
        %s
        %s
		}`, f.name, strings.Join(inputs, ", "), encode, handlError))
	}

	t, err := template.New("template").Parse(templateText)
	if err != nil {
		panic(err)
	}

	input := map[string]interface{}{
		"Addresses": strings.Join(addresses, "\n"),
		"Functions": strings.Join(functions, "\n"),
		"Structs":   strings.Join(structs, "\n"),
	}

	var outputRaw bytes.Buffer
	if err = t.Execute(&outputRaw, input); err != nil {
		panic(err)
	}

	fmt.Println(outputRaw.String())
}

var templateText = `pragma solidity ^0.8.8;

library Suave {
    error PeekerReverted(address, bytes);

{{.Structs}}

{{.Addresses}}

{{.Functions}}
}
`
