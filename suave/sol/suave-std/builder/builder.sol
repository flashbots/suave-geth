// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "../../libraries/Suave.sol";

contract Builder {
    struct Log {
        bytes32[] topics;
    }

    struct ExecResult {
        Log logs;
    }

    function execTransaction() external view returns (ExecResult memory) {
        ExecResult memory result;
        return result;
    }

    function call() external view {
    }
}
