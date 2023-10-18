// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "./Suave.sol";
import "forge-std/Test.sol";

contract SuaveForge is Test {
    function doForge(bytes4 sig, bytes memory input) public returns (bytes memory) {
        bytes memory data = bytes.concat(sig, input);

        console.logBytes(data);
        
        string[] memory inputs = new string[](3);
        inputs[0] = "suave";
        inputs[1] = "forge";
        inputs[2] = string(data);

        bytes memory res = vm.ffi(inputs);
        return res;
    }

    function buildEthBlock(Suave.BuildBlockArgs memory param1, Suave.BidId param2, string memory param3)
        public
        view
        returns (bytes memory, bytes memory)
    {}

    function confidentialInputs() public returns (bytes memory) {
        bytes memory result = doForge(this.confidentialInputs.selector, abi.encode());

        console.logBytes(result);
        
        return new bytes(0);
    }

    function confidentialStoreRetrieve(Suave.BidId param1, string memory param2)
        public
        view
        returns (bytes memory)
    {}

    function confidentialStoreStore(Suave.BidId param1, string memory param2, bytes memory param3) public view {}

    function extractHint(bytes memory param1) public view returns (bytes memory) {}

    function fetchBids(uint64 param1, string memory param2) public view returns (Suave.Bid[] memory) {}

    function isConfidential() public view returns (bool) {}

    function newBid(uint64 param1, address[] memory param2, address[] memory param3, string memory param4)
        public
        view
        returns (Suave.Bid memory)
    {}

    function simulateBundle(bytes memory param1) public view returns (uint64) {}

    function submitEthBlockBidToRelay(string memory param1, bytes memory param2)
        public
        view
        returns (bytes memory)
    {}
}
