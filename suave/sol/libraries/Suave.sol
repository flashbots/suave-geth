// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

contract Suave {
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

    function buildEthBlock(BuildBlockArgs param1, BidId param2, string param3) returns (bytes, bytes) {}

    function confidentialInputs() returns (bytes) {}

    function confidentialStoreRetrieve(BidId param1, string param2) returns (bytes) {}

    function confidentialStoreStore(BidId param1, string param2, bytes param3) {}

    function extractHint(bytes param1) returns (bytes) {}

    function fetchBids(uint64 param1, string param2) returns (Bid[]) {}

    function isConfidential() returns (bool) {}

    function newBid(uint64 param1, address[] param2, address[] param3, string param4) returns (Bid) {}

    function simulateBundle(bytes param1) returns (uint64) {}

    function submitEthBlockBidToRelay(string param1, bytes param2) returns (bytes) {}
}
