pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract ExampleEthCallSource {
    function callTarget(address target, uint256 expected) public {
        bytes memory output = Suave.ethcall(target, abi.encodeWithSignature("get()"));
        (uint256 num) = abi.decode(output, (uint64));
        require(num == expected);
    }
}

contract ExampleEthCallTarget {
    event Nil();

    function get() public payable returns (uint256) {
        emit Nil();
        return 101;
    }
}

contract ExampleSimulateTransaction {
    function callback() public payable {
    }

    function run(bytes memory txn) external payable returns (bytes memory) {
        Suave.SimulateTransactionResult memory result = Suave.simulateTransaction(txn);
        require(result.logs.length == 1);
        require(result.logs[0].topics[0] == bytes32(abi.encodeWithSignature("get()")));
        return abi.encodeWithSelector(this.callback.selector);
    }
}
