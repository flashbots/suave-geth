Formatted output:
pragma solidity ^0.8.8;

library Suave {
    error PeekerReverted(address, bytes);

    struct Bid {
        uint256 Amount;
        uint256 Price;
    }

    struct Withdrawal {
        uint256 Index;
        uint256 Validator;
        string Address;
        uint256 Amount;
    }

    struct BuildBlockArgs {
        uint256 Slot;
        bytes ProposerPubkey;
        bytes32 Parent;
        uint256 Timestamp;
        string FeeRecipient;
        uint256 GasLimit;
        bytes32 Random;
        Withdrawal[] Withdrawals;
    }

    address public constant IS_OFFCHAIN_ADDR = 0x0000000000000000000000000000000042010000;

    address public constant CONFIDENTIAL_INPUTS = 0x0000000000000000000000000000000042010001;

    address public constant NEW_BID = 0x0000000000000000000000000000000042030000;

    address public constant FETCH_BIDS = 0x0000000000000000000000000000000042030001;

    address public constant CONFIDENTIAL_STORE_STORE = 0x0000000000000000000000000000000042020000;

    address public constant CONFIDENTIAL_STORE_RETRIEVE = 0x0000000000000000000000000000000042020001;

    address public constant SIMULATE_BUNDLE = 0x0000000000000000000000000000000042100000;

    address public constant EXTRACT_HINT = 0x0000000000000000000000000000000042100037;

    address public constant BUILD_ETH_BLOCK = 0x0000000000000000000000000000000042100001;

    address public constant SUBMIT_ETH_BLOCK_BID_TO_RELAY = 0x0000000000000000000000000000000042100002;

    // Returns whether execution is off- or on-chain
    function isOffchain() internal view returns (bool b) {
        (bool success, bytes memory isOffchainBytes) = IS_OFFCHAIN_ADDR.staticcall("");
        if (!success) {
            revert PeekerReverted(IS_OFFCHAIN_ADDR, isOffchainBytes);
        }
        assembly {
            // Load the length of data (first 32 bytes)
            let len := mload(isOffchainBytes)
            // Load the data after 32 bytes, so add 0x20
            b := mload(add(isOffchainBytes, 0x20))
        }
    }

    function confidentialInputs() internal view returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_INPUTS.staticcall(abi.encode());
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_INPUTS, data);
        }
        return data;
    }

    function newBid(uint256 decryptionCondition, address[] memory allowedPeekers, string memory bidType)
        internal
        view
        returns (Bid memory)
    {
        (bool success, bytes memory data) = NEW_BID.staticcall(abi.encode(decryptionCondition, allowedPeekers, bidType));
        if (!success) {
            revert PeekerReverted(NEW_BID, data);
        }
        return abi.decode(data, (Bid));
    }

    function fetchBids(uint256 cond, string memory namespace) internal view returns (Bid[] memory) {
        (bool success, bytes memory data) = FETCH_BIDS.staticcall(abi.encode(cond, namespace));
        if (!success) {
            revert PeekerReverted(FETCH_BIDS, data);
        }
        return abi.decode(data, (Bid[]));
    }

    function confidentialStoreStore(bytes16 bidId, string memory key, bytes memory data) internal view {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE_STORE.staticcall(abi.encode(bidId, key, data));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE_STORE, data);
        }
    }

    function confidentialStoreRetrieve(bytes16 bidId, string memory key) internal view returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_STORE_RETRIEVE.staticcall(abi.encode(bidId, key));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE_RETRIEVE, data);
        }
        return data;
    }

    function simulateBundle(bytes memory bundleData) internal view returns (uint256) {
        (bool success, bytes memory data) = SIMULATE_BUNDLE.staticcall(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(SIMULATE_BUNDLE, data);
        }
        return abi.decode(data, (uint256));
    }

    function extractHint(bytes memory bundleData) internal view returns (bytes memory) {
        (bool success, bytes memory data) = EXTRACT_HINT.staticcall(abi.encode(bundleData));
        if (!success) {
            revert PeekerReverted(EXTRACT_HINT, data);
        }
        return data;
    }

    function buildEthBlock(BuildBlockArgs memory blockArgs, bytes16 bidId, string memory namespace)
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

    function submitEthBlockBidToRelay(string memory relayUrl, bytes memory builderBid)
        internal
        view
        returns (bytes memory)
    {
        (bool success, bytes memory data) = SUBMIT_ETH_BLOCK_BID_TO_RELAY.staticcall(abi.encode(relayUrl, builderBid));
        if (!success) {
            revert PeekerReverted(SUBMIT_ETH_BLOCK_BID_TO_RELAY, data);
        }
        return data;
    }
}


