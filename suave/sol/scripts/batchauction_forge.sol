// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";
import "../libraries/secp256k1.sol";
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

	{
	    bytes32 sk = bytes32(0x4646464646464646464646464646464646464646464646464646464646464646);
	    //bytes32 sk = bytes32(0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d);
	    (uint qx, uint qy) = Secp256k1.derivePubKey(uint(sk));
	    address a = auction.deriveAddress(uint(sk));
	    bytes memory ser = bytes.concat(bytes32(qx), bytes32(qy));
	    console.logAddress(a);
	}


	bytes32 secretKey = bytes32(uint(0x4646464646464646464646464646464646464646464646464646464646464646));
	bytes32 r = bytes32(uint(0x1231251)); // This would be sampled randomly by client

	bytes memory message = bytes("hello there suave,      #32bytes");

	// Encrypt the message to the auction contract
	bytes memory ciphertext = auction.encryptMessage(message, r);
	console.log("Ciphertext:");
	console.logBytes(ciphertext);

	// Decrypt the message (using hardcoded auction contract secretkey):
	bytes memory message2 = PKE.decrypt(secretKey, ciphertext);
	console.log(string(message2));
	console.log("Encryption/Decryption ok");

	// Test some on-chain behavior

	// Check that the message is signed 
	
        vm.startBroadcast();

	EthContract e = new EthContract(auction.suappPubkey());
	auction.init(address(e));
	console.log("Suapp Pubkey:");
	console.logAddress(auction.suappPubkey());

	auction.submitOrder(ciphertext);
	auction.submitOrder(ciphertext);

	bytes memory txn = auction.completeBatch(0, 21 *10**9, 1 *10**6);
	console.logBytes(txn);
	
	vm.stopBroadcast();	
	
	console.log("Ok.");
    }
}
