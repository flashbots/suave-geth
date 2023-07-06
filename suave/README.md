## SUAVE node

### Node on the SUAVE testchain network

Simply execute as you would any other network, with the additional `--suave` flag passed in. This will make the node load the default suave chain genesis and configuration.

### Suave ethereum execution node

To run this node as a remote Ethereum RPC backend for SUAVE ethereum offchain API, run the node as you usually would for the Ethereum network you wish to provide RPC to. No additional flags required.
Add this node's RPC to your SUAVE chain node using `suave.eth.remote_endpoint`.

## Offchain transaction types

These new transaction types unify off-chain execution with the SUAVE chain and its mempool. The transactions track the usage of gas in off-chain computation, and contain (or reference) the result of the computation in a chain-friendly manner.

We introduce two new transaction types: `OffchainTxType` and `OffchainExecutedTxType`.  

`OffchainTxType` expresses the request for off-chain computation, and `OffchainExecutedTxType` - its outcome to be applied on-chain.  
`OffchainTxType` is only an intermediary between the user requesting off-chain computation and the RPC node, and is not propagated through the mempool.  

```
type OffchainTx struct {
	ExecutionNode common.Address
	Wrapped  Transaction
}
```

`OffchainExecutedTxType` transactions are propagated through the suave mempool and applied to the suave chain, unifying all of the interactions on suave, both off-chain and on-chain.  

```
type OffchainExecutedTx struct {
	ExecutionNode  common.Address `json:"executionNode" gencodec:"required"`
	Wrapped        Transaction    `json:"wrapped" gencodec:"required"`
	OffchainResult []byte         // Should post-execution transaction be its own transaction type / be the main off-chain transaction type?
	/* Signature fields */
}
```

The off-chain execution result is placed in the `OffchainResult` field, which is further used instead of the original transaction's calldata for on-chain execution.  
The basic flow is as follows:
1. User crafts a usual legacy/dynamic transaction, which calls the off-chain contract of their liking
2. User crafts the `OffchainTx`:
    1. User choses an execution node of their liking, that is an address whose signature over the offchain results will be trusted
    2. User embeds the transaction from (1.) into an `OffchainTx` together with the desired execution node's address
    3. User signs and sends the off-chain transation to an RPC via `eth_sendRawTransaction` (possibly passing in additional condifential data)
3. The RPC executes the transaction in an off-chain mode, providing access to the usual off-chain APIs
4. RPC creates an `OffchainExecutedTx` using the off-chain execution result and the off-chain request `OffchainTx`, signs and submits the whole transaction into the mempool
5. The transaction makes its way into a block, by executing the `OffchainResult` as calldata, as long as the execution node's signature matches the requested execution node in (2a.)


## Off-chain APIs

We introdude [off-chain APIs](core/types.go) that are available to off-chain precompiles through `SuaveOffchainBackend`.
Consult the file for most up-to-date information. For reference:
```
type ConfiendialStoreBackend interface {
	Initialize(bid Bid, key string, value []byte) (Bid, error)
	Store(bidId BidId, caller common.Address, key string, value []byte) (Bid, error)
	Retrieve(bid BidId, caller common.Address, key string) ([]byte, error)
}

type MempoolBackend interface {
	SubmitBid(Bid) error
	FetchBids(blockNumber uint64) []Bid
	FetchBidById(BidId) (Bid, error)
}

type OffchainEthBackend interface {
	BuildEthBlock(ctx context.Context, args *BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)
}
```

## Changes to RPC methods

1. New `IsOffchain` and `ExecutionNode` fields are added to TransactionArgs, used in `eth_sendTransaction` and `eth_call` methods.  
If `IsOffchain` is set to true, the call will be performed as an off-chain call, using the `ExecutionNode` passed in for constructing `OffchainTx`.  
`OffchainExecutedTx` is the result of `eth_sendTransaction`!

2. New optional argument - `confidential_data` is added to `eth_sendRawTransaction`, `eth_sendTransaction` and `eth_call` methods.  
The confidential data is made available to the EVM durin off-chain execution via a precompile, but does not become a part of the transaction that makes it to chain. This allows performing off-chain computation based on confidential data (like simulating a bundle, putting the data into confidential store).

## Other notable differences from standard issue go-ethereum

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

We implement a basic rpc method that builds an Ethereum block from a list of transactions by simply applying them in order.
This method is accessible through [BlockChainAPI](../internal/ethapi/api.go) as `func (s *BlockChainAPI) BuildEth2Block(ctx context.Context, buildArgs *types.BuildBlockArgs, txs types.Transactions) (*engine.ExecutionPayloadEnvelope, error)`
The method is implemeinted in the [worker](../miner/worker.go) as `func (w *worker) buildBlockFromTxs(ctx context.Context, args *types.BuildBlockArgs, txs types.Transactions) (*types.Block, *big.Int, error)`.
The method is exposed to off-chain EVM execution through the `OffchainEthBackend` interface.


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
| Inputs | string bundleArgs (json) |
| Outputs | (bool success, uint64 egp) |

Simulates the bundle by building a block containing it, returns whether the apply was successful and the EGP of the resulting block.

### BuildEthBlock

|   |   |
|---|---|
| Address | `0x42100001` |
| Inputs | (Suave.BuildBlockArgs blockArgs, Suave.BidId bidId) |
| Outputs | (bytes builderBid, bytes blockPayload) |


Builds an Ethereum block based on the bid passed in.  
The bid can either hold `ethBundle` in its confidential store, or be a "merged bid", ie contain a list of bids in `mergedBids` in its confidential store. The merged bids should themselves hold `ethBundle`.
The block is built *in order*, without any attepmts at re-ordering. The block will contain the transactions unless they failed to apply. The caller should check whether the bids applied successfully, ie whether they revert only if are allowed to.  

### Suave library

We provide convenient access to the SUAVE precompiles through the new [SUAVE solidity library](sol/libraries/Suave.sol).

```
library Suave {
    type BidId is bytes16;

    struct Bid {
        BidId id;
        uint64 decryptionCondition;
        address[] allowedPeekers;
    }

    function isOffchain() internal view returns (bool b)
    function confidentialInputs() internal view returns (bytes memory)
    function newBid(uint64 decryptionCondition, address[] memory allowedPeekers) internal view returns (Bid memory)
    function confidentialStoreStore(BidId bidId, string memory key, bytes memory data) internal view
    function confidentialStoreRetrieve(BidId bidId, string memory key) internal view returns (bytes memory)
    function fetchBids(uint64 cond) internal view returns (Bid[] memory)
    function simulateBundle(bytes memory bundleData) internal view returns (bool, uint64)
    function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bid) internal view returns (bytes memory, bytes memory)
}
```
