// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";

contract Example is Script {
    address[] public addressList = [Suave.ANYALLOWED];

    function run() public {
        Suave.Bid memory bid = SuaveForge.newBid(0, addressList, addressList, "default:v0:ethBundles");

        Suave.Bid[] memory allShareMatchBids = SuaveForge.fetchBids(0, "default:v0:ethBundles");
        console.log(allShareMatchBids.length);

        SuaveForge.confidentialStore(bid.id, "a", abi.encodePacked("bbbbbb"));
        bytes memory result = SuaveForge.confidentialRetrieve(bid.id, "a");
        console.logBytes(result);
    }
}
