// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "./Suave.sol";

interface Vm {
    function ffi(string[] calldata commandInput) external view returns (bytes memory result);
}

library SuaveForge {
    Vm constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function forgeIt(string memory addr, bytes memory data) internal view returns (bytes memory) {
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

    function buildEthBlock(Suave.BuildBlockArgs memory param1, Suave.BidId param2, string memory param3)
        internal
        view
        returns (bytes memory, bytes memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100001", abi.encode(param1, param2, param3));
        return abi.decode(data, (bytes, bytes));
    }

    function confidentialInputs() internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042010001", abi.encode());
        return abi.decode(data, (bytes));
    }

    function confidentialStoreRetrieve(Suave.BidId param1, string memory param2) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042020001", abi.encode(param1, param2));
        return abi.decode(data, (bytes));
    }

    function confidentialStoreStore(Suave.BidId param1, string memory param2, bytes memory param3) internal view {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042020000", abi.encode(param1, param2, param3));
        return abi.decode(data, ());
    }

    function ethcall(address param1, bytes memory param2) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100003", abi.encode(param1, param2));
        return abi.decode(data, (bytes));
    }

    function extractHint(bytes memory param1) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100037", abi.encode(param1));
        return abi.decode(data, (bytes));
    }

    function fetchBids(uint64 param1, string memory param2) internal view returns (Suave.Bid[] memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042030001", abi.encode(param1, param2));
        return abi.decode(data, (Suave.Bid[]));
    }

    function fillMevShareBundle(Suave.BidId param1) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000043200001", abi.encode(param1));
        return abi.decode(data, (bytes));
    }

    function isConfidential() internal view returns (bool) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042010000", abi.encode());
        return abi.decode(data, (bool));
    }

    function newBid(uint64 param1, address[] memory param2, address[] memory param3, string memory param4)
        internal
        view
        returns (Suave.Bid memory)
    {
        bytes memory data =
            forgeIt("0x0000000000000000000000000000000042030000", abi.encode(param1, param2, param3, param4));
        return abi.decode(data, (Suave.Bid));
    }

    function signEthTransaction(bytes memory param1, string memory param2, string memory param3)
        internal
        view
        returns (bytes memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000040100001", abi.encode(param1, param2, param3));
        return abi.decode(data, (bytes));
    }

    function simulateBundle(bytes memory param1) internal view returns (uint64) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100000", abi.encode(param1));
        return abi.decode(data, (uint64));
    }

    function submitBundleJsonRPC(string memory param1, string memory param2, bytes memory param3)
        internal
        view
        returns (bytes memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000043000001", abi.encode(param1, param2, param3));
        return abi.decode(data, (bytes));
    }

    function submitEthBlockBidToRelay(string memory param1, bytes memory param2) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100002", abi.encode(param1, param2));
        return abi.decode(data, (bytes));
    }
}
