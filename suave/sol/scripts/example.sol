// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "forge-std/Script.sol";
import "../libraries/SuaveForge.sol";

contract PurchaseEdition is Script {
    Suave suave;

    function run() public {
        suave.confidentialInputs();
        vm.startBroadcast();
        vm.stopBroadcast();
    }
}
