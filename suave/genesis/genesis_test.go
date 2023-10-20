package genesis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	_, err := Load("test")
	require.NoError(t, err)

	_, err = Load("./fixtures/genesis-test.json")
	require.NoError(t, err)

	_, err = Load("not-exists.json")
	require.Error(t, err)

	_, err = Load("not-exists")
	require.Error(t, err)
}
