// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

//import "../../../libraries/Suave.sol";
import "../jsonrpc/jsonrpc.sol";
import "../../utils/Strings.sol";
import "solady/src/utils/JSONParserLib.sol";

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
        Suave.simpleConsole(abi.encode(2));

        // 1. Make a POST request
        bytes memory response = ss.post("eth_getBlockByNumber", abi.encodePacked('["', Strings.toHexString(number), '", false]'));
        //Suave.simpleConsole(bytes(example.value()));

        string memory s = string(response);
        Suave.simpleConsole(bytes(s));

        // 2. Parse the response. Uncomment this line an everything crashes, 1 is not done!?!?!
        // JSONParserLib.Item memory item = JSONParserLib.parse(s);

        /*
        // loop over the children of item and return the one with key 'result'
        JSONParserLib.Item[] memory children = item.children();
        for (uint256 i = 0; i < children.length; i++) {
            if (Strings.equal(children[i].key(), "result")) {
                Suave.simpleConsole(bytes(children[i].value()));
            }
        }
        */

        return 1;
    }
}
