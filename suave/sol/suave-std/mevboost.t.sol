// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "./mevboost.sol";

contract MevBoostTest is Test {
    function testEncodeBid() public {
        MevBoost.Bid memory bid;

        bytes memory res = MevBoost.encode(bid);
        assertEq(string(res), '{"slot":0,"parentHash":"0x","blockHash":"0x","builderPubkey":"0x","proposerPubkey":"0x","proposerFeeRecipient":"0x0000000000000000000000000000000000000000","gasLimit":0,"gasUsed":0,"value":0}');
    }

    function testEncodePayload() public {
        MevBoost.Payload memory payload;

        bytes memory res = MevBoost.encode(payload);
        assertEq(string(res), '{"parentHash":"0x","feeRecipient":"0x0000000000000000000000000000000000000000","stateRoot":"0x","receiptsRoot":"0x","logsBloom":"0x","prevRandao":"0x","blockNumber":0,"gasLimit":0,"gasUsed":0,"timestamp":0,"extraData":"0x","baseFeePerGas":"0x","blockHash":"0x","transactions":[],"withdrawals":[]}');
    }
}
