package offchain_test

import (
	"log"
	"os"
	"os/exec"
	"testing"

	"github.com/ethereum/go-ethereum/suave/offchain"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/stretchr/testify/require"
)

var api iface.CoreAPI

func TestMain(m *testing.M) {
	// Check if IPFS is available in the environment before attempting
	// to run the integration tests.
	cmd := exec.Command("which", "ipfs")
	switch err := cmd.Run().(type) {
	case *exec.ExitError:
		if status := err.ExitCode(); status > 0 {
			log.Println("ipfs not found in $PATH.  Skipping...")
		} else {
			os.Exit(status) // abort; we still don't know if IPFS is available
		}

	case error:
		log.Fatal(err)
	}

	os.Exit(m.Run())
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

}
