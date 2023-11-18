pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract ConfidentialStore {
    event Event (uint64 indexed num);

    function callback(uint64 num) external payable {
        emit Event(num);
    }

    function example() external payable returns (bytes memory) {
        uint64 num = Suave.getBlockNumber();
        return abi.encodeWithSelector(this.callback.selector, num);
    }
}
