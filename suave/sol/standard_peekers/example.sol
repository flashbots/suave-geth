pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "../suave-std/api/ethereum/ethereum.sol";
import "../suave-std/builder/builder.sol";
import "solady/src/utils/JSONParserLib.sol";
import "../suave-std/utils/Strings.sol";
import "solady/src/utils/LibString.sol";

contract ExampleEthCallSource {
    using JSONParserLib for *;

    function callTarget(address target, uint256 expected) public {
        bytes memory output = Suave.ethcall(target, abi.encodeWithSignature("get()"));
        (uint256 num) = abi.decode(output, (uint64));
        require(num == expected);
    }

    function getExample(string memory url) public {
        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "a:b";

        Suave.httpGet(url, config);
    }

    function postExample(string memory url, bytes memory body) public {
        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "b:c";

        Suave.httpPost(url, body, config);
    }

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

    struct SimulateResult {
        // TODO: There are other things here.
        uint256 blockValue;
    }

    function example5(string memory url, Transaction memory txn) public {
        simulateTxn(url, txn);
    }

    function simulateTxn(string memory url, Transaction memory txn) public returns (SimulateResult memory) {
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

        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "Content-Type:application/json";

        txnEncoded = abi.encodePacked('{"jsonrpc": "2.0", "method": "suavex_buildEthBlock", "params": [null, [',txnEncoded,']], "id": 1}');
        bytes memory response = Suave.httpPost(url, bytes(txnEncoded), config);

        // --- DECODE ---
        string memory sss;
        JSONParserLib.Item memory item;

        sss = string(response);
        item = sss.parse();

        uint256 profit = JSONParserLib.parseUintFromHex(removeQuotes(item.children()[2].at('"blockValue"').value()));

        SimulateResult memory result = SimulateResult(profit);
        return result;
    }

    function removeQuotes(string memory input) private pure returns (string memory) {
        bytes memory inputBytes = bytes(input);
        require(inputBytes.length >= 2 && inputBytes[0] == '"' && inputBytes[inputBytes.length - 1] == '"', "Invalid input");

        bytes memory result = new bytes(inputBytes.length - 2);

        for (uint i = 1; i < inputBytes.length - 1; i++) {
            result[i - 1] = inputBytes[i];
        }

        return string(result);
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

    function sampleJSON(string memory url) public {
        Suave.simpleConsole(abi.encode(11));

        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "Content-Type:application/json";

        string memory request = string(abi.encodePacked('{"jsonrpc": "2.0", "method": "eth_getBlockByNumber", "params": ["0x1", false], "id": 1}'));
        bytes memory response = Suave.httpPost(url, bytes(request), config);

        Suave.simpleConsole(response);

        string memory s;
        JSONParserLib.Item memory item;

        s = string(response);
        item = s.parse();

        Suave.simpleConsole(abi.encode(123));
        
        Suave.simpleConsole(bytes(item.children()[0].key()));
        Suave.simpleConsole(bytes(item.children()[0].value()));
        Suave.simpleConsole(bytes(item.children()[1].key()));
        Suave.simpleConsole(bytes(item.children()[1].value()));
        Suave.simpleConsole(bytes(item.children()[2].key()));
        Suave.simpleConsole(bytes(item.children()[2].value()));
        Suave.simpleConsole(bytes(item.children()[2].children()[0].key()));
        Suave.simpleConsole(bytes(item.children()[2].children()[1].key()));
        Suave.simpleConsole(bytes(item.children()[2].children()[2].key()));
        Suave.simpleConsole(bytes(item.children()[2].children()[3].key()));
        Suave.simpleConsole(bytes(item.children()[2].children()[4].key()));
        Suave.simpleConsole(bytes(item.children()[2].children()[5].key()));

        Suave.simpleConsole(abi.encode(22));
    }

    function other() public {

    }

    function returnChildrenKeys(JSONParserLib.Item memory item) private {
        JSONParserLib.Item[] memory children = JSONParserLib.children(item);
        for (uint256 i = 0; i < children.length; i++) {
            Suave.simpleConsole(abi.encode("key"));
            Suave.simpleConsole(bytes(JSONParserLib.key(children[i])));
        }
    }
}

contract BuilderExample {
    function example() public {
        Builder bb = new Builder();
        bb.execTransaction();
        bb.call();
    }
}

contract ExampleEthCallTarget {
    function get() public view returns (uint256) {
        return 101;
    }
}
