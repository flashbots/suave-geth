// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";

import {BatchAuction, EthContract, PKE, Curve, EthTransaction} from "../batchauction.sol";

contract TransactionExample is Script {
    function run() public {
	
	// Construct a transaction
	EthTransaction.Transaction memory t = EthTransaction.Transaction({
	    nonce: 0,
	    gasPrice: 200000,
	    gasLimit: 1000000,
	    to: address(0), // This should be the suave contract 
	    value: 0,
	    data: bytes(abi.encode(0xdeadbeef)), // Calldata, this will be the bit to append
	    v: 0, r: 0, s: 0});

	bytes memory x = EthTransaction.serialize(t);
	console.logBytes(x);
    }
}


contract EncryptionExample is Script {
    function run() public {

	BatchAuction auction = new BatchAuction();

	bytes32 secretKey = bytes32(uint(0x424242));
	bytes32 r = bytes32(uint(0x1231251)); // This would be sampled randomly by client

	bytes memory message = bytes("hello there suave,      #32bytes");

	// Encrypt the message to the auction contract
	(uint X, uint Y) = auction.publicKey();
	Curve.G1Point memory pub = Curve.G1Point(X,Y);
	bytes memory ciphertext = PKE.encrypt(pub, r, message);
	console.log("Ciphertext:");
	console.logBytes(ciphertext);

	// Decrypt the message (using hardcoded auction contract secretkey):
	bytes memory message2 = PKE.decrypt(secretKey, ciphertext);
	console.log(string(message2));
	console.log("Encryption/Decryption ok");

	// Test some on-chain behavior

	// Check that the message is signed 
	
        vm.startBroadcast();

	EthContract e = new EthContract(address(auction));
	auction.init(address(e));

	auction.submitOrder(ciphertext);
	auction.submitOrder(ciphertext);

	bytes[] memory msgs = auction.completeBatch();
	for (uint i = 0; i < msgs.length; i++)
	    console.log(string(msgs[i]));

	vm.stopBroadcast();	
	
	console.log("Ok.");
    }
}
