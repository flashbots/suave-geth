// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "../../../libraries/Suave.sol";
import "../jsonrpc/jsonrpc.sol";
import "../../utils/Strings.sol";

contract Ethereum {
    Jsonrpc ss;

    constructor(string memory url) {
        ss = new Jsonrpc(url);
    }

    function version() public view returns (uint64) {
        ss.post("net_version", abi.encode());
        return 1;
    }

    function blockNumber() public view returns (uint64) {
        ss.post("eth_blockNumber", abi.encode());
        return 1;
    }
    
    function blockByNumber(uint64 number) public view returns (uint64) {
        ss.post("eth_getBlockByNumber", abi.encodePacked('["', Strings.toHexString(number), '", false]'));
        return 1;
    }
}
