package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"html/template"
	"os"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/suave/artifacts"
)

type fieldInput struct {
	Name string
	Typ  abi.Type
}

type methodElem struct {
	Name            string
	Inputs          []fieldInput
	Outputs         []fieldInput
	IsComplexOutput bool
}

type structElem struct {
	Name   string
	Fields []fieldInput
}

type generator struct {
	Methods []*methodElem
	Structs []*structElem
	Hash    string
}

func (g *generator) encodeMethod(abiMethod abi.Method) {
	mtd := &methodElem{
		Name:   abiMethod.Name,
		Inputs: []fieldInput{},
	}

	for _, input := range abiMethod.Inputs {
		g.findStructs(input.Type)

		mtd.Inputs = append(mtd.Inputs, fieldInput{
			Name: input.Name,
			Typ:  input.Type,
		})
	}

	for _, output := range abiMethod.Outputs {
		g.findStructs(output.Type)

		mtd.Outputs = append(mtd.Outputs, fieldInput{
			Name: output.Name,
			Typ:  output.Type,
		})
	}

	g.Methods = append(g.Methods, mtd)
}

func (g *generator) findStructs(typ abi.Type) {
	// there can only be structs either in a slice/array or
	// as sub-elements of another struct
	if typ.T == abi.TupleTy {
		name := typ.TupleRawName
		name = strings.TrimPrefix(name, "Suave")

		// do not encode twice the same struct
		for _, st := range g.Structs {
			if st.Name == name {
				return
			}
		}

		elem := &structElem{
			Name: name,
		}
		for indx, field := range typ.TupleElems {
			elem.Fields = append(elem.Fields, fieldInput{
				Name: strings.Title(typ.TupleRawNames[indx]),
				Typ:  *field,
			})
			// search for structs inside the struct itself
			g.findStructs(*field)
		}
		g.Structs = append(g.Structs, elem)
	} else if typ.T == abi.SliceTy || typ.T == abi.ArrayTy {
		g.findStructs(*typ.Elem)
	}
}

func encodeType(typ abi.Type, insideTypes, slicePointers bool) string {
	// if the type is a tuple, encode as a struct
	if typ.T == abi.TupleTy {
		name := typ.TupleRawName
		name = strings.TrimPrefix(name, "Suave")

		if insideTypes {
			return name
		}
		return "types." + name
	}

	// if the type is an array, return "[]" + the type of the element
	if typ.T == abi.SliceTy {
		if typ.Elem.T == abi.TupleTy && slicePointers {
			// slice of struct pointers
			return "[]*" + encodeType(*typ.Elem, insideTypes, slicePointers)
		}
		return "[]" + encodeType(*typ.Elem, insideTypes, slicePointers)
	}

	// otherwise, it is a basic element
	switch typ.T {
	case abi.UintTy:
		return "uint64"
	case abi.BytesTy:
		return "[]byte"
	case abi.StringTy:
		return "string"
	case abi.FixedBytesTy:
		if typ.Size == 16 {
			return fmt.Sprintf("[%d]byte", typ.Size)
		} else if typ.Size == 32 {
			return "common.Hash"
		}
	case abi.AddressTy:
		return "common.Address"
	}

	panic(fmt.Sprintf("input not done for type: %s", typ.String()))
}

func applyTemplate(templateText string, input interface{}, out string) error {
	funcMap := template.FuncMap{
		"ityp": func(param interface{}) string {
			return encodeType(param.(abi.Type), true, true)
		},
		"typ": func(param interface{}) string {
			return encodeType(param.(abi.Type), false, false)
		},
		"title": func(param interface{}) string {
			return strings.Title(param.(string))
		},
		"encodeArgs": func(params interface{}) string {
			method := params.(*methodElem)

			var (
				inputs  []string
				outputs []string
			)

			if len(method.Inputs) != 0 {
				for _, input := range method.Inputs {
					inputs = append(inputs, fmt.Sprintf("%s %s", input.Name, encodeType(input.Typ, false, false)))
				}
			}
			if len(method.Outputs) != 0 {
				for indx, output := range method.Outputs {
					outputs = append(outputs, fmt.Sprintf("resp%d %s", indx, encodeType(output.Typ, false, false)))
				}
			}
			outputs = append(outputs, "err error")

			return fmt.Sprintf("(%s) (%s)", strings.Join(inputs, ", "), strings.Join(outputs, ", "))
		},
		"encodeClient": func(params interface{}) string {
			method := params.(*methodElem)

			var (
				body   []string
				inputs []string
			)

			for _, input := range method.Inputs {
				inputs = append(inputs, input.Name)
			}

			addr, ok := addresses[method.Name]
			if !ok {
				panic(fmt.Sprintf("address not found for method: %s", method.Name))
			}

			body = append(body, fmt.Sprintf(`var resp []interface{}
			var ok bool

			if resp, err = c.call("%s", %s, []interface{}{%s}); err != nil {
				err = fmt.Errorf("failed to make rpc request: %%v", err)
				return
			}`, addr, fmt.Sprintf(`"%s"`, method.Name), strings.Join(inputs, ", ")))

			// if there are no outputs, return nil
			body = append(body, `_ = resp
			_ = ok`)

			// decode outputs
			if len(method.Outputs) != 0 {
				for indx, output := range method.Outputs {
					if output.Typ.T == abi.TupleTy || output.Typ.T == abi.SliceTy && output.Typ.Elem.T == abi.TupleTy {
						// if it is a complex type with structs, use mapstructure
						body = append(body, fmt.Sprintf(`if err = mapstructure.Decode(resp[%d], &resp%d); err != nil {
						return
					}`, indx, indx))
					} else {
						// use an interface deconstructor
						body = append(body, fmt.Sprintf(`resp%d, ok = resp[%d].(%s)
					if !ok {
						err = fmt.Errorf("failed to decode argument %d")
						return
					}`, indx, indx, encodeType(output.Typ, false, false), indx))
					}
				}
			}

			body = append(body, "return")
			return strings.Join(body, "\n\n")
		},
		"encodeStub": func(params interface{}) string {
			method := params.(*methodElem)
			str := []string{
				"var err error",
			}

			var inputs []string

			// [Step 1]: Unpack if:
			// 1. If there are more than two input items.
			// 2. There is one input and the type is not bytes.
			// 3. The 'extractHint' function is a specific case which takes []input and also unpacks it. TODO: Fix.
			if len(method.Inputs) >= 2 || (len(method.Inputs) == 1 && method.Inputs[0].Typ.T != abi.BytesTy) || method.Name == "extractHint" {
				str = append(str, fmt.Sprintf(`unpacked, err := artifacts.SuaveAbi.Methods["%s"].Inputs.Unpack(input)
				if err != nil {
					return nil, err
				}`, method.Name))

				// If it unpacked, we have to deserialize the elements
				for indx, input := range method.Inputs {
					// if some of the types are struct, we have to use mapstructure to unpack it
					if input.Typ.T == abi.TupleTy {
						str = append(str, fmt.Sprintf(`var %s %s
						if err := mapstructure.Decode(unpacked[%d], &%s); err != nil {
							return nil, err
						}`, input.Name, encodeType(input.Typ, false, false), indx, input.Name))

						inputs = append(inputs, input.Name)
					} else {
						inputs = append(inputs, fmt.Sprintf(`unpacked[%d].(%s)`, indx, encodeType(input.Typ, false, false)))
					}
				}
			} else {
				// The input to the backend is the []byte input itself
				inputs = []string{"input"}
			}

			outputs := []string{}
			for indx := range method.Outputs {
				outputs = append(outputs, fmt.Sprintf("res%d", indx))
			}
			outputs = append(outputs, "err") // all the backend emit an error

			// [Step 2]: Declare the output variables with their types.
			if len(method.Outputs) != 0 {
				outputDecl := []string{}
				for indx, output := range method.Outputs {
					outputDecl = append(outputDecl, fmt.Sprintf(`res%d %s`, indx, encodeType(output.Typ, false, false)))
				}
				str = append(str, `var (
					`+strings.Join(outputDecl, "\n")+`
				)`)
			}

			// [Step 3]: Call the backend
			str = append(str, fmt.Sprintf(`if %s = b.impl.%s(%s); err != nil {
				return nil, err
			}`, strings.Join(outputs, ", "), method.Name, strings.Join(inputs, ", ")))

			// [Step 4]: Pack and return
			if len(method.Outputs) >= 2 || (len(method.Outputs) == 1 && method.Outputs[0].Typ.T != abi.BytesTy) {
				// Pack if:
				// 1. There are two or more items.
				// 2. There is one item and the type is not bytes.
				str = append(str, fmt.Sprintf(`packedRes, err := artifacts.SuaveAbi.Methods["%s"].Outputs.Pack(%s)
				if err != nil {
					return nil, err
				}
				return packedRes, nil`, method.Name, strings.Join(outputs[:len(outputs)-1], ", ")))
			} else if len(method.Outputs) == 1 {
				// Only one output which is of type []byte, return it
				str = append(str, fmt.Sprintf(`return %s, nil`, outputs[0]))
			} else {
				// The backend does not have output, return nil
				str = append(str, "return nil, nil")
			}

			return strings.Join(str, "\n\n")
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

	//fmt.Println(str)

	srcFormatted, err := format.Source([]byte(str))
	if err != nil {
		return err
	}

	if err := os.WriteFile(out, srcFormatted, 0755); err != nil {
		return err
	}
	return nil
}

func main() {
	g := &generator{
		Methods: []*methodElem{},
		Structs: []*structElem{},
	}

	for _, abiMethod := range artifacts.SuaveAbi.Methods {
		g.encodeMethod(abiMethod)
	}

	// compute the hash as the keccak of the json ABI
	// and add it to the generator
	abiBytes, err := json.Marshal(artifacts.SuaveAbi)
	if err != nil {
		panic(err)
	}
	g.Hash = fmt.Sprintf("%x", crypto.Keccak256(abiBytes))

	// sort both methods and structs by name to generate
	// a deterministic output
	sort.Slice(g.Methods, func(i, j int) bool {
		return g.Methods[i].Name < g.Methods[j].Name
	})
	sort.Slice(g.Structs, func(i, j int) bool {
		return g.Structs[i].Name < g.Structs[j].Name
	})

	// Because of circular imports, we need to generate the structs first
	// in the types folder and then the stub in the vm folder.
	if err := applyTemplate(structsTemplate, g, "./core/types/suave_structs.go"); err != nil {
		panic(err)
	}

	if err := applyTemplate(stubTemplate, g, "./core/vm/contracts_suave_runtime_stub.go"); err != nil {
		panic(err)
	}

	if err := applyTemplate(clientTemplate, g, "./suave/gen/examples/client.go"); err != nil {
		panic(err)
	}
}

var structsTemplate = `// Code generated by suave/gen. DO NOT EDIT.
// Hash: {{.Hash}}
package types

import "github.com/ethereum/go-ethereum/common"

// Structs
{{range .Structs}}
type {{.Name}} struct {
	{{range .Fields}}{{.Name}} {{ityp .Typ}}
	{{end}}
}
{{end}}
`

var stubTemplate = `// Code generated by suave/gen. DO NOT EDIT.
// Hash: {{.Hash}}
package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/mitchellh/mapstructure"
)

type BackendImpl interface {
	{{range .Methods}}
	{{.Name}}({{range .Inputs}}{{.Name}} {{typ .Typ}},{{end}}) ({{range .Outputs}} {{typ .Typ}}, {{end}} error){{end}}
}

type BackendStub struct {
	impl BackendImpl
}

{{range .Methods}}
func (b *BackendStub) {{.Name}}(input []byte) ([]byte, error) {
	{{encodeStub .}}
}
{{end}}
`

var clientTemplate = `// Hash: {{.Hash}}
package examples

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/mitchellh/mapstructure"
)

type Client struct {
	rpc *rpc.Client
}

func NewClient(rpc *rpc.Client) *Client {
	return &Client{rpc: rpc}
}

func (c *Client) call(addr string, methodName string, args []interface{}) ([]interface{}, error) {
	method := artifacts.SuaveAbi.Methods[methodName]

	input, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}

	addrD := common.HexToAddress(addr)
	msg := ethapi.TransactionArgs{
		To:         &addrD,
		IsOffchain: true,
		Data:       (*hexutil.Bytes)(&input),
	}

	var respBytes hexutil.Bytes
	if err := c.rpc.Call(&respBytes, "eth_call", msg, "latest"); err != nil {
		return nil, err
	}

	resp, err := method.Outputs.Unpack(respBytes)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

{{range .Methods}}
func (c *Client) {{title .Name}} {{encodeArgs .}} {
	{{encodeClient .}}
}
{{end}}
`

var addresses = map[string]common.Address{
	"buildEthBlock":             common.HexToAddress("0x0000000000000000000000000000000042100001"),
	"confidentialStoreRetrieve": common.HexToAddress("0x0000000000000000000000000000000042020001"),
	"confidentialStoreStore":    common.HexToAddress("0x0000000000000000000000000000000042020000"),
	"extractHint":               common.HexToAddress("0x0000000000000000000000000000000042100037"),
	"fetchBids":                 common.HexToAddress("0x0000000000000000000000000000000042030001"),
	"newBid":                    common.HexToAddress("0x0000000000000000000000000000000042030000"),
	"simulateBundle":            common.HexToAddress("0x0000000000000000000000000000000042100000"),
	"submitEthBlockBidToRelay":  common.HexToAddress("0x0000000000000000000000000000000042100002"),
}
