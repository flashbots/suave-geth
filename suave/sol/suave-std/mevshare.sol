pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "./utils/Strings.sol";
import "./utils/RLPWriter.sol";
import "solady/src/utils/LibString.sol";
import "solady/src/utils/JSONParserLib.sol";

library MevShare {
    //using JSONParserLib for *;

    struct Bundle {
        string version;
        uint64 inclusionBlock;
        bytes[] bodies;
        bool[] canRevert;
        uint8[] refundPercents;
    }

    // encodes following the schema in https://github.com/flashbots/mev-share/blob/main/specs/bundles/v0.1.md#json-rpc-request-scheme
    function encodeBundle(Bundle memory bundle) internal view returns (bytes memory) {
        require(bundle.bodies.length == bundle.canRevert.length, "MevShare: bodies and canRevert length mismatch");

        bytes memory body = abi.encodePacked(
            '{"jsonrpc":"2.0","method":"mev_sendBundle","params":[{'
        );

        // -> inclusion
        body = abi.encodePacked(
            body,
            '"inclusion":{"block":"', LibString.toMinimalHexString(bundle.inclusionBlock), '"},'
        );

        // -> body
        body = abi.encodePacked(body, '"body":[');

        for (uint i = 0; i < bundle.bodies.length; i++) {
            body = abi.encodePacked(
                body,
                '{"tx":"',
                toMinimalHexString(bundle.bodies[i]),
                '","canRevert":',
                bundle.canRevert[i] ? "true" : "false",
                "}"
            );

            if (i < bundle.bodies.length - 1) {
                body = abi.encodePacked(body, ",");
            }
        }

        body = abi.encodePacked(body, "],");
        
        // -> validity
        body = abi.encodePacked(body, '"validity":{"refund":[');

        for (uint i = 0; i < bundle.refundPercents.length; i++) {
            body = abi.encodePacked(
                body,
                '{"bodyIdx":',
                Strings.toString(i),
                ',"percent":',
                Strings.toString(bundle.refundPercents[i]),
                "}"
            );

            if (i < bundle.refundPercents.length - 1) {
                body = abi.encodePacked(body, ",");
            }
        }

        body = abi.encodePacked(body, "]}");
        return body;
    }

    function sendBundle(string memory url, Bundle memory bundle) internal view {
        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "Content-Type:application/json";

        bytes memory request = encodeBundle(bundle);
        bytes memory response = Suave.httpPost(url, request, config);
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
}
