// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SimpleEncryption {
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
