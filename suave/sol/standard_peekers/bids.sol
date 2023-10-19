pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract AnyBidContract {
	Suave constant suave = Suave(0x1100000000000000000000000000000042100002);

	event BidEvent(
		Suave.BidId bidId,
		uint64 decryptionCondition,
		address[] allowedPeekers
	);

	function fetchBidConfidentialBundleData() public returns (bytes memory) {
		require(suave.isConfidential());
		
		bytes memory confidentialInputs = suave.confidentialInputs();
		return abi.decode(confidentialInputs, (bytes));
	}

	// Bids to this contract should not be trusted!
	function emitBid(Suave.Bid calldata bid) public {
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
	}
}

contract BundleBidContract is AnyBidContract {

	function newBid(uint64 decryptionCondition, address[] memory bidAllowedPeekers, address[] memory bidAllowedStores) external payable returns (bytes memory) {
		require(suave.isConfidential());

		bytes memory bundleData = this.fetchBidConfidentialBundleData();

		uint64 egp = suave.simulateBundle(bundleData);

		Suave.Bid memory bid = suave.newBid(decryptionCondition, bidAllowedPeekers, bidAllowedStores, "default:v0:ethBundles");

		suave.confidentialStoreStore(bid.id, "default:v0:ethBundles", bundleData);
		suave.confidentialStoreStore(bid.id, "default:v0:ethBundleSimResults", abi.encode(egp));

		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		return bytes.concat(this.emitBid.selector, abi.encode(bid));
	}
}

contract MevShareBidContract is AnyBidContract {

	event HintEvent(
		Suave.BidId bidId,
		bytes hint
	);

	event MatchEvent(
		Suave.BidId matchBidId,
		bytes bidhint,
		bytes matchHint
	);

	function newBid(uint64 decryptionCondition, address[] memory bidAllowedPeekers, address[] memory bidAllowedStores) external payable returns (bytes memory) {
		// 0. check confidential execution
		require(suave.isConfidential());

		// 1. fetch bundle data
		bytes memory bundleData = this.fetchBidConfidentialBundleData();

		// 2. sim bundle
		uint64 egp = suave.simulateBundle(bundleData);
		
		// 3. extract hint
		bytes memory hint = suave.extractHint(bundleData);
		
		// // 4. store bundle and sim results
		Suave.Bid memory bid = suave.newBid(decryptionCondition, bidAllowedPeekers, bidAllowedStores, "mevshare:v0:unmatchedBundles");
		suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundles", bundleData);
		suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundleSimResults", abi.encode(egp));
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		emit HintEvent(bid.id, hint);

		// // 5. return "callback" to emit hint onchain
		return bytes.concat(this.emitBidAndHint.selector, abi.encode(bid, hint));
	}

	function emitBidAndHint(Suave.Bid calldata bid, bytes memory hint) public {
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		emit HintEvent(bid.id, hint);
	}

	function newMatch(uint64 decryptionCondition, address[] memory bidAllowedPeekers, address[] memory bidAllowedStores, Suave.BidId shareBidId) external payable returns (bytes memory) {
		// WARNING : this function will copy the original mev share bid
		// into a new key with potentially different permsissions
		
		require(suave.isConfidential());
		// 1. fetch confidential data
		bytes memory matchBundleData = this.fetchBidConfidentialBundleData();

		// 2. sim match alone for validity
		uint64 egp = suave.simulateBundle(matchBundleData);

		// 3. extract hint
		bytes memory matchHint = suave.extractHint(matchBundleData);
		
		Suave.Bid memory bid = suave.newBid(decryptionCondition, bidAllowedPeekers, bidAllowedStores, "mevshare:v0:matchBids");
		suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundles", matchBundleData);
		suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundleSimResults", abi.encode(0));

		//4. merge bids
		Suave.BidId[] memory bids = new Suave.BidId[](2);
		bids[0] = shareBidId;
		bids[1] = bid.id;
		suave.confidentialStoreStore(bid.id, "mevshare:v0:mergedBids", abi.encode(bids));

		return bytes.concat(this.emitBid.selector, abi.encode(bid));
	}

	function emitMatchBidAndHint(Suave.Bid calldata bid, bytes memory bidHint, bytes memory matchHint) public {
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		emit MatchEvent(bid.id, bidHint, matchHint);
	}
}

/* Not tested or implemented on the precompile side */
struct EgpBidPair {
	uint64 egp; // in wei, beware overflow
	Suave.BidId bidId;
}

contract EthBlockBidContract is AnyBidContract {

	event BuilderBoostBidEvent(
		Suave.BidId bidId,
		bytes builderBid
	);
	
	function idsEqual(Suave.BidId _l, Suave.BidId _r) public pure returns (bool) {
		bytes memory l = abi.encodePacked(_l);
		bytes memory r = abi.encodePacked(_r);
		for (uint i = 0; i < l.length; i++) {
			if (bytes(l)[i] != r[i]) {
				return false;
			}
		}

		return true;
	}

	function buildMevShare(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight) public returns (bytes memory) {
		require(suave.isConfidential());

		Suave.Bid[] memory allShareMatchBids = suave.fetchBids(blockHeight, "mevshare:v0:matchBids");
		Suave.Bid[] memory allShareUserBids = suave.fetchBids(blockHeight, "mevshare:v0:unmatchedBundles");

		if (allShareUserBids.length == 0) {
			revert Suave.PeekerReverted(address(this), "no bids");
		}

		Suave.Bid[] memory allBids = new Suave.Bid[](allShareUserBids.length);
		for (uint i = 0; i < allShareUserBids.length; i++) {
			// TODO: sort matches by egp first!
			Suave.Bid memory bidToInsert = allShareUserBids[i]; // will be updated with the best match if any
			for (uint j = 0; j < allShareMatchBids.length; j++) {
				// TODO: should be done once at the start and sorted
				Suave.BidId[] memory mergedBidIds = abi.decode(suave.confidentialStoreRetrieve(allShareMatchBids[j].id, "mevshare:v0:mergedBids"), (Suave.BidId[]));
				if (idsEqual(mergedBidIds[0], allShareUserBids[i].id)) {
					bidToInsert = allShareMatchBids[j];
					break;
				}
			}
			allBids[i] = bidToInsert;
		}

		EgpBidPair[] memory bidsByEGP = new EgpBidPair[](allBids.length);
		for (uint i = 0; i < allBids.length; i++) {
			bytes memory simResults = suave.confidentialStoreRetrieve(allBids[i].id, "mevshare:v0:ethBundleSimResults");
			uint64 egp = abi.decode(simResults, (uint64));
			bidsByEGP[i] = EgpBidPair(egp, allBids[i].id);
		}

		// Bubble sort, cause why not
		uint n = bidsByEGP.length;
		for (uint i = 0; i < n - 1; i++) {
			for (uint j = i + 1; j < n; j++) {
				if (bidsByEGP[i].egp < bidsByEGP[j].egp) {
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

		return buildAndEmit(blockArgs, blockHeight, allBidIds, "mevshare:v0");
	}

	function buildFromPool(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight) public returns (bytes memory) {
		require(suave.isConfidential());

		Suave.Bid[] memory allBids = suave.fetchBids(blockHeight, "default:v0:ethBundles");
		if (allBids.length == 0) {
			revert Suave.PeekerReverted(address(this), "no bids");
		}

		EgpBidPair[] memory bidsByEGP = new EgpBidPair[](allBids.length);
		for (uint i = 0; i < allBids.length; i++) {
			bytes memory simResults = suave.confidentialStoreRetrieve(allBids[i].id, "default:v0:ethBundleSimResults");
			uint64 egp = abi.decode(simResults, (uint64));
			bidsByEGP[i] = EgpBidPair(egp, allBids[i].id);
		}

		// Bubble sort, cause why not
		uint n = bidsByEGP.length;
		for (uint i = 0; i < n - 1; i++) {
			for (uint j = i + 1; j < n; j++) {
				if (bidsByEGP[i].egp < bidsByEGP[j].egp) {
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

		return buildAndEmit(blockArgs, blockHeight, allBidIds, "");
	}

	function buildAndEmit(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight, Suave.BidId[] memory bids, string memory namespace) public virtual returns (bytes memory) {
		require(suave.isConfidential());

		(Suave.Bid memory blockBid, bytes memory builderBid) = this.doBuild(blockArgs, blockHeight, bids, namespace);

		emit BuilderBoostBidEvent(blockBid.id, builderBid);
		emit BidEvent(blockBid.id, blockBid.decryptionCondition, blockBid.allowedPeekers);
		return bytes.concat(this.emitBuilderBidAndBid.selector, abi.encode(blockBid, builderBid));
	}

	function doBuild(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight, Suave.BidId[] memory bids, string memory namespace) public view returns (Suave.Bid memory, bytes memory) {
		address[] memory allowedPeekers = new address[](2);
		allowedPeekers[0] = address(this);

		Suave.Bid memory blockBid = suave.newBid(blockHeight, allowedPeekers, allowedPeekers, "default:v0:mergedBids");
		suave.confidentialStoreStore(blockBid.id, "default:v0:mergedBids", abi.encode(bids));
		 
		(bytes memory builderBid, bytes memory payload) = suave.buildEthBlock(blockArgs, blockBid.id, namespace);
		suave.confidentialStoreStore(blockBid.id, "default:v0:builderPayload", payload); // only through this.unlock

		return (blockBid, builderBid);
	}

	function emitBuilderBidAndBid(Suave.Bid memory bid, bytes memory builderBid) public returns (Suave.Bid memory, bytes memory) {
		emit BuilderBoostBidEvent(bid.id, builderBid);
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		return (bid, builderBid);
	}

	function unlock(Suave.BidId bidId, bytes memory signedBlindedHeader) public view returns (bytes memory) {
		require(suave.isConfidential());

		// TODO: verify the header is correct
		// TODO: incorporate protocol name
		bytes memory payload = suave.confidentialStoreRetrieve(bidId, "default:v0:builderPayload");
		return payload;
	}
}

contract EthBlockBidSenderContract is EthBlockBidContract {
	string boostRelayUrl;

	constructor(string memory boostRelayUrl_) {
		boostRelayUrl = boostRelayUrl_;
	}

	function buildAndEmit(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight, Suave.BidId[] memory bids, string memory namespace) public virtual override returns (bytes memory) {
		require(suave.isConfidential());

		(Suave.Bid memory blockBid, bytes memory builderBid) = this.doBuild(blockArgs, blockHeight, bids, namespace);
		suave.submitEthBlockBidToRelay(boostRelayUrl, builderBid);

		emit BidEvent(blockBid.id, blockBid.decryptionCondition, blockBid.allowedPeekers);
		return bytes.concat(this.emitBid.selector, abi.encode(blockBid));
	}
}
