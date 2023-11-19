pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

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
}

contract ExampleEthCallTarget {
    function get() public view returns (uint256) {
        return 101;
    }
}
