// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "./Suave.sol";
import "forge-std/Test.sol";
import "forge-std/console.sol";

contract SuaveForge is Suave, Test {
    function buildEthBlock(BuildBlockArgs memory param1, BidId param2, string memory param3)
        external
        view
        returns (bytes memory, bytes memory)
    {}

    function confidentialInputs() external view returns (bytes memory) {
        string[] memory inputs = new string[](3);
        inputs[0] = "echo";
        inputs[1] = "-n";
        // ABI encoded "gm", as a hex string
        inputs[2] = "0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002676d000000000000000000000000000000000000000000000000000000000000";

        bytes memory res = vm.ffi(inputs);
        console.log(res);
    }

    function confidentialStoreRetrieve(BidId param1, string memory param2) external view returns (bytes memory) {}

    function confidentialStoreStore(BidId param1, string memory param2, bytes memory param3) external view {}

    function extractHint(bytes memory param1) external view returns (bytes memory) {}

    function fetchBids(uint64 param1, string memory param2) external view returns (Bid[] memory) {}

    function isConfidential() external view returns (bool) {}

    function newBid(uint64 param1, address[] memory param2, address[] memory param3, string memory param4)
        external
        view
        returns (Bid memory)
    {}

    function simulateBundle(bytes memory param1) external view returns (uint64) {}

    function submitEthBlockBidToRelay(string memory param1, bytes memory param2) external view returns (bytes memory) {}
}
