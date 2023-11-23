pragma solidity ^0.8.8;

import "./utils/Strings.sol";
import "solady/src/utils/LibString.sol";
import "solady/src/utils/JSONParserLib.sol";

library Types {
    using JSONParserLib for *;

    struct Transaction {
        address to;
        uint64 gas;
        uint64 gasPrice;
        uint64 value;
        uint64 nonce;
        bytes data;
        uint64 chainId;
        bytes r;
        bytes s;
        bytes v;
    }

    function encode(Transaction memory txn) internal pure returns (bytes memory) {
        // encode transaction in json
        bytes memory txnEncoded;

        // dynamic fields
        if (txn.data.length != 0) {
            txnEncoded = abi.encodePacked(txnEncoded, '"input":', string(txn.data), ",");
        } else {
            txnEncoded = abi.encodePacked(txnEncoded, '"input": "0x",');
        }
        if (txn.to != address(0)) {
            txnEncoded = abi.encodePacked(txnEncoded, '"to":', Strings.toHexString(txn.to), ",");
        }

        // fixed fields
        txnEncoded = abi.encodePacked(
            txnEncoded,
            '"gas":"', LibString.toMinimalHexString(txn.gas), '",'
            '"gasPrice":"', LibString.toMinimalHexString(txn.gasPrice), '",'
            '"nonce":"', LibString.toMinimalHexString(txn.nonce), '",'
            '"value":"', LibString.toMinimalHexString(txn.value), '",'
            '"chainId":"', LibString.toMinimalHexString(txn.chainId), '",'
            '"r":"', toMinimalHexString(txn.r), '",'
            '"s":"', toMinimalHexString(txn.s), '",'
        );

        txnEncoded = abi.encodePacked(txnEncoded, '"v":"', toMinimalHexString(txn.v), '"');
        txnEncoded = abi.encodePacked("{", txnEncoded, "}");

        return txnEncoded;
    }

    /// @dev Returns the hexadecimal representation of `value`.
    /// The output is prefixed with "0x".
    /// The output excludes leading "0" from the `toHexString` output.
    /// `0x00: "0x0", 0x01: "0x1", 0x12: "0x12", 0x123: "0x123"`.
    function toMinimalHexString(bytes memory value) internal pure returns (string memory str) {
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

    function decodeJsonRPCResponse(bytes memory response) internal pure returns (JSONParserLib.Item memory) {
        string memory sss;
        JSONParserLib.Item memory item;

        sss = string(response);
        item = sss.parse();
        
        return item.at('"result"');
    }
}
