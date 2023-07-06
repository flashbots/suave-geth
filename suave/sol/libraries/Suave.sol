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

    // Not implemented yet!
    address public constant SIMULATE_BUNDLE_PEEKER =
        0x0000000000000000000000000000000042100000;
    address public constant BUILD_ETH_BLOCK_PEEKER =
        0x0000000000000000000000000000000042100001;

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
        require(success);
        assembly {
            // Load the length of data (first 32 bytes)
            let len := mload(isOffchainBytes)
            // Load the data after 32 bytes, so add 0x20
            b := mload(add(isOffchainBytes, 0x20))
        }
    }

    function confidentialInputs() internal view returns (bytes memory) {
		require(isOffchain());

        (bool success, bytes memory inputs) = CONFIDENTIAL_INPUTS.staticcall("");
        require(success);
        return inputs;
    }

    // Generates a new, random id and stores the bid
    function newBid(uint64 decryptionCondition, address[] memory allowedPeekers) internal view returns (Bid memory) {
		require(isOffchain());

        (bool success, bytes memory bidBytes) = NEW_BID.staticcall(abi.encode(decryptionCondition, allowedPeekers));
        if (!success) {
            revert PeekerReverted(NEW_BID, bidBytes);
        }
        return abi.decode(bidBytes, (Bid));
    }

    // Stores confidential bid's data
    // Both the caller AND store peeker have to be allowed on the bid!
    function confidentialStoreStore(BidId bidId, string memory key, bytes memory data) internal view {
		require(isOffchain());

        (bool success, bytes memory resp) = CONFIDENTIAL_STORE.staticcall(abi.encode(bidId, key, data));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_STORE, resp);
        }
    }

    function confidentialStoreRetrieve(BidId bidId, string memory key) internal view returns (bytes memory) {
		require(isOffchain());

        (bool success, bytes memory data) = CONFIDENTIAL_RETRIEVE.staticcall(abi.encode(bidId, key));
        if (!success) {
            revert PeekerReverted(CONFIDENTIAL_RETRIEVE, data);
        }
        return data;
    }

	// Returns bids matching the decryption condition.
	function fetchBids(uint64 cond) internal view returns (Bid[] memory) {
		require(isOffchain());

        (bool success, bytes memory packedBids) = FETCH_BIDS.staticcall(
            abi.encode(cond)
        );
        if (!success) {
            revert PeekerReverted(FETCH_BIDS, packedBids);
        }
        Bid[] memory bids = abi.decode(packedBids, (Bid[]));
        return bids;
    }

    function simulateBundle(bytes memory bundleData) internal view returns (bool, uint64) { // returns egp
		require(isOffchain());

        (bool success, bytes memory simResults) = SIMULATE_BUNDLE_PEEKER.staticcall(bundleData);
        return (success, abi.decode(simResults, (uint64)));
    }

	// Builds a block based on the single bid passed in.
	// Returns a bid which contains the built block and can be queried.
	// The input bid may contain some intermediate cached results
	// speeding up the subsequent block building calls.
	function buildEthBlock(BuildBlockArgs memory blockArgs, BidId bid) internal view returns (bytes memory, bytes memory) {
		require(isOffchain());

        (bool success, bytes memory builderBid) = BUILD_ETH_BLOCK_PEEKER.staticcall(
            abi.encode(blockArgs, bid)
        );
        if (!success) {
            revert PeekerReverted(BUILD_ETH_BLOCK_PEEKER, builderBid);
        }
        return abi.decode(builderBid, (bytes, bytes));
    }
}
