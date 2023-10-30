pragma solidity ^0.8.13;

import "./libraries/bn256g1.sol";
import "./libraries/encryption.sol";

contract Eth {
    event Fulfilled(address from, bytes32 nonce, bytes message);
    uint public ordersFulfilled = 0;
    
    function fulfillOrders(address[] froms,
			   bytes messages) public view
    {
	require(froms.length == nonces.length);
	require(froms.length == message.length);
	for (uint i = 0; i < froms.length; i++) {
	    emit
	}
	ordersFulfilled += 
    }

    function numberFulfilled() public view returns (uint) {
    }
}

contract BatchAuction {
    
    Curve.G1Point public publicKey;

    // Confidential
    // secretKey confidential;
    
    constructor() public {
	// Normally (in Oasis, Secret) we would initialize this on-chain.
	// Without this avialable in SUAVE, we'll just hardcode it for hackathon
	// secretKey := pseudoRandomBytes32();
	bytes32 secretKey;
	secretKey = bytes32(uint(0x424242));
	publicKey = Curve.g1mul(Curve.P1(), uint(secretKey));
    }

    mapping (address => mapping (uint => bool)) public canceled;
    mapping (address => mapping (uint => bytes)) public payloads;

    function submitOrder(uint nonce, bytes memory encryptedPayload) public {
	require(encryptedPayload.length != 0);
	require(canceled[msg.sender][nonce] == false);
	require(payloads[msg.sender][nonce].length == 0);
	payloads[msg.sender][nonce] = encryptedPayload;
    }

    function submitBundle() {
    }

    function call() public {
    }    
}
