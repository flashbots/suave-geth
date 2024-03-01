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

    function emptyCallback() public payable {}

    function sessionE2ETest(bytes memory subTxn, bytes memory subTxn2) public payable returns (bytes memory) {
        string memory id = Suave.newBuilder();

        Suave.SimulateTransactionResult memory sim1 = Suave.simulateTransaction(id, subTxn);
        require(sim1.success == true);
        require(sim1.logs.length == 1);

        // simulate the same transaction again should fail because the nonce is the same
        Suave.SimulateTransactionResult memory sim2 = Suave.simulateTransaction(id, subTxn);
        require(sim2.success == false);

        // now, simulate the transaction with the correct nonce
        Suave.SimulateTransactionResult memory sim3 = Suave.simulateTransaction(id, subTxn2);
        require(sim3.success == true);
        require(sim3.logs.length == 2);

        return abi.encodeWithSelector(this.emptyCallback.selector);
    }

    event OffchainLogs(bytes data);

    function emitLogCallback(uint256 num) public {
        // From the msg.input, the 'confidential context' sequence
        // starts at index 37 (4 signbature bytes + 32 bytes for uint256 + 3 bytes for the magic sequence)
        uint256 magicSequenceIndex = 4 + 32 + 3;

        bytes memory inputData = msg.data;
        uint256 dataLength = inputData.length - magicSequenceIndex;

        // Initialize memory for the data to decode
        bytes memory dataToDecode = new bytes(dataLength);

        // Copy the data to decode into the memory array
        for (uint256 i = 0; i < dataLength; i++) {
            dataToDecode[i] = inputData[magicSequenceIndex + i];
        }

        emit OffchainLogs(dataToDecode);
    }

    // Event with no indexed parameters
    event EventAnonymous() anonymous;
    event EventTopic1();
    event EventTopic2(uint256 indexed num1, uint256 numNoIndex);
    event EventTopic3(uint256 indexed num1, uint256 indexed num2, uint256 numNoIndex);
    event EventTopic4(uint256 indexed num1, uint256 indexed num2, uint256 indexed num3, uint256 numNoIndex);

    function emitLog() public payable returns (bytes memory) {
        emit EventAnonymous();
        emit EventTopic1();
        emit EventTopic2(1, 1);
        emit EventTopic3(1, 2, 2);
        emit EventTopic4(1, 2, 3, 3);

        return abi.encodeWithSelector(this.emitLogCallback.selector, 10);
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
