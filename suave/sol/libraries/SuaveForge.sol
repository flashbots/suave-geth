// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "./Suave.sol";
import "forge-std/Test.sol";
import "forge-std/console.sol";

contract SuaveForge is Test {
    function forgeIt(string memory addr, bytes memory data) internal returns (bytes memory) {
        string memory dataHex = iToHex(data);

        string[] memory inputs = new string[](4);
        inputs[0] = "suave";
        inputs[1] = "forge";
        inputs[2] = addr;
        inputs[3] = dataHex;

        bytes memory res = vm.ffi(inputs);
        return res;
    }

    function iToHex(bytes memory buffer) public pure returns (string memory) {
        bytes memory converted = new bytes(buffer.length * 2);

        bytes memory _base = "0123456789abcdef";

        for (uint256 i = 0; i < buffer.length; i++) {
            converted[i * 2] = _base[uint8(buffer[i]) / _base.length];
            converted[i * 2 + 1] = _base[uint8(buffer[i]) % _base.length];
        }

        return string(abi.encodePacked("0x", converted));
    }

    function buildEthBlock(Suave.BuildBlockArgs memory blockArgs, Suave.BidId bidId, string memory namespace)
        external
        payable
        returns (bytes memory, bytes memory)
    {
        bytes memory data =
            forgeIt("0x0000000000000000000000000000000042100001", abi.encode(blockArgs, bidId, namespace));

        return abi.decode(data, (bytes, bytes));
    }

    function confidentialInputs() external payable returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042010001", abi.encode());

        return data;
    }

    function confidentialStoreRetrieve(Suave.BidId bidId, string memory key) external payable returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042020001", abi.encode(bidId, key));

        return data;
    }

    function confidentialStoreStore(Suave.BidId bidId, string memory key, bytes memory data1) external payable {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042020000", abi.encode(bidId, key, data1));
    }

    function ethcall(address contractAddr, bytes memory input1) external payable returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100003", abi.encode(contractAddr, input1));

        return abi.decode(data, (bytes));
    }

    function extractHint(bytes memory bundleData) external payable returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100037", abi.encode(bundleData));

        return data;
    }

    function fetchBids(uint64 cond, string memory namespace) external payable returns (Suave.Bid[] memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042030001", abi.encode(cond, namespace));

        return abi.decode(data, (Suave.Bid[]));
    }

    function newBid(
        uint64 decryptionCondition,
        address[] memory allowedPeekers,
        address[] memory allowedStores,
        string memory bidType
    ) external payable returns (Suave.Bid memory) {
        bytes memory data = forgeIt(
            "0x0000000000000000000000000000000042030000",
            abi.encode(decryptionCondition, allowedPeekers, allowedStores, bidType)
        );

        return abi.decode(data, (Suave.Bid));
    }

    function simulateBundle(bytes memory bundleData) external payable returns (uint64) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100000", abi.encode(bundleData));

        return abi.decode(data, (uint64));
    }

    function submitEthBlockBidToRelay(string memory relayUrl, bytes memory builderBid)
        external
        payable
        returns (bytes memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100002", abi.encode(relayUrl, builderBid));

        return data;
    }
}
