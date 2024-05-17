package spellcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"strings"
	"time"

	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
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
	artifactsDirFlag = &cli.StringFlag{
		Name:  "artifacts",
		Usage: "The directory where the contract artifacts are located",
		Value: "out",
	}
	confidentialInput = &cli.StringFlag{
		Name:  "confidential-input",
		Usage: "The confidential input to use for the confidential request",
	}
)

var (
	Cmd = &cli.Command{
		Name:        "spell",
		Usage:       "Send a confidential request to a contract",
		Description: "Send a confidential request to a contract",
		Subcommands: []*cli.Command{
			deployCmd,
			confRequestCmd,
		},
	}

	deployCmd = &cli.Command{
		Name:        "deploy",
		Usage:       "Deploy a contract",
		Description: "Deploy a contract",
		Flags: []cli.Flag{
			kettleAddressFlag,
			privateKeyFlag,
			rpcFlag,
			artifactsDirFlag,
		},
		Action: func(ctx *cli.Context) error {
			args := ctx.Args().Slice()
			if len(args) < 1 {
				return fmt.Errorf("expected at least 1 argument (contract artifact), got %d", len(args))
			}

			clt, err := getClient(ctx)
			if err != nil {
				return err
			}

			artifact, err := resolveArtifact(ctx, args[0])
			if err != nil {
				return err
			}

			buf, err := hexutil.Decode(artifact.Bytecode.Object)
			if err != nil {
				return err
			}
			res, err := sdk.DeployContract(buf, clt)
			if err != nil {
				panic(err)
			}

			log.Info("Hash of the result onchain transaction", "hash", res.Hash().String())
			log.Info("Waiting for the transaction to be mined...")

			receipt, err := waitForTxn(clt, res.Hash())
			if err != nil {
				return err
			}
			if receipt.Status == types.ReceiptStatusFailed {
				return fmt.Errorf("the txn did not succeed")
			}

			log.Info("Transaction mined", "status", receipt.Status, "blockNum", receipt.BlockNumber)
			log.Info("Contract deployed", "address", receipt.ContractAddress)

			return nil
		},
	}

	confRequestCmd = &cli.Command{
		Name:        "conf-request",
		Usage:       "Send a confidential request to a contract",
		Description: "Send a confidential request to a contract",
		Flags: []cli.Flag{
			kettleAddressFlag,
			privateKeyFlag,
			rpcFlag,
			artifactsDirFlag,
			confidentialInput,
		},
		Action: func(ctx *cli.Context) error {
			args := ctx.Args().Slice()
			if len(args) < 2 {
				return fmt.Errorf("expected at least 2 arguments (contract address, method signature), got %d", len(args))
			}

			clt, err := getClient(ctx)
			if err != nil {
				return err
			}

			var confInput []byte
			if input := ctx.String(confidentialInput.Name); input != "" {
				if strings.HasPrefix(input, "0x") {
					if confInput, err = hexutil.Decode(input); err != nil {
						return fmt.Errorf("failed to decode hex confidential input: %w", err)
					}
				} else {
					confInput = []byte(input)
				}
				log.Info("Confidential input provided", "input", confInput)
			} else {
				log.Info("No confidential input provided, using empty string")
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

			var methodArgs []interface{}
			if len(args) == 3 {
				// arguments are passed as an array of items '(0x000,1,2)'
				methodArgsStr := args[2]

				// verify it has brackets and remove them
				if !strings.HasPrefix(methodArgsStr, "(") {
					return fmt.Errorf("expected method arguments to start with '('")
				}
				if !strings.HasSuffix(methodArgsStr, ")") {
					return fmt.Errorf("expected method arguments to end with ')'")
				}

				methodArgsStr = methodArgsStr[1 : len(methodArgsStr)-1]
				parts := strings.Split(methodArgsStr, ",")

				for _, part := range parts {
					methodArgs = append(methodArgs, strings.TrimSpace(part))
				}
			}

			calldata, err := method.Encode(methodArgs)
			if err != nil {
				return err
			}

			log.Info("Sending offchain confidential compute request", "kettle", clt.KettleAddress().String())

			hash, err := sendConfRequest(clt, contractAddr, calldata, confInput)
			if err != nil {
				return err
			}

			log.Info("Hash of the result onchain transaction", "hash", hash.String())
			log.Info("Waiting for the transaction to be mined...")

			receipt, err := waitForTxn(clt, hash)
			if err != nil {
				return err
			}
			if receipt.Status == types.ReceiptStatusFailed {
				return fmt.Errorf("the txn did not succeed")
			}

			log.Info("Transaction mined", "status", receipt.Status, "blockNum", receipt.BlockNumber)

			if len(receipt.Logs) != 0 {
				// If possible, try to use the artifact output folder to type decode the logs emitted
				artifactEvents, err := resolveEventsInArtifactsFolder(ctx)
				if err != nil {
					log.Warn("could not decode events from artifacts")
				}

				log.Info("Logs emitted in the onchain transaction", "numLogs", len(receipt.Logs))
				for _, rLog := range receipt.Logs {
					prettyLogEmitted := false
					if artifactEvents != nil {
						prettyLogEmitted = artifactEvents.decodeLog(rLog)
					}

					// fallback to emit the log raw if the event was not found in the artifacts
					if !prettyLogEmitted {
						topic1 := "<none>"
						if len(rLog.Topics) >= 1 {
							topic1 = rLog.Topics[0].Hex()[:5]
						}
						log.Info("Log emitted", "address", rLog.Address.Hex(), "numTopics", len(rLog.Topics), "topic1", topic1)
					}
				}
			}

			return nil
		},
	}
)

func getClient(ctx *cli.Context) (*sdk.Client, error) {
	rpcClient, err := rpc.Dial(ctx.String(rpcFlag.Name))
	if err != nil {
		return nil, err
	}

	var kettleAddress common.Address
	if ctx.IsSet(kettleAddressFlag.Name) {
		kettleAddress = common.HexToAddress(ctx.String(kettleAddressFlag.Name))
	} else {
		// get the kettle address from the rpc endpoint
		var addrs []common.Address
		if err := rpcClient.Call(&addrs, "eth_kettleAddress"); err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("no kettle address found")
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
			log.Info("Running with local devchain settings")
			privKeyString = devchainPrivateKey
		} else {
			return nil, fmt.Errorf("no private key set")
		}
	}
	privKey, err := crypto.HexToECDSA(privKeyString)
	if err != nil {
		return nil, err
	}

	clt := sdk.NewClient(rpcClient, privKey, kettleAddress)
	return clt, nil
}

func sendConfRequest(client *sdk.Client, addr common.Address, calldata, confBytes []byte) (common.Hash, error) {
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
			KettleAddress: client.KettleAddress(),
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

type forgeArtifact struct {
	Abi      *ethgoabi.ABI `json:"abi"`
	Bytecode struct {
		Object string `json:"object"`
	} `json:"bytecode"`
}

func resolveArtifact(ctx *cli.Context, contractRef string) (*forgeArtifact, error) {
	parts := strings.Split(contractRef, ":")
	sourceName, contractName := parts[0], parts[1]
	outDir := ctx.String(artifactsDirFlag.Name)

	contractDir := filepath.Join(outDir, sourceName, contractName+".json")

	data, err := os.ReadFile(contractDir)
	if err != nil {
		return nil, err
	}

	var artifact forgeArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, err
	}
	return &artifact, nil
}

// artifactEvents contains all the events in the artifacts folder
type artifactEvents []*ethgoabi.Event

func (a artifactEvents) decodeLog(rawLog *types.Log) bool {
	ethgoLog := &ethgo.Log{
		Data:   rawLog.Data,
		Topics: make([]ethgo.Hash, len(rawLog.Topics)),
	}
	for indx, topic := range rawLog.Topics {
		ethgoLog.Topics[indx] = ethgo.Hash(topic)
	}

	// try to find the topic0 id
	topic1 := ethgoLog.Topics[0]
	for _, event := range a {
		if event.ID() == topic1 {
			decoded, err := event.ParseLog(ethgoLog)
			if err != nil {
				log.Warn("failed to parse log", "err", err)
				return false
			}

			decodedList := []interface{}{
				"name", event.Sig(),
			}
			for _, xx := range event.Inputs.TupleElems() {
				decodedList = append(decodedList, xx.Name, decoded[xx.Name])
			}
			log.Info("Log emitted", decodedList...)
			return true
		}
	}
	return false
}

func resolveEventsInArtifactsFolder(ctx *cli.Context) (artifactEvents, error) {
	outDir := ctx.String(artifactsDirFlag.Name)

	// check if the directory exists or not
	info, err := os.Stat(outDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("is not dir")
	}

	var events []*ethgoabi.Event
	err = filepath.WalkDir(outDir, func(path string, d os.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var artifact *forgeArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			return err
		}

		for _, evnt := range artifact.Abi.Events {
			events = append(events, evnt)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return artifactEvents(events), nil
}
