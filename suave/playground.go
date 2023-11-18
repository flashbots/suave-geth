package main

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/e2e"
	"github.com/ethereum/go-ethereum/suave/sdk"
)

var (
	ExNodeEthAddr = common.HexToAddress("b5feafbdd752ad52afb7e1bd2e40432a485bbb7f")
	ExNodeNetAddr = "http://localhost:8545"

	// This account is funded in both devnev networks
	// address: 0xBE69d72ca5f88aCba033a063dF5DBe43a4148De0
	FundedAccount = newPrivKeyFromHex("91ab9a7e53c220e6210460b65a7a3bb2ca181412a8a7b43ff336b3df1737ce12")
)

func main() {
	artifact := e2e.NewArtifact("../suave/artifacts/example.sol/ConfidentialStore.json")
	rpc, _ := rpc.Dial(ExNodeNetAddr)

	clt := sdk.NewClient(rpc, FundedAccount.Priv, ExNodeEthAddr)
	result, err := sdk.DeployContract(artifact.Code, clt)
	if err != nil {
		panic(err)
	}
	receipt, err := result.Wait()
	if err != nil {
		panic(err)
	}
	fmt.Println(receipt.ContractAddress)

	contract := sdk.GetContract(receipt.ContractAddress, artifact.Abi, clt)
	result, err = contract.SendTransaction("example", []interface{}{}, nil)
	if err != nil {
		panic(err)
	}
	receipt, err = result.Wait()
	if err != nil {
		panic(err)
	}

	fmt.Println(receipt.Logs[0].Topics[1])

	//blockNumAsHash := receipt.Logs[0]
	//fmt.Println(blockNumAsHash)
}

type privKey struct {
	Priv *ecdsa.PrivateKey
}

func (p *privKey) Address() common.Address {
	return crypto.PubkeyToAddress(p.Priv.PublicKey)
}

func (p *privKey) MarshalPrivKey() []byte {
	return crypto.FromECDSA(p.Priv)
}

func newPrivKeyFromHex(hex string) *privKey {
	key, err := crypto.HexToECDSA(hex)
	if err != nil {
		panic(fmt.Sprintf("failed to parse private key: %v", err))
	}
	return &privKey{Priv: key}
}
