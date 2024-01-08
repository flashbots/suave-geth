package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/urfave/cli/v2"
)

var defaultRemoteSuaveHost = "http://localhost:8545"

var (
	forgeCommand = &cli.Command{
		Name:        "forge",
		Usage:       "Internal command for MEVM forge commands",
		ArgsUsage:   "",
		Description: `Internal command used by MEVM precompiles in forge to access the MEVM API utilities.`,
		Subcommands: []*cli.Command{
			forgeStatusCmd,
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
