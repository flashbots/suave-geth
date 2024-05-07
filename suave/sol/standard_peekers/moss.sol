pragma solidity ^0.8.8;

import "forge-std/console2.sol";
import "suave-std/Context.sol";
import "suave-std/Transactions.sol";

// methods available for Suapps during CCR requests
interface CCRMoss {
    struct MossBundle {
        address to;
        bytes data;
        uint64 blockNumber;
        uint64 maxBlockNumber;
    }

    function sendBundle(MossBundle memory bundle) external;

    function sendTransaction(bytes memory txn) external;
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
        console2.log("execute");

        for (uint256 i = 0; i < bundle.txns.length; i++) {
            Transaction memory txn = bundle.txns[i];

            workerCtx.addTransaction(txn.txn);
        }
    }
}

contract Bundle2 is Suapp {
    uint256 public count;

    function mint() public {
        console2.log("X");
        console2.logUint(count);
        count += 10;
        console2.logUint(count);
    }

    function coprocess() public isCCR {
        // do some random stuff...

        // send a transaction that interacts with the contract
        bytes memory signingKey = Context.confidentialInputs();

        Transactions.EIP155Request memory txnWithToAddress = Transactions.EIP155Request({
            to: address(this),
            gas: 1000000,
            gasPrice: 500,
            value: 0,
            nonce: 0,
            data: abi.encodeWithSelector(this.mint.selector),
            chainId: 1337
        });

        Transactions.EIP155 memory txn = Transactions.signTxn(txnWithToAddress, string(signingKey));

        // create a txn to execute to mint the asset
        bytes memory raw = Transactions.encodeRLP(txn);
        ccrCtx.sendTransaction(raw);
    }
}
