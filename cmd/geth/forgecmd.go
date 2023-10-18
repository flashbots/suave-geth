package main

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/ethereum/go-ethereum/suave/sdk"
	"github.com/urfave/cli/v2"
)

var runtimeAddr = common.HexToAddress("0x1100000000000000000000000000000042100002")

var (
	forgeCommand = &cli.Command{
		Name:        "forge",
		Usage:       "Do things with forge",
		ArgsUsage:   "",
		Description: `Do things with forge.`,
		Action: func(ctx *cli.Context) error {
			arg := ctx.Args().First()

			buf, err := hexutil.Decode(arg)
			if err != nil {
				panic(err)
			}

			sig, input := buf[:4], buf[4:]

			// find the signature
			var method *abi.Method
			for _, target := range artifacts.SuaveAbi.Methods {
				if bytes.Equal(target.ID, sig) {
					method = &target
					break
				}
			}
			if method == nil {
				panic("could not find method")
			}

			rpcClient, err := rpc.Dial("http://localhost:8545")
			if err != nil {
				panic(err)
			}

			clt := sdk.NewClient(rpcClient, nil, common.Address{})
			contract := sdk.GetContract(runtimeAddr, artifacts.SuaveAbi, clt)
			fmt.Println(contract.CallRaw(method.Name, input))

			return nil
		},
	}
)
