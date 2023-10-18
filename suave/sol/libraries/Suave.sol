// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

contract Suave {
    error PeekerReverted(address, bytes);

    struct Bid {
        BidId id;
        BidId salt;
        uint64 decryptionCondition;
        address[] allowedPeekers;
        address[] allowedStores;
        string version;
    }

    type BidId is bytes16;

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

    function buildEthBlock(BuildBlockArgs memory param1, BidId param2, string memory param3)
        external
        view
        returns (bytes memory, bytes memory)
    {}

    function confidentialInputs() external view returns (bytes memory) {}

    function confidentialStoreRetrieve(BidId param1, string memory param2) external view returns (bytes memory) {}

    function confidentialStoreStore(BidId param1, string memory param2, bytes memory param3) external view {}

    function extractHint(bytes memory param1) external view returns (bytes memory) {}

    function fetchBids(uint64 param1, string memory param2) external view returns (Bid[] memory) {}

    function isConfidential() external view returns (bool) {}

    function newBid(uint64 param1, address[] memory param2, address[] memory param3, string memory param4)
        external
        view
        returns (Bid memory)
    {}

    function simulateBundle(bytes memory param1) external view returns (uint64) {}

    function submitEthBlockBidToRelay(string memory param1, bytes memory param2) external view returns (bytes memory) {}
}
