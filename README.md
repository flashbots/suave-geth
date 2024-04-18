# SUAVE

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/suave-geth)](https://goreportcard.com/report/github.com/flashbots/suave-geth)
[![CI status](https://github.com/flashbots/suave-geth/workflows/Checks/badge.svg?branch=main)](https://github.com/flashbots/suave-geth/actions/workflows/checks.yml)

[SUAVE](https://writings.flashbots.net/mevm-suave-centauri-and-beyond) is designed to decentralize the MEV supply chain by enabling centralized infrastructure (builders, relays, centralized RFQ routing, etc.) to be programmed as smart contracts on a decentralized blockchain.

`suave-geth` is a work-in-progress Golang SUAVE client consisting of two separable components: chain nodes and execution nodes. SUAVE clients offer confidential execution for smart contracts, allowing confidential processing with extended precompiles for enhanced MEV functionalities, including transaction simulation via geth RPC, block building, and relay boosting, all handled by dedicated execution nodes.

For a deeper dive, check out the following links:

- [Suave Specs](https://github.com/flashbots/suave-specs)
- [Simple MEV-share walk through](suave/cmd/suavecli/README.md)
- [Demo video from EthCC](https://drive.google.com/file/d/1IHuLtxwjRvRpYjMG3oRuAgS5MUZtmAXq/view?usp=sharing).
- [Suapp Examples](https://github.com/flashbots/suapp-examples)

---

## Getting Started

### Starting a local devnet

You can use `suave-geth` to start a local SUAVE devnet.

There's multiple ways to get `suave-geth`:

1. Install the latest release binary
2. Build from source
3. Use Docker

#### Install the latest suave-geth release binary

```bash
curl -L https://suaveup.flashbots.net | bash
```

#### Building from source

```bash
# build the binary
$ make suave
```

Now you can go to /build/bin and:

Start the local devnet like this:

```bash
$ ./suave-geth --suave.dev
```

Start the Rigil testnet like this:

```bash
$ ./suave-geth --rigil
```

#### Using Docker

```bash
# spin up the local devnet with docker-compose
$ make devnet-up

# check that the containers are running
$ docker ps

# you can stop the local devnet like this
$ make devnet-down
```

#### Testing the devnet

Create a few example transactions:

```bash
$ go run suave/devenv/cmd/main.go
```

Execute a RPC request with curl like this:

```bash
$ curl 'http://localhost:8545' --header 'Content-Type: application/json' --data '{ "jsonrpc":"2.0", "method":"eth_blockNumber", "params":[], "id":83 }'
```

## What next?

- [suapp-examples](https://github.com/flashbots/suapp-examples) is a collection of example SUAVE apps and boilerplate to get started quickly and right-footed.
- [suave-specs](https://github.com/flashbots/suave-specs) is the spec repository for SUAVE which contains all the technical documentation.