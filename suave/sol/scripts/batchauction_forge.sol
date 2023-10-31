// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";

import {BatchAuction, EthContract, PKE, Curve, EthTransaction} from "../batchauction.sol";

contract TransactionExample is Script {
    function run() public {
	// Compare with test vector here:
	//   https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
	
	
	// Construct a transaction
	EthTransaction.Transaction memory t = EthTransaction.Transaction({
	    nonce: 9,
	    gasPrice: 20 * 10**9,
	    gasLimit: 21000,
	    to: address(0x3535353535353535353535353535353535353535),
	    // This should be the suave contract 
	    value: 10**18,
	    data: bytes(''), // Calldata, this will be the bit to append
	    v: 0, r: 0, s: 0});

	// Hash the transaction
	bytes memory x = EthTransaction.serializeNew(t, 0x1);
	bytes32 hashedTx = keccak256(x);
	console.logBytes(x);
	console.logBytes32(hashedTx);
    }
}


contract EncryptionExample is Script {
    function run() public {

	BatchAuction auction = new BatchAuction();

	bytes32 secretKey = bytes32(uint(0x4646464646464646464646464646464646464646464646464646464646464646));
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

	bytes memory txn = auction.completeBatch(0, 21 *10**9, 1 *10**6);
	console.logBytes(txn);
	
	vm.stopBroadcast();	
	
	console.log("Ok.");
    }
}
