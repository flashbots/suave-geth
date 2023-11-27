// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import "forge-std/Test.sol";
import "forge-std/console.sol";
import "./std.sol";

contract TransactionTest is Test {
    function testIncrementxxx() public view {
        Types.Transaction memory t = Types.Transaction({
            to: address(0),
            gas: 0,
            gasPrice: 0,
            value: 0,
            nonce: 0,
            data: bytes(""),
            chainId: 0,
            v: abi.encode(1),
            r: abi.encode(1),
            s: abi.encode(1)
        });

        bytes memory x = Types.encode(t);
        console.log(string(x));

        bytes memory rlp = Types.encodeRLP(t);
        console.logBytes(rlp);
    }

    function testBundle() public view {
        Types.Transaction memory t = Types.Transaction({
            to: address(0),
            gas: 0,
            gasPrice: 0,
            value: 0,
            nonce: 0,
            data: bytes(""),
            chainId: 0,
            v: abi.encode(1),
            r: abi.encode(1),
            s: abi.encode(1)
        });

        Types.Transaction[] memory txs = new Types.Transaction[](1);
        txs[0] = t;

        Types.SBundle memory b = Types.SBundle({
            txs: txs,
            blockNumber: 0,
            revertingHashes: new bytes32[](1),
            refundPercent: 0
        });

        bytes memory x = Types.encode(b);
        console.log(string(x));
    }
}
