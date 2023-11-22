// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "../../../libraries/Suave.sol";
//import "solady/src/utils/JSONParserLib.sol";
//import "../../utils/Strings.sol";

contract Jsonrpc {
    //using JSONParserLib for *;

    string url;

    constructor(string memory _url) {
        url = _url;
    }

    function post(string memory method, bytes memory body) external view returns (bytes memory) {
        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "Content-Type:application/json";

        if (body.length == 0) {
            body = bytes("[]");
        }
        string memory request = string(abi.encodePacked('{"jsonrpc": "2.0", "method": "', method, '", "params": ', string(body), ', "id": 1}'));
        bytes memory response = Suave.httpPost(url, bytes(request), config);

        return response;

        /*
        string memory s = string(response);
        JSONParserLib.Item memory item;

        item = s.parse();

        // loop over the children of item and return the one with key 'result'
        JSONParserLib.Item[] memory children = item.children();
        for (uint256 i = 0; i < children.length; i++) {
            if (Strings.equal(children[i].key(), "result")) {
                return children[i];
            }
        }
        */

        revert("not found");
    }
}
