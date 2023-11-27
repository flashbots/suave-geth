pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "./utils/Strings.sol";
import "./std.sol";
import "solady/src/utils/LibString.sol";
import "solady/src/utils/JSONParserLib.sol";

library MevBoost {
    using JSONParserLib for *;

    struct Bid {
        uint64 slot;
        bytes32 parentHash;
        bytes32 blockHash;
        bytes builderPubkey; // bytes48
        bytes proposerPubkey; // bytes48
        address proposerFeeRecipient;
        uint64 gasLimit;
        uint64 gasUsed;
        uint256 value;
    }

    struct Withdrawal {
        uint64 index;
        uint64 validatorIndex;
        address addr;
        uint256 amount;
    }

    struct Payload {
        bytes32 parentHash;
        address feeRecipient;
        bytes32 stateRoot;
        bytes32 receiptsRoot;
        bytes logsBloom;
        bytes32 prevRandao;
        uint64 blockNumber;
        uint64 gasLimit;
        uint64 gasUsed;
        uint64 timestamp;
        bytes extraData;
        bytes32 baseFeePerGas;
        bytes32 blockHash;
        bytes[] transactions;
        Withdrawal[] withdrawals;
    }

    struct SubmitBlockRequest {
        Bid bid;
        Payload payload;
        bytes32 signature;
    }

    function submitBlock(string memory url, SubmitBlockRequest memory bundle) internal {
        // TODO
    }
}
