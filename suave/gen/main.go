package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	goformat "go/format"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/yaml.v2"
)

var (
	formatFlag bool
	writeFlag  bool
)

func applyTemplate(templateText string, input desc, out string) error {
	// hash the content of the description
	raw, err := yaml.Marshal(input)
	if err != nil {
		return err
	}
	hash := crypto.Keccak256(raw)

	funcMap := template.FuncMap{
		"hash": func() string {
			return hex.EncodeToString(hash)
		},
		"typ2": func(param interface{}) string {
			return encodeTypeToGolang(param.(string), false, false)
		},
		"typ3": func(param interface{}) string {
			return encodeTypeToGolang(param.(string), true, true)
		},
		"title": func(param interface{}) string {
			return strings.Title(param.(string))
		},
		"isComplex": func(param interface{}) bool {
			_, err := abi.NewType(param.(string), "", nil)
			return err != nil
		},
		"encodeAddrName": func(param interface{}) string {
			return toAddressName(param.(string))
		},
		"styp": func(param interface{}) string {
			typName := param.(string)
			if isMemoryType(typName) && typName != "BidId" {
				return typName + " memory"
			}
			return typName
		},
	}

	t, err := template.New("template").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return err
	}

	var outputRaw bytes.Buffer
	if err = t.Execute(&outputRaw, input); err != nil {
		return err
	}

	// escape any quotes
	str := outputRaw.String()
	str = strings.Replace(str, "&#34;", "\"", -1)
	str = strings.Replace(str, "&amp;", "&", -1)
	str = strings.Replace(str, ", )", ")", -1)

	if formatFlag || writeFlag {
		// The output is always formatted if it is going to be written
		ext := filepath.Ext(out)
		if ext == ".go" {
			if str, err = formatGo(str); err != nil {
				return err
			}
		} else if ext == ".sol" {
			if str, err = formatSolidity(str); err != nil {
				return err
			}
		}
	}

	if err := outputFile(out, str); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.BoolVar(&formatFlag, "format", false, "format the output")
	flag.BoolVar(&writeFlag, "write", false, "write the output to the file")
	flag.Parse()

	data, err := os.ReadFile("./suave/gen/suave_spec.yaml")
	if err != nil {
		panic(err)
	}
	var ff desc
	if err := yaml.Unmarshal(data, &ff); err != nil {
		panic(err)
	}

	// sort the structs by name
	sort.Slice(ff.Structs, func(i, j int) bool {
		return ff.Structs[i].Name < ff.Structs[j].Name
	})

	// sort the methods by name
	sort.Slice(ff.Functions, func(i, j int) bool {
		return ff.Functions[i].Name < ff.Functions[j].Name
	})

	// Because of circular imports, we need to generate the structs first
	// in the types folder and then the stub in the vm folder.
	if err := applyTemplate(structsTemplate, ff, "./core/types/suave_structs.go"); err != nil {
		panic(err)
	}

	if err := applyTemplate(stubTemplate, ff, "./core/vm/contracts_suave_runtime_stub.go"); err != nil {
		panic(err)
	}

	if err := applyTemplate(suaveMethodsGoTemplate, ff, "./suave/artifacts/addresses.go"); err != nil {
		panic(err)
	}

	if err := applyTemplate(suaveLibTemplate, ff, "./suave/sol/libraries/Suave.sol"); err != nil {
		panic(err)
	}

	if err := generateABI("./suave/artifacts/Suave.json", ff); err != nil {
		panic(err)
	}
}

func encodeTypeToGolang(str string, insideTypes bool, slicePointers bool) string {
	typ, err := abi.NewType(str, "", nil)
	if err == nil {
		// basic type that has an easy match with Go
		if typ.T == abi.SliceTy {
			return "[]" + encodeTypeToGolang(typ.Elem.String(), insideTypes, slicePointers)
		}

		switch str {
		case "uint256":
			return "*big.Int"
		case "address":
			return "common.Address"
		case "bytes":
			return "[]byte"
		case "bytes32":
			return "common.Hash"
		case "bool":
			return "bool"
		case "string":
			return "string"
		}

		if strings.HasPrefix(str, "uint") {
			// uint8, uint16, uint32, uint64 are encoded the same way in Go
			return str
		}
		if strings.HasPrefix(str, "bytes") {
			// fixed bytesX are encoded as [X]byte
			return fmt.Sprintf("[%s]byte", strings.TrimPrefix(str, "bytes"))
		}
	} else {
		var ref string
		if !insideTypes {
			ref = "types."
		}

		// complex type with a struct. If it a slice (i.e. Struct[])
		// convert to []*Struct.
		if strings.HasSuffix(str, "[]") {
			if slicePointers {
				// This is a hack to keep compatibility with the old generated code
				return fmt.Sprintf("[]*%s%s", ref, strings.TrimSuffix(str, "[]"))
			} else {
				return fmt.Sprintf("[]%s%s", ref, strings.TrimSuffix(str, "[]"))
			}
		}
		return ref + str
	}

	panic(fmt.Sprintf("input not done for type: %s", str))
}

var structsTemplate = `
// Hash: {{hash}}
package types

import "github.com/ethereum/go-ethereum/common"

{{range .Types}}
type {{.Name}} {{typ3 .Typ}}
{{end}}

// Structs
{{range .Structs}}
type {{.Name}} struct {
	{{range .Fields}}{{title .Name}} {{typ3 .Typ}}
	{{end}}
}
{{end}}
`

var stubTemplate = `
// Hash: {{hash}}
package vm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/mitchellh/mapstructure"
)

var (
	errFailedToUnpackInput = fmt.Errorf("failed to decode input")
	errFailedToDecodeField = fmt.Errorf("failed to decode field")
	errFailedToPackOutput = fmt.Errorf("failed to encode output")
)

type BackendImpl interface {
	{{range .Functions}}
	{{.Name}}({{range .Input}}{{.Name}} {{typ2 .Typ}}, {{end}}) ({{range .Output.Fields}}{{typ2 .Typ}}, {{end}}error){{end}}
}

type BackendStub struct {
	impl BackendImpl
}

{{range .Functions}}
func (b *BackendStub) {{.Name}}(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["{{.Name}}"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		{{range .Input}}{{.Name}} {{typ2 .Typ}}
		{{end}})
	
	{{range $index, $item := .Input}}{{ if isComplex .Typ }}
	if err = mapstructure.Decode(unpacked[{{$index}}], &{{.Name}}); err != nil {
		err = errFailedToDecodeField
		return
	}
	{{else}}{{.Name}} = unpacked[{{$index}}].({{typ2 .Typ}}){{end}}
	{{end}}

	var (
		{{range .Output.Fields}}{{.Name}} {{typ2 .Typ}}
		{{end}})
	
	if {{range .Output.Fields}}{{.Name}},{{end}} err = b.impl.{{.Name}}({{range .Input}}{{.Name}}, {{end}}); err != nil {
		return
	}

	{{if .Output.None}}
	return nil, nil
	{{else if .Output.Plain}}
	result = {{range .Output.Fields}}{{.Name}} {{end}}
	return result, nil
	{{else}}
	result, err = artifacts.SuaveAbi.Methods["{{.Name}}"].Outputs.Pack({{range .Output.Fields}}{{.Name}}, {{end}})
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil
	{{end}}
}
{{end}}
`

var suaveMethodsGoTemplate = `
// Hash: {{hash}}
package artifacts

import (
	"github.com/ethereum/go-ethereum/common"
)

var SuaveMethods = map[string]common.Address{
{{range .Functions}}"{{.Name}}": common.HexToAddress("{{.Address}}"),
{{end}}}
`

var suaveLibTemplate = `pragma solidity ^0.8.8;

library Suave {
    error PeekerReverted(address, bytes);

{{range .Types}}
type {{.Name}} is {{.Typ}};
{{end}}

{{range .Structs}}
struct {{.Name}} {
	{{range .Fields}}{{.Typ}} {{.Name}};
	{{end}} }
{{end}}

address public constant IS_OFFCHAIN_ADDR =
0x0000000000000000000000000000000042010000;
{{range .Functions}}
address public constant {{encodeAddrName .Name}} =
{{.Address}};
{{end}}

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

{{range .Functions}}
function {{.Name}}({{range .Input}}{{styp .Typ}} {{.Name}}, {{end}}) internal view returns ({{range .Output.Fields}}{{styp .Typ}}, {{end}}) {
	(bool success, bytes memory data) = {{encodeAddrName .Name}}.staticcall(abi.encode({{range .Input}}{{.Name}}, {{end}}));
	if (!success) {
		revert PeekerReverted({{encodeAddrName .Name}}, data);
	}
	{{if .Output.None}}
	{{else if .Output.Plain}}
	return data;
	{{else}}
	return abi.decode(data, ({{range .Output.Fields}}{{.Typ}}, {{end}}));
	{{end}}
}
{{end}}

}
`

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
	Name string
	Typ  string `yaml:"type"`
}

type structsDef struct {
	Name   string
	Fields []typ
}

type desc struct {
	Types     []typ
	Structs   []structsDef
	Functions []functionDef
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

func formatGo(code string) (string, error) {
	srcFormatted, err := goformat.Source([]byte(code))
	if err != nil {
		return "", err
	}
	return string(srcFormatted), nil
}

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

func generateABI(out string, dd desc) error {
	abiEncode := []*abiField{}

	var encodeType func(name, typ string) arguments

	encodeType = func(name, typ string) arguments {
		arg := arguments{
			Name: name,
		}
		_, err := abi.NewType(typ, "", nil)
		if err == nil {
			// basic type
			arg.Type = typ
			arg.InternalType = typ
		} else {
			// struct type
			arg.InternalType = fmt.Sprintf("struct Suave.%s", typ)
			if strings.HasSuffix(typ, "[]") {
				arg.Type = "tuple[]"
				typ = strings.TrimSuffix(typ, "[]")
			} else {
				arg.Type = "tuple"
			}

			var subElem structsDef
			var found bool

			for _, f := range dd.Structs {
				if f.Name == typ {
					subElem = f
					found = true
					break
				}
			}
			if found {
				for _, ff := range subElem.Fields {
					arg.Components = append(arg.Components, encodeType(ff.Name, ff.Typ))
				}
			} else {
				// try to search as an alias
				for _, a := range dd.Types {
					if a.Name == typ {
						arg.Type = a.Typ
					}
				}
			}
		}

		return arg
	}

	for _, f := range dd.Functions {
		field := &abiField{
			Name:   f.Name,
			Type:   "function",
			Inputs: []arguments{},
		}

		for _, i := range f.Input {
			field.Inputs = append(field.Inputs, encodeType(i.Name, i.Typ))
		}
		for _, i := range f.Output.Fields {
			field.Outputs = append(field.Outputs, encodeType(i.Name, i.Typ))
		}

		abiEncode = append(abiEncode, field)
	}

	// marshal the object
	raw, err := json.Marshal(abiEncode)
	if err != nil {
		return err
	}

	// try to decode the output with abi.ABI to validate
	// that the result is correct
	if _, err := abi.JSON(bytes.NewReader(raw)); err != nil {
		return err
	}

	if err := outputFile(out, string(raw)); err != nil {
		return err
	}
	return nil
}

func outputFile(out string, str string) error {
	if !writeFlag {
		fmt.Println("=> " + out)
		fmt.Println(str)
	} else {
		fmt.Println("Write: " + out)
		// write file to output and create any parent directories if necessary
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(out, []byte(str), 0644); err != nil {
			return err
		}
	}
	return nil
}

func isMemoryType(s string) bool {
	typ, err := abi.NewType(s, "", nil)
	if err != nil {
		return true
	}
	// string, bytes, slices are types in memory
	return typ.T == abi.StringTy || typ.T == abi.BytesTy || typ.T == abi.SliceTy
}
