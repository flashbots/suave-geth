pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract ExampleEthCallSource {
    function callTarget(address target, uint256 expected) public {
        Suave.CallResult memory output = Suave.ethcall(target, abi.encodeWithSignature("get()"));
        (uint256 num) = abi.decode(output.returnData, (uint64));
        require(num == expected);
    }
}

contract ExampleEthCallTarget {
    event Event(
        uint64 num
    );

    function get() public payable returns (uint256) {
        emit Event(1);
        return 101;
    }
}
