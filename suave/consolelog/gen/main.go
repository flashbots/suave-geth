package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// read the console.sol Solidity file from input args
	console, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	rxp := regexp.MustCompile("abi.encodeWithSignature\\(\"log(.*)\"")
	matches := rxp.FindAllStringSubmatch(string(console), -1)

	methodMap := map[string]string{}
	for _, match := range matches {
		signature := match[1]

		// signature of the call. Use the version without the bytes in 'uint'.
		sig := crypto.Keccak256([]byte("log" + match[1]))[:4]
		methodMap["log"+signature] = hex.EncodeToString(sig)
	}

	raw, err := json.MarshalIndent(methodMap, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
}
