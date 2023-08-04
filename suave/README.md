# SUAVE Chain

We introduce SUAVE, a new hard-fork based on Shapella. The node, compatible with Ethereum, operates just like any Ethereum network node. The defining feature of SUAVE chain nodes is the off-chain execution capability, enabling users to both define and use smart contracts executed before their transactions are sent to the public chain. Off-chain execution gives contracts access to additional precompiles, expanding MEV-related functionalities. This includes transaction and bundle simulation using geth RPC, building blocks, and sending blocks to boost relays. Nodes providing off-chain execution are known as "execution nodes".

Currently, the SUAVE chain operates as a good old PoA chain (Proof of Authority), with Flashbots acting as the authority.


## Why would I use the suave chain?

The SUAVE Chain, an integral part of the [Centauri release](https://writings.flashbots.net/mevm-suave-centauri-and-beyond), introduces [MEVM](#suave-runtime-mevm), an augmented version of the Ethereum Virtual Machine (EVM) that lowers entry barriers for MEV application development and encourages the creation of innovative mechanisms. It aims to decentralize the traditionally centralized MEV infrastructure by rendering every MEV supply chain primitive programmable in smart contracts on a decentralized blockchain.

SUAVE positions itself as a marketplace for mechanisms, promoting an array of novel applications previously unachievable on Ethereum. It is designed to cater to applications needing private data, intra-block time coordination, and eventually, access to real-time off-chain data, and high compute resources. SUAVE assures privacy, credible computation, and composability, with the MEVM performing computations of sensitive data off-chain.

## How do I use the suave chain?

1. **Deploy smart contracts with off-chain execution capabilities.**  
   Similiar to the Ethereum network, the same rules apply to smart contracts in SUAVE. However, contracts on the SUAVE Chain have the added advantage of being able to access additional precompiles during off-chain request execution. To find the off-chain precompile reference, check the [**SUAVE library**](# SUAVE library).

2. **NEW! Request off-chain execution using the new off-chain transaction feature.**  
   After off-chain execution, the result replaces the calldata for on-chain execution. This grants different behaviors to off-chain and on-chain transactions since off-chain APIs are inaccessible during on-chain computation. 

   Contracts requested using off-chain transactions have access to off-chain data and APIs through SUAVE precompiles. Off-chain execution is *not* reproducible on-chain, thus, users are required to whitelist a specific execution node trusted to provide the result. Eventually proofs and trusted enclaves will help to verify the results of execution.

   See [Off-chain transactions](#Off-chain-transactions) for more details.

### How do I execute a contract off-chain?

Let’s take a look at how you can send an off-chain transaction through an execution node.

1. Pick your favorite execution node. You’ll need its URL and wallet address. Note that the execution node is fully trusted to provide the result of your off-chain computation.

    The execution node ran by Flashbots is at `soon™️` and its wallet address is `0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B`.

2. Craft your off-chain request. This is a regular Ethereum transaction, where you specify the contract address and it’s (public) calldata. I’m assuming you have found or deployed a smart contract which you intend to call off-chain. Don’t sign the transaction quite yet.

    ```go
    allowedPeekers := []common.Address{newBlockBidPeeker, newBundleBidPeeker, buildEthBlockPeeker}
    offchainInnerTx := &types.LegacyTx{
        Nonce:    suaveAccNonce,
        To:       &newBundleBidAddress,
        Value:    nil,
        Gas:      1000000,
        GasPrice: 50,
        Data:     bundleBidAbi.Pack("newBid", targetBlock, allowedPeekers)
    }
    ```

3. Wrap your regular transaction into the new `OffchainTransaction` transaction, and specify the execution node’s wallet address as the `ExecutionNode` field. Sign the transaction with your suave wallet.

    ```go
    offchainTx := types.SignTx(types.NewTx(&types.OffchainTx{
        ExecutionNode: "0x4E2B0c0e428AE1CDE26d5BcF17Ba83f447068E5B",
        Wrapped:       *types.NewTx(&offchainInnerTx),
    }), suaveSigner, privKey)
    ```

4. Submit your off-chain transaction along with your confidential data to the execution node you chose via `eth_sendRawTransaction`.

    ```go
    confidentialDataBytes := hexutil.Encode(ethBundle)
    suaveClient.Call("eth_sendRawTransaction", offchainTx, confidentialDataBytes)
    ```

5. All done! Once the execution node processes your off-chain transaction, the execution node will submit it as `OffchainExecutedTransaction` to the suave chain’s mempool.

For more on off-chain transactions see [Off-chain transactions](# Off-chain transactions).

### How do I run a suave chain node?

1. Check out https://github.com/flashbots/suave-geth on the `main` branch. If you are running a different version, make sure your genesis is the same!
2. Build the client with `make geth`.
3. Run the node. Pass in `--suave` to enable the suave network. Example:

    ```go
    ./build/bin/geth --suave --syncmode=full --http --ws --allow-insecure-unlock --unlock "0xd52d1935D1239ADf94C59fA0F586fE00250694d5"
    ```

4. Do your thing! Run your node as an execution node (see below), or simply process blocks as they come.

### How do I run a suave execution node?

Not all suave chain nodes provide off-chain execution. You’ll need:
- A suave chain node (see above).
- An account. If you are doing this for testing, simply run `geth --suave account new`. Take note of the address.
- Access to Ethereum’s RPC. When starting your node, pass in `--suave.eth.remote_endpoint` to point to your Ethereum RPC for off-chain execution.
    ```go
    ./build/bin/geth --dev --dev.gaslimit 30000000 --datadir dev_data --http --allow-insecure-unlock --unlock "0x<YOUR_PUBKEY>" --ws --suave.eth.remote_endpoint "http://<EXECUTION_NODE_IP>"
    ```
- Users! Advertise your execution node’s address and how to access it.
Note that simply enabling http jsonrpc and allowing direct access might not be the wisest. Look into proxyd and other restricted access solutions.

# Suave technical details

## Suave Runtime (MEVM)

`SuaveOffchainBackend` 🤝 EVM = MEVM

More specifically, `SuaveOffchainBackend` and `Runtime` add functionality to the stock EVM which allows it to interact with off-chain operations.

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
    A-->|11|U[NewRuntimeSuaveOffchainBackend]
    U-->|12|V((SuaveOffchainBackend))
    V-->|13|F
    class A,B,C,D,E,F yellow
    class G,H,I,J,K,L,M,N,O red
    class P,Q green
    class R blue
    class S orange
    class T,U purple
    class Z,V lightgreen
    classDef yellow fill:#f6c347,stroke:#333,stroke-width:2px;
    classDef red fill:#eaafa1,stroke:#333,stroke-width:2px;
    classDef green fill:#96bb7c,stroke:#333,stroke-width:2px;
    classDef blue fill:#f7ea00,stroke:#333,stroke-width:2px;
    classDef orange fill:#ffcdab,stroke:#333,stroke-width:2px;
    classDef purple fill:#b992c9,stroke:#333,stroke-width:2px;
    classDef lightgreen fill:#dbd56e,stroke:#333,stroke-width:2px;
```

The capabilities enabled by this modified runtime are exposed via the APIs `ConfiendialStoreBackend` , `MempoolBackend`, `ConfiendialStoreBackend`, as well as access to `confidentialInputs` to offchain txns and `callerStack`. 

```go
func NewRuntimeSuaveOffchainBackend(evm *EVM, caller common.Address) *SuaveOffchainBackend {
	if !evm.Config.IsOffchain {
		return nil
	}

	return &SuaveOffchainBackend{
		ConfiendialStoreBackend: evm.suaveOffchainBackend.ConfiendialStoreBackend,
		MempoolBackned:          evm.suaveOffchainBackend.MempoolBackned,
		OffchainEthBackend:      evm.suaveOffchainBackend.OffchainEthBackend,
		confidentialInputs:      evm.suaveOffchainBackend.confidentialInputs,
		callerStack:             append(evm.suaveOffchainBackend.callerStack, &caller),
	}
}
```

All of these newly offered APIs are available to your solidity smart contract through the use of precompiles! See below for how offchain executions and smart contracts interact.

## Off-chain execution of smart contracts

Virtual machine inside suave nodes has two modes of operation: off-chain and on-chain. On-chain environment is your usual Ethereum virtual machine environment.

Off-chain environment is available to users through `eth_sendRawTransaction`, `eth_sendTransaction` and `eth_call`. Transactions of a new type - `OffchainTx` - are executed in the off-chain mode, and so are simulations (`eth_call`) requested with a new optional argument `IsOffchain`. For more on off-chain transactions see [Off-chain transactions](#Off-chain-transactions).

The off-chain environment provides additional precompiles, both directly and through a convenient [library](#SUAVE-library). Off-chain execution is *not* verifiable on-chain, instead the result of the off-chain execution is cached in the transaction (`OffchainExecutedTx`). Users submitting off-chain transactions specify which execution nodes they trust with execution, and the execution nodes' signature is used for verifying the transaction on-chain.

The cached result of off-chain execution is used as calldata once the transaction makes its way on the suave chain.

Other than the new precompiles, the off-chain contracts are written as usual in Solidity (or any other language) and compiled to EVM bytecode.

## Off-chain transactions

These new transaction types unify off-chain execution with the SUAVE chain and its mempool. The transactions track the usage of gas in off-chain computation, and contain (or reference) the result of the computation in a chain-friendly manner.

We introduce two new transaction types: `OffchainTx` and `OffchainExecutedTx`.

`OffchainTx` expresses the request for off-chain computation, and `OffchainExecutedTx` - its outcome to be applied on-chain.

![image](docs/offchain_tx_flow.png)

`OffchainTx` is only an intermediary between the user requesting off-chain computation and the RPC node, and is not propagated through the mempool.

```go
type OffchainTx struct {
    ExecutionNode common.Address
    Wrapped  Transaction
}
```

`OffchainExecutedTx` transactions are propagated through the suave mempool and applied to the suave chain, unifying all of the interactions on suave, both off-chain and on-chain.

```go
type OffchainExecutedTx struct {
    ExecutionNode  common.Address
    Wrapped        Transaction
    OffchainResult []byte
    /* Execution node's signature fields */
}
```

The off-chain execution result is placed in the `OffchainResult` field, which is further used instead of the original transaction's calldata for on-chain execution.

The basic flow is as follows:

1. User crafts a usual legacy/dynamic transaction, which calls the off-chain contract of their liking
2. User crafts the `OffchainTx`:
    1. User choses an execution node of their liking, that is an address whose signature over the offchain results will be trusted
    2. User embeds the transaction from (1.) into an `OffchainTx` together with the desired execution node's address
    3. User signs and sends the off-chain transaction to an RPC via `eth_sendRawTransaction` (possibly passing in additional confidential data)
3. The RPC executes the transaction in an off-chain mode, providing access to the usual off-chain APIs
4. RPC creates an `OffchainExecutedTx` using the off-chain execution result and the off-chain request `OffchainTx`, signs and submits the whole transaction into the mempool
5. The transaction makes its way into a block, by executing the `OffchainResult` as calldata, as long as the execution node's signature matches the requested executor node in (2a.)

The user passes in any confidential data through the new `confidential_data` field of the `eth_sendRawTransaction` RPC method. The initial, off-chain execution has access to both the public and confidential data, but only the public data becomes part of the transaction propagated through the mempool.

Architecture reference
![image](docs/execution_node_architecture.png)

Mind, that the results are not reproducible as they are based on confidential data that is dropped after execution, and off-chain data that might change with time. On-chain state transition only depends on the result of the off-chain computation.

## SUAVE Bids

On the SUAVE chain, bids serve as the primary transaction unit, and are used for interactions between smart contracts and the Confidential Store. 

A `Bid` is a data structure encapsulating key information about a transaction on the SUAVE chain.

```go
type Bid struct {
	Id                  BidId            `json:"id"`
	DecryptionCondition uint64           `json:"decryptionCondition"` 
	AllowedPeekers      []common.Address `json:"allowedPeekers"`
	Version             string           `json:"version"`
}
```

Each `Bid` has an `Id`, a `DecryptionCondition`, an array of `AllowedPeekers`, and a `Version`. The `DecryptionCondition` signifies the block number at which the bid can be decrypted and is typically derived from the source contract or may even be a contract itself. The `AllowedPeekers` are the addresses that are permitted to access the data associated with the bid, providing an added layer of access control. The `Version` indicates the version of the protocol used for the bid.

## SUAVE library

Along the SUAVE precompiles, we provide a convenient wrapper for calling them from Solidity. The [library](sol/libraries/Suave.sol) makes the precompiles easier to call by providing the signatures, and the library functions themselves simply perform a `staticcall` of the requested precompile.

```solidity
library Suave {
    error PeekerReverted(address, bytes);

    type BidId is bytes16;

    struct Bid {
        BidId id; 
        uint64 decryptionCondition;
        address[] allowedPeekers;
    }   

    function isOffchain() internal view returns (bool b)
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

## Offchain APIs

Off-chain precompiles have access to the following [off-chain APIs](core/types.go) during execution.

```go
type ConfiendialStoreBackend interface {
    Initialize(bid Bid, key string, value []byte) (Bid, error)
    Store(bidId BidId, caller common.Address, key string, value []byte) (Bid, error)
    Retrieve(bid BidId, caller common.Address, key string) ([]byte, error)
}

type MempoolBackend interface {
    SubmitBid(Bid) error
    FetchBidById(BidId) (Bid, error)
    FetchBidsByProtocolAndBlock(blockNumber uint64, namespace string) []Bid
}

type OffchainEthBackend interface {
    BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
    BuildEthBlockFromBundles(ctx context.Context, args *BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error)
}
```

## Confidential Store

The Confidential Store is an integral part of the SUAVE chain, designed to facilitate secure and privacy-preserving transactions and smart contract interactions. It functions as a key-value store where users can safely store and retrieve confidential data related to their bids. This data is managed through off-chain smart contracts, ensuring optimal confidentiality and security. The Confidential Store restricts access (both reading and writing) only to the allowed peekers of each bid, allowing developers to define the entire data model of their application!

The current, and certainly not final, implementation of the Confidential Store is managed by the `LocalConfidentialStore` struct. It provides thread-safe access to the bids' confidential data. The `LocalConfidentialStore` struct is composed of a mutex lock and a map of bid data, `ACData`, indexed by a `BidId`. 

```go
type LocalConfidentialStore struct {
	lock sync.Mutex
	bids map[suave.BidId]ACData
}
```
`ACData` is another struct that contains a `bid` and a `dataMap`. The `dataMap` is a key-value store that holds the actual confidential data of the bids.

```go
type ACData struct {
	bid     suave.Bid
	dataMap map[string][]byte
}
```

The `LocalConfidentialStore` provides the following key methods:

- **Initialize**: This method is used to initialize a bid with a given `bid.Id`. If no `bid.Id` is provided, a new one is created. The method is trusted, meaning it is not directly accessible through precompiles.
- **Store**: This method stores a given value under a specified key in a bid's `dataMap`. Access is restricted only to addresses listed in the bid's `AllowedPeekers`.
- **Retrieve**: This method retrieves data associated with a given key from a bid's `dataMap`. Similar to the `Store` method, access is restricted only to addresses listed in the bid's `AllowedPeekers`.

It is important to note that the actual implementation of the Confidential Store will vary depending on future requirements and the privacy mechanisms used.

## SUAVE Mempool

The SUAVE mempool is a temporary storage pool for transactions waiting to be added to the blockchain. This mempool, `MempoolOnConfidentialStore`, operates on the Confidential Store, hence facilitating the privacy-preserving handling of bid transactions. The `MempoolOnConfidentialStore` is designed to handle SUAVE bids, namely the submission, retrieval, and grouping of bids by decryption condition such as block number and protocol. It provides a secure and efficient mechanism for managing these transactions while preserving their confidentiality.

The `MempoolOnConfidentialStore` interacts directly with the `ConfiendialStoreBackend` interface.

```go
type MempoolOnConfidentialStore struct {
	cs suave.ConfiendialStoreBackend
}
```
It is initialized with a predefined `mempoolConfidentialStoreBid` that's only accessible by a particular address `mempoolConfStoreAddr`.

```go
mempoolConfidentialStoreBid = suave.Bid{Id: mempoolConfStoreId, AllowedPeekers: []common.Address{mempoolConfStoreAddr}}
```
The `MempoolOnConfidentialStore` includes three primary methods:

- **SubmitBid**: This method submits a bid to the mempool. The bid is stored in the Confidential Store with its ID as the key. Additionally, the bid is grouped by block number and protocol, which are also stored in the Confidential Store.

- **FetchBidById**: This method retrieves a bid from the mempool using its ID.

- **FetchBidsByProtocolAndBlock**: This method fetches all bids from a particular block and matching a specified protocol.

The mempool operates on the underlying Confidential Store, thereby maintaining the confidentiality of the bids throughout the transaction process. As such, all data access is subject to the Confidential Store's security controls, ensuring privacy and integrity. Please note that while this initial implementation provides an idea of the ideal functionality, the final version will most likely incorporate additional features or modifications.

## Notable differences from standard issue go-ethereum

### Changes to RPC methods

1. New `IsOffchain` and `ExecutionNode` fields are added to TransactionArgs, used in `eth_sendTransaction` and `eth_call` methods.  
If `IsOffchain` is set to true, the call will be performed as an off-chain call, using the `ExecutionNode` passed in for constructing `OffchainTx`.  
`OffchainExecutedTx` is the result of `eth_sendTransaction`!

2. New optional argument - `confidential_data` is added to `eth_sendRawTransaction`, `eth_sendTransaction` and `eth_call` methods.  
The confidential data is made available to the EVM durin off-chain execution via a precompile, but does not become a part of the transaction that makes it to chain. This allows performing off-chain computation based on confidential data (like simulating a bundle, putting the data into confidential store).


### SuavePrecompiledContract

We introduce a new interface [SuavePrecompiledContract](../core/vm/contracts.go) for SUAVE precompiles.

```
type SuavePrecompiledContract interface {
	PrecompiledContract
	RunOffchain(backend *SuaveOffchainBackend, input []byte) ([]byte, error)
```

The method `RunOffchain` is invoked during off-chain execution, and the off-chain backend providing access to off-chain APIs is passed in as input.

### Off-chain precompile wrapper

We introduce [OffchainPrecompiledContractWrapper](../core/vm/suave.go) implementing the `PrecompiledContract` interface. The new structure captures the off-chain APIs in its constructor, and passes the off-chain APIs during the usual contract's `Run` method to a separate method - `RunOffchain`


### SuaveOffchainBackend

We introduce [SuaveOffchainBackend](../core/vm/suave.go), which allows access to off-chain capabilities during (off-chain) EVM execution:
* Access to off-chain APIs
* Access to confidential input
* Caller stack tracing

The backend is only available to off-chain execution!

### EVM Interpreter

The [EVM interpreter](../core/vm/interpreter.go) is modified to allow for off-chain computation's needs:
* We introduce `IsOffchain` to the interpreter's config
* We modify the `Run` function to accept off-chain APIs `func (in *EVMInterpreter) Run(*SuaveOffchainBackend, *Contract, []byte, bool) ([]byte, err)`
* We modify the `Run` function to trace the caller stack


Like `eth_sendTransaction`, this method accepts an additional, optional confidential inputs argument.


### Basic Eth block building RPC

We implement two rpc methods that allow building Ethereum blocks from a list of either transactions or bundles: `BuildEth2Block` and `BuildEth2BlockFromBundles`.

This methods are defined in [BlockChainAPI](../internal/ethapi/api.go)  

```go
func (s *BlockChainAPI) BuildEth2Block(ctx context.Context, buildArgs *types.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
func (s *BlockChainAPI) BuildEth2BlockFromBundles(ctx context.Context, buildArgs *types.BuildBlockArgs, bundles []types.SBundle) (*engine.ExecutionPayloadEnvelope, error)

```  

The methods are implemented in [worker](../miner/worker.go), by `buildBlockFromTxs` and `buildBlockFromBundles` respectively.

`buildBlockFromTxs` will simply build a block out of the transactions provided, while `buildBlockFromBundles` will in addition forward the block profit to the requested fee recipient, as needed for boost relay payments.


## Suave precompiles

Additional precompiles available via the EVM.  
Only `IsOffchain` is available during on-chain execution, and simply returns false.  

For details and implementation see [contracts_suave.go](../core/vm/contracts_suave.go)  

### IsOffchain

|   |   |
|---|---|
| Address | `0x42010000` |
| Inputs | None |
| Outputs | boolean |

Outputs whether execution is on- or off-chain.


### ConfidentialInputs

|   |   |
|---|---|
| Address | `0x42010001` |
| Inputs | None |
| Outputs | bytes |

Outputs the confidential inputs passed in with the off-chain transaction.


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
