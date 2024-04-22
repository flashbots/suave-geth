pragma solidity ^0.8.8;

import "forge-std/console.sol";
import "../libraries/Suave.sol";

contract Bundle1 {
    struct Bundle {
        bytes txn1;
        bytes txn2;
    }

    function emitTheBundle(bytes memory txn) public {
        // call with a confidential compute request
        Bundle memory bundle;
        bundle.txn1 = txn;

        Suave.mossSendBundle(abi.encodeWithSelector(this.applyFn.selector, bundle));
    }

    function applyFn(Bundle memory bundle) public {
        console.log("execute");

        if (bundle.txn1.length > 0) {
            Suave.mossAddTransaction(bundle.txn1);
        }
        if (bundle.txn2.length > 0) {
            Suave.mossAddTransaction(bundle.txn2);
        }
    }
}
