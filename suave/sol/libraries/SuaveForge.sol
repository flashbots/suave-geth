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

    function buildEthBlock(Suave.BuildBlockArgs memory blockArgs, Suave.DataId dataId, string memory namespace)
        internal
        view
        returns (bytes memory, bytes memory)
    {
        bytes memory data =
            forgeIt("0x0000000000000000000000000000000042100001", abi.encode(blockArgs, dataId, namespace));

        return abi.decode(data, (bytes, bytes));
    }

    function confidentialInputs() internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042010001", abi.encode());

        return data;
    }

    function confidentialRetrieve(Suave.DataId dataId, string memory key) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042020001", abi.encode(dataId, key));

        return data;
    }

    function confidentialStore(Suave.DataId dataId, string memory key, bytes memory data1) internal view {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042020000", abi.encode(dataId, key, data1));
    }

    function doHTTPRequest(Suave.HttpRequest memory request) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000043200002", abi.encode(request));

        return abi.decode(data, (bytes));
    }

    function ethcall(address contractAddr, bytes memory input1) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100003", abi.encode(contractAddr, input1));

        return abi.decode(data, (bytes));
    }

    function extractHint(bytes memory bundleData) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100037", abi.encode(bundleData));

        return data;
    }

    function fetchDataRecords(uint64 cond, string memory namespace) internal view returns (Suave.DataRecord[] memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042030001", abi.encode(cond, namespace));

        return abi.decode(data, (Suave.DataRecord[]));
    }

    function fillMevShareBundle(Suave.DataId dataId) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000043200001", abi.encode(dataId));

        return data;
    }

    function newBuilder() internal view returns (string memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000053200001", abi.encode());

        return abi.decode(data, (string));
    }

    function newDataRecord(
        uint64 decryptionCondition,
        address[] memory allowedPeekers,
        address[] memory allowedStores,
        string memory dataType
    ) internal view returns (Suave.DataRecord memory) {
        bytes memory data = forgeIt(
            "0x0000000000000000000000000000000042030000",
            abi.encode(decryptionCondition, allowedPeekers, allowedStores, dataType)
        );

        return abi.decode(data, (Suave.DataRecord));
    }

    function signEthTransaction(bytes memory txn, string memory chainId, string memory signingKey)
        internal
        view
        returns (bytes memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000040100001", abi.encode(txn, chainId, signingKey));

        return abi.decode(data, (bytes));
    }

    function signMessage(bytes memory digest, string memory signingKey) internal view returns (bytes memory) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000040100003", abi.encode(digest, signingKey));

        return abi.decode(data, (bytes));
    }

    function simulateBundle(bytes memory bundleData) internal view returns (uint64) {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100000", abi.encode(bundleData));

        return abi.decode(data, (uint64));
    }

    function simulateTransaction(string memory session, bytes memory txn)
        internal
        view
        returns (Suave.SimulateTransactionResult memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000053200002", abi.encode(session, txn));

        return abi.decode(data, (Suave.SimulateTransactionResult));
    }

    function submitBundleJsonRPC(string memory url, string memory method, bytes memory params)
        internal
        view
        returns (bytes memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000043000001", abi.encode(url, method, params));

        return data;
    }

    function submitEthBlockToRelay(string memory relayUrl, bytes memory builderBid)
        internal
        view
        returns (bytes memory)
    {
        bytes memory data = forgeIt("0x0000000000000000000000000000000042100002", abi.encode(relayUrl, builderBid));

        return data;
    }
}
