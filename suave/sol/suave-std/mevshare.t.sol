// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "./mevshare.sol";

contract MevShareTest is Test {
    function testEncodeMevShare() public {
        MevShare.Bundle memory bundle;
        bundle.version = "";
        bundle.inclusionBlock = 1;

        bundle.bodies = new bytes[](1);
        bundle.bodies[0] = abi.encode(1234);

        bundle.canRevert = new bool[](1);
        bundle.canRevert[0] = true;

        bundle.refundPercents = new uint8[](1);
        bundle.refundPercents[0] = 10;

        bytes memory res = MevShare.encodeBundle(bundle);
        assertEq(string(res), '{"jsonrpc":"2.0","method":"mev_sendBundle","params":[{"inclusion":{"block":"0x1"},"body":[{"tx":"0x0000000000000000000000000000000000000000000000000000000000004d2","canRevert":true}],"validity":{"refund":[{"bodyIdx":0,"percent":10}]}');
    }
}
