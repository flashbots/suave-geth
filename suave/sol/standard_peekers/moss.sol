pragma solidity ^0.8.8;

import "forge-std/console2.sol";
import "suave-std/Context.sol";
import "suave-std/Transactions.sol";
import "suave-std/suavelib/Suave.sol";

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

    function emitTopic(string memory topicName, bytes memory data) external;
}

// methods available for Suapps during the block building process.
interface WorkerMoss {
    struct TransactionResult {
        bool err;
    }

    function coinbase() external view returns (address);
    function getBalance(address addr) external view returns (uint256);
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

contract MevShare is Suapp {
    mapping(bytes32 => bytes) public userTxns;

    struct Hint {
        bytes32 txnId;
        address to;
        bytes data;
    }

    struct Bundle {
        bytes userTxn;
        bytes matchTxn;
    }

    function callback() public {}

    function sendTransaction() public isCCR returns (bytes memory) {
        // decode the txn from confidential inputs
        bytes memory txnRaw = Suave.confidentialInputs();
        Transactions.EIP155 memory txn = Transactions.decodeRLP_EIP155(txnRaw);

        // store the txn
        bytes32 txnId = keccak256(txnRaw);
        userTxns[txnId] = txnRaw;

        // extract the hints
        Hint memory hint = Hint({txnId: txnId, to: txn.to, data: txn.data});
        bytes memory bytesHint = abi.encode(hint);

        // emit the hints over the p2p layer
        ccrCtx.emitTopic("mev-share", bytesHint);
        return abi.encodeWithSelector(this.callback.selector);
    }

    function matchBundle(bytes32 txnId) public isWorker isCCR returns (bytes memory) {
        // We have to validate that applyting 'originalTxn' and then 'matchTxn' produces
        // a refund on the coinbase account.
        bytes memory backrunTx = Suave.confidentialInputs();
        uint256 preBalance = workerCtx.getBalance(workerCtx.coinbase());

        bytes memory userTxn = userTxns[txnId];

        workerCtx.addTransaction(userTxn);
        workerCtx.addTransaction(backrunTx);

        uint256 postBalance = workerCtx.getBalance(workerCtx.coinbase());
        if (postBalance <= preBalance) {
            revert("No refund");
        }

        //console2.logBytes(userTxn);
        //console2.logBytes(backrunTx);
        //console2.log("Pre balance: ", preBalance);
        //console2.log("Post balance: ", postBalance);

        // create the bundle and emit it
        Bundle memory bundle = Bundle({userTxn: userTxn, matchTxn: backrunTx});

        CCRMoss.MossBundle memory mossBundle = CCRMoss.MossBundle({
            to: address(this),
            data: abi.encodeWithSelector(this.applyFn.selector, bundle),
            blockNumber: 0, // this gets set internally if empty
            maxBlockNumber: 0
        });
        ccrCtx.sendBundle(mossBundle);

        return abi.encodeWithSelector(this.callback.selector);
    }

    function applyFn(Bundle memory bundle) public isWorker {
        uint256 preBalance = workerCtx.getBalance(workerCtx.coinbase());

        workerCtx.addTransaction(bundle.userTxn);
        workerCtx.addTransaction(bundle.matchTxn);

        uint256 postBalance = workerCtx.getBalance(workerCtx.coinbase());
        if (postBalance <= preBalance) {
            revert("No refund");
        }

        bytes memory privkey = Suave.contextGet("privkey");

        // apply the refund
        Transactions.EIP155Request memory txnWithToAddress = Transactions.EIP155Request({
            to: workerCtx.coinbase(),
            gas: 1000000,
            gasPrice: 500,
            value: postBalance - 1000,
            nonce: 0,
            data: "",
            chainId: 1337
        });

        // sign it.
        Transactions.EIP155 memory txn = Transactions.signTxn(txnWithToAddress, string(privkey));
        workerCtx.addTransaction(Transactions.encodeRLP(txn));
    }
}
