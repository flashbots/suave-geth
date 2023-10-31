pragma solidity ^0.8.13;

import "./libraries/bn256g1.sol";
import "./libraries/encryption.sol";
import "./libraries/ethtransaction.sol";

contract EthContract {
    event Fulfilled(string message);
    uint public ordersFulfilled = 0;

    address public suaveContract;
    
    constructor (address _suaveContract) {
	suaveContract = _suaveContract;
    }

    function fulfillOrders(bytes[] calldata messages) public
    {
	// Only called by SUAVE
	require(msg.sender == suaveContract);

	for (uint i = 0; i < messages.length; i++) {
	    emit Fulfilled(string(messages[i]));
	}
	ordersFulfilled += messages.length;
    }
}

contract BatchAuction {
    
    Curve.G1Point public publicKey;
    EthContract ethL1contract;

    // Confidential
    // secretKey confidential;

    function init(address _ethL1contract) public {
	require(address(ethL1contract) == address(0));
	ethL1contract = EthContract(_ethL1contract);
    }
    
    constructor() {
	// Normally (in Oasis, Secret) we would initialize this on-chain.
	// Without this avialable in SUAVE, we'll just hardcode it for hackathon
	// secretKey := pseudoRandomBytes32();
	bytes32 secretKey = bytes32(uint(0x424242));
	publicKey = Curve.g1mul(Curve.P1(), uint(secretKey));
    }

    mapping (uint => bytes) orders;
    uint orderCount;
    uint fulfilled;

    event OrderSubmitted(address sender, uint idx, bytes);
    function submitOrder(bytes memory encryptedOrder) public {
	require(encryptedOrder.length != 0);
	orders[orderCount] = encryptedOrder;
	orderCount++;
	emit OrderSubmitted(msg.sender, orderCount-1, encryptedOrder);
    }

    // This should be called offline in confidential mode
    function completeBatch() public returns(bytes[] memory) {
	bytes[] memory msgs = new bytes[](orderCount-fulfilled);

	// Confidential!!!!
	bytes32 secretKey = bytes32(uint(0x424242));
	
	for (uint i = fulfilled; i < orderCount; i++) {
	    // Try to decrypt. If it fails, put "" in its place
	    bytes memory message = PKE.decrypt(secretKey, orders[i]);
	    msgs[i-fulfilled] = message;
	}

	// TODO: 
	// 1.Now construct calldata by encoding these messages
	// 2.Now construct the TX with this as calldata
	// 3.Now sign the transaction
	return msgs;
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
