package castcmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	ethgoabi "github.com/umbracle/ethgo/abi"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/suave/sdk"

	"github.com/urfave/cli/v2"
)

var (
	devchainKettleAddress = common.HexToAddress("0xB5fEAfbDD752ad52Afb7e1bD2E40432A485bBB7F")
	devchainPrivateKey    = "91ab9a7e53c220e6210460b65a7a3bb2ca181412a8a7b43ff336b3df1737ce12"
)

var (
	rpcFlag = &cli.StringFlag{
		Name:  "rpc",
		Usage: `The rpc endpoint to use`,
		Value: "http://localhost:8545",
	}
	kettleAddressFlag = &cli.StringFlag{
		Name:  "kettle-address",
		Usage: `The address of the kettle contract`,
	}
	privateKeyFlag = &cli.StringFlag{
		Name:  "private-key",
		Usage: `The private key to use for signing the confidential request`,
	}
)

var (
	Cmd = &cli.Command{
		Name:        "cast",
		Usage:       "Send a confidential request to a contract",
		Description: "Send a confidential request to a contract",
		Flags: []cli.Flag{
			kettleAddressFlag,
			privateKeyFlag,
			rpcFlag,
		},
		Action: func(ctx *cli.Context) error {
			rpcClient, err := rpc.Dial(ctx.String(rpcFlag.Name))
			if err != nil {
				return err
			}

			var kettleAddress common.Address
			if ctx.IsSet(kettleAddressFlag.Name) {
				kettleAddress = common.HexToAddress(ctx.String(kettleAddressFlag.Name))
			} else {
				// get the kettle address from the rpc endpoint
				var addrs []common.Address
				if err := rpcClient.Call(&addrs, "eth_kettleAddress"); err != nil {
					return err
				}
				if len(addrs) == 0 {
					return fmt.Errorf("no kettle address found")
				}
				kettleAddress = addrs[0]
			}

			// derive the private key for the cast, if none is set return an error
			// except that the target is the local devchain where we can use the default private key
			var privKeyString string
			if ctx.IsSet(privateKeyFlag.Name) {
				privKeyString = ctx.String(privateKeyFlag.Name)
			} else {
				if kettleAddress == devchainKettleAddress {
					log.Info("Running with local devchain")
					privKeyString = devchainPrivateKey
				} else {
					return fmt.Errorf("no private key set")
				}
			}
			privKey, err := crypto.HexToECDSA(privKeyString)
			if err != nil {
				return err
			}

			clt := sdk.NewClient(rpcClient, privKey, kettleAddress)

			args := ctx.Args().Slice()
			if len(args) < 2 {
				return fmt.Errorf("expected at least 2 arguments (contract address, method signature), got %d", len(args))
			}

			var contractAddr common.Address
			if err := contractAddr.UnmarshalText([]byte(args[0])); err != nil {
				return err
			}

			log.Info(fmt.Sprintf("Contract at address %s", contractAddr.String()))

			methodSig := args[1]
			if !strings.HasPrefix("function ", methodSig) {
				// ethgo requires the method signature to start with "function "
				methodSig = "function " + methodSig
			}
			method, err := ethgoabi.NewMethod(methodSig)
			if err != nil {
				return fmt.Errorf("failed to parse method signature: %w", err)
			}

			calldata, err := method.Encode([]interface{}{})
			if err != nil {
				return err
			}

			log.Info("Sending offchain confidential compute request", "kettle", kettleAddress.String())
			hash, err := sendConfRequest(clt, devchainKettleAddress, contractAddr, calldata, nil)
			if err != nil {
				return err
			}

			log.Info("Hash of the result onchain transaction", "hash", hash.String())
			log.Info("Waiting for the transaction to be mined...")

			receipt, err := waitForTxn(clt, hash)
			if err != nil {
				return err
			}

			log.Info("Transaction mined", "status", receipt.Status, "blockNum", receipt.BlockNumber)

			return nil
		},
	}
)

func sendConfRequest(client *sdk.Client, kettleAddr, addr common.Address, calldata, confBytes []byte) (common.Hash, error) {
	signer, err := client.GetSigner()
	if err != nil {
		return common.Hash{}, err
	}

	nonce, err := client.RPC().PendingNonceAt(context.Background(), client.SenderAddr())
	if err != nil {
		return common.Hash{}, err
	}

	gasPrice, err := client.RPC().SuggestGasPrice(context.Background())
	if err != nil {
		return common.Hash{}, err
	}

	computeRequest, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
		ConfidentialComputeRecord: types.ConfidentialComputeRecord{
			KettleAddress: kettleAddr,
			Nonce:         nonce,
			To:            &addr,
			Value:         nil,
			GasPrice:      gasPrice,
			Gas:           10000000,
			Data:          calldata,
		},
		ConfidentialInputs: confBytes,
	}), signer, client.Key())
	if err != nil {
		return common.Hash{}, err
	}

	computeRequestBytes, err := computeRequest.MarshalBinary()
	if err != nil {
		return common.Hash{}, err
	}

	var hash common.Hash
	if err = client.RPC().Client().Call(&hash, "eth_sendRawTransaction", hexutil.Encode(computeRequestBytes)); err != nil {
		return common.Hash{}, err
	}
	return hash, nil
}

var (
	txnMinedTimeout = 5 * time.Minute
)

func waitForTxn(client *sdk.Client, hash common.Hash) (*types.Receipt, error) {
	timer := time.NewTimer(txnMinedTimeout)

	var receipt *types.Receipt
	var err error

	for {
		select {
		case <-timer.C:
			return nil, fmt.Errorf("timeout")
		case <-time.After(100 * time.Millisecond):
			receipt, err = client.RPC().TransactionReceipt(context.Background(), hash)
			if err != nil && err != ethereum.NotFound {
				return nil, err
			}
			if receipt != nil {
				return receipt, nil
			}
		}
	}
}
