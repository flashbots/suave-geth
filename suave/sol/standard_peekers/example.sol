pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "../suave-std/api/ethereum/ethereum.sol";
import "../suave-std/builder/builder.sol";

contract ExampleEthCallSource {
    function callTarget(address target, uint256 expected) public {
        bytes memory output = Suave.ethcall(target, abi.encodeWithSignature("get()"));
        (uint256 num) = abi.decode(output, (uint64));
        require(num == expected);
    }

    function getExample(string memory url) public {
        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "a:b";

        Suave.httpGet(url, config);
    }

    function postExample(string memory url, bytes memory body) public {
        Suave.HttpConfig memory config;
        config.headers = new string[](1);
        config.headers[0] = "b:c";

        Suave.httpPost(url, body, config);
    }

    function sampleJSON(string memory url) public {
        Suave.simpleConsole(abi.encode(11));

        Ethereum jj = new Ethereum(url);
        // jj.version();
        // jj.blockNumber();
        jj.blockByNumber(20);
        Suave.simpleConsole(abi.encode(22));
    }

    function other() public {

    }
}

contract BuilderExample {
    function example() public {
        Builder bb = new Builder();
        bb.execTransaction();
        bb.call();
    }
}

contract ExampleEthCallTarget {
    function get() public view returns (uint256) {
        return 101;
    }
}
