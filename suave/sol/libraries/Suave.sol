// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

library Suave {
    error PeekerReverted(address, bytes);

    type DataId is bytes16;

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

    struct DataRecord {
        DataId id;
        DataId salt;
        uint64 decryptionCondition;
        address[] allowedPeekers;
        address[] allowedStores;
        string version;
    }

    struct HttpRequest {
        string url;
        string method;
        string[] headers;
        bytes body;
        bool withFlashbotsSignature;
    }

    struct SimulateTransactionResult {
        uint64 egp;
        SimulatedLog[] logs;
        bool success;
        string error;
    }

    struct SimulatedLog {
        bytes data;
        address addr;
        bytes32[] topics;
    }

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

    address public constant DO_HTTPREQUEST = 0x0000000000000000000000000000000043200002;

    address public constant ETHCALL = 0x0000000000000000000000000000000042100003;

    address public constant EXTRACT_HINT = 0x0000000000000000000000000000000042100037;

    address public constant FETCH_DATA_RECORDS = 0x0000000000000000000000000000000042030001;

    address public constant FILL_MEV_SHARE_BUNDLE = 0x0000000000000000000000000000000043200001;

    address public constant NEW_BUILDER = 0x0000000000000000000000000000000053200001;

    address public constant NEW_DATA_RECORD = 0x0000000000000000000000000000000042030000;

    address public constant PRIVATE_KEY_GEN = 0x0000000000000000000000000000000053200003;

    address public constant SIGN_ETH_TRANSACTION = 0x0000000000000000000000000000000040100001;

    address public constant SIGN_MESSAGE = 0x0000000000000000000000000000000040100003;

    address public constant SIMULATE_BUNDLE = 0x0000000000000000000000000000000042100000;

    address public constant SIMULATE_TRANSACTION = 0x0000000000000000000000000000000053200002;

    address public constant SUBMIT_BUNDLE_JSON_RPC = 0x0000000000000000000000000000000043000001;

    address public constant SUBMIT_ETH_BLOCK_TO_RELAY = 0x0000000000000000000000000000000042100002;

    // Returns whether execution is off- or on-chain
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

    function confidentialInputs() internal returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_INPUTS.call(abi.encode());
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_INPUTS, data);
        }

        return data;
    }

    function confidentialRetrieve(DataId dataId, string memory key) internal returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_RETRIEVE.call(abi.encode(dataId, key));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_RETRIEVE, data);
        }

        return data;
    }

    function confidentialStore(DataId dataId, string memory key, bytes memory value) internal {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE.call(abi.encode(dataId, key, value));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE, data);
        }
    }

    function doHTTPRequest(HttpRequest memory request) internal returns (bytes memory) {
        (bool success, bytes memory data) = DO_HTTPREQUEST.call(abi.encode(request));
        if (!success) {
            revert PeekerReverted(DO_HTTPREQUEST, data);
        }

        return abi.decode(data, (bytes));
    }

    function ethcall(address contractAddr, bytes memory input1) internal returns (bytes memory) {
        (bool success, bytes memory data) = ETHCALL.call(abi.encode(contractAddr, input1));
        if (!success) {
            revert PeekerReverted(ETHCALL, data);
        }

        return abi.decode(data, (bytes));
    }

    function extractHint(bytes memory bundleData) internal returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = EXTRACT_HINT.call(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(EXTRACT_HINT, data);
        }

        return data;
    }

    function fetchDataRecords(uint64 cond, string memory namespace) internal returns (DataRecord[] memory) {
        (bool success, bytes memory data) = FETCH_DATA_RECORDS.call(abi.encode(cond, namespace));
        if (!success) {
            revert PeekerReverted(FETCH_DATA_RECORDS, data);
        }

        return abi.decode(data, (DataRecord[]));
    }

    function fillMevShareBundle(DataId dataId) internal returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = FILL_MEV_SHARE_BUNDLE.call(abi.encode(dataId));
        if (!success) {
            revert PeekerReverted(FILL_MEV_SHARE_BUNDLE, data);
        }

        return data;
    }

    function newBuilder() internal returns (string memory) {
        (bool success, bytes memory data) = NEW_BUILDER.call(abi.encode());
        if (!success) {
            revert PeekerReverted(NEW_BUILDER, data);
        }

        return abi.decode(data, (string));
    }

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

    function privateKeyGen() internal returns (string memory) {
        (bool success, bytes memory data) = PRIVATE_KEY_GEN.call(abi.encode());
        if (!success) {
            revert PeekerReverted(PRIVATE_KEY_GEN, data);
        }

        return abi.decode(data, (string));
    }

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

    function signMessage(bytes memory digest, string memory signingKey) internal returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = SIGN_MESSAGE.call(abi.encode(digest, signingKey));
        if (!success) {
            revert PeekerReverted(SIGN_MESSAGE, data);
        }

        return abi.decode(data, (bytes));
    }

    function simulateBundle(bytes memory bundleData) internal returns (uint64) {
        (bool success, bytes memory data) = SIMULATE_BUNDLE.call(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(SIMULATE_BUNDLE, data);
        }

        return abi.decode(data, (uint64));
    }

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

    function submitEthBlockToRelay(string memory relayUrl, bytes memory builderBid) internal returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = SUBMIT_ETH_BLOCK_TO_RELAY.call(abi.encode(relayUrl, builderBid));
        if (!success) {
            revert PeekerReverted(SUBMIT_ETH_BLOCK_TO_RELAY, data);
        }

        return data;
    }
}
