pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "forge-std/console2.sol";

contract ExampleEthCallSource {
    uint64 state;

    function callTarget(address target, uint256 expected) public {
        bytes memory output = Suave.ethcall(target, abi.encodeWithSignature("get()"), "");
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

    function emptyCallback() public payable {}

    function sessionE2ETest(bytes memory subTxn, bytes memory subTxn2) public payable returns (bytes memory) {
        string memory id = Suave.newBuilder();

        Suave.SimulateTransactionResult memory sim1 = Suave.simulateTransaction(id, subTxn, "");
        require(sim1.success == true);
        require(sim1.logs.length == 1);

        // simulate the same transaction again should fail because the nonce is the same
        Suave.SimulateTransactionResult memory sim2 = Suave.simulateTransaction(id, subTxn, "");
        require(sim2.success == false);

        // now, simulate the transaction with the correct nonce
        Suave.SimulateTransactionResult memory sim3 = Suave.simulateTransaction(id, subTxn2, "");
        require(sim3.success == true);
        require(sim3.logs.length == 2);

        return abi.encodeWithSelector(this.emptyCallback.selector);
    }
}

contract ExampleEthCallTarget {
    uint256 stateCount;

    function get() public view returns (uint256) {
        return 101;
    }

    event Example(uint256 num);

    function func1() public payable {
        stateCount++;

        for (uint256 i = 0; i < stateCount; i++) {
            emit Example(1);
        }
    }
}
