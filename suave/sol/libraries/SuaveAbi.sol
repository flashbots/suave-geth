pragma solidity ^0.8.8;

import {Suave} from "./Suave.sol";

contract SuaveAbi {
    error PeekerReverted(address, bytes);

    function newBid(uint64 decryptionCondition, address[] memory allowedPeekers, address[] memory allowedStores, string memory BidType) external view returns (Suave.Bid memory) {}
	function fetchBids(uint64 cond, string memory namespace) external view returns (Suave.Bid[] memory) {}
    function confidentialStore(Suave.BidId bidId, string memory key, bytes memory data) external view {}
    function confidentialRetrieve(Suave.BidId bidId, string memory key) external view returns (bytes memory) {}
    function signEthTransaction(bytes memory txn, string memory chainId, string memory signingKey) external view returns (bytes memory) {}
    function simulateBundle(bytes memory bundleData) external view returns (uint64) {}
    function extractHint(bytes memory bundleData) external view returns (bytes memory) {}
	function buildEthBlock(Suave.BuildBlockArgs memory blockArgs, Suave.BidId bid, string memory namespace) external view returns (bytes memory, bytes memory) {}
    function submitEthBlockBidToRelay(string memory relayUrl, bytes memory builderBid) external view returns (bytes memory) {}
    function fillMevShareBundle(Suave.BidId bidId) external view returns (bytes memory) {}
    function submitBundleJsonRPC(string memory url, string memory method, bytes memory params) external view returns (bytes memory) {}
}
