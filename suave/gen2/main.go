package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"os/exec"
	"reflect"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
)

type function struct {
	name    string
	address common.Address
	input   []field
	output  output
}

type output struct {
	plain  bool
	none   bool
	fields []field
}

type field struct {
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

var structs []string

func generateSolidityStructFromInterface(iface interface{}, structName string) string {
	// check if the struct was generated already
	for _, s := range structs {
		if strings.Contains(s, structName) {
			return ""
		}
	}

	ifaceType := reflect.TypeOf(iface).Elem()
	structCode := fmt.Sprintf("struct %s {\n", structName)

	for i := 0; i < ifaceType.NumField(); i++ {
		field := ifaceType.Field(i)
		structCode += fmt.Sprintf("    %s %s;\n", goTypeToSolidityType(field.Type), strings.Title(field.Name))
	}

	structCode += "}\n"
	return structCode
}

func goTypeToSolidityType(goType reflect.Type) string {
	switch goType.Kind() {
	case reflect.Ptr:
		return goTypeToSolidityType(goType.Elem())
	case reflect.Struct:
		structName := goType.Name()
		structs = append(structs, generateSolidityStructFromInterface(reflect.New(goType).Interface(), structName))
		return structName

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return "uint256"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Slice:
		if goType.Elem().Kind() == reflect.Uint8 {
			return "bytes"
		}
		return goTypeToSolidityType(goType.Elem()) + "[]"
	case reflect.Array:
		if goType.Elem().Kind() == reflect.Uint8 {
			if goType.Len() == 20 {
				// address
				return "address"
			}
			return fmt.Sprintf("bytes%d", goType.Len())
		}
		return fmt.Sprintf("%s[%d]", goTypeToSolidityType(goType.Elem()), goType.Len())
	default:
		panic(fmt.Sprintf("unsupported type %s", goType))
	}
}

func isDynamicType(goType reflect.Type) bool {
	switch goType.Kind() {
	case reflect.Ptr:
		return isDynamicType(goType.Elem())
	case reflect.Struct:
		return true
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return false
	case reflect.String:
		return true
	case reflect.Bool:
		return false
	case reflect.Slice:
		return true
	case reflect.Array:
		return false
	default:
		panic(fmt.Sprintf("unsupported type %s", goType))
	}
}

type Withdrawal struct {
	Index     uint64
	Validator uint64
	Address   string
	Amount    uint64
}

type BuildBlockArgs struct {
	Slot           uint64
	ProposerPubkey []byte
	Parent         [32]byte
	Timestamp      uint64
	FeeRecipient   string
	GasLimit       uint64
	Random         [32]byte
	Withdrawals    []Withdrawal
}

type Bid struct {
	Amount uint64
	Price  uint64
}

func main() {
	var format bool

	flag.BoolVar(&format, "format", false, "format output")
	flag.Parse()

	ff := []function{
		{
			name:    "confidentialInputs",
			address: common.HexToAddress("0x0000000000000000000000000000000042010001"),
			output: output{
				plain: true,
			},
		},
		{
			name:    "newBid",
			address: common.HexToAddress("0x0000000000000000000000000000000042030000"),
			input: []field{
				{
					name: "decryptionCondition",
					typ:  uint64(0),
				},
				{
					name: "allowedPeekers",
					typ:  []common.Address{},
				},
				{
					name: "bidType",
					typ:  string(""),
				},
			},
			output: output{
				fields: []field{
					{
						name: "bid",
						typ:  &Bid{},
					},
				},
			},
		},
		{
			name:    "fetchBids",
			address: common.HexToAddress("0x0000000000000000000000000000000042030001"),
			input: []field{
				{
					name: "cond",
					typ:  uint64(0),
				},
				{
					name: "namespace",
					typ:  string(""),
				},
			},
			output: output{
				fields: []field{
					{
						name: "bid",
						typ:  []Bid{},
					},
				},
			},
		},
		{
			name:    "confidentialStoreStore",
			address: common.HexToAddress("0x0000000000000000000000000000000042020000"),
			input: []field{
				{
					name: "bidId",
					typ:  [16]byte{},
				},
				{
					name: "key",
					typ:  string(""),
				},
				{
					name: "data",
					typ:  []byte{},
				},
			},
			output: output{
				none: true,
			},
		},
		{
			name:    "confidentialStoreRetrieve",
			address: common.HexToAddress("0x0000000000000000000000000000000042020001"),
			input: []field{
				{
					name: "bidId",
					typ:  [16]byte{},
				},
				{
					name: "key",
					typ:  string(""),
				},
			},
			output: output{
				plain: true,
			},
		},
		{
			name:    "simulateBundle",
			address: common.HexToAddress("0x0000000000000000000000000000000042100000"),
			input: []field{
				{
					name: "bundleData",
					typ:  []byte{},
				},
			},
			output: output{
				fields: []field{
					{
						name: "output1",
						typ:  uint64(0),
					},
				},
			},
		},
		{
			name:    "extractHint",
			address: common.HexToAddress("0x0000000000000000000000000000000042100037"),
			input: []field{
				{
					name: "bundleData",
					typ:  []byte{},
				},
			},
			output: output{
				plain: true,
			},
		},
		{
			name:    "buildEthBlock",
			address: common.HexToAddress("0x0000000000000000000000000000000042100001"),
			input: []field{
				{
					name: "blockArgs",
					typ:  &BuildBlockArgs{},
				},
				{
					name: "bidId",
					typ:  [16]byte{},
				},
				{
					name: "namespace",
					typ:  string(""),
				},
			},
			output: output{
				fields: []field{
					{
						name: "output1",
						typ:  []byte{},
					},
					{
						name: "output2",
						typ:  []byte{},
					},
				},
			},
		},
		{
			name:    "submitEthBlockBidToRelay",
			address: common.HexToAddress("0x0000000000000000000000000000000042100002"),
			input: []field{
				{
					name: "relayUrl",
					typ:  string(""),
				},
				{
					name: "builderBid",
					typ:  []byte{},
				},
			},
			output: output{
				plain: true,
			},
		},
	}

	var (
		addresses []string
		functions []string
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
				inputs = append(inputs, fmt.Sprintf(`%s memory %s`, name, input.name))
			} else {
				var loc string
				if isDynamicType(reflect.TypeOf(input.typ)) {
					loc = "memory"
				}
				inputs = append(inputs, fmt.Sprintf(`%s %s %s`, goTypeToSolidityType(reflect.TypeOf(input.typ)), loc, input.name))
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
		outputTypesFunc := []string{}
		outputTypes := []string{}
		for _, output := range f.output.fields {
			var loc string
			if isDynamicType(reflect.TypeOf(output.typ)) {
				loc = " memory"
			}
			outputTypesFunc = append(outputTypesFunc, goTypeToSolidityType(reflect.TypeOf(output.typ))+loc)
			outputTypes = append(outputTypes, goTypeToSolidityType(reflect.TypeOf(output.typ)))
		}

		var output string
		if f.output.none {
			// do nothting
		} else if f.output.plain {
			// return the output
			outputTypesFunc = []string{"bytes memory"}
			output = `return data;`
		} else {
			output = fmt.Sprintf(`return abi.decode(data, (%s));`, strings.Join(outputTypes, ", "))
		}

		retFunc := ""
		if len(outputTypesFunc) > 0 {
			retFunc = fmt.Sprintf("returns (%s)", strings.Join(outputTypesFunc, ", "))
		}

		// list of functions
		functions = append(functions, fmt.Sprintf(`function %s(%s) internal view %s {
        %s
        %s
		%s
		}`, f.name, strings.Join(inputs, ", "), retFunc, encode, handlError, output))
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

	output := outputRaw.String()
	if format {
		output, err = formatSolidity(output)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println(output)
}

var templateText = `pragma solidity ^0.8.8;

library Suave {
    error PeekerReverted(address, bytes);

{{.Structs}}

address public constant IS_OFFCHAIN_ADDR =
0x0000000000000000000000000000000042010000;

{{.Addresses}}

// Returns whether execution is off- or on-chain
function isOffchain() internal view returns (bool b) {
	(bool success, bytes memory isOffchainBytes) = IS_OFFCHAIN_ADDR.staticcall("");
	if (!success) {
		revert PeekerReverted(IS_OFFCHAIN_ADDR, isOffchainBytes);
	}
	assembly {
		// Load the length of data (first 32 bytes)
		let len := mload(isOffchainBytes)
		// Load the data after 32 bytes, so add 0x20
		b := mload(add(isOffchainBytes, 0x20))
	}
}

{{.Functions}}
}
`

func formatSolidity(code string) (string, error) {
	// Check if "forge" command is available in PATH
	_, err := exec.LookPath("forge")
	if err != nil {
		return "", fmt.Errorf("forge command not found in PATH: %v", err)
	}

	// Command and arguments for forge fmt
	command := "forge"
	args := []string{"fmt", "--raw", "-"}

	// Create a command to run the forge fmt command
	cmd := exec.Command(command, args...)

	// Set up input from stdin
	cmd.Stdin = bytes.NewBufferString(code)

	// Set up output buffer
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	// Run the command
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error running command: %v", err)
	}

	// Print the formatted output
	fmt.Println("Formatted output:")
	fmt.Println(outBuf.String())

	return "", nil
}
