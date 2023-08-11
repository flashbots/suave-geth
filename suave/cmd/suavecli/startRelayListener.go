package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const hyperCube = `
	SUAVE CENTAURI!
				         ()
	     __________     // 
	()  |\          \  //
     \\__ \ _________\//
	  \__) |          |
		|  |  goerli  |
		 \ |  block   |
		  \|__________|
		  //    \\
		 ((     ||
		  \\    ||
		( ()    ||
		        () )
`

func cmdRelayListener() {
	flagset := flag.NewFlagSet("startRelayListener", flag.ExitOnError)

	var (
		suaveWs            = flagset.String("suave_ws", "ws://127.0.0.1:8546", "address of suave ws")
		verbosity          = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		blockSenderAddrHex = flagset.String("block_sender_addr", "0x42042042028AE1CDE26d5BcF17Ba83f447068E5B", "address of mev share contract")
		boostRelayUrl      = flagset.String("relay_url", "https://boost-relay.flashbots.net/", "address of boost relay that the contract will send blocks to")
	)
	flagset.Parse(os.Args[2:])

	blockHashChan := make(chan string)
	nextBlockHashChan := make(chan string)
	payloadDeliveredChan := make(chan bool)

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	targetAddress := common.HexToAddress(*blockSenderAddrHex)

	go blockHashListener(blockHashChan, *suaveWs, targetAddress)
	go blockHashChecker(blockHashChan, nextBlockHashChan, *boostRelayUrl)
	go payloadDeliveryChecker(nextBlockHashChan, payloadDeliveredChan, *boostRelayUrl)

	// Run until success
	<-payloadDeliveredChan
	log.Info("Success!")
}

// blockHashListener listens for blocks and when it detects one, sifts through the transactions looking for block bid event
func blockHashListener(blockHashChan chan<- string, suaveWs string, targetAddress common.Address) {
	client, err := rpc.Dial(suaveWs)
	if err != nil {
		log.Error("Failed to connect to the Ethereum client", "err", err)
		return
	}

	ethClient := ethclient.NewClient(client)
	if ethClient == nil {
		log.Error("Failed to create new ethclient instance")
		return
	}

	log.Info("started hint listener")

	headers := make(chan *types.Header)
	sub, err := ethClient.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Error("Failed to subscribe to new heads: %v", err)
		return
	}
	log.Info("Subscribed to NewHead events")

	for {
		select {
		case err := <-sub.Err():
			log.Error("Subscription error: %v", err)
		case header := <-headers:
			block, err := ethClient.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Error("Failed to get block by hash: %v", err)
				return
			} else {
				for _, tx := range block.Transactions() {
					if tx.To().Hex() == targetAddress.Hex() {
						receipt, err := ethClient.TransactionReceipt(context.Background(), tx.Hash())
						if err != nil {
							log.Error("Failed to get transaction receipt: %v", err)
							return
						} else {
							for _, log := range receipt.Logs {
								fmt.Println(log) // For now, simply print out the logs
							}
						}
						blockHashChan <- block.Hash().Hex()
						log.Info(hyperCube) // Handle one block for now
					}
				}
			}
		}
	}
}

// blockHashChecker is a goroutine that takes the block hash that was emitted and checks the specific endpoint
func blockHashChecker(blockHashChan <-chan string, nextBlockHashChan chan<- string, boostRelayUrl string) {
	for blockHash := range blockHashChan {
		url := fmt.Sprintf("%s/relay/v1/data/bidtraces/builder_blocks_received?block_hash=%s", boostRelayUrl, blockHash)
		for i := 0; i < 12; i++ {
			time.Sleep(time.Second)
			resp, err := http.Get(url)
			if err != nil {
				log.Error("Failed to send GET request: %v", err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error("Failed to read response body: %v", err)
				continue
			}

			if strings.Contains(string(body), blockHash) {
				nextBlockHashChan <- blockHash
				break
			}
		}
	}
}

// payloadDeliveryChecker is a goroutine that checks if the proposer payload was delivered using the block hash
func payloadDeliveryChecker(nextBlockHashChan <-chan string, payloadDeliveredChan chan<- bool, boostRelayUrl string) {
	for blockHash := range nextBlockHashChan {
		if blockHash == "" {
			continue
		}
		url := fmt.Sprintf("%s/relay/v1/data/bidtraces/proposer_payload_delivered?block_hash=%s", boostRelayUrl, blockHash)
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			resp, err := http.Get(url)
			if err != nil {
				log.Error("Failed to send GET request: %v", err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error("Failed to read response body: %v", err)
				continue
			}

			if strings.Contains(string(body), blockHash) {
				payloadDeliveredChan <- true
				return
			}
		}
		payloadDeliveredChan <- false
	}
}
