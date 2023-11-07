# SUAVE

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/suave-geth)](https://goreportcard.com/report/github.com/flashbots/suave-geth)
[![CI status](https://github.com/flashbots/suave-geth/workflows/Checks/badge.svg?branch=main)](https://github.com/flashbots/suave-geth/actions/workflows/checks.yml)

[SUAVE](https://writings.flashbots.net/mevm-suave-centauri-and-beyond) is designed to decentralize the MEV supply chain by enabling centralized infrastructure (builders, relays, centralized RFQ routing, etc.) to be programmed as smart contracts on a decentralized blockchain.

`suave-geth` is a work-in-progress Golang SUAVE client consisting of two separable components: chain nodes and execution nodes. SUAVE clients offer confidential execution for smart contracts, allowing confidential processing with extended precompiles for enhanced MEV functionalities, including transaction simulation via geth RPC, block building, and relay boosting, all handled by dedicated execution nodes.

For a deeper dive, check out the [technical details section](#suave-geth-technical-details), [simple MEV-share walk through](suave/cmd/suavecli/README.md), and the [demo video from EthCC](https://drive.google.com/file/d/1IHuLtxwjRvRpYjMG3oRuAgS5MUZtmAXq/view?usp=sharing).

---

**Table of Contents**

1. [Getting Started](#getting-started)
    1. [How do I use the SUAVE?](#how-do-i-use-suave)
    1. [How do I execute a contract confidentially?](#how-do-i-execute-a-contract-confidentially)
    1. [How do I run a SUAVE chain node?](#how-do-i-run-a-suave-chain-node)
    1. [How do I run a SUAVE execution node?](#how-do-i-run-a-suave-execution-node)
1. [suave-geth technical details](#suave-geth-technical-details)
    1. [SUAVE Runtime (MEVM)](#suave-runtime-mevm)
    1. [Confidential execution of smart contracts](#confidential-execution-of-smart-contracts)
    1. [Confidential compute requests](#confidential-compute-requests)
    1. [SUAVE Bids](#suave-bids)
    1. [SUAVE library](#suave-library)
    1. [Confidential APIs](#confidential-apis)
    1. [Confidential Store](#confidential-store)
    1. [SUAVE Mempool](#suave-mempool)
    1. [Notable differences from standard issue go-ethereum](#notable-differences-from-standard-issue-go-ethereum)
    1. [Suave precompiles](#suave-precompiles)

---

## Getting Started

### How do I use SUAVE?

1. **Deploy confidential smart contracts.**
   Smart contracts on SUAVE follow the same rules as on Ethereum with the added advantage of being able to access additional precompiles during confidential execution. Precompiles are available through the [SUAVE library](#suave-library).

2. **NEW! Request confidential execution using the new confidential computation request.**
   Contracts called using confidential compute requests have access to confidential data and APIs through SUAVE precompiles. Confidential computation is *not* reproducible on-chain, thus, users are required to whitelist a specific execution node trusted to provide the result. Eventually proofs and trusted enclaves will help to verify the results of execution.
      After the initial confidential computation, its result replaces the calldata for on-chain execution. This grants different behaviors to confidential computation and regular on-chain transactions since confidential APIs are inaccessible during regular chain state transition.


   See [confidential compute requests](#confidential-compute-requests) for more details.

### How do I execute a contract confidentially?

Let’s take a look at how you can request confidential computation through an execution node.  

1. Pick your favorite execution node. You’ll need its URL and wallet address. Note that the execution node is fully trusted to provide the result of your confidential computation.

2. Craft your confidential computation record. This is a regular Ethereum transaction (fields are similar to `LegacyTx`), where you specify the desired contract address and its (public) calldata. I’m assuming you have found or deployed a smart contract which you intend to call. Don’t sign the transaction quite yet!

    ```go
    allowedPeekers := []common.Address{newBlockBidPeeker, newBundleBidPeeker, buildEthBlockPeeker} // express which contracts should have access to your data (by their addresses)
    confidentialComputeRecord := &types.ConfidentialComputeRecord{
        KettleAddress: "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B",
        Nonce:    suaveAccNonce,
        To:       &newBundleBidAddress,
        Value:    nil,
        Gas:      1000000,
        GasPrice: 50,
        Data:     bundleBidAbi.Pack("newBid", targetBlock, allowedPeekers)
    }
    ```

3. Wrap your compute record into a `ConfidentialComputeRequest` transaction type, and specify the confidential data.

    ```go
    confidentialDataBytes := hexutil.Encode(ethBundle)
    confidentialComputeRequest := types.SignTx(types.NewTx(&types.ConfidentialComputeRequest{
        ConfidentialComputeRecord: confidentialComputeRecord,
        ConfidentialInputs: confidentialDataBytes,
    }), suaveSigner, privKey)
    ```

4. Request confidential computation by submitting your transaction to the execution node you chose via `eth_sendRawTransaction`.

    ```go
    suaveClient.Call("eth_sendRawTransaction", confidentialComputeRequest)
    ```

5. All done! Once the execution node processes your computation request, the execution node will submit it as `SuaveTransaction` to the mempool.

For more on confidential compute requests see [confidential compute requests](#confidential-compute-requests).

### How do I run a SUAVE chain node?

1. Build the client with `make geth`.
2. Run the node. Pass in `--dev` to enable local devnet. Example:

    ```go
    ./build/bin/geth --dev --dev.gaslimit 30000000 --datadir suave_dev --http --ws --allow-insecure-unlock --unlock "0xd52d1935D1239ADf94C59fA0F586fE00250694d5"
    ```

3. Do your thing!

### How do I run a SUAVE execution node?

Not all nodes serve confidential compute requests. You’ll need:
- A SUAVE node (see above).
- An account. If you are doing this for testing, simply run `geth --suave account new`. Take note of the address.
- Access to Ethereum’s RPC. When starting your node, pass in `--suave.eth.remote_endpoint` to point to your Ethereum RPC.
    ```go
    ./build/bin/geth --dev --dev.gaslimit 30000000 --datadir suave_dev --http --allow-insecure-unlock --unlock "0x<YOUR_PUBKEY>" --ws --suave.eth.remote_endpoint "http://<EXECUTION_NODE_IP>"
    ```
Note that simply enabling http jsonrpc and allowing direct access might not be the wisest. Look into proxyd and other restricted access solutions.

## suave-geth technical details

### SUAVE Runtime (MEVM)

[`SuaveExecutionBackend`](#suaveexecutionbackend) 🤝 EVM = MEVM

More specifically, `SuaveExecutionBackend` and `Runtime` add functionality to the stock EVM which allows it both confidential computation and interaction with APIs.

```mermaid
graph TB
    A[EVM]-->|1|B((StateDB))
    A-->|2|C((Context))
    A-->|3|D((chainConfig))
    A-->|4|E((Config))
    A-->|5|F((interpreter))
    D-->|6|R[ChainRules]
    E-->|7|S[Tracer]
    A-->|8|T[NewRuntime]
    T-->|9|Z((Runtime))
    Z-->|10|F
    A-->|11|U[NewRuntimeSuaveExecutionBackend]
    U-->|12|V((SuaveExecutionBackend))
    V-->|13|F
    class A,B,C,D,E,F yellow
    class G,H,I,J,K,L,M,N,O red
    class P,Q green
    class R blue
    class S orange
    class T,U purple
    class Z,V lightgreen
    classDef yellow fill:#f5cf58,stroke:#444,stroke-width:2px, color:#333;
    classDef red fill:#d98686,stroke:#444,stroke-width:2px, color:#333;
    classDef green fill:#82a682,stroke:#444,stroke-width:2px, color:#333;
    classDef blue fill:#9abedc,stroke:#444,stroke-width:2px, color:#333;
    classDef orange fill:#f3b983,stroke:#444,stroke-width:2px, color:#333;
    classDef purple fill:#ab92b5,stroke:#444,stroke-width:2px, color:#333;
    classDef lightgreen fill:#b3c69f,stroke:#444,stroke-width:2px, color:#333;
```

The capabilities enabled by this modified runtime are exposed to the virtual machine via `SuaveContext` and its components.

```go
type SuaveContext struct {
    Backend                      *SuaveExecutionBackend
    ConfidentialComputeRequestTx *types.Transaction
    ConfidentialInputs           []byte
    CallerStack                  []*common.Address
}

type SuaveExecutionBackend struct {
    ConfidentialStoreEngine *suave.ConfidentialStoreEngine
    MempoolBackend          suave.MempoolBackend
    ConfidentialEthBackend  suave.ConfidentialEthBackend
}
```

All of these newly offered APIs are available to your solidity smart contract through the use of precompiles! See below for how confidential computation and smart contracts interact.

### Confidential execution of smart contracts

The virtual machine (MEVM) inside SUAVE nodes have two modes of operation: regular and confidential. Regular on-chain environment is your usual Ethereum virtual machine environment.

Confidential environment is available to users through a new type of transaction - `ConfidentialComputeRequest` via the usual jsonrpc methods `eth_sendRawTransaction`, `eth_sendTransaction` and `eth_call`. Simulations (`eth_call`) requested with a new optional argument `IsConfidential` are also executed in the confidential mode. For more on confidential requests see [confidential compute requests](#confidential-compute-requests).

The confidential execution environment provides additional precompiles, both directly and through a convenient [library](#suave-library). Confidential execution is *not* verifiable during on-chain state transition, instead the result of the confidential execution is cached in the transaction (`SuaveTransaction`). Users requesting confidential compute requests specify which execution nodes they trust with execution, and the execution nodes' signature is used for verifying the transaction on-chain.

The cached result of confidential execution is used as calldata in the transaction that inevitably makes its way onto the SUAVE chain.

Other than ability to access new precompiles, the contracts aiming to be executed confidentially are written as usual in Solidity (or any other language) and compiled to EVM bytecode.

### Confidential compute requests

We introduce a few new transaction types.

* `ConfidentialComputeRecord`

    This type serves as an onchain record of computation. It's a part of both the [Confidential Compute Request](#confidential-compute-request) and [Suave Transaction](#suave-transaction).

    ```go
    type ConfidentialComputeRecord struct {
        KettleAddress          common.Address
        ConfidentialInputsHash common.Hash

        // LegacyTx fields
        Nonce    uint64
        GasPrice *big.Int
        Gas      uint64
        To       *common.Address `rlp:"nil"`
        Value    *big.Int
        Data     []byte

        // Signature fields
    }
    ```

* `ConfidentialComputeRequest`

    This type facilitates users in interacting with the MEVM through the `eth_sendRawTransaction` method. After processing, the request's `ConfidentialComputeRecord` is embedded into `SuaveTransaction.ConfidentialComputeRequest` and serves as an onchain record of computation.  

    ```go
    type ConfidentialComputeRequest struct {
        ConfidentialComputeRecord
        ConfidentialInputs []byte
    }
    ```

* `SuaveTransaction`

    A specialized transaction type that encapsulates the result of a confidential computation request. It includes the `ConfidentialComputeRequest`, signed by the user, which ensures that the result comes from the expected computor, as the `SuaveTransaction`'s signer must match the `KettleAddress`.  

    ```go
    type SuaveTransaction struct {
        KettleAddress              common.Address
        ConfidentialComputeRequest ConfidentialComputeRecord
        ConfidentialComputeResult  []byte
        /* Execution node's signature fields */
    }
    ```

![image](suave/docs/conf_comp_request_flow.png)


The basic flow is as follows:

1. User crafts a confidential computation request (`ConfidentialComputeRequest`):
    1. Sets their GasPrice, GasLimit, To address and calldata as they would for a `LegacyTx`
    2. Choses an execution node of their liking, that is an address whose signature over the confidential computation result will be trusted
    3. The above becomes the `ConfidentialComputeRecord` that will eventually make its way onto the chain
    4. Sets the `ConfidentialInputs` of the request (if any)
    5. Signs and sends the confidential computation request (consisting of (3 and 4) to an execution node via `eth_sendRawTransaction`
2. The execution node executes the transaction in the confidential mode, providing access to the usual confidential APIs
3. The execution node creates a `SuaveTransaction` using the confidential computation request and the result of its execution, the node then signs and submits the transaction into the mempool
4. The transaction makes its way into a block, by executing the `ConfidentialComputeResult` as calldata, as long as the execution node's signature matches the requested executor node in (1.2.)

The initial confidential computation has access to both the public and confidential data, but only the public data becomes part of the transaction propagated through the mempool. Any confidential data passed in by the user is discarded after the execution.  

Architecture reference
![image](suave/docs/execution_node_architecture.png)

Mind, that the results are not reproducible as they are based on confidential data that is dropped after execution. On-chain state transition only depends on the result of the confidential computation, so it is fully reproducible.

### SUAVE Bids

On the SUAVE chain, bids serve as the primary transaction unit, and are used for interactions between smart contracts and the Confidential Store.

A `Bid` is a data structure encapsulating key information about a transaction on the SUAVE chain.

```go
type Bid struct {
	Id                  BidId            `json:"id"`
	Salt                BidId            `json:"salt"`
	DecryptionCondition uint64           `json:"decryptionCondition"`
	AllowedPeekers      []common.Address `json:"allowedPeekers"`
	AllowedStores       []common.Address `json:"allowedStores"`
	Version             string           `json:"version"`
}
```

Each `Bid` has a `Id` (uuid v5), a `Salt` (random uuid v4),  a `DecryptionCondition`, an array of `AllowedPeekers` and `AllowedStores`, and a `Version`. The `DecryptionCondition` signifies the block number at which the bid can be decrypted and is typically derived from the source contract or may even be a contract itself. The `AllowedPeekers` are the addresses that are permitted to access the data associated with the bid, providing an added layer of access control. The `AllowedStores` are the confidential stores which should be granted access to the bid's data (currently not enforced). The `Version` indicates the version of the protocol used for the bid.

### SUAVE library

Along with the SUAVE precompiles, we provide a convenient wrapper for calling them from Solidity. The [library](suave/sol/libraries/Suave.sol) makes the precompiles easier to call by providing the signatures, and the library functions themselves simply perform a `staticcall` of the requested precompile.

```solidity
library Suave {
    error PeekerReverted(address, bytes);

    type BidId is bytes16;

    struct Bid {
        BidId id;
        BidId salt;
        uint64 decryptionCondition;
        address[] allowedPeekers;
        address[] allowedStores;
        string version;
    }

    function isConfidential() internal view returns (bool b)
    function confidentialInputs() internal view returns (bytes memory)
    function newBid(uint64 decryptionCondition, address[] memory allowedPeekers, string memory BidType) internal view returns (Bid memory)
    function fetchBids(uint64 cond, string memory namespace) internal view returns (Bid[] memory)
    function confidentialStoreStore(BidId bidId, string memory key, bytes memory data) internal view
    function confidentialStoreRetrieve(BidId bidId, string memory key) internal view returns (bytes memory)
    function simulateBundle(bytes memory bundleData) internal view returns (bool, uint64)
    function extractHint(bytes memory bundleData) internal view returns (bytes memory)
    function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bid, string memory namespace) internal view returns (bytes memory, bytes memory)
    function submitEthBlockBidToRelay(string memory relayUrl, bytes memory builderBid) internal view returns (bool, bytes memory)
}
```

### Confidential APIs

Confidential precompiles have access to the following [Confidential APIs](suave/core/types.go) during execution.  
This is subject to change!  

```go
type ConfidentialStoreEngine interface {
    Initialize(bid Bid, creationTx *types.Transaction, key string, value []byte) (Bid, error)
    Store(bidId BidId, sourceTx *types.Transaction, caller common.Address, key string, value []byte) (Bid, error)
    Retrieve(bid BidId, caller common.Address, key string) ([]byte, error)
}

type MempoolBackend interface {
    SubmitBid(Bid) error
    FetchBidById(BidId) (Bid, error)
    FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []Bid
}

type ConfidentialEthBackend interface {
    BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
    BuildEthBlockFromBundles(ctx context.Context, args *BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error)
}
```

### Confidential Store

The Confidential Store is an integral part of the SUAVE chain, designed to facilitate secure and privacy-preserving transactions and smart contract interactions. It functions as a key-value store where users can safely store and retrieve confidential data related to their bids. The Confidential Store restricts access (both reading and writing) only to the allowed peekers of each bid, allowing developers to define the entire data model of their application!

The current, and certainly not final, implementation of the Confidential Store is managed by the `ConfidentialStoreEngine`. The engine consists of a storage backend, which holds the raw data, and a transport topic, which relays synchronization messages between nodes.  
We provide two storage backends to the confidential store engine: the `LocalConfidentialStore`, storing data in memory in a simple dictionary, and `RedisStoreBackend`, storing data in redis. To enable redis as the storage backed, pass redis endpoint via `--suave.confidential.redis-store-endpoint`.  
For synchronization of confidential stores via transport we provide an implementation using a shared Redis PubSub in `RedisPubSubTransport`, as well as a *crude* synchronization protocol. To enable redis transport, pass redis endpoint via `--suave.confidential.redis-transport-endpoint`. Note that Redis transport only synchronizes *current* state, there is no initial synchronization - a newly connected node will not have access to old data.  
Redis as either storage backend or transport is *temporary* and will be removed once we have a well-tested p2p solution.  

![image](suave/docs/confidential_store_engine.png)

The `ConfidentialStoreEngine` provides the following key methods:

- **Initialize**: This method is used to initialize a bid. The method is trusted, meaning it is not directly accessible through precompiles. The method returns the initialized bid, importantly with the `Id` field set.
- **Store**: This method stores a given value under a specified key in a bid's `dataMap`. Access is restricted only to addresses listed in the bid's `AllowedPeekers`.
- **Retrieve**: This method retrieves data associated with a given key from a bid's `dataMap`. Similar to the `Store` method, access is restricted only to addresses listed in the bid's `AllowedPeekers`.

It is important to note that the actual implementation of the Confidential Store will vary depending on future requirements and the privacy mechanisms used.

### SUAVE Mempool

The SUAVE mempool is a temporary storage pool for transactions waiting to be added to the blockchain. This mempool, `MempoolOnConfidentialStore`, operates on the Confidential Store, hence facilitating the privacy-preserving handling of bid transactions. The `MempoolOnConfidentialStore` is designed to handle SUAVE bids, namely the submission, retrieval, and grouping of bids by decryption condition such as block number and protocol. It provides a secure and efficient mechanism for managing these transactions while preserving their confidentiality.

The `MempoolOnConfidentialStore` interacts directly with the `ConfidentialStoreBackend` interface.

```go
type MempoolOnConfidentialStore struct {
	cs suave.ConfidentialStoreBackend
}
```
It is initialized with a predefined `mempoolConfidentialStoreBid` that's only accessible by a particular address `mempoolConfStoreAddr`.

```go
mempoolConfidentialStoreBid = suave.Bid{Id: mempoolConfStoreId, AllowedPeekers: []common.Address{mempoolConfStoreAddr}}
```
The `MempoolOnConfidentialStore` includes three primary methods:

- **SubmitBid**: This method submits a bid to the mempool. The bid is stored in the Confidential Store with its ID as the key. Additionally, the bid is grouped by block number and protocol, which are also stored in the Confidential Store.

- **FetchBidById**: This method retrieves a bid from the mempool using its ID.

- **FetchBidsByProtocolAndBlock**: This method fetches all bids from a particular block that match a specified protocol.

The mempool operates on the underlying Confidential Store, thereby maintaining the confidentiality of the bids throughout the transaction process. As such, all data access is subject to the Confidential Store's security controls, ensuring privacy and integrity. Please note that while this initial implementation provides an idea of the ideal functionality, the final version will most likely incorporate additional features or modifications.

## Notable differences from standard issue go-ethereum

### Changes to RPC methods

1. New `IsConfidential` and `KettleAddress` fields are added to TransactionArgs, used in `eth_sendTransaction` and `eth_call` methods.
If `IsConfidential` is set to true, the call will be performed as a confidential call, using the `KettleAddress` passed in for constructing `ConfidentialComputeRequest`.
`SuaveTransaction` is the result of `eth_sendTransaction`!

2. New optional argument - `confidential_data` is added to `eth_sendRawTransaction`, `eth_sendTransaction` and `eth_call` methods.
The confidential data is made available to the EVM in the confidential mode via a precompile, but does not become a part of the transaction that makes it to chain. This allows performing computation based on confidential data (like simulating a bundle, putting the data into confidential store).


### SuavePrecompiledContract

We introduce a new interface [SuavePrecompiledContract](core/vm/contracts.go) for SUAVE precompiles.

```
type SuavePrecompiledContract interface {
	PrecompiledContract
	RunConfidential(backend *SuaveExecutionBackend, input []byte) ([]byte, error)
}
```

The method `RunConfidential` is invoked during confidential execution, and the suave execution backend which provides access to confidential APIs is passed in as input.

### SUAVE precompile wrapper

We introduce [SuavePrecompiledContractWrapper](core/vm/suave.go) implementing the `PrecompiledContract` interface. The new structure captures the confidential APIs in its constructor, and passes the confidential APIs during the usual contract's `Run` method to a separate method - `RunConfidential`


### SuaveExecutionBackend

We introduce [SuaveExecutionBackend](core/vm/suave.go), which allows access to confidential capabilities during execution:
* Access to confidential APIs
* Access to confidential input
* Caller stack tracing

The backend is only available to confidential execution!

### EVM Interpreter

The [EVM interpreter](core/vm/interpreter.go) is modified to allow for confidential computation's needs:
* We introduce `IsConfidential` to the interpreter's config
* We modify the `Run` function to accept confidential APIs `func (in *EVMInterpreter) Run(*SuaveExecutionBackend, *Contract, []byte, bool) ([]byte, err)`
* We modify the `Run` function to trace the caller stack


Like `eth_sendTransaction`, this method accepts an additional, optional confidential inputs argument.


### Basic Eth block building RPC

We implement two rpc methods that allow building Ethereum blocks from a list of either transactions or bundles: `BuildEth2Block` and `BuildEth2BlockFromBundles`.

These methods are defined in [BlockChainAPI](internal/ethapi/api.go)

```go
func (s *BlockChainAPI) BuildEth2Block(ctx context.Context, buildArgs *types.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
func (s *BlockChainAPI) BuildEth2BlockFromBundles(ctx context.Context, buildArgs *types.BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error)

```

The methods are implemented in [worker](miner/worker.go), by `buildBlockFromTxs` and `buildBlockFromBundles` respectively.

`buildBlockFromTxs` will simply build a block out of the transactions provided, while `buildBlockFromBundles` will in addition forward the block profit to the requested fee recipient, as needed for boost relay payments.


## SUAVE precompiles

Additional precompiles available via the EVM.
Only `IsConfidential` is available during on-chain execution, and simply returns false.

For details and implementation see [contracts_suave.go](core/vm/contracts_suave.go)

### IsConfidential

|   |   |
|---|---|
| Address | `0x42010000` |
| Inputs | None |
| Outputs | boolean |

Outputs whether execution mode is regular (on-chain) or confidential.


### ConfidentialInputs

|   |   |
|---|---|
| Address | `0x42010001` |
| Inputs | None |
| Outputs | bytes |

Outputs the confidential inputs passed in with the confidential computation request.


NOTE: currently all precompiles have access to the data passed in. This might change in the future.

### ConfidentialStore

|   |   |
|---|---|
| Address | `0x42020000` |
| Inputs | (Suave.BidId bidId, string key, bytes data) |
| Outputs | None |


Stores the value in underlying confidential store.
Requires that the caller is present in the `AllowedPeekers` of the bid passed in!

### ConfidentialRetrieve

|   |   |
|---|---|
| Address | `0x42020001` |
| Inputs | (Suave.BidId bidId, string key) |
| Outputs | bytes |


Retrieves the value from underlying confidential store.
Requires that the caller is present in the `AllowedPeekers` of the bid passed in!


### NewBid


|   |   |
|---|---|
| Address | `0x42030000` |
| Inputs | (uint64 decryptionCondition, string[] allowedPeekers) |
| Outputs | Suave.Bid |

Initializes the bid in ConfidentialStore. All bids must be initialized before attempting to store data on them.
Initialization of bids can *only* be done through this precompile!

### FetchBids

|   |   |
|---|---|
| Address | `0x42030001` |
| Inputs | uint64 DecryptionCondition |
| Outputs | Suave.Bid[] |

Returns all bids matching the decryption condition.
This method is subject to change! In the near future bids will be stored in a different way, possibly changing how they are accessed.

### SimulateBundle

|   |   |
|---|---|
| Address | `0x42100000` |
| Inputs | bytes bundleArgs (json) |
| Outputs | (bool success, uint64 egp) |

Simulates the bundle by building a block containing it, returns whether the apply was successful and the EGP of the resulting block.

### ExtractHint

|   |   |
|---|---|
| Address | `0x42100037` |
| Inputs | bytes bundleData (json) |
| Outputs | bytes hintData (json) |

Parses the bundle data and extracts the hint - "To" address and the calldata.

The return structure is encoded as follows:
```
struct {
    To   common.Address
    Data []byte
}
```


### BuildEthBlock

|   |   |
|---|---|
| Address | `0x42100001` |
| Inputs | (Suave.BuildBlockArgs blockArgs, Suave.BidId bidId) |
| Outputs | (bytes builderBid, bytes blockPayload) |


Builds an Ethereum block based on the bid passed in.
The bid can either hold `ethBundle` in its confidential store, or be a "merged bid", ie contain a list of bids in `mergedBids` in its confidential store. The merged bids should themselves hold `ethBundle`.
The block is built *in order*, without any attepmts at re-ordering. The block will contain the transactions unless they failed to apply. The caller should check whether the bids applied successfully, ie whether they revert only if are allowed to.

### SubmitEthBlockBidToRelay

|   |   |
|---|---|
| Address | `0x42100002` |
| Inputs | (string relayUrl, bytes builderBid (json) |
| Outputs | (bytes error) |

Submits provided builderBid to a boost relay. If the submission is successful, returns nothing, otherwise returns an error string.

---

Made with ☀️ by the ⚡🤖 collective.
