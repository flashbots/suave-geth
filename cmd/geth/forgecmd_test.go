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

	// read context from non-existent config file
	ctx.Set("config", "./testdata/forge_not_exists.toml")

	_, err := readContext(ctx)
	require.Error(t, err)

	// read context from valid config toml file WITHOUT suave section
	// it should fallback to the default values
	ctx.Set("config", "./testdata/forge_noconfig.toml")

	sCtx, err := readContext(ctx)
	require.NoError(t, err)

	require.Len(t, sCtx.Backend.ExternalWhitelist, 0)
	require.Len(t, sCtx.Backend.DnsRegistry, 0)

	// read context from config toml file
	ctx.Set("config", "./testdata/forge.toml")

	sCtx, err = readContext(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, sCtx.Backend.ExternalWhitelist)
	require.Equal(t, map[string]string{"a": "b", "c": "d"}, sCtx.Backend.ServiceAliasRegistry)
	require.Equal(t, "suave", sCtx.Backend.ConfidentialEthBackend.(*suave_backends.RemoteEthBackend).Endpoint())

	// override the config if the flags are set
	ctx.Set("eth-backend", "http://localhost:8545")
	ctx.Set("whitelist", "c,d")
	ctx.Set("service-alias", "e=f,g=h")

	sCtx, err = readContext(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"c", "d"}, sCtx.Backend.ExternalWhitelist)
	require.Equal(t, map[string]string{"e": "f", "g": "h"}, sCtx.Backend.ServiceAliasRegistry)
	require.Equal(t, "http://localhost:8545", sCtx.Backend.ConfidentialEthBackend.(*suave_backends.RemoteEthBackend).Endpoint())

	// set flags to null and use default values
	ctx = cli.NewContext(nil, flagSet(t, forgeCommand.Flags), nil)

	sCtx, err = readContext(ctx)
	require.NoError(t, err)
	require.Len(t, sCtx.Backend.ExternalWhitelist, 0)

	_, ok := sCtx.Backend.ConfidentialEthBackend.(*suave_backends.EthMock)
	require.True(t, ok)
}
