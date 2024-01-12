package offchain_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/suave/offchain"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/stretchr/testify/require"
)

var api iface.CoreAPI

func TestMain(m *testing.M) {
	// Check if IPFS is available in the environment before attempting
	// to run the integration tests.
	cmd := exec.Command("xxx", "version")
	switch err := cmd.Run().(type) {
	case nil:
		os.Exit(m.Run())
	case *exec.ExitError:
		// Application error
		// skip; let the CI pipeline continue

	case *exec.Error:
		// OS error
		// skip; let the CI pipeline contineu

	case error:
		fmt.Println(reflect.TypeOf(err))
		log.Fatal(err)
	}
}

func TestBlockstore(t *testing.T) {
	t.Parallel()

	env := offchain.Env{
		IPFS: api,
	}
	require.NoError(t, env.Start(), "failed to bind offchain environment")
	defer func() {
		require.NoError(t, env.Stop(), "failed to release offchain environment")
	}()

	data := []byte("hello, world")

	cid, err := env.Datastore().Put(context.TODO(), bytes.NewReader(data))
	require.NoError(t, err)
	require.NotZero(t, cid)

	r, err := env.Datastore().Get(context.TODO(), cid)
	require.NoError(t, err)
	require.NotNil(t, r)

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NotNil(t, b)

	require.Equal(t, data, b)
}
