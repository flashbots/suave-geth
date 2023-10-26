// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";

contract Example is Script {
    address[] public addressList = [0x0000000000000000000000000000000000000000];

    function run() public {
        Suave.Bid memory bid = SuaveForge.newBid(0, addressList, addressList, "default:v0:ethBundles");

        Suave.Bid[] memory allShareMatchBids = SuaveForge.fetchBids(0, "default:v0:ethBundles");
        console.log(allShareMatchBids.length);

        SuaveForge.confidentialStoreStore(bid.id, "a", abi.encodePacked("bbbbbb"));
        bytes memory result = SuaveForge.confidentialStoreRetrieve(bid.id, "a");
        console.logBytes(result);
    }
}
