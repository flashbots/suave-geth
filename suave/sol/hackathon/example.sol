pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract ConfidentialStore {
    function callback() external payable {}

    function example() external payable returns (bytes memory) {
        address[] memory allowedList = new address[](1);
        allowedList[0] = address(this);

        Suave.aaa();
        
        Suave.Bid memory bid = Suave.newBid(
            10,
            allowedList,
            allowedList,
            "namespace"
        );

        Suave.confidentialStore(bid.id, "key1", abi.encode(1));
        Suave.confidentialStore(bid.id, "key2", abi.encode(2));

        bytes memory value = Suave.confidentialRetrieve(bid.id, "key1");
        require(keccak256(value) == keccak256(abi.encode(1)));

        Suave.Bid[] memory allShareMatchBids = Suave.fetchBids(10, "namespace");
        return abi.encodeWithSelector(this.callback.selector);
    }
}