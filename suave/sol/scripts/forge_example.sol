// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";

contract Example is Script {
    address[] public addressList = [Suave.ANYALLOWED];

    function run() public {
        Suave.DataRecord memory record = SuaveForge.newDataRecord(0, addressList, addressList, "default:v0:ethBundles");

        Suave.DataRecord[] memory allShareMatchRecords = SuaveForge.fetchDataRecords(0, "default:v0:ethBundles");
        console.log(allShareMatchRecords.length);

        SuaveForge.confidentialStore(record.id, "a", abi.encodePacked("bbbbbb"));
        bytes memory result = SuaveForge.confidentialRetrieve(record.id, "a");
        console.logBytes(result);
    }
}
