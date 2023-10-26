// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "forge-std/Test.sol";
import "forge-std/console.sol";

contract SuaveForge is Test {
    function confidentialInputs() external payable returns (bytes memory) {
        string[] memory inputs = new string[](3);
        inputs[0] = "echo";
        inputs[1] = "-n";
        // ABI encoded "gm", as a hex string
        inputs[2] = "0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002676d000000000000000000000000000000000000000000000000000000000000";

        bytes memory res = vm.ffi(inputs);
        console.logBytes(res);
    }
}
