pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "./utils/Strings.sol";
import "./std.sol";
import "solady/src/utils/LibString.sol";
import "solady/src/utils/JSONParserLib.sol";

library MevShare {
    using JSONParserLib for *;

    struct Bundle {
        string version;
        uint64 inclusionBlock;
        bytes[] bodies;
    }

    function sendBundle(string memory url, Bundle memory bundle) internal {
        // TODO
    }
}
