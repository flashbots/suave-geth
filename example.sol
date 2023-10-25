=> ./suave/sol/libraries/Suave2.sol
// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.8;

library Suave {
	error PeekerReverted(address, bytes);

	
	
	type BidId is bytes16;
	
	

	address public constant IS_CONFIDENTIAL_ADDR =
	0x0000000000000000000000000000000042010000;
	
	address public constant CONF_STORE_STORE =
	0x0000000000000000000000000000000042020000;
	
	address public constant ETH_CALL_PRECOMPILE =
	0x0000000000000000000000000000000042100003;
	

	// Returns whether execution is off- or on-chain
	function isConfidential() internal view returns (bool b) {
		(bool success, bytes memory isConfidentialBytes) = IS_CONFIDENTIAL_ADDR.staticcall("");
		if (!success) {
			revert PeekerReverted(IS_CONFIDENTIAL_ADDR, isConfidentialBytes);
		}
		assembly {
			// Load the length of data (first 32 bytes)
			let len := mload(isConfidentialBytes)
			// Load the data after 32 bytes, so add 0x20
			b := mload(add(isConfidentialBytes, 0x20))
		}
	}

	
	function confStoreStore (  BidId param1,  string memory param2,  bytes memory param3) external view returns ( ) {
		(bool success, bytes memory data) = CONF_STORE_STORE.staticcall(abi.encode(param1, param2, param3));
		if (!success) {
			revert PeekerReverted(0x, data);
		}
		return abi.decode(data, ());
	}
	
	function ethCallPrecompile (  address param1,  bytes memory param2) external view returns (  bytes memory) {
		(bool success, bytes memory data) = ETH_CALL_PRECOMPILE.staticcall(abi.encode(param1, param2));
		if (!success) {
			revert PeekerReverted(0x, data);
		}
		return abi.decode(data, (bytes));
	}
	
}

