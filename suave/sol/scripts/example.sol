// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "forge-std/Script.sol";
import "../libraries/Suave.sol";
import "../libraries/SuaveForge.sol";

contract Example is Script {
    function run() public {
        SuaveForge suave = new SuaveForge();
        bytes memory confidentialInputs = suave.confidentialInputs();

        console.logBytes(confidentialInputs);
        console.log(1);
    }
}
