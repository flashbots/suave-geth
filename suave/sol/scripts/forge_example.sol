// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";

contract Example is Script {
    address[] public addressList = [0xC8df3686b4Afb2BB53e60EAe97EF043FE03Fb829];

    function run() public {
        Suave.Bid memory bid = SuaveForge.newBid(0, addressList, addressList, "default:v0:ethBundles");

        Suave.Bid[] memory allShareMatchBids = SuaveForge.fetchBids(0, "default:v0:ethBundles");
        console.log(allShareMatchBids.length);

        SuaveForge.confidentialStoreStore(bid.id, "a", abi.encodePacked("bbbbbb"));
        bytes memory result = SuaveForge.confidentialStoreRetrieve(bid.id, "a");
        console.logBytes(result);

        uint256 EthPriceNow = SuaveForge.getBinancePrice("ETHUSDT");
        // uint256 EthPriceNow = 0;
        console.log(EthPriceNow);
    }
}
