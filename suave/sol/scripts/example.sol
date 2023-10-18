// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "forge-std/Script.sol";
import "../libraries/Suave.sol";
import "../libraries/SuaveForge.sol";

contract PurchaseEdition is Script {
    function run() public {
        Suave suave = SuaveForge(0x1100000000000000000000000000000042100002);
        suave.confidentialInputs();
    }
}
