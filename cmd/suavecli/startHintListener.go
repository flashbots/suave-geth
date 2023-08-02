package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

const cube = `
          +------------+,
         /            /|,
        /            / |,
       +------------+  |,
       |            |  |,
       |   Suave    |  +,
       |   Block    | /,
       |            |/,
       +------------+
	   `
const hint = `
            ______              
         .-'      '-.           
       .'            '.         
      /                \        
     |      Hint       |;       
     '\               / ;       
      \'.           .' /        
       '.'-._____.-' .'         
         / /'_____.-'           
        / / /                   
       / / /
      / / /
     / / /
     \/_/
`

func cmdHintListener() {
	flagset := flag.NewFlagSet("sendBundle", flag.ExitOnError)

	var (
		suaveWs   = flagset.String("suave_ws", "ws://127.0.0.1:8546", "address of suave ws")
		verbosity = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
	)

	flagset.Parse(os.Args[2:])

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	go subscribeToWs(*suaveWs)

	select {}
}

func subscribeToWs(suaveWs string) {
	log.Info("started hint listener")
	for {
		client, err := rpc.Dial(suaveWs)
		if err != nil {
			log.Error("Failed to connect to the Ethereum client", "err", err)
			time.Sleep(2 * time.Second)
			continue
		}

		ethClient := ethclient.NewClient(client)
		if ethClient == nil {
			log.Error("Failed to create new ethclient instance")
			time.Sleep(2 * time.Second)
			continue
		}

		httpClient, err := rpc.Dial("http://127.0.0.1:8545")
		if err != nil {
			log.Error("Failed to connect to the Ethereum client", "err", err)
			time.Sleep(2 * time.Second)
			continue
		}

		httpEthClient := ethclient.NewClient(httpClient)
		if ethClient == nil {
			log.Error("Failed to create new ethclient instance")
			time.Sleep(2 * time.Second)
			continue
		}

		headers := make(chan *types.Header)
		sub, err := ethClient.SubscribeNewHead(context.Background(), headers)
		if err != nil {
			log.Error("Failed to subscribe to new heads", "err", err)
			time.Sleep(2 * time.Second)
			subscribeToWs(suaveWs)
			continue
		}
		log.Info("Subscribed to NewHead events")

		for {
			select {
			case err := <-sub.Err():
				log.Error("Subscription error", "err", err)
				log.Info("Attempting to resubscribe in 2 seconds")
				time.Sleep(2 * time.Second)
				continue
			case header := <-headers:
				headerHashString := header.Hash().String()
				log.Info(headerHashString)
				log.Info(cube)
				block, err := httpEthClient.BlockByHash(context.Background(), header.Hash())
				if err != nil {
					log.Error("Failed to get block", "err", err)
					continue
				}
				log.Info(cube)

				for _, tx := range block.Transactions() {
					r, err := ethClient.TransactionReceipt(context.Background(), tx.Hash())
					if err != nil {
						log.Error("Failed to get transaction receipt", "err", err)
						continue
					}

					// Check for contract creation
					if tx.To() == nil {
						log.Info("Contract created", "Contract Address", r.ContractAddress)
					}

					// we know mevshare will emit 2 events and dont want to break on other txns
					// as well may not know mevshare addr at demo time
					if len(r.Logs) == 2 {
						log.Info(hint)
						log.Info("Detected event in tx", "tx_hash", tx.Hash().Hex(), "to_addr", tx.To())
						// extract first hint
						hint1, err := mevShareABI.Events["HintEvent"].Inputs.Unpack(r.Logs[0].Data)
						if err != nil {
							log.Error("Failed to unpack log", "err", err)
						}
						// extract second hint
						hint2, err := mevShareABI.Events["BidEvent"].Inputs.Unpack(r.Logs[1].Data)
						if err != nil {
							log.Error("Failed to unpack log", "err", err)
						}
						bidId := hint2[0].([16]byte)
						log.Info("Hint event", "hint1", hint1, "hint2", hint2, "bidId", bidId)
					}
				}
			}
		}
	}
}
