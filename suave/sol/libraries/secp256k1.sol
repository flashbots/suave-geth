pragma solidity ^0.8.0;

import "./EllipticCurve.sol";

library Secp256k1 {

    uint256 public constant GX = 0x79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798;
    uint256 public constant GY = 0x483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8;
    uint256 public constant AA = 0;
    uint256 public constant BB = 7;
    uint256 public constant PP = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F;

    function derivePubKey(uint256 privKey) public pure returns(uint256 qx, uint256 qy) {
	(qx, qy) = EllipticCurve.ecMul(
	    privKey,
	    GX,
	    GY,
	    AA,
	    PP
	);
    }
}
