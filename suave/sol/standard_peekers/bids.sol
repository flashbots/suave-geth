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

	function fetchBidConfidentialBundleData() public view returns (bytes memory) {
		require(Suave.isOffchain());

		bytes memory confidentialInputs = Suave.confidentialInputs();
		return abi.decode(confidentialInputs, (bytes));
	}

	function newBid(uint64 decryptionCondition, address[] memory bidAllowedPeekers) external payable returns (bytes memory) {
		require(Suave.isOffchain());

		bytes memory bundleData = this.fetchBidConfidentialBundleData();

		(bool simOk, uint64 egp) = Suave.simulateBundle(bundleData);
		require(simOk, "bundle does not simulate correctly");

		Suave.Bid memory bid = Suave.newBid(decryptionCondition, bidAllowedPeekers, "default:v0:ethBundles");

		Suave.confidentialStoreStore(bid.id, "default:v0:ethBundles", bundleData);
		Suave.confidentialStoreStore(bid.id, "default:v0:ethBundleSimResults", abi.encode(egp));

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

	function fetchBidConfidentialBundleData() public returns (bytes memory) {
		bytes memory confidentialInputs = Suave.confidentialInputs();
		return abi.decode(confidentialInputs, (bytes));
	}

	// function newBid(uint64 decryptionCondition, address[] memory bidAllowedPeekers, bytes memory hintConfig) external payable returns (bytes memory) {
	function newBid(uint64 decryptionCondition, address[] memory bidAllowedPeekers) external payable returns (bytes memory) {
		// 0. check offchain execution
		require(Suave.isOffchain());

		// 1. fetch bundle data
		bytes memory bundleData = this.fetchBidConfidentialBundleData();

		// 2. sim bundle
		(bool simOk, uint64 egp) = Suave.simulateBundle(bundleData);
		require(simOk, "bundle does not simulate correctly");

		
		// 3. extract hint
		// TODO : It may be useful to store hints for future
		bytes memory hint = Suave.extractHint(bundleData);
		
		// // 4. store bundle and sim results
		Suave.Bid memory bid = Suave.newBid(decryptionCondition, bidAllowedPeekers, "mevshare:v0:ethBundles");
		Suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundles", bundleData);
		Suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundleSimResults", abi.encode(egp));
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		emit HintEvent(bid.id, hint);

		// // 5. return "callback" to emit hint onchain
		return bytes.concat(this.emitBidAndHint.selector, abi.encode(bid, hint));
	}

	function emitBidAndHint(Suave.Bid calldata bid, bytes memory hint) public {
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		emit HintEvent(bid.id, hint);
	}

	function newMatch(uint64 decryptionCondition, address[] memory bidAllowedPeekers, Suave.BidId shareBidId) external payable returns (bytes memory) {
		// WARNING : this function will copy the original mev share bid
		// into a new key with different permsissions
		
		require(Suave.isOffchain());
		// 1. fetch confidential data
		bytes memory matchBundleData = this.fetchBidConfidentialBundleData();

		// 2. sim match alone for validity
		(bool simOk, uint64 egp) = Suave.simulateBundle(matchBundleData);
		require(simOk, "bundle does not simulate correctly");

		// 3. extract hint
		bytes memory matchHint = Suave.extractHint(matchBundleData);
		
		Suave.Bid memory bid = Suave.newBid(decryptionCondition, bidAllowedPeekers, "mevshare:v0:ethBundles");
		Suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundles", matchBundleData);
		Suave.confidentialStoreStore(bid.id, "mevshare:v0:ethBundleSimResults", abi.encode(egp));

		//4. merge bids
		Suave.BidId[] memory bids = new Suave.BidId[](2);
		bids[0] = shareBidId;
		bids[1] = bid.id;
		Suave.Bid memory mergeBid = Suave.newBid(decryptionCondition, bidAllowedPeekers, "mevshare:v0:mergedBids");
		Suave.confidentialStoreStore(mergeBid.id, "mevshare:v0:mergedBids", abi.encode(bids));

		//5. grab original share bid and extract hint
		// TODO : store hints ?
		bytes memory shareBundleData = Suave.confidentialStoreRetrieve(shareBidId, "mevshare:v0:ethBundles");
		bytes memory bidHint = Suave.extractHint(shareBundleData);

		
		return bytes.concat(this.emitMatchBidAndHint.selector, abi.encode(bid, bidHint, matchHint));
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
	
	function buildMevShare(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight) public returns (bytes memory){
		require(Suave.isOffchain());

		Suave.Bid[] memory allBids = Suave.fetchBids(blockHeight, "mevshare:v0:ethBundles");

		EgpBidPair[] memory bidsByEGP = new EgpBidPair[](allBids.length);
		for (uint i = 0; i < allBids.length; i++) {
			bytes memory simResults = Suave.confidentialStoreRetrieve(allBids[i].id, "mevshare:v0:ethBundleSimResults");
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

		return buildAndEmit(blockArgs, blockHeight, allBidIds, "mevshare:v0");
	}

	function buildFromPool(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight) public returns (bytes memory) {
		require(Suave.isOffchain());

		Suave.Bid[] memory allBids = Suave.fetchBids(blockHeight, "default:v0:ethBundles");

		// TODO: handle merged bids
		EgpBidPair[] memory bidsByEGP = new EgpBidPair[](allBids.length);
		for (uint i = 0; i < allBids.length; i++) {
			bytes memory simResults = Suave.confidentialStoreRetrieve(allBids[i].id, "default:v0:ethBundleSimResults");
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

		return buildAndEmit(blockArgs, blockHeight, allBidIds, "default:v0");
	}

	function buildAndEmit(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight, Suave.BidId[] memory bids, string memory namespace) public virtual returns (bytes memory) {
		require(Suave.isOffchain());

		(Suave.Bid memory blockBid, bytes memory builderBid) = this.doBuild(blockArgs, blockHeight, bids, namespace);

		emit BuilderBoostBidEvent(blockBid.id, builderBid);
		emit BidEvent(blockBid.id, blockBid.decryptionCondition, blockBid.allowedPeekers);
		return bytes.concat(this.emitBuilderBidAndBid.selector, abi.encode(blockBid, builderBid));
	}

	function doBuild(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight, Suave.BidId[] memory bids, string memory namespace) public view returns (Suave.Bid memory, bytes memory) {
		address[] memory allowedPeekers = new address[](2);
		allowedPeekers[0] = address(this);
		allowedPeekers[1] = Suave.BUILD_ETH_BLOCK_PEEKER;

		Suave.Bid memory blockBid = Suave.newBid(blockHeight, allowedPeekers, "default:v0:mergedBids");
		Suave.confidentialStoreStore(blockBid.id, "default:v0:mergedBids", abi.encode(bids));
		 
		(bytes memory builderBid, bytes memory payload) = Suave.buildEthBlock(blockArgs, blockBid.id, namespace);
		Suave.confidentialStoreStore(blockBid.id, "default:v0:builderPayload", payload); // only through this.unlock

		return (blockBid, builderBid);
	}

	function emitBuilderBidAndBid(Suave.Bid memory bid, bytes memory builderBid) public returns (Suave.Bid memory, bytes memory) {
		emit BuilderBoostBidEvent(bid.id, builderBid);
		emit BidEvent(bid.id, bid.decryptionCondition, bid.allowedPeekers);
		return (bid, builderBid);
	}

	function unlock(Suave.BidId bidId, bytes memory signedBlindedHeader) public view returns (bytes memory) {
		require(Suave.isOffchain());

		// TODO: verify the header is correct
		// TODO: incorporate protocol name
		bytes memory payload = Suave.confidentialStoreRetrieve(bidId, "default:v0:builderPayload");
		return payload;
	}
}

contract EthBlockBidSenderContract is EthBlockBidContract {
	string boostRelayUrl;

	constructor(string memory boostRelayUrl_) {
		boostRelayUrl = boostRelayUrl_;
	}

	function buildAndEmit(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight, Suave.BidId[] memory bids, string memory namespace) public virtual override returns (bytes memory) {
		require(Suave.isOffchain());

		(Suave.Bid memory blockBid, bytes memory builderBid) = this.doBuild(blockArgs, blockHeight, bids, namespace);
		(bool ok, bytes memory err) = Suave.submitEthBlockBidToRelay(boostRelayUrl, builderBid);
		if (!ok) {
			revert Suave.PeekerReverted(address(this), err);
		}

		emit BidEvent(blockBid.id, blockBid.decryptionCondition, blockBid.allowedPeekers);
		return bytes.concat(this.emitBid.selector, abi.encode(blockBid));
	}
}
