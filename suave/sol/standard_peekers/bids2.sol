// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

import "../libraries/Suave.sol";
import "../suave-std/suavex.sol";
import "../suave-std/mevshare.sol";

contract OFAPrivate {
    address[] public addressList = [0xC8df3686b4Afb2BB53e60EAe97EF043FE03Fb829];

    // Struct to hold hint-related information for an order.
    struct HintOrder {
        Suave.BidId id;
        bytes[] hints;
    }

    event HintEvent (
        Suave.BidId id,
        bytes[] hints
    );

    // Internal function to save order details and generate a hint.
    function saveOrder(string memory url, Types.SBundle memory bundle) internal view returns (HintOrder memory) {
        // Simulate the bundle and extract its score.
        uint256 egp = Suavex.simulateTxn(url, bundle.txs).blockValue;

        // Store the bundle and the simulation results in the confidential datastore.
        Suave.Bid memory bid = Suave.newBid(10, addressList, addressList, "");
        Suave.confidentialStore(bid.id, "mevshare:v0:ethBundles", abi.encode(bundle));
        Suave.confidentialStore(bid.id, "mevshare:v0:ethBundleSimResults", abi.encode(egp));

        // decode hints from the bundle
        bytes[] memory hints = new bytes[](bundle.txs.length);
        for (uint256 i = 0; i < bundle.txs.length; i++) {
            hints[i] = abi.encode(bundle.txs[i].to, bundle.txs[i].data);
        }

        HintOrder memory hintOrder;
        hintOrder.id = bid.id;
        hintOrder.hints = hints;

        return hintOrder;
    }

    function emitHint(HintOrder memory order) public payable {
        emit HintEvent(order.id, order.hints);
    }

    // Function to create a new user order
    function newOrder(string memory url, Types.SBundle memory bundle) external payable returns (bytes memory) {
        HintOrder memory hintOrder = saveOrder(url, bundle);
        return abi.encodeWithSelector(this.emitHint.selector, hintOrder);
    }

    // Function to match and backrun another bid.
    function newMatch(Suave.BidId shareBidId, string memory url, Types.SBundle memory bundle) external payable returns (bytes memory) {
        HintOrder memory hintOrder = saveOrder(url, bundle);

        // Merge the bids and store them in the confidential datastore.
        // The 'fillMevShareBundle' precompile will use this information to send the bundles.
        Suave.BidId[] memory bids = new Suave.BidId[](2);
        bids[0] = shareBidId;
        bids[1] = hintOrder.id;
        Suave.confidentialStore(hintOrder.id, "mevshare:v0:mergedBids", abi.encode(bids));
        
        return abi.encodeWithSelector(this.emitHint.selector, hintOrder);
    }

    function emitMatchBidAndHintCallback() external payable {
    }

    function emitMatchBidAndHint(string memory builderUrl, Suave.BidId bidId) external payable returns (bytes memory) {
        /*
        // retrieve the bids of 'bidId' that we are going to send 
        Suave.BidId[] memory bids = abi.decode(Suave.confidentialRetrieve(bidId, "mevshare:v0:mergedBids"), (Suave.BidId[]));

        // retrieve both bundles
        Types.SBundle memory bundle1 = abi.decode(Suave.confidentialRetrieve(bids[0], "mevshare:v0:ethBundles"), (Types.SBundle));
        Types.SBundle memory bundle2 = abi.decode(Suave.confidentialRetrieve(bids[1], "mevshare:v0:ethBundles"), (Types.SBundle));

        bytes[] memory bodies = new bytes[](2);
        bodies[0] = Types.encodeRLP(bundle1.txs[0]);
        bodies[1] = Types.encodeRLP(bundle2.txs[0]);

        // build the mev share bundle
        MevShare.Bundle memory mevBundle = MevShare.Bundle({
            version: "v0.1",
            inclusionBlock: 0,
            bodies: bodies
            // TODO: refunds
        });

        MevShare.sendBundle(builderUrl, mevBundle);
        */
        
        return abi.encodeWithSelector(this.emitMatchBidAndHintCallback.selector);
    }
}
