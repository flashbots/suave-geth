package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/flashbots/go-boost-utils/bls"
)

func main() {
	{
		sk, err := crypto.GenerateKey()
		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("\nECDSA key: %s (public: %s) (address: %s)", hexutil.Encode(crypto.FromECDSA(sk)), hexutil.Encode(crypto.CompressPubkey(&sk.PublicKey)), crypto.PubkeyToAddress(sk.PublicKey).Hex())
	}

	{
		sk, pk, err := bls.GenerateNewKeypair()
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("\nBLS key: %s (public: %s)", hexutil.Encode(bls.SecretKeyToBytes(sk)), hexutil.Encode(bls.PublicKeyToBytes(pk)))
	}
}
