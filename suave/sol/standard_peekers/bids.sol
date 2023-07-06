pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract AnyBidContract {
	event BidEvent(
		Suave.BidId bidId,
		uint64 decryptionCondition,
		address[] allowedPeekers
	);

	// Bids to this contract should not be trusted!
	function emitBid(Suave.Bid calldata bid) public {
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
	}
}

contract BundleBidContract is AnyBidContract {
	function fetchBidConfidentialBundleData() public returns (bytes memory) {
		require(Suave.isOffchain());

		bytes memory confidentialInputs = Suave.confidentialInputs();
		return abi.decode(confidentialInputs, (bytes));
	}

	function newBid(uint64 decryptionCondition, address[] memory bidAllowedPeekers) external payable returns (bytes memory) {
		require(Suave.isOffchain());

		bytes memory bundleData = this.fetchBidConfidentialBundleData();

		(bool simOk, uint64 egp) = Suave.simulateBundle(bundleData);
		require(simOk, "bundle does not simulate correctly");

		Suave.Bid memory bid = Suave.newBid(decryptionCondition, bidAllowedPeekers);

		Suave.confidentialStoreStore(bid.id, "ethBundle", bundleData);
		Suave.confidentialStoreStore(bid.id, "ethBundleSimResults", abi.encode(egp));

		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		return bytes.concat(this.emitBid.selector, abi.encode(bid));
	}
}

struct EgpBidPair {
	uint64 egp; // in wei, beware overflow
	Suave.BidId bidId;
}

error RevertBytes(address, bytes);
error RevertString(string);

contract EthBlockBidContract is AnyBidContract {
	event BuilderBoostBidEvent(
		Suave.BidId bidId,
		bytes builderBid
	);

	function buildFromPool(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight) public returns (bytes memory) {
		require(Suave.isOffchain());

		Suave.Bid[] memory allBids = Suave.fetchBids(blockHeight);

		// TODO: handle merged bids
		EgpBidPair[] memory bidsByEGP = new EgpBidPair[](allBids.length);
		for (uint i = 0; i < allBids.length; i++) {
			bytes memory simResults = Suave.confidentialStoreRetrieve(allBids[i].id, "ethBundleSimResults");
			uint64 egp = abi.decode(simResults, (uint64));
			bidsByEGP[i] = EgpBidPair(egp, allBids[i].id);
		}

		// Bubble sort, cause why not
		uint n = bidsByEGP.length;
		for (uint i = 0; i < n - 1; i++) {
			for (uint j = i + 1; j < n; j++) {
				if (bidsByEGP[i].egp > bidsByEGP[j].egp) {
					EgpBidPair memory temp = bidsByEGP[i];
					bidsByEGP[i] = bidsByEGP[j];
					bidsByEGP[j] = temp;
				}
			}
		}

		Suave.BidId[] memory allBidIds = new Suave.BidId[](allBids.length);
		for (uint i = 0; i < bidsByEGP.length; i++) {
			allBidIds[i] = bidsByEGP[i].bidId;
		}

		return this.build(blockArgs, blockHeight, allBidIds);
	}

	function build(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight, Suave.BidId[] memory bids) public returns (bytes memory) {
		require(Suave.isOffchain());

		address[] memory allowedPeekers = new address[](2);
		allowedPeekers[0] = address(this);
		allowedPeekers[1] = Suave.BUILD_ETH_BLOCK_PEEKER;

		Suave.Bid memory blockBid = Suave.newBid(blockHeight, allowedPeekers);
		Suave.confidentialStoreStore(blockBid.id, "mergedBids", abi.encode(bids));
		
		(bytes memory builderBid, bytes memory payload) = Suave.buildEthBlock(blockArgs, blockBid.id);
		Suave.confidentialStoreStore(blockBid.id, "builderPayload", payload); // only through this.unlock

		emit BuilderBoostBidEvent(blockBid.id, builderBid);
		emit BidEvent(blockBid.id, blockBid.decryptionCondition, blockBid.allowedPeekers);
		return bytes.concat(this.emitBuilderBidAndBid.selector, abi.encode(blockBid, builderBid));
		// this makes the builder bid (block profit) public, which doesn't have to be the case
	}

	function emitBuilderBidAndBid(Suave.Bid memory bid, bytes memory builderBid) public returns (Suave.Bid memory, bytes memory) {
		emit BuilderBoostBidEvent(bid.id, builderBid);
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		return (bid, builderBid);
	}

	function unlock(Suave.BidId bidId, bytes memory signedBlindedHeader) public returns (bytes memory) {
		require(Suave.isOffchain());

		// TODO: verify the header is correct

		bytes memory payload = Suave.confidentialStoreRetrieve(bidId, "builderPayload");
		return payload;
	}
}
