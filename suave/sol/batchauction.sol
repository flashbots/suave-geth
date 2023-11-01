pragma solidity ^0.8.13;

import "./libraries/Suave.sol";
import "./libraries/bn256g1.sol";
import "./libraries/encryption.sol";
import "./libraries/ethtransaction.sol";
import "./libraries/secp256k1.sol";

contract EthContract {
    event Fulfilled(string message);
    uint public ordersFulfilled = 0;

    address public suappPubkey;
    
    constructor (address _suappPubkey) {
	suappPubkey = _suappPubkey;
    }

    function fulfillOrders(bytes[] calldata messages) public
    {
	// Can only be invoked by the SuApp itself
	require(msg.sender == suappPubkey);

	for (uint i = 0; i < messages.length; i++) {
	    emit Fulfilled(string(messages[i]));
	}
	ordersFulfilled += messages.length;
    }
}

contract BatchAuction {
    
    Curve.G1Point public publicKey;
    address public suappPubkey;
    EthContract ethL1contract;

    // Confidential:
    // secretKey confidential;

    function init(address _ethL1contract) public {
	require(address(ethL1contract) == address(0));
	require(EthContract(_ethL1contract).suappPubkey() == suappPubkey);
	ethL1contract = EthContract(_ethL1contract);
    }

    function deriveAddress(uint secretKey) pure public returns(address) {
	(uint qx, uint qy) = Secp256k1.derivePubKey(secretKey);
	bytes memory ser = bytes.concat(bytes32(qx), bytes32(qy));
	return address(uint160(uint256(keccak256(ser))));
    }
    
    constructor() {
	// Normally (in Oasis, Secret) we would initialize this on-chain.
	// It will be possible to do this through a round of off-chain communication
	// in SUAVE.
	// For now in the hackathon, we'll just hardcode it
	// secretKey := pseudoRandomBytes32();
	bytes32 secretKey = bytes32(uint(0x4646464646464646464646464646464646464646464646464646464646464646));
	publicKey = Curve.g1mul(Curve.P1(), uint(secretKey));
	suappPubkey = deriveAddress(uint(secretKey));
    }

    mapping (uint => bytes) orders;
    uint orderCount;
    uint fulfilled;

    function encryptMessage(bytes memory message, bytes32 nonce) public view returns (bytes memory) {
	//require(message.length / 32 == 0);
	return PKE.encrypt(publicKey, nonce, message);
    }

    event OrderSubmitted(address sender, uint idx, bytes);
    function submitOrder(bytes memory encryptedOrder) public {
	require(encryptedOrder.length != 0);
	orders[orderCount] = encryptedOrder;
	orderCount++;
	emit OrderSubmitted(msg.sender, orderCount-1, encryptedOrder);
    }

    // This should be called offline in confidential mode
    function completeBatch(uint nonce, uint gasPrice, uint gasLimit) public view returns(bytes memory) {
	bytes[] memory msgs = new bytes[](orderCount-fulfilled);

	// Confidential!!!!
	bytes32 secretKey = bytes32(uint(0x4646464646464646464646464646464646464646464646464646464646464646));
	
	for (uint i = fulfilled; i < orderCount; i++) {
	    // Try to decrypt. If it fails, put "" in its place
	    bytes memory message = PKE.decrypt(secretKey, orders[i]);
	    msgs[i-fulfilled] = message;
	}

	// 1.Now construct calldata by encoding these messages
	bytes memory data = abi.encodeWithSignature("fulfillOrders(bytes[])", msgs);
	
	// 2.Now construct the TX with this as calldata
	EthTransaction.Transaction memory t = EthTransaction.Transaction({
	    nonce: nonce,
	    gasPrice: gasPrice,
	    gasLimit: gasLimit,
	    to: address(ethL1contract),
	    value: 0,
	    data: data,
	    v: 0, r: 0, s: 0});
	bytes memory txn = EthTransaction.serializeNew(t, 0x5);
	
	// 3. TODO: Now sign the transaction
	bytes memory t2 = Suave.signEthTransaction(txn, "5", "4646464646464646464646464646464646464646464646464646464646464646");

	// 4. TODO: Finally send the bundle
	bytes memory bundle = t2;
	Suave.submitBundleJsonRPC("https://rpc-goerli.flashbots.net", "eth_sendBundle", bundle);
	
	return txn;
    }

    // This will be called to have the SUAVE chain catch up with its
    // view of the Ethereum chain
    function advanceBatch() public {
	uint ethL1ordersFulfilled = 0;
	// TODO: Call the Eth L1
	// ethL1ordersFulfilled = suave.ethcall("ordersFulfilled");

	for (uint i = fulfilled; i < ethL1ordersFulfilled; i++) {
	    delete orders[i];
	}
	fulfilled = ethL1ordersFulfilled;
    }
}
