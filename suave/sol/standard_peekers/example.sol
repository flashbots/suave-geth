pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract ExampleEthCallSource {
    uint64 state;

    function callTarget(address target, uint256 expected) public {
        bytes memory output = Suave.ethcall(target, abi.encodeWithSignature("get()"));
        (uint256 num) = abi.decode(output, (uint64));
        require(num == expected);
    }

    function ilegalStateTransition() public payable {
        state++;
    }
}

contract ExampleEthCallTarget {
    function get() public view returns (uint256) {
        return 101;
    }
}
