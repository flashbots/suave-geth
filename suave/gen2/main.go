package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v2"
)

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

type functionDef struct {
	Name       string
	Address    string
	Input      []field
	Output     output
	IsOffchain bool `yaml:"isOffchain"`
}

type output struct {
	Plain  bool
	None   bool
	Fields []field
}

type field struct {
	Name string
	Typ  string `yaml:"type"`
}

type typ struct {
	Name    string
	TypName string `yaml:"type"`
}

type structsDef struct {
	Name   string
	Fields []typ
}

type desc struct {
	Structs   []structsDef
	Functions []functionDef
}

func isMemoryType(s string) bool {
	typ, err := abi.NewType(s, "", nil)
	if err != nil {
		return true
	}
	// string, bytes, slices are types in memory
	return typ.T == abi.StringTy || typ.T == abi.BytesTy || typ.T == abi.SliceTy
}

func main() {
	var format bool

	flag.BoolVar(&format, "format", false, "format output")
	flag.Parse()

	var ff desc

	data, err := os.ReadFile("./suave/gen2/suave.yaml")
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(data, &ff); err != nil {
		panic(err)
	}

	var (
		addresses []string
		functions []string
		structs   []string
	)

	for _, ss := range ff.Structs {
		structRes := []string{
			fmt.Sprintf("struct %s {\n", ss.Name),
		}
		for _, f := range ss.Fields {
			structRes = append(structRes, fmt.Sprintf("    %s %s;\n", f.TypName, f.Name))
		}
		structRes = append(structRes, "}\n")
		structs = append(structs, strings.Join(structRes, "\n"))
	}

	for _, f := range ff.Functions {
		// list of addresses
		addr := common.HexToAddress(f.Address)
		addresses = append(addresses, fmt.Sprintf(`address public constant %s = %s;`, toAddressName(f.Name), addr.Hex()))

		// struct types
		inputs := []string{}
		inputNames := []string{}

		for _, input := range f.Input {
			inputNames = append(inputNames, input.Name)

			var loc string
			if isMemoryType(input.Typ) {
				loc = "memory"
			}

			inputs = append(inputs, fmt.Sprintf(`%s %s %s`, input.Typ, loc, input.Name))
		}

		// body of the function. It has three stages:
		// 1. encode input and call the contract
		encode := fmt.Sprintf(`(bool success, bytes memory data) = %s.staticcall(
            abi.encode(%s)
        );`, toAddressName(f.Name), strings.Join(inputNames, ", "))

		// 2. handle error
		handlError := fmt.Sprintf(`if (!success) {
            revert PeekerReverted(%s, data);
        }`, toAddressName(f.Name))

		// 3. decode output (if output type is defined) and return
		outputTypes := []string{}
		for _, output := range f.Output.Fields {
			outputTypes = append(outputTypes, output.Typ)
		}

		var output string
		if f.Output.None {
			// do nothting
		} else if f.Output.Plain {
			outputTypes = []string{"bytes"}

			// return the output
			output = `return data;`
		} else {
			output = fmt.Sprintf(`return abi.decode(data, (%s));`, strings.Join(outputTypes, ", "))
		}

		retFunc := ""
		if len(outputTypes) != 0 {
			// same as 'outputTypes' with with 'memory' if they are dynamic
			outputTypesWithLoc := []string{}
			for _, output := range outputTypes {
				var loc string
				if isMemoryType(output) {
					loc = "memory"
				}
				outputTypesWithLoc = append(outputTypesWithLoc, fmt.Sprintf("%s %s", output, loc))
			}
			retFunc = fmt.Sprintf("returns (%s)", strings.Join(outputTypesWithLoc, ", "))
		}

		var isOffchain string
		if f.IsOffchain {
			isOffchain = `require(isOffchain());`
		}

		// list of functions
		functions = append(functions, fmt.Sprintf(`function %s(%s) internal view %s {
		%s
        %s
        %s
		%s
		}`, f.Name, strings.Join(inputs, ", "), retFunc, isOffchain, encode, handlError, output))
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
	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("error running command: %v", err)
	}

	return outBuf.String(), nil
}
