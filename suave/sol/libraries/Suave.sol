pragma solidity ^0.8.8;

library Suave {
    error PeekerReverted(address, bytes);

    address public constant IS_OFFCHAIN_ADDR =
        0x0000000000000000000000000000000042010000;
    address public constant CONFIDENTIAL_INPUTS =
        0x0000000000000000000000000000000042010001;


    address public constant CONFIDENTIAL_STORE =
        0x0000000000000000000000000000000042020000;
    address public constant CONFIDENTIAL_RETRIEVE =
        0x0000000000000000000000000000000042020001;

    address public constant NEW_BID =
        0x0000000000000000000000000000000042030000;
    address public constant FETCH_BIDS =
        0x0000000000000000000000000000000042030001;

    address public constant SIMULATE_BUNDLE_PEEKER =
        0x0000000000000000000000000000000042100000;
    address public constant EXTRACT_HINT =
        0x0000000000000000000000000000000042100037;
    address public constant BUILD_ETH_BLOCK_PEEKER =
        0x0000000000000000000000000000000042100001;
    address public constant SUBMIT_ETH_BLOCK_BID_TO_RELAY =
        0x0000000000000000000000000000000042100002;

	type BidId is bytes16;

    struct Bid {
        BidId id;
        uint64 decryptionCondition;
        address[] allowedPeekers;
    }

    struct Withdrawal {
         uint64 index;
         uint64 validator;
         address Address;
         uint64 amount;
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

    // Temporay with this call
    function confidentialInputs() internal view returns (bytes memory) {
        (bool success, bytes memory inputs) = CONFIDENTIAL_INPUTS.staticcall("");
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_INPUTS, inputs);
        }
        return inputs;
    }

    // Generates a new, random id and stores the bid
    function newBid(uint64 decryptionCondition, address[] memory allowedPeekers, string memory BidType) internal view returns (Bid memory) {
        (bool success, bytes memory bidBytes) = NEW_BID.staticcall(abi.encode(decryptionCondition, allowedPeekers, BidType));
        if (!success) {
            revert PeekerReverted(NEW_BID, bidBytes);
        }
        return abi.decode(bidBytes, (Bid));
    }

	// Returns bids matching the decryption condition.
	function fetchBids(uint64 cond, string memory namespace) internal view returns (Bid[] memory) {
        (bool success, bytes memory packedBids) = FETCH_BIDS.staticcall(
            abi.encode(cond, namespace)
        );
        if (!success) {
            revert PeekerReverted(FETCH_BIDS, packedBids);
        }
        Bid[] memory bids = abi.decode(packedBids, (Bid[]));
        return bids;
    }

     // Stores confidential bid's data
    // Both the caller AND store peeker have to be allowed on the bid!
    function confidentialStoreStore(BidId bidId, string memory key, bytes memory data) internal view {
        (bool success, bytes memory resp) = CONFIDENTIAL_STORE.staticcall(abi.encode(bidId, key, data));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE, resp);
        }
    }

    function confidentialStoreRetrieve(BidId bidId, string memory key) internal view returns (bytes memory) {
        (bool success, bytes memory data) = CONFIDENTIAL_RETRIEVE.staticcall(abi.encode(bidId, key));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_RETRIEVE, data);
        }
        return data;
    }

    function simulateBundle(bytes memory bundleData) internal view returns (bool, uint64) { // returns egp
        (bool success, bytes memory simResults) = SIMULATE_BUNDLE_PEEKER.staticcall(bundleData);
        if (!success) {
            return (false, 0);
        }
        return (success, abi.decode(simResults, (uint64)));
    }

    function extractHint(bytes memory bundleData) internal view returns (bytes memory) {
		require(isOffchain());

        (bool success, bytes memory data) = EXTRACT_HINT.staticcall(abi.encode(bundleData));

        if (!success) {
            revert PeekerReverted(EXTRACT_HINT, data);
        }
        return data;
    }

	// Builds a block based on the single bid passed in.
	// Returns a bid which contains the built block and can be queried.
	// The input bid may contain some intermediate cached results
	// speeding up the subsequent block building calls.
	function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bid, string memory namespace) internal view returns (bytes memory, bytes memory) {
        (bool success, bytes memory builderBid) = BUILD_ETH_BLOCK_PEEKER.staticcall(
            abi.encode(blockArgs, bid, namespace)
        );
        if (!success) {
            revert PeekerReverted(BUILD_ETH_BLOCK_PEEKER, builderBid);
        }
        return abi.decode(builderBid, (bytes, bytes));
    }

    function submitEthBlockBidToRelay(string memory relayUrl, bytes memory builderBid) internal view returns (bool, bytes memory) {
		require(isOffchain());

        (bool success, bytes memory err) = SUBMIT_ETH_BLOCK_BID_TO_RELAY.staticcall(
            abi.encode(relayUrl, builderBid)
        );

        return (success, err);
    }
}

function idsEqual(Suave.BidId _l, Suave.BidId _r) pure returns (bool) {
    bytes memory l = abi.encodePacked(_l);
    bytes memory r = abi.encodePacked(_r);
    for (uint i = 0; i < l.length; i++) {
        if (bytes(l)[i] != r[i]) {
            return false;
        }
    }

    return true;
}
