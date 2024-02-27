pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "forge-std/console2.sol";

contract ExampleEthCallSource {
    uint64 state;

    struct Log {
        address addr;
    }

    event LogEvent(address addr); // just for testingn

    function dummy(Log[] memory logs) public {} // doing this to be able to output the json of Log, not proud.

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

    function findStartIndex(bytes memory data) private pure returns (uint256) {
        for (uint256 i = 0; i < data.length; i++) {
            if (data[i] == bytes1(0xff)) {
                return i;
            }
        }
        return data.length; // Not found
    }

    event XX(uint256 indexed num, bytes data);

    modifier decodeLogs() {
        // revert("2");

        bytes memory inputData = msg.data;
        uint256 magicSequenceIndex = findStartIndex(inputData);
        require(magicSequenceIndex != inputData.length, "Magic sequence not found");

        // because we have to skip the magic number
        magicSequenceIndex += 1;

        // Calculate the length of the data to decode
        uint256 dataLength = inputData.length - magicSequenceIndex;

        // Initialize memory for the data to decode
        bytes memory dataToDecode = new bytes(dataLength);

        // Copy the data to decode into the memory array
        for (uint256 i = 0; i < dataLength; i++) {
            dataToDecode[i] = inputData[magicSequenceIndex + i];
        }

        emit XX(magicSequenceIndex, dataToDecode);

        // Decode logs from the extracted data
        Log[] memory logs = abi.decode(dataToDecode, (Log[]));

        /*
        for (uint256 i = 0; i < logs.length; i++) {
            emit LogEvent(logs[i].addr);
        }
        */

        // Call the function with decoded logs
        _;
    }

    function emitLogCallback(uint256 num) public decodeLogs {
        // revert("a");
    }

    event Example(uint256 num, uint256 indexed num2);

    function emitLog() public payable returns (bytes memory) {
        emit Example(1, 2);

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
