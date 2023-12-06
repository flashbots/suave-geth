pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract AppMevShare {
	event HintEvent(
		Suave.BidId bidId,
		bytes hint
	);
    
	struct Bundle {
		Suave.STransaction txn;
		uint64 refund;
		uint64 blockNum;
	}

	function newBidCallback(Suave.BidId bid) public payable {
		emit HintEvent(bid, abi.encode(1));
	}

	function newBid(Bundle memory bundle) public returns (bytes memory) {
		address[] memory allowedList = new address[](1);
        allowedList[0] = address(this);

		Suave.STransaction[] memory txns = new Suave.STransaction[](1);
		txns[0] = bundle.txn;

		uint64 egp = Suave.simulateTransactions(txns);

		Suave.Bid memory bid = Suave.newBid(0, allowedList, allowedList, "ofa-private");
		Suave.confidentialStore(bid.id, "namespace:data", abi.encode(bundle));
		Suave.confidentialStore(bid.id, "namespace:data:score", abi.encode(egp));

		return abi.encodeWithSelector(this.newBidCallback.selector, bid.id);
	}

	function newMatch(Bundle memory bundle, Suave.BidId shareBidId) public returns (bytes memory) {
		address[] memory allowedList = new address[](1);
        allowedList[0] = address(this);

		Suave.STransaction[] memory txns = new Suave.STransaction[](1);
		txns[0] = bundle.txn;

		uint64 egp = Suave.simulateTransactions(txns);

		Suave.Bid memory bid = Suave.newBid(0, allowedList, allowedList, "ofa-private");
		Suave.confidentialStore(bid.id, "namespace:data", abi.encode(bundle));
		Suave.confidentialStore(bid.id, "namespace:data:score", abi.encode(egp));
		Suave.confidentialStore(bid.id, "namespace:data:shareBidId", abi.encode(shareBidId));
		
		return abi.encodeWithSelector(this.newBidCallback.selector, bid.id);
	}

	function emitBundle(string memory url, Suave.BidId matchBidId) public {
		// retrieve the match bundle
		Bundle memory matchComp = abi.decode(Suave.confidentialRetrieve(matchBidId, "namespace:data"), (Bundle));

		// retrieve the frontrun
		Suave.BidId shareBidId = abi.decode(Suave.confidentialRetrieve(matchBidId, "namespace:data:shareBidId"), (Suave.BidId));
		Bundle memory share = abi.decode(Suave.confidentialRetrieve(shareBidId, "namespace:data"), (Bundle));

		// create the mev-share bundle
		Suave.MevShareBundle memory bundle;
		bundle.inclusionBlock = share.blockNum;

		bundle.transactions = new bytes[](2);
		bundle.transactions[0] = Suave.encodeRLPTxn(share.txn);
		bundle.transactions[1] = Suave.encodeRLPTxn(matchComp.txn);

		// add the refund for the frontrun
		bundle.refundPercents = new uint8[](1);
		bundle.refundPercents[0] = uint8(share.refund);

		Suave.sendMevShareBundle(url, bundle);
	}
}
