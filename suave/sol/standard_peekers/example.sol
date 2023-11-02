pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract ExampleEthCallSource {
    function callTarget(address target, uint256 expected) public {
        bytes memory output = Suave.ethcall(
            target,
            abi.encodeWithSignature("get()")
        );
        uint256 num = abi.decode(output, (uint64));
        require(num == expected);
    }
}

contract ExampleEthCallTarget {
    function get() public view returns (uint256) {
        return 101;
    }
}

contract OnlyConfidential {
    event SimResultEvent(uint64 egp);

    function fetchBidConfidentialBundleData() public returns (bytes memory) {
        require(Suave.isConfidential());

        bytes memory confidentialInputs = Suave.confidentialInputs();
        return abi.decode(confidentialInputs, (bytes));
    }

    // note: because of confidential execution,
    // you will not see your input as input to the function
    function helloWorld() external payable {
        // 0. ensure confidential execution
        require(Suave.isConfidential());

        // 1. fetch bundle data
        bytes memory bundleData = this.fetchBidConfidentialBundleData();

        // 2. sim bundle and get effective gas price
        uint64 effectiveGasPrice = Suave.simulateBundle(bundleData);

        emit SimResultEvent(effectiveGasPrice);

        // note: this function doesn't return anything
        // so this computation result will never land onchain
    }
}
