pragma solidity ^0.8.19;

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

    function serialize(Transaction memory _tx) public pure returns (bytes memory) {
        return abi.encodePacked(
	    _tx.nonce,
	    _tx.gasPrice,
	    _tx.gasLimit,
	    _tx.to,
	    _tx.value,
	    _tx.data
	);
    }

    function deserialize(bytes memory _data) public pure returns (Transaction memory) {
        (uint256 nonce, uint256 gasPrice, uint256 gasLimit, address to, uint256 value, bytes memory data) = abi.decode(_data, (uint256, uint256, uint256, address, uint256, bytes));

        Transaction memory _tx;
	_tx.nonce = nonce;
	_tx.gasPrice = gasPrice;
	_tx.gasLimit = gasLimit;
	_tx.to = to;
	_tx.value = value;
	_tx.data = data;

        return _tx;
    }

    function verifyTransactionSignature(address _signer, Transaction memory _tx) public pure returns (bool) {
        bytes32 hashedTx = keccak256(serialize(_tx));
	bytes32 prefixedHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", hashedTx));

        address recovered = ecrecover(prefixedHash, _tx.v, _tx.r, _tx.s);
	return recovered == _signer;
    }
}
