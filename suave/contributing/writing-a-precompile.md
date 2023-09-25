
# Writing a precompile

SUAVE uses custom precompiles to extend the EVM with specific MEV functions.

It uses a specification in `yaml` to describe each precompile with its inputs and outputs. From this specification, two bindings are auto-generated, one in Solidity (the client) and another one in Go (the server).

The Solidity binding, also known as `SuaveLib` is a Solidity library that Suave apps can use to call the precompiles. The Golang counterpart runs in the EVM and handles the Solidity calls to the SUAVE precompiles. It generates an skeleton interface that the developer needs to implement with the logic of the precompiles.

The bindings abstract from the developer any encoding/decoding and error management and provide an standard format for both runtimes to communicate with each other.

You can find the full `yaml` specification [here](../gen/suave_spec.yaml).

## Specification

This is the specification of the yaml file.

```yaml
types:
  - name: BidId
    type: bytes16
structs:
  - name: Bid
    fields:
      - name: id
        type: BidId
      - name: decryptionCondition
        type: uint64
...
functions:
  - name: confidentialInputs
    address: "0x0000000000000000000000000000000042010001"
    output:
      packed: true
      fields:
        - name: output1
          type: bytes
  - name: newBid
    address: "0x0000000000000000000000000000000042030000"
    input:
      - name: decryptionCondition
        type: uint64
      - name: allowedPeekers
        type: address[]
      - name: bidType
        type: string
    output:
      fields:
        - name: bid
          type: Bid
```

- types: List of user-defined value types:
    - Name: Name of the type.
    - Type: The basic type associated with the new alias in Solidity format (i.e. bytes16).
- Structs: List of user-defined structs:
    - Name: Name of the struct.
    - Fields: Array of fields for the struct.
        - Name (string): Name of the field.
        - Type (string): Type of the field.
            - It can be a basic Solidity type (address), a composite type (address[]), or a reference to any of the custom types and structs (i.e. Struct, Struct[]). It has to be written in the same format as it would be in Solidity.
- Functions: List of precompiles:
    - Name: Name of the precompile.
    - Address: Address of the precompile.
    - Input: Array of input fields for the precompile:
        - It follows the same rules as Structs.Fields.
    - Output: Configuration of the output.
        - Fields: Array of output fields for the precompile.
            - It follows the same rules as Structs.Fields.
        - Packed (bool): Whether to pack the output. Only available if it returns a single array of bytes.

## How to write one

Now, we are going to write a custom SUAVE precompile to perform the "add" operation.

First, modify the SUAVE precompile [specification](../gen/suave_spec.yaml) and add a new entry in the `functions` section:

````yaml
functions:
  - name: add
    address: "0x0000000000000000000000000000000042010009"
    input:
      - name: a
        type: uint64
      - name: b
        type: uint64
    output:
      fields:
        - name: output1
          type: uint64
````

Second, run the code generator:

```bash
$ go run suave/gen/main.go --write
```

If there are no errors and the `--write` flag is set, the bindings will be regenerated [here](../sol/libraries/Suave.sol) and [here](../../core/vm/contracts_suave_runtime_adapter.go).

In the Golang skeleton, a new `Add` function has been created:

```go
type SuaveRuntime interface {
    ...
    Add(a uint64, b uint64) (uint64, error)
    ...
}
```

The new function follows the same typing as defined in the SUAVE specification.

As it is right now, the system will error because we have not provided yet an implementation for the precompile. We do it in the `suaveRuntime` struct.

````go
func (b *suaveRuntime) Add(a uint64, b uint64) (uint64, error) {
    return a+b, nil
}
````
