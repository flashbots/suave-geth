// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

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

    struct BuildBlockArgs {
        uint64 slot;
        bytes proposerPubkey;
        bytes32 parent;
        uint64 timestamp;
        address feeRecipient;
        uint64 gasLimit;
        bytes32 random;
        Withdrawal[] withdrawals;
    }

    struct Withdrawal {
        uint64 index;
        uint64 validator;
        address Address;
        uint64 amount;
    }

    address public constant IS_CONFIDENTIAL_ADDR = 0x0000000000000000000000000000000042010000;

    address public constant BUILD_ETH_BLOCK = 0x0000000000000000000000000000000042100001;

    address public constant CONFIDENTIAL_INPUTS = 0x0000000000000000000000000000000042010001;

    address public constant CONFIDENTIAL_STORE_RETRIEVE = 0x0000000000000000000000000000000042020001;

    address public constant CONFIDENTIAL_STORE_STORE = 0x0000000000000000000000000000000042020000;

    address public constant ETHCALL = 0x0000000000000000000000000000000042100003;

    address public constant EXTRACT_HINT = 0x0000000000000000000000000000000042100037;

    address public constant FETCH_BIDS = 0x0000000000000000000000000000000042030001;

    address public constant NEW_BID = 0x0000000000000000000000000000000042030000;

    address public constant SIMULATE_BUNDLE = 0x0000000000000000000000000000000042100000;

    address public constant SUBMIT_ETH_BLOCK_BID_TO_RELAY = 0x0000000000000000000000000000000042100002;

    address public constant SUBMIT_ETH_BUNDLE_TO_BUILDER = 0x0000000000000000000000000000000042100004;

    // Returns whether execution is off- or on-chain
    function isConfidential() internal view returns (bool b) {
        (bool success, bytes memory isConfidentialBytes) = IS_CONFIDENTIAL_ADDR.staticcall("");
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

    function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bidId, string memory namespace)
        internal
        view
        returns (bytes memory, bytes memory)
    {
        (bool success, bytes memory data) = BUILD_ETH_BLOCK.staticcall(abi.encode(blockArgs, bidId, namespace));
        if (!success) {
            revert PeekerReverted(BUILD_ETH_BLOCK, data);
        }

        return abi.decode(data, (bytes, bytes));
    }

    function confidentialInputs() internal view returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_INPUTS.staticcall(abi.encode());
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_INPUTS, data);
        }

        return data;
    }

    function confidentialStoreRetrieve(BidId bidId, string memory key) internal view returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE_RETRIEVE.staticcall(abi.encode(bidId, key));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE_RETRIEVE, data);
        }

        return data;
    }

    function confidentialStoreStore(BidId bidId, string memory key, bytes memory data1) internal view {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE_STORE.staticcall(abi.encode(bidId, key, data1));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE_STORE, data);
        }
    }

    function ethcall(address contractAddr, bytes memory input1) internal view returns (bytes memory) {
        (bool success, bytes memory data) = ETHCALL.staticcall(abi.encode(contractAddr, input1));
        if (!success) {
            revert PeekerReverted(ETHCALL, data);
        }

        return abi.decode(data, (bytes));
    }

    function extractHint(bytes memory bundleData) internal view returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = EXTRACT_HINT.staticcall(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(EXTRACT_HINT, data);
        }

        return data;
    }

    function fetchBids(uint64 cond, string memory namespace) internal view returns (Bid[] memory) {
        (bool success, bytes memory data) = FETCH_BIDS.staticcall(abi.encode(cond, namespace));
        if (!success) {
            revert PeekerReverted(FETCH_BIDS, data);
        }

        return abi.decode(data, (Bid[]));
    }

    function newBid(
        uint64 decryptionCondition,
        address[] memory allowedPeekers,
        address[] memory allowedStores,
        string memory bidType
    ) internal view returns (Bid memory) {
        (bool success, bytes memory data) =
            NEW_BID.staticcall(abi.encode(decryptionCondition, allowedPeekers, allowedStores, bidType));
        if (!success) {
            revert PeekerReverted(NEW_BID, data);
        }

        return abi.decode(data, (Bid));
    }

    function simulateBundle(bytes memory bundleData) internal view returns (uint64) {
        (bool success, bytes memory data) = SIMULATE_BUNDLE.staticcall(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(SIMULATE_BUNDLE, data);
        }

        return abi.decode(data, (uint64));
    }

    function submitEthBlockBidToRelay(string memory relayUrl, bytes memory builderBid)
        internal
        view
        returns (bytes memory)
    {
        require(isConfidential());
        (bool success, bytes memory data) = SUBMIT_ETH_BLOCK_BID_TO_RELAY.staticcall(abi.encode(relayUrl, builderBid));
        if (!success) {
            revert PeekerReverted(SUBMIT_ETH_BLOCK_BID_TO_RELAY, data);
        }

        return data;
    }

    function submitEthBundleToBuilder(string memory builderUrl, BidId bid) internal view returns (bytes memory) {
        require(isConfidential());
        (bool success, bytes memory data) = SUBMIT_ETH_BUNDLE_TO_BUILDER.staticcall(abi.encode(builderUrl, bid));
        if (!success) {
            revert PeekerReverted(SUBMIT_ETH_BUNDLE_TO_BUILDER, data);
        }

        return data;
    }
}
