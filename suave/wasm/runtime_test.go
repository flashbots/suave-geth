package wasm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeXXX(t *testing.T) {
	r, err := NewRuntime()
	require.NoError(t, err)

	fmt.Println(r)
}
