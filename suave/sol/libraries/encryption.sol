// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./bn256g1.sol";

library Signing {
    function sign(uint256 privateKey, bytes32 digest) external returns (uint8 v, bytes32 r, bytes32 s) {
	return (0,0,0);
    }
}

library PKE {
    // Improvised... replace with ECIES or similar later
    function encrypt(Curve.G1Point memory pub, bytes32 r, bytes memory m) public view
    returns (bytes memory) {
	Curve.G1Point memory sk    = Curve.g1mul(pub,        uint(r));
	Curve.G1Point memory mypub = Curve.g1mul(Curve.P1(), uint(r));

	// Encrypt using the curve point as secret key
	bytes32 key = bytes32(sk.X);
	(bytes memory ciphertext, bytes32 tag) = SimpleEncryption.encrypt(key, m);
	return abi.encode(mypub.X, mypub.Y, ciphertext, tag);
    }

    function decrypt(bytes32 secretKey, bytes memory ciph) public view
    returns (bytes memory) {
	(uint X, uint Y, bytes memory ciphertext, bytes32 tag) =
	    abi.decode(ciph, (uint, uint, bytes, bytes32));
	Curve.G1Point memory mypub = Curve.G1Point(X,Y);	

	Curve.G1Point memory sk = Curve.g1mul(mypub, uint(secretKey));
	// Decrypt using the curve point as secret key
	bytes32 key = bytes32(sk.X);
	bytes memory message = SimpleEncryption.decrypt(key, ciphertext, tag);
	return message;
    }
}

library SimpleEncryption {
    // This function produces a masking stream using a PRF (in this case, keccak256)
    function produceMaskingStream(bytes32 key, uint256 counter) private pure returns (bytes32) {
	return keccak256(abi.encodePacked(key, counter));
    }
    
    function encrypt(bytes32 key, bytes memory message) public pure returns (bytes memory ciphertext, bytes32 tag) {
        require(message.length % 32 == 0, "Message length should be a multiple of 32");

        ciphertext = new bytes(message.length);

        // Encrypt the message using the masking stream
	for (uint256 i = 0; i < message.length; i += 32) {
	    bytes32 mask = produceMaskingStream(key, i / 32);
	    for (uint256 j = 0; j < 32; j++) {
		ciphertext[i + j] = message[i + j] ^ mask[j];
	    }
	}

        // Compute the MAC
	tag = keccak256(abi.encodePacked(key, ciphertext));

        return (ciphertext, tag);
    }

    function decrypt(bytes32 key, bytes memory ciphertext, bytes32 tag) public pure returns (bytes memory) {
        require(ciphertext.length % 32 == 0, "Ciphertext length should be a multiple of 32");

        // Verify the MAC
	require(keccak256(abi.encodePacked(key, ciphertext)) == tag, "Invalid tag, decryption failed");

        bytes memory decryptedMessage = new bytes(ciphertext.length);

        // Decrypt the message using the masking stream
	for (uint256 i = 0; i < ciphertext.length; i += 32) {
	    bytes32 mask = produceMaskingStream(key, i / 32);
	    for (uint256 j = 0; j < 32; j++) {
		decryptedMessage[i + j] = ciphertext[i + j] ^ mask[j];
	    }
	}

        return decryptedMessage;
    }
}
