package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	suave_backends "github.com/ethereum/go-ethereum/suave/backends"
	suave "github.com/ethereum/go-ethereum/suave/core"
	"github.com/flashbots/go-boost-utils/bls"
	"github.com/naoina/toml"
	"github.com/urfave/cli/v2"
)

var defaultRemoteSuaveHost = "http://localhost:8545"

var (
	isLocalForgeFlag = &cli.BoolFlag{
		Name:  "local",
		Usage: `Whether to run the query command locally`,
	}
	whiteListForgeFlag = &cli.StringSliceFlag{
		Name:  "whitelist",
		Usage: `The whitelist external endpoints to call`,
	}
	dnsRegistryForgeFlag = &cli.StringSliceFlag{
		Name:  "dns-registry",
		Usage: `The DNS registry to resolve aliases to endpoints`,
	}
	ethBackendForgeFlag = &cli.StringFlag{
		Name:  "eth-backend",
		Usage: `The endpoint of the confidential eth backend`,
	}
	tomlConfigForgeFlag = &cli.StringFlag{
		Name:  "config",
		Usage: `The path to the forge toml config file`,
	}
)

type suaveForgeConfig struct {
	Whitelist   []string          `toml:"whitelist"`
	DnsRegistry map[string]string `toml:"dns_registry"`
	EthBackend  string            `toml:"eth_backend"`
}

func readContext(ctx *cli.Context) (*vm.SuaveContext, error) {
	// try to read the config from the toml config file
	cfg := &suaveForgeConfig{}

	if ctx.IsSet(tomlConfigForgeFlag.Name) {
		// read the toml config file
		data, err := os.ReadFile(ctx.String(tomlConfigForgeFlag.Name))
		if err != nil {
			return nil, err
		}

		// this is decoding
		// [profile.suave]
		var config struct {
			Profile struct {
				Suave *suaveForgeConfig
			}
		}

		tomlConfig := toml.DefaultConfig
		tomlConfig.MissingField = func(rt reflect.Type, field string) error {
			return nil
		}
		if err := tomlConfig.NewDecoder(bytes.NewReader(data)).Decode(&config); err != nil {
			return nil, err
		}
		cfg = config.Profile.Suave
	}

	// override the config if the flags are set
	if ctx.IsSet(ethBackendForgeFlag.Name) {
		cfg.EthBackend = ctx.String(ethBackendForgeFlag.Name)
	}
	if ctx.IsSet(whiteListForgeFlag.Name) {
		cfg.Whitelist = ctx.StringSlice(whiteListForgeFlag.Name)
	}
	if ctx.IsSet(dnsRegistryForgeFlag.Name) {
		dnsRegistry := make(map[string]string)
		for _, endpoint := range ctx.StringSlice(dnsRegistryForgeFlag.Name) {
			parts := strings.Split(endpoint, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid value for remote backend endpoint: %s", endpoint)
			}
			chainId := new(big.Int)
			if _, ok := chainId.SetString(parts[0], 10); !ok {
				return nil, fmt.Errorf("invalid chain id: %s", parts[0])
			}
			rpcUrl := parts[1]
			dnsRegistry[chainId.String()] = rpcUrl
		}
		cfg.DnsRegistry = dnsRegistry
	}

	// create the suave context
	var suaveEthBackend suave.ConfidentialEthBackend
	if cfg.EthBackend != "" {
		suaveEthBackend = suave_backends.NewRemoteEthBackend(cfg.EthBackend)
	} else {
		suaveEthBackend = &suave_backends.EthMock{}
	}

	ecdsaKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	blsKey, err := bls.GenerateRandomSecretKey()
	if err != nil {
		return nil, err
	}

	// NOTE: the confidential store precompiles are not enabled since they are stateless
	backend := &vm.SuaveExecutionBackend{
		ExternalWhitelist:      cfg.Whitelist,
		ConfidentialEthBackend: suaveEthBackend,
		EthBlockSigningKey:     blsKey,
		EthBundleSigningKey:    ecdsaKey,
	}
	suaveCtx := &vm.SuaveContext{
		Backend: backend,
	}
	return suaveCtx, nil
}

var (
	forgeCommand = &cli.Command{
		Name:        "forge",
		Usage:       "Internal command for MEVM forge commands",
		ArgsUsage:   "",
		Description: `Internal command used by MEVM precompiles in forge to access the MEVM API utilities.`,
		Flags: []cli.Flag{
			isLocalForgeFlag,
			whiteListForgeFlag,
			ethBackendForgeFlag,
			tomlConfigForgeFlag,
		},
		Subcommands: []*cli.Command{
			forgeStatusCmd,
			resetConfStore,
		},
		Action: func(ctx *cli.Context) error {
			args := ctx.Args()
			if args.Len() == 0 {
				return fmt.Errorf("expected at least 1 argument (address), got %d", args.Len())
			}

			// The first argument of the command is used to identify the precompile
			// contract to be called, it can either be:
			// 1. The address of the precompile
			// 2. The name of the precompile.
			addr := args.Get(0)
			if !strings.HasPrefix(addr, "0x") {
				mAddr, ok := artifacts.SuaveMethods[addr]
				if !ok {
					return fmt.Errorf("unknown precompile name '%s'", addr)
				}
				addr = mAddr.Hex()
			}

			inputStr := "0x"
			if args.Len() > 1 {
				inputStr = args.Get(1)
			}

			input, err := hexutil.Decode(inputStr)
			if err != nil {
				return fmt.Errorf("failed to decode input: %w", err)
			}

			if ctx.IsSet(isLocalForgeFlag.Name) {
				suaveCtx, err := readContext(ctx)
				if err != nil {
					return fmt.Errorf("failed to read context: %w", err)
				}

				result, err := vm.NewSuavePrecompiledContractWrapper(common.HexToAddress(addr), suaveCtx).Run(input)
				if err != nil {
					return fmt.Errorf("failed to run precompile: %w", err)
				}
				fmt.Println(hex.EncodeToString(result))
			} else {
				rpcClient, err := rpc.Dial(defaultRemoteSuaveHost)
				if err != nil {
					return fmt.Errorf("failed to dial rpc: %w", err)
				}

				ethClient := ethclient.NewClient(rpcClient)

				chainIdRaw, err := ethClient.ChainID(context.Background())
				if err != nil {
					return fmt.Errorf("failed to get chain id: %w", err)
				}

				chainId := hexutil.Big(*chainIdRaw)
				toAddr := common.HexToAddress(addr)

				callArgs := ethapi.TransactionArgs{
					To:             &toAddr,
					IsConfidential: true,
					ChainID:        &chainId,
					Data:           (*hexutil.Bytes)(&input),
				}
				var simResult hexutil.Bytes
				if err := rpcClient.Call(&simResult, "eth_call", setTxArgsDefaults(callArgs), "latest"); err != nil {
					return err
				}

				// return the result without the 0x prefix
				fmt.Println(simResult.String()[2:])
			}

			return nil
		},
	}
)

func setTxArgsDefaults(args ethapi.TransactionArgs) ethapi.TransactionArgs {
	gas := hexutil.Uint64(1000000)
	args.Gas = &gas

	nonce := hexutil.Uint64(0)
	args.Nonce = &nonce

	gasPrice := big.NewInt(1)
	args.GasPrice = (*hexutil.Big)(gasPrice)

	value := big.NewInt(0)
	args.Value = (*hexutil.Big)(value)

	return args
}

var forgeStatusCmd = &cli.Command{
	Name:        "status",
	Usage:       "Internal command to return whether the remote Suave node is enabled",
	ArgsUsage:   "",
	Description: `Internal command used by MEVM precompiles in forge to access the MEVM API utilities.`,
	Subcommands: []*cli.Command{},
	Action: func(ctx *cli.Context) error {
		handleErr := func(err error) error {
			fmt.Printf("not-ok: %s", err.Error())
			return nil
		}

		rpcClient, err := rpc.Dial(defaultRemoteSuaveHost)
		if err != nil {
			return handleErr(err)
		}

		// just make any random call for an endpoint that is always enabled
		var chainID hexutil.Big
		if err := rpcClient.Call(&chainID, "eth_chainId"); err != nil {
			return handleErr(err)
		}
		return nil
	},
}

var resetConfStore = &cli.Command{
	Name:  "reset-conf-store",
	Usage: "Internal command to reset the confidential store",
	Action: func(ctx *cli.Context) error {
		rpcClient, err := rpc.Dial(defaultRemoteSuaveHost)
		if err != nil {
			return err
		}
		if err := rpcClient.Call(nil, "suavey_resetConfStore"); err != nil {
			return err
		}
		return nil
	},
}
