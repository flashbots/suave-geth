# SUAVE

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/suave-geth)](https://goreportcard.com/report/github.com/flashbots/suave-geth)
[![CI status](https://github.com/flashbots/suave-geth/workflows/Checks/badge.svg?branch=main)](https://github.com/flashbots/suave-geth/actions/workflows/checks.yml)

[SUAVE](https://writings.flashbots.net/the-future-of-mev-is-suave) is designed to keep the benefits of MEV well distributed by enabling centralized infrastructure (builders, relays, centralized RFQ routing, etc.) to be programmed as smart contracts on a decentralized blockchain.

`suave-geth` is a work-in-progress Golang SUAVE client consisting of two separable components: chain nodes and MEVM nodes. SUAVE clients offer confidential execution for smart contracts, enabled by extended precompiles for MEV applications, including confidential requests, transaction simulation, block building, and relay boosting.

Please [visit our documentation](https://suave.flashbots.net) for further details.

---

## Getting Started

### With Docker

1. Clone this repo:
```bash
git clone https://github.com/flashbots/suave-geth.git
cd suave-geth
```
2. Run SUAVE (depending on your docker setup, you may need to run this as `sudo`):
```bash
make devnet-up 
```

#### Optional testing

4. If you'd like to test SUAVE by deploying a contract and sending it some transactions, you can do so easily by running:
```bash
go run suave/devenv/cmd/main.go
```

### Build the binaries yourself

1. Clone the repo and build SUAVE:
```bash
git clone https://github.com/flashbots/suave-geth.git
cd suave-geth
make suave
```
2. SUAVE clients are really two nodes in a trench coat. Run the MEVM node first:
```bash
./build/bin/suave --dev --dev.gaslimit 30000000 --datadir suave --http --ws \
--allow-insecure-unlock --unlock "0xb5feafbdd752ad52afb7e1bd2e40432a485bbb7f" \
--keystore ./suave/devenv/suave-ex-node/keystore/
```
3. Press `Enter` when prompted for a password
4. In a new terminal, run the chain node:
```bash
./build/bin/suave --dev --dev.gaslimit 30000000 --http --http.port 8555 --ws --ws.port 8556 --authrpc.port 8561
```
5. You can now run any SUAVE command you like. Start by generating a new account (in another terminal):
```bash
./build/bin/suave --suave account new
```

If the `--datadir` flag is not set, a geth client stores data in the `$HOME/.ethereum` directory. Depending on the chain you use, it creates a subdirectory. For example, if you run Sepolia, geth creates `$HOME/.ethereum/sepolia/`. So, if you use the `--suave` flag, your data ends up in `$HOME/.ethereum/suave/...`.

## Contributing

We welcome your contributions. If MEV only benefits the few, we cannot realise the permissionless, censorship-resistant systems that crypto promises.

Please read our [Contributing Guide](./CONTRIBUTING.md) to get oriented.

---

Made with ☀️ by the ⚡🤖 collective.
