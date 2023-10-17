// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

contract Suave {
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

    function newBid(uint64 decryptionCondition, address[] memory allowedPeekers, address[] memory allowedStores, string memory BidType) external view returns (Suave.Bid memory) {}
	function fetchBids(uint64 cond, string memory namespace) external view returns (Suave.Bid[] memory) {}
    function isConfidential() external view returns (bool) {}
    function confidentialInputs() external view returns (bytes memory) {}
    function confidentialStoreStore(Suave.BidId bidId, string memory key, bytes memory data) external view {}
    function confidentialStoreRetrieve(Suave.BidId bidId, string memory key) external view returns (bytes memory) {}
    function simulateBundle(bytes memory bundleData) external view returns (uint64) {}
    function extractHint(bytes memory bundleData) external view returns (bytes memory) {}
	function buildEthBlock(Suave.BuildBlockArgs memory blockArgs, Suave.BidId bid, string memory namespace) external view returns (bytes memory, bytes memory) {}
    function submitEthBlockBidToRelay(string memory relayUrl, bytes memory builderBid) external view returns (bytes memory) {}
}
