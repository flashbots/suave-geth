package abi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFuzz_GenerateRandomTypes(t *testing.T) {
	// use the packUnpackTests to make sure the type generator
	// can generate structs/objects that the abi can pack
	for _, test := range packUnpackTests {
		inDef := fmt.Sprintf(`[{ "name" : "method", "type": "function", "inputs": %s}]`, test.def)

		inAbi, err := JSON(strings.NewReader(inDef))
		require.NoError(t, err)

		// generate random types for the method
		inputs := GenerateRandomTypeForMethod(inAbi.Methods["method"])

		// pack the inputs into ABI
		_, err = inAbi.Methods["method"].Inputs.Pack(inputs...)
		require.NoError(t, err)
	}
}
