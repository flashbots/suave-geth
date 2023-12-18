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

    function remoteCall(Suave.HttpRequest memory request) public {
        Suave.doHTTPRequest(request);
    }

    function sessionE2ETest(bytes memory subTxn) public payable {
        string memory id = Suave.newBuilder();
        Suave.simulateTransaction(id, subTxn);
    }
}

contract ExampleEthCallTarget {
    function get() public view returns (uint256) {
        return 101;
    }

    event Example (
        uint256 num
    );

    function func1() public payable {
        emit Example(1);
    }
}
