pragma solidity ^0.8.19;

import "./RLPEncode.sol";

library EthTransaction {

    struct Transaction {
        uint256 nonce;
	uint256 gasPrice;
	uint256 gasLimit;
	address to;
	uint256 value;
	bytes data;
	// Signature
	uint8 v;
	bytes32 r;
	bytes32 s;
    }

    function serializeNew(Transaction memory _tx, uint chainid) public pure returns (bytes memory) {
	// See https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
	// (nonce, gasprice, startgas, to, value, data, chainid, 0, 0)
	bytes[] memory t = new bytes[](9);
	t[0] = RLPEncode.encodeUint(_tx.nonce);
	t[1] = RLPEncode.encodeUint(_tx.gasPrice);
	t[2] = RLPEncode.encodeUint(_tx.gasLimit);
	t[3] = RLPEncode.encodeAddress(_tx.to);
	t[4] = RLPEncode.encodeUint(_tx.value);
	t[5] = RLPEncode.encodeBytes(_tx.data);
	t[6] = RLPEncode.encodeUint(chainid);
	t[7] = RLPEncode.encodeUint(0);
	t[8] = RLPEncode.encodeUint(0);
	return RLPEncode.encodeList(t);
    }

    function serialize(Transaction memory _tx) public pure returns (bytes memory) {
	// (nonce, gasprice, startgas, to, value, data)
	bytes[] memory t = new bytes[](6);
	t[0] = RLPEncode.encodeUint(_tx.nonce);
	t[1] = RLPEncode.encodeUint(_tx.gasPrice);
	t[2] = RLPEncode.encodeUint(_tx.gasLimit);
	t[3] = RLPEncode.encodeAddress(_tx.to);
	t[4] = RLPEncode.encodeUint(_tx.value);
	t[5] = RLPEncode.encodeBytes(_tx.data);
	return RLPEncode.encodeList(t);
    }

    function verifyTransactionSignature(address _signer, Transaction memory _tx) public pure returns (bool) {
        bytes32 hashedTx = keccak256(serializeNew(_tx, 0x1));
        address recovered = ecrecover(hashedTx, _tx.v, _tx.r, _tx.s);
	return recovered == _signer;
    }
}
