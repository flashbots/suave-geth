pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "./utils/Strings.sol";
import "./std.sol";
import "solady/src/utils/LibString.sol";
import "solady/src/utils/JSONParserLib.sol";

library Suavex {
    using JSONParserLib for *;

    struct SimulateResult {
        // TODO: There are other things here.
        uint256 blockValue;
    }

    function simulateTxn(string memory url, Types.Transaction[] memory txns) internal view returns (SimulateResult memory) {
        // encode both transactions as an array
        bytes memory txnEncoded;
        for (uint i = 0; i < txns.length; i++) {
            if (i == 0) {
                txnEncoded = abi.encodePacked(Types.encode(txns[i]));
            } else {
                txnEncoded = abi.encodePacked(Types.encode(txns[i]), ",");
            }
        }
        
        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "Content-Type:application/json";

        txnEncoded = abi.encodePacked('{"jsonrpc": "2.0", "method": "suavex_buildEthBlock", "params": [null, [',txnEncoded,']], "id": 1}');
        bytes memory response = Suave.httpPost(url, bytes(txnEncoded), config);

        JSONParserLib.Item memory body = Types.decodeJsonRPCResponse(response);
        uint256 profit = JSONParserLib.parseUintFromHex(removeQuotes(body.at('"blockValue"').value()));

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
}
