pragma solidity ^0.8.8;

import "forge-std/console.sol";

// methods available for Suapps during CCR requests
interface CCRMoss {
    // sendBundle sends a Moss bundle to the Suave chain transaction pool to be picked
    // up by miners during the block building process.
    function sendBundle(bytes memory txn) external;
}

// methods available for Suapps during the block building process.
interface WorkerMoss {
    struct TransactionResult {
        bool err;
    }

    function addTransaction(bytes memory txn) external returns (TransactionResult memory);
}

contract Suapp {
    WorkerMoss workerCtx;
    CCRMoss ccrCtx;

    modifier isCCR() {
        ccrCtx = CCRMoss(0x1234567890123456789012345678901234567890);
        _;
    }

    modifier isWorker() {
        workerCtx = WorkerMoss(0x1234567890123456789012345678901234567891);
        _;
    }
}

contract Bundle1 is Suapp {
    uint256 public count;

    struct Bundle {
        Transaction[] txns;
    }

    struct Transaction {
        bytes txn;
    }

    function incr() public {
        count++;
    }

    function applyFn(Bundle memory bundle) public isWorker {
        console.log("execute");

        for (uint256 i = 0; i < bundle.txns.length; i++) {
            Transaction memory txn = bundle.txns[i];

            workerCtx.addTransaction(txn.txn);
        }
    }
}
