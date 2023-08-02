# Suave CLI

This repository contains the Command Line Interface (CLI) for interacting with a Suave Geth node for live testing and demo'ing purposes. The Suave CLI allows users to deploy contracts, send bids, and perform various actions related to MEV-Share and Block Building.

As this repository is under development these commands may not always be up to date but they should be a good starting point for interacting with a SUAVE node.

## Getting Started

Before running the Suave CLI, ensure you have the following:

- Golang (Go) installed
- Ethereum account with private key (for testing)
- Suave RPC endpoint (local or remote)
- Goerli RPC endpoint (local or remote)
- Goerli Beacon RPC endpoint (local or remote)
- Boost Relay URL

To get started with the Suave CLI, follow these steps:

1. Clone this repository to your local machine.

2. Install the necessary dependencies by running `go mod tidy`.

3. Build the CLI using `go build -o suavecli ./suave-geth/cmd/suavecli`.

4. Run the Suave CLI with the desired command and subcommand.

For example:
```
./suavecli deployBlockSenderContract
```

## Commands

### Deploy Commands:

1. `deployBlockSenderContract`: Deploys the BlockSender contract to the Suave network. This contract is used to send constructed blocks for execution via the Boost Relay.

2. `deployMevShareContract`: Deploys the MevShare contract to the Suave network. This contract is used for sharing Maximum Extractable Value (MEV) profits with the MevExtractor contract.

### Send Commands:

1. `sendBundle`: Sends a bundle of transactions to specified MEVM contract.

2. `sendMevShareBundle`: Sends a MEVShare bundle to specified MEVM contract.

3. `sendMevShareMatch`: Sends a MEV share match transaction to the Suave network via the Boost Relay for matching MEV share recipients with their corresponding transactions.

4. `sendBuildShareBlock`: Sends a transaction to build a Goerli block using MEV-Share orderflow and sends to specified Goerli relay.

### Demo Helper Commands:

1. `startHintListener`: Starts a hint listener for demo purposes. This command listens for hints emmited from MEV-Share on the Suave Chain.

2. `subscribeBeaconAndBoost`: Subscribes to events from the Beacon Chain and Boost for demo purposes.

3. `startRelayListener`: Starts a relay listener for demo purposes. This command listens for block submisisons and deliveries from the Boost Relay.

### End-to-End (e2e) Test Commands:

1. `testDeployAndShare`: Performs an end-to-end test scenario that includes contract deployment and block sharing.

2. `buildGoerliBlocks`: Performs an end-to-end test scenario for building and sharing blocks on the Goerli network.

