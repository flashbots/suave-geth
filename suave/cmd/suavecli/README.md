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

3. Build the CLI using `go build -o suavecli ./suave-geth/suave/cmd/suavecli`.

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

## Simple MEV-Share Walkthrough

To get a better understanding of how the MEVM works, let's delve into the deployment of a simple version of `[mev-share](https://github.com/flashbots/mev-share)`, a protocol for orderflow auctions, defined via smart contract on SUAVE. Our journey below will guide you through the steps of deploying simple mev-share and block builder contracts, interacting with them, and ultimately seeing a block land onchain.

- [Intro](#simple-mev-share-walkthrough)
- [Prerequisites 🛠](#prerequisites-)
- [Walkthrough Overview 🚀](#walkthrough-overview-)
    - [1. Deploy Simple MEV-Share Contract 📜](#1-deploy-simple-mev-share-contract-)
    - [2. Deploy Block Builder Contract 📜](#2-deploy-block-builder-contract-)
    - [3. Send Mevshare Bundles 📨](#3-send-mevshare-bundles-)
    - [4. Send Mevshare Matches 🎯](#4-send-mevshare-matches-)
    - [5. Build Block and Relay 🧱](#5-build-block-and-relay-)

## Prerequisites 🛠

In this walkthrough we will go through a script located inside the [suave cli tool](https://github.com/flashbots/suave-geth/blob/suave-poc/cmd/suavecli/testDeployAndShare.go), so don't worry about running the below code, it’s mainly for conceptual purposes. To follow along and use the tool you will need:

- [SUAVE chain node and SUAVE execution node setup](https://github.com/flashbots/suave-geth/tree/main/suave).
- Basic knowledge of the Ethereum's Golang libraries and the [Ethereum RPC methods](https://github.com/flashbots/suave-geth/tree/main/suave).

Ensure these details for the command line tool are on hand:

- `suave_rpc` : address of suave rpc
- `goerli_rpc` : address of goerli execution node rpc
- `goerli_beacon_rpc` : address of goerli beacon rpc
- `kettleAddress` : wallet address of execution node
- `privKeyHex` : private key as hex (for testing)
- `relay_url` : address of boost relay that the contract will send blocks to

## Walktrhough

### 1. Deploy Simple MEV-Share Contract 📜

Our first step is to deploy the [compiled byte code](https://github.com/flashbots/suave-geth/blob/suave-poc/cmd/suavecli/deployMevShareByteCode.go) from our `mev-share` contract. As you will see, deploying on SUAVE feels just like deploying on any other EVM chain. First we gather our transaction details, nounce and gas price, sign the transaction, and then send using the normal `eth_sendRawTransaction` using your `suaveClient`

```go
	mevShareAddrPtr, txHash, err := sendMevShareCreationTx(suaveClient, suaveSigner, privKey)
	if err != nil {
		panic(err.Error())
	}

	waitForTransactionToBeConfirmed(suaveClient, txHash)
	mevShareAddr := *mevShareAddrPtr

```

Now we take a look under the hood of `sendMevShareCreationTx`.

```go
func sendMevShareCreationTx(suaveClient *rpc.Client, suaveSigner types.Signer, privKey *ecdsa.PrivateKey) (*common.Address, *common.Hash, error) {
	var suaveAccNonceBytes hexutil.Uint64
	err := suaveClient.Call(
            &suaveAccNonceBytes,
            "eth_getTransactionCount",
            crypto.PubkeyToAddress(privKey.PublicKey),
            "latest"
        )
	suaveAccNonce := uint64(suaveAccNonceBytes)

	var suaveGp hexutil.Big
	err = suaveClient.Call(&suaveGp, "eth_gasPrice")

	calldata := hexutil.MustDecode(mevshareContractBytecode)
	mevshareContractBytecode)
	ccTxData := &types.LegacyTx{
		Nonce:    suaveAccNonce,
		To:       nil, // contract creation
		Value:    big.NewInt(0),
		Gas:      10000000,
		GasPrice: (*big.Int)(&suaveGp),
		Data:     calldata,
	}

	tx, err := types.SignTx(types.NewTx(ccTxData), suaveSigner, privKey)

	from, _ := types.Sender(suaveSigner, tx)
	mevshareAddr := crypto.CreateAddress(from, tx.Nonce())
	log.Info("contract address will be", "addr", mevshareAddr)

	txBytes, err := tx.MarshalBinary()

	var txHash common.Hash
	err = suaveClient.Call(
            &txHash,
            "eth_sendRawTransaction",
            hexutil.Encode(txBytes)
        )

	return &mevshareAddr, &txHash, nil
}

```

Later, we'll incorporate the `mevshareAddr` into our transaction's allowed contracts, granting access for the contract to compute over our confidential data.

### 2. Deploy Block Builder Contract 📜

Next we deploy a [simple block builder contract](https://github.com/flashbots/suave-geth/blob/4ad40edc58d8374c24a2e88c0b6fe2a7d8363ae3/suave/sol/standard_peekers/bids.sol#L140) which we will also store to later grant access to. The block builder takes in a `boostRelayUrl` which is where it will send blocks to when finished building.

```go

	blockSenderAddrPtr, txHash, err := sendBlockSenderCreationTx(
            suaveClient,
            suaveSigner,
            privKey,
            boostRelayUrl
    )
	if err != nil {
		panic(err.Error())
	}

	waitForTransactionToBeConfirmed(suaveClient, txHash)
	blockSenderAddr := *blockSenderAddrPtr

```

Similar as above, `sendBlockSenderCreationTx` operates like any other contract deployment on an EVM chain.

### 3. Send Mevshare Bundles 📨

Once our contracts have been succesfully deployed we will craft a goerli bundle and send it to our newly deployed mev-share contract.

```go
	mevShareTx, err := sendMevShareBidTx(suaveClient, goerliClient, suaveSigner, goerliSigner, 5, mevShareAddr, blockSenderAddr, kettleAddress, privKey)
	if err != nil {
		err = errors.Wrap(err, unwrapPeekerError(err).Error())
		panic(err.Error())
	}

	waitForTransactionToBeConfirmed(suaveClient, &mevShareTx.txHash)

```

Let's take a deeper look at `sendMevShareBidTx` which looks similar to a normal Ethereum transaction but has a couple key differences. We explore those below the following code snippet.

```go
func sendMevShareBidTx(
    // function inputs removed for brevity
) (mevShareBidData, error) {

	var startingGoerliBlockNum uint64
	err = goerliClient.Call(
            (*hexutil.Uint64)(&startingGoerliBlockNum),
            "eth_blockNumber"
        )
	if err != nil {
		utils.Fatalf("could not get goerli block: %v", err)
	}

	_, ethBundleBytes, err := prepareEthBundle(
            goerliClient,
            goerliSigner,
            privKey
        )

	// Prepare bundle bid
	var suaveAccNonce hexutil.Uint64
	err = suaveClient.Call(
            &suaveAccNonce,
            "eth_getTransactionCount",
            crypto.PubkeyToAddress(privKey.PublicKey),
            "pending"
        )

	confidentialDataBytes, err := mevShareABI.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(ethBundleBytes)

	allowedPeekers := []common.Address{
            newBlockBidAddress,
            extractHintAddress,
            buildEthBlockAddress,
            mevShareAddr,
            blockBuilderAddr
        }

	calldata, err := mevShareABI.Pack("newBid", blockNum, allowedPeekers)
	if err != nil {
		return mevShareBidData{}, err
	}

        wrappedTxData := &types.DynamicFeeTx{
		Nonce:     suaveAccNonce,
		To:        &mevShareAddr,
		Value:     nil,
		Gas:       10000000,
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(33000000000),
		Data:      calldata,
	}

	mevShareTx, err := types.SignTx(types.NewTx(&types.ConfidentialComputeRequestTx{
		KettleAddress: kettleAddress,
		Wrapped:       *types.NewTx(wrappedTxData),
	}), suaveSigner, privKey)
	if err != nil {
		return nil, nil, err
	}

	mevShareTxBytes, err := mevShareTx.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	var confidentialRequestTxHash common.Hash
	err = suaveClient.Call(
            &confidentialRequestTxHash,
            "eth_sendRawTransaction",
            hexutil.Encode(mevShareTxBytes),
            hexutil.Encode(confidentialDataBytes)
        )
	if err != nil {
		return mevShareBidData{}, err
	}

	mevShareTxHash= mevShareBidData{blockNumber: blockNum, txHash: confidentialRequestTxHash}

	return mevShareTxHash, nil
}

```

A SUAVE transaction, referred to as a mevshare bid in the code, takes in two extra arguments: `allowedPeekers` and `kettleAddress`. These arguement are to utilize a new transaction primitive `types.ConfidentialComputeRequest`, which you can read more about [here](https://github.com/flashbots/suave-geth/tree/suave-poc/suave#confidential-compute-requests).  The role of `allowedPeekers` is to dictate which contracts can view the confidential data, in our scenario, the goerli bundle being submitted. Meanwhile, `kettleAddress` points to the intended execution node for the transaction. Lastly, Suave nodes have a modified `ethSendRawTransaction` to support this new transaction type.

### 4. Send Mevshare Matches 🎯

Now that a MEV-share bid has been sent in we can simulate sending in a match. Once live on a testnet, searchers can monitor the SUAVE chain looking for hints emitted as logs for protocols they specialize in. In our example you could monitor the `mevShareAddr` for emitted events. Using these hints they can get a `BidId` to reference in their match. Below we see the code.

```go
    bidIdBytes, err := extractBidId(suaveClient, mevShareTx.txHash)
    if err != nil {
        panic(err.Error())
    }

    _, err = sendMevShareMatchTx(
        suaveClient,
        goerliClient,
        suaveSigner,
        goerliSigner,
        mevShareTx.blockNumber,
        mevShareAddr,
        blockSenderAddr,
        kettleAddress,
        bidIdBytes,
        privKey,
    )
    if err != nil {
        err = errors.Wrap(err, unwrapPeekerError(err).Error())
        panic(err.Error())
    }

```

### 5. Build Block and Relay 🧱

Now that our SUAVE node's bidpool has a mevshare bid and match, we can trigger block building to combine these transactions, simulate for validity, and insert the refund transaction.

```go
    _, err = sendBuildShareBlockTx(suaveClient, suaveSigner, privKey, kettleAddress, blockSenderAddr, payloadArgsTuple, uint64(goerliBlockNum)+1)
    if err != nil {
        err = errors.Wrap(err, unwrapPeekerError(err).Error())
        if strings.Contains(err.Error(), "no bids") {
            log.Error("Failed to build a block, no bids")
        }
        log.Error("Failed to send BuildShareBlockTx", "err", err)
    }

```

Once the execution node has received this transaction it will build your block and send it off to a relay. If you used the flashbots goerli relay you should be able to check it out using the [builder blocks received endpoint](https://boost-relay-goerli.flashbots.net/).
