// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "../../../libraries/Suave.sol";

contract Jsonrpc {
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

        Suave.jsonUnmarshal(string(response));
        return response;
    }
}
