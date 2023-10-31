pragma solidity ^0.8.13;

import "./libraries/bn256g1.sol";
import "./libraries/encryption.sol";
import "./libraries/ethtransaction.sol";

contract EthContract {
    event Fulfilled(string message, uint strikeprice);
    uint public ordersFulfilled = 0;

    function fulfillOrders(bytes[] calldata messages, uint strikeprice) public
    {
	for (uint i = 0; i < messages.length; i++) {
	    emit Fulfilled(string(messages[i]), strikeprice);
	}
	ordersFulfilled += messages.length;
    }

    
}

contract BatchAuction {
    
    Curve.G1Point public publicKey;

    address addr;

    // Confidential
    // secretKey confidential;
    
    constructor() {
	// Normally (in Oasis, Secret) we would initialize this on-chain.
	// Without this avialable in SUAVE, we'll just hardcode it for hackathon
	// secretKey := pseudoRandomBytes32();
	bytes32 secretKey = bytes32(uint(0x424242));
	publicKey = Curve.g1mul(Curve.P1(), uint(secretKey));
    }

    mapping (address => mapping (uint => bool)) public canceled;
    mapping (address => mapping (uint => bytes)) public payloads;

    struct Order {
	address a;
	bytes m;
	bytes32 v; bytes32 r; bytes32 s;
    }

    Order[] orderqueue;

    event OrderSubmitted(address sender, uint nonce);
    function submitOrder(uint nonce, bytes memory encryptedPayload) public {
	require(encryptedPayload.length != 0);
	require(canceled[msg.sender][nonce] == false);
	require(payloads[msg.sender][nonce].length == 0);
	payloads[msg.sender][nonce] = encryptedPayload;
	emit OrderSubmitted(msg.sender, nonce);
    }


    function completeBatch() public {
	// All the 
    }

    function receivePayment() public {
	// The point is that I want to authorize a payment only after a value is confirmed in EthL1.
	
    }
}
