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
        bytes parentHash;
        bytes blockHash;
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
        bytes parentHash;
        address feeRecipient;
        bytes stateRoot;
        bytes receiptsRoot;
        bytes logsBloom;
        bytes prevRandao;
        uint64 blockNumber;
        uint64 gasLimit;
        uint64 gasUsed;
        uint64 timestamp;
        bytes extraData;
        bytes baseFeePerGas;
        bytes blockHash;
        bytes[] transactions;
        Withdrawal[] withdrawals;
    }

    struct SubmitBlockRequest {
        Bid bid;
        Payload payload;
        bytes signature;
    }

    function encode(Payload memory payload) internal view returns (bytes memory) {
        bytes memory body;

        body = abi.encodePacked(
            '{',
            '"parentHash":"', toMinimalHexString(payload.parentHash), '",'
        );
        body = abi.encodePacked(body,
            '"feeRecipient":"', Strings.toHexString(payload.feeRecipient), '",'
        );
        body = abi.encodePacked(body,
            '"stateRoot":"', toMinimalHexString(payload.stateRoot), '",'
        );
        body = abi.encodePacked(body,
            '"receiptsRoot":"', toMinimalHexString(payload.receiptsRoot), '",'
        );
        body = abi.encodePacked(body,
            '"logsBloom":"', toMinimalHexString(payload.logsBloom), '",'
        );
        body = abi.encodePacked(body,
            '"prevRandao":"', toMinimalHexString(payload.prevRandao), '",'
        );
        body = abi.encodePacked(body,
            '"blockNumber":', Strings.toString(payload.blockNumber), ','
        );
        body = abi.encodePacked(body,
            '"gasLimit":', Strings.toString(payload.gasLimit), ','
        );
        body = abi.encodePacked(body,
            '"gasUsed":', Strings.toString(payload.gasUsed), ','
        );
        body = abi.encodePacked(body,
            '"timestamp":', Strings.toString(payload.timestamp), ','
        );
        body = abi.encodePacked(body,
            '"extraData":"', toMinimalHexString(payload.extraData), '",',
            '"baseFeePerGas":"', toMinimalHexString(payload.baseFeePerGas), '",',
            '"blockHash":"', toMinimalHexString(payload.blockHash), '",',
            '"transactions":['
        );

        for (uint i = 0; i < payload.transactions.length; i++) {
            body = abi.encodePacked(body,
                '"', toMinimalHexString(payload.transactions[i]), '"'
            );
            if (i < payload.transactions.length - 1) {
                body = abi.encodePacked(body, ',');
            }
        }

        body = abi.encodePacked(body, '],');

        body = abi.encodePacked(body,
            '"withdrawals":['
        );

        for (uint i = 0; i < payload.withdrawals.length; i++) {
            body = abi.encodePacked(body,
                '{',
                '"index":', Strings.toString(payload.withdrawals[i].index), ',',
                '"validatorIndex":', Strings.toString(payload.withdrawals[i].validatorIndex), ',',
                '"addr":"', Strings.toHexString(payload.withdrawals[i].addr), '",',
                '"amount":', Strings.toString(payload.withdrawals[i].amount),
                '}'
            );
            if (i < payload.withdrawals.length - 1) {
                body = abi.encodePacked(body, ',');
            }
        }
        body = abi.encodePacked(body, ']}');

        return body;
    }

    function encode(Bid memory bid) internal view returns (bytes memory) {
        bytes memory body;

        body = abi.encodePacked(
            '{',
            '"slot":', Strings.toString(bid.slot), ',',
            '"parentHash":"', toMinimalHexString(bid.parentHash), '",',
            '"blockHash":"', toMinimalHexString(bid.blockHash), '",',
            '"builderPubkey":"', toMinimalHexString(bid.builderPubkey), '",'
        );

        body = abi.encodePacked(body,
            '"proposerPubkey":"', toMinimalHexString(bid.proposerPubkey), '",'
        );

        body = abi.encodePacked(body,
            '"proposerFeeRecipient":"', Strings.toHexString(bid.proposerFeeRecipient), '",'
        );

        body = abi.encodePacked(body,
            '"gasLimit":', Strings.toString(bid.gasLimit), ','
        );

        body = abi.encodePacked(body,
            '"gasUsed":', Strings.toString(bid.gasUsed), ','
        );

        body = abi.encodePacked(body,
            '"value":', Strings.toString(bid.value),
            '}'
        );

        return body;
    }

    function encode(SubmitBlockRequest memory bundle) internal view returns (bytes memory) {
        bytes memory body;

        body = abi.encodePacked(body,
            '{',
            '"bid":', encode(bundle.bid), ',',
            '"payload":', encode(bundle.payload), ',',
            '"signature":"', toMinimalHexString(bundle.signature), '"',
            '}'
        );

        return body;
    }

    function submitBlock(string memory baseUrl, SubmitBlockRequest memory bundle) internal view {
        string memory url = string(abi.encodePacked(baseUrl, "/relay/v1/builder/blocks"));

        Suave.HttpConfig memory config;

        bytes memory request = encode(bundle);
        bytes memory response = Suave.httpPost(url, request, config);
    }

    /// @dev Returns the hexadecimal representation of `value`.
    /// The output is prefixed with "0x".
    /// The output excludes leading "0" from the `toHexString` output.
    /// `0x00: "0x0", 0x01: "0x1", 0x12: "0x12", 0x123: "0x123"`.
    function toMinimalHexString(bytes memory value) private pure returns (string memory str) {
        str = LibString.toHexStringNoPrefix(value);
        /// @solidity memory-safe-assembly
        assembly {
            let o := eq(byte(0, mload(add(str, 0x20))), 0x30) // Whether leading zero is present.
            let strLength := add(mload(str), 2) // Compute the length.
            mstore(add(str, o), 0x3078) // Write the "0x" prefix, accounting for leading zero.
            str := sub(add(str, o), 2) // Move the pointer, accounting for leading zero.
            mstore(str, sub(strLength, o)) // Write the length, accounting for leading zero.
        }
    }
}
