package main

import (
	"flag"
	"io"
	"testing"

	suave_backends "github.com/ethereum/go-ethereum/suave/backends"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func flagSet(t *testing.T, flags []cli.Flag) *flag.FlagSet {
	set := flag.NewFlagSet("test", flag.ContinueOnError)

	for _, f := range flags {
		if err := f.Apply(set); err != nil {
			t.Fatal(err)
		}
	}
	set.SetOutput(io.Discard)
	return set
}

func TestForgeReadConfig(t *testing.T) {
	t.Parallel()

	ctx := cli.NewContext(nil, flagSet(t, forgeCommand.Flags), nil)

	// read context from config toml file
	ctx.Set("config", "./testdata/forge.toml")

	sCtx, err := readContext(ctx)
	require.NoError(t, err)
	require.Equal(t, sCtx.Backend.ExternalWhitelist, []string{"a", "b"})
	require.Equal(t, sCtx.Backend.DnsRegistry, map[string]string{"a": "b", "c": "d"})
	require.Equal(t, sCtx.Backend.ConfidentialEthBackend.(*suave_backends.RemoteEthBackend).Endpoint(), "suave")

	// override the config if the flags are set
	ctx.Set("eth-backend", "http://localhost:8545")
	ctx.Set("whitelist", "c,d")
	ctx.Set("dns-registry", "e=f,g=h")

	sCtx, err = readContext(ctx)
	require.NoError(t, err)
	require.Equal(t, sCtx.Backend.ExternalWhitelist, []string{"c", "d"})
	require.Equal(t, sCtx.Backend.DnsRegistry, map[string]string{"e": "f", "g": "h"})
	require.Equal(t, sCtx.Backend.ConfidentialEthBackend.(*suave_backends.RemoteEthBackend).Endpoint(), "http://localhost:8545")

	// set flags to null and use default values
	ctx = cli.NewContext(nil, flagSet(t, forgeCommand.Flags), nil)

	sCtx, err = readContext(ctx)
	require.NoError(t, err)
	require.Len(t, sCtx.Backend.ExternalWhitelist, 0)

	_, ok := sCtx.Backend.ConfidentialEthBackend.(*suave_backends.EthMock)
	require.True(t, ok)
}
