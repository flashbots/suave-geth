pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "forge-std/console2.sol";

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

    function consoleLog() public payable {
        console2.log(1, 2, 3);
    }

    function remoteCall(Suave.HttpRequest memory request) public {
        Suave.doHTTPRequest(request);
    }
}

contract ExampleEthCallTarget {
    function get() public view returns (uint256) {
        return 101;
    }
}
