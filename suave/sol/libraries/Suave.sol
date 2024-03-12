// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

/// @notice Library to interact with the Suave MEVM precompiles.
library Suave {
    error PeekerReverted(address, bytes);

    enum CryptoSignature {
        SECP256,
        BLS
    }

    type DataId is bytes16;

    /// @notice Arguments to build the block.
    /// @param slot Slot number of the block
    /// @param proposerPubkey Public key of the proposer
    /// @param parent Hash of the parent block
    /// @param timestamp Timestamp of the block
    /// @param feeRecipient Address of the fee recipient
    /// @param gasLimit Gas limit of the block
    /// @param random Randomness of the block
    /// @param withdrawals List of withdrawals
    /// @param extra Extra data of the block
    /// @param beaconRoot Root of the beacon chain
    /// @param fillPending Whether to fill the block with pending transactions
    struct BuildBlockArgs {
        uint64 slot;
        bytes proposerPubkey;
        bytes32 parent;
        uint64 timestamp;
        address feeRecipient;
        uint64 gasLimit;
        bytes32 random;
        Withdrawal[] withdrawals;
        bytes extra;
        bytes32 beaconRoot;
        bool fillPending;
    }

    /// @notice A record of data stored in the ConfidentialStore.
    /// @param id ID of the data record
    /// @param salt Salt used to derive the encryption key
    /// @param decryptionCondition Up to which block this data record is valid
    /// @param allowedPeekers Addresses which can get data
    /// @param allowedStores Addresses can set data
    /// @param version Namespace of the data record
    struct DataRecord {
        DataId id;
        DataId salt;
        uint64 decryptionCondition;
        address[] allowedPeekers;
        address[] allowedStores;
        string version;
    }

    /// @notice Description of an HTTP request.
    /// @param url Target url of the request
    /// @param method HTTP method of the request
    /// @param headers HTTP Headers
    /// @param body Body of the request (if Post or Put)
    /// @param withFlashbotsSignature Whether to include the Flashbots signature
    struct HttpRequest {
        string url;
        string method;
        string[] headers;
        bytes body;
        bool withFlashbotsSignature;
    }

    /// @notice Result of a simulated transaction.
    /// @param egp Effective Gas Price of the transaction
    /// @param logs Logs emitted during the simulation
    /// @param success Whether the transaction was successful or not
    /// @param error Error message if any
    struct SimulateTransactionResult {
        uint64 egp;
        SimulatedLog[] logs;
        bool success;
        string error;
    }

    /// @notice A log emitted during the simulation of a transaction.
    /// @param data Data of the log
    /// @param addr Address of the contract that emitted the log
    /// @param topics Topics of the log
    struct SimulatedLog {
        bytes data;
        address addr;
        bytes32[] topics;
    }

    /// @notice A withdrawal from the beacon chain.
    /// @param index Index of the withdrawal
    /// @param validator ID of the validator
    /// @param Address Address to withdraw to
    /// @param amount Amount to be withdrawn
    struct Withdrawal {
        uint64 index;
        uint64 validator;
        address Address;
        uint64 amount;
    }

    address public constant ANYALLOWED = 0xC8df3686b4Afb2BB53e60EAe97EF043FE03Fb829;

    address public constant IS_CONFIDENTIAL_ADDR = 0x0000000000000000000000000000000042010000;

    address public constant BUILD_ETH_BLOCK = 0x0000000000000000000000000000000042100001;

    address public constant CONFIDENTIAL_INPUTS = 0x0000000000000000000000000000000042010001;

    address public constant CONFIDENTIAL_RETRIEVE = 0x0000000000000000000000000000000042020001;

    address public constant CONFIDENTIAL_STORE = 0x0000000000000000000000000000000042020000;

    address public constant CONTEXT_GET = 0x0000000000000000000000000000000053300003;

    address public constant DO_HTTPREQUEST = 0x0000000000000000000000000000000043200002;

    address public constant ETHCALL = 0x0000000000000000000000000000000042100003;

    address public constant EXTRACT_HINT = 0x0000000000000000000000000000000042100037;

    address public constant FETCH_DATA_RECORDS = 0x0000000000000000000000000000000042030001;

    address public constant FILL_MEV_SHARE_BUNDLE = 0x0000000000000000000000000000000043200001;

    address public constant NEW_BUILDER = 0x0000000000000000000000000000000053200001;

    address public constant NEW_DATA_RECORD = 0x0000000000000000000000000000000042030000;

    address public constant PRIVATE_KEY_GEN = 0x0000000000000000000000000000000053200003;

    address public constant RANDOM_BYTES = 0x000000000000000000000000000000007770000b;

    address public constant SIGN_ETH_TRANSACTION = 0x0000000000000000000000000000000040100001;

    address public constant SIGN_MESSAGE = 0x0000000000000000000000000000000040100003;

    address public constant SIMULATE_BUNDLE = 0x0000000000000000000000000000000042100000;

    address public constant SIMULATE_TRANSACTION = 0x0000000000000000000000000000000053200002;

    address public constant SUBMIT_BUNDLE_JSON_RPC = 0x0000000000000000000000000000000043000001;

    address public constant SUBMIT_ETH_BLOCK_TO_RELAY = 0x0000000000000000000000000000000042100002;

    /// @notice Returns whether execution is off- or on-chain
    /// @return b Whether execution is off- or on-chain
    function isConfidential() internal returns (bool b) {
        (bool success, bytes memory isConfidentialBytes) = IS_CONFIDENTIAL_ADDR.call("");
        if (!success) {
            revert PeekerReverted(IS_CONFIDENTIAL_ADDR, isConfidentialBytes);
        }
        assembly {
            // Load the length of data (first 32 bytes)
            let len := mload(isConfidentialBytes)
            // Load the data after 32 bytes, so add 0x20
            b := mload(add(isConfidentialBytes, 0x20))
        }
    }

    /// @notice Constructs an Ethereum block based on the provided data records.
    /// @param blockArgs Arguments to build the block
    /// @param dataId ID of the data record with mev-share bundle data
    /// @param namespace deprecated
    /// @return blockBid Block Bid encoded in JSON
    /// @return executionPayload Execution payload encoded in JSON
    function buildEthBlock(BuildBlockArgs memory blockArgs, DataId dataId, string memory namespace)
        internal
        returns (bytes memory, bytes memory)
    {
        (bool success, bytes memory data) = BUILD_ETH_BLOCK.call(abi.encode(blockArgs, dataId, namespace));
        if (!success) {
            revert PeekerReverted(BUILD_ETH_BLOCK, data);
        }

        return abi.decode(data, (bytes, bytes));
    }

    /// @notice Provides the confidential inputs associated with a confidential computation request. Outputs are in bytes format.
    /// @return confindentialData Confidential inputs
    function confidentialInputs() internal returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_INPUTS.call(abi.encode());
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_INPUTS, data);
        }

        return data;
    }

    /// @notice Retrieves data from the confidential store. Also mandates the caller's presence in the `AllowedPeekers` list.
    /// @param dataId ID of the data record to retrieve
    /// @param key Key slot of the data to retrieve
    /// @return value Value of the data
    function confidentialRetrieve(DataId dataId, string memory key) internal returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_RETRIEVE.call(abi.encode(dataId, key));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_RETRIEVE, data);
        }

        return data;
    }

    /// @notice Stores data in the confidential store. Requires the caller to be part of the `AllowedPeekers` for the associated data record.
    /// @param dataId ID of the data record to store
    /// @param key Key slot of the data to store
    /// @param value Value of the data to store
    function confidentialStore(DataId dataId, string memory key, bytes memory value) internal {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE.call(abi.encode(dataId, key, value));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE, data);
        }
    }

    /// @notice Retrieves a value from the context
    /// @param key Key of the value to retrieve
    /// @return value Value of the key
    function contextGet(string memory key) internal returns (bytes memory) {
        (bool success, bytes memory data) = CONTEXT_GET.call(abi.encode(key));
        if (!success) {
            revert PeekerReverted(CONTEXT_GET, data);
        }

        return abi.decode(data, (bytes));
    }

    /// @notice Performs an HTTP request and returns the response. `request` is the request to perform.
    /// @param request Request to perform
    /// @return httpResponse Body of the response
    function doHTTPRequest(HttpRequest memory request) internal returns (bytes memory) {
        (bool success, bytes memory data) = DO_HTTPREQUEST.call(abi.encode(request));
        if (!success) {
            revert PeekerReverted(DO_HTTPREQUEST, data);
        }

        return abi.decode(data, (bytes));
    }

    /// @notice Uses the `eth_call` JSON RPC method to let you simulate a function call and return the response.
    /// @param contractAddr Address of the contract to call
    /// @param input1 Data to send to the contract
    /// @return callOutput Output of the contract call
    function ethcall(address contractAddr, bytes memory input1) internal returns (bytes memory) {
        (bool success, bytes memory data) = ETHCALL.call(abi.encode(contractAddr, input1));
        if (!success) {
            revert PeekerReverted(ETHCALL, data);
        }

        return abi.decode(data, (bytes));
    }

    /// @notice Interprets the bundle data and extracts hints, such as the `To` address and calldata.
    /// @param bundleData Bundle object encoded in JSON
    /// @return hints List of hints encoded in JSON
    function extractHint(bytes memory bundleData) internal returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = EXTRACT_HINT.call(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(EXTRACT_HINT, data);
        }

        return data;
    }

    /// @notice Retrieves all data records correlating with a specified decryption condition and namespace
    /// @param cond Filter for the decryption condition
    /// @param namespace Filter for the namespace of the data records
    /// @return dataRecords List of data records that match the filter
    function fetchDataRecords(uint64 cond, string memory namespace) internal returns (DataRecord[] memory) {
        (bool success, bytes memory data) = FETCH_DATA_RECORDS.call(abi.encode(cond, namespace));
        if (!success) {
            revert PeekerReverted(FETCH_DATA_RECORDS, data);
        }

        return abi.decode(data, (DataRecord[]));
    }

    /// @notice Joins the user's transaction and with the backrun, and returns encoded mev-share bundle. The bundle is ready to be sent via `SubmitBundleJsonRPC`.
    /// @param dataId ID of the data record with mev-share bundle data
    /// @return encodedBundle Mev-Share bundle encoded in JSON
    function fillMevShareBundle(DataId dataId) internal returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = FILL_MEV_SHARE_BUNDLE.call(abi.encode(dataId));
        if (!success) {
            revert PeekerReverted(FILL_MEV_SHARE_BUNDLE, data);
        }

        return data;
    }

    /// @notice Initializes a new remote builder session
    /// @return sessionid ID of the remote builder session
    function newBuilder() internal returns (string memory) {
        (bool success, bytes memory data) = NEW_BUILDER.call(abi.encode());
        if (!success) {
            revert PeekerReverted(NEW_BUILDER, data);
        }

        return abi.decode(data, (string));
    }

    /// @notice Initializes data records within the ConfidentialStore. Prior to storing data, all data records should undergo initialization via this precompile.
    /// @param decryptionCondition Up to which block this data record is valid. Used during `fillMevShareBundle` precompie.
    /// @param allowedPeekers Addresses which can get data
    /// @param allowedStores Addresses can set data
    /// @param dataType Namespace of the data
    /// @return dataRecord Data record that was created
    function newDataRecord(
        uint64 decryptionCondition,
        address[] memory allowedPeekers,
        address[] memory allowedStores,
        string memory dataType
    ) internal returns (DataRecord memory) {
        (bool success, bytes memory data) =
            NEW_DATA_RECORD.call(abi.encode(decryptionCondition, allowedPeekers, allowedStores, dataType));
        if (!success) {
            revert PeekerReverted(NEW_DATA_RECORD, data);
        }

        return abi.decode(data, (DataRecord));
    }

    /// @notice Generates a private key in ECDA secp256k1 format
    /// @param crypto Type of the private key to generate
    /// @return privateKey Hex encoded string of the ECDSA private key. Exactly as a signMessage precompile wants.
    function privateKeyGen(CryptoSignature crypto) internal returns (string memory) {
        (bool success, bytes memory data) = PRIVATE_KEY_GEN.call(abi.encode(crypto));
        if (!success) {
            revert PeekerReverted(PRIVATE_KEY_GEN, data);
        }

        return abi.decode(data, (string));
    }

    /// @notice Generates a number of random bytes, given by the argument numBytes.
    /// @param numBytes Number of random bytes to generate
    /// @return value Randomly-generated bytes
    function randomBytes(uint8 numBytes) internal returns (bytes memory) {
        (bool success, bytes memory data) = RANDOM_BYTES.call(abi.encode(numBytes));
        if (!success) {
            revert PeekerReverted(RANDOM_BYTES, data);
        }

        return abi.decode(data, (bytes));
    }

    /// @notice Signs an Ethereum Transaction, 1559 or Legacy, and returns raw signed transaction bytes. `txn` is binary encoding of the transaction.
    /// @param txn Transaction to sign (RLP encoded)
    /// @param chainId Id of the chain to sign for (hex encoded, with 0x prefix)
    /// @param signingKey Hex encoded string of the ECDSA private key (without 0x prefix)
    /// @return signedTxn Signed transaction encoded in RLP
    function signEthTransaction(bytes memory txn, string memory chainId, string memory signingKey)
        internal
        returns (bytes memory)
    {
        (bool success, bytes memory data) = SIGN_ETH_TRANSACTION.call(abi.encode(txn, chainId, signingKey));
        if (!success) {
            revert PeekerReverted(SIGN_ETH_TRANSACTION, data);
        }

        return abi.decode(data, (bytes));
    }

    /// @notice Signs a message and returns the signature.
    /// @param digest Message to sign
    /// @param crypto Type of the private key to generate
    /// @param signingKey Hex encoded string of the ECDSA private key
    /// @return signature Signature of the message with the private key
    function signMessage(bytes memory digest, CryptoSignature crypto, string memory signingKey)
        internal
        returns (bytes memory)
    {
        require(isConfidential());
        (bool success, bytes memory data) = SIGN_MESSAGE.call(abi.encode(digest, crypto, signingKey));
        if (!success) {
            revert PeekerReverted(SIGN_MESSAGE, data);
        }

        return abi.decode(data, (bytes));
    }

    /// @notice Performs a simulation of the bundle by building a block that includes it.
    /// @param bundleData Bundle encoded in JSON
    /// @return effectiveGasPrice Effective Gas Price of the resultant block
    function simulateBundle(bytes memory bundleData) internal returns (uint64) {
        (bool success, bytes memory data) = SIMULATE_BUNDLE.call(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(SIMULATE_BUNDLE, data);
        }

        return abi.decode(data, (uint64));
    }

    /// @notice Simulates a transaction on a remote builder session
    /// @param sessionid ID of the remote builder session
    /// @param txn Txn to simulate encoded in RLP
    /// @return simulationResult Result of the simulation
    function simulateTransaction(string memory sessionid, bytes memory txn)
        internal
        returns (SimulateTransactionResult memory)
    {
        (bool success, bytes memory data) = SIMULATE_TRANSACTION.call(abi.encode(sessionid, txn));
        if (!success) {
            revert PeekerReverted(SIMULATE_TRANSACTION, data);
        }

        return abi.decode(data, (SimulateTransactionResult));
    }

    /// @notice Submits bytes as JSONRPC message to the specified URL with the specified method. As this call is intended for bundles, it also signs the params and adds `X-Flashbots-Signature` header, as usual with bundles. Regular eth bundles don't need any processing to be sent.
    /// @param url URL to send the request to
    /// @param method JSONRPC method to call
    /// @param params JSONRPC input params encoded in RLP
    /// @return errorMessage Error message if any
    function submitBundleJsonRPC(string memory url, string memory method, bytes memory params)
        internal
        returns (bytes memory)
    {
        require(isConfidential());
        (bool success, bytes memory data) = SUBMIT_BUNDLE_JSON_RPC.call(abi.encode(url, method, params));
        if (!success) {
            revert PeekerReverted(SUBMIT_BUNDLE_JSON_RPC, data);
        }

        return data;
    }

    /// @notice Submits a given builderBid to a mev-boost relay.
    /// @param relayUrl URL of the relay to submit to
    /// @param builderBid Block bid to submit encoded in JSON
    /// @return blockBid Error message if any
    function submitEthBlockToRelay(string memory relayUrl, bytes memory builderBid) internal returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = SUBMIT_ETH_BLOCK_TO_RELAY.call(abi.encode(relayUrl, builderBid));
        if (!success) {
            revert PeekerReverted(SUBMIT_ETH_BLOCK_TO_RELAY, data);
        }

        return data;
    }
}
