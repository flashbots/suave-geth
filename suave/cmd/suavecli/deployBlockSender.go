package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func cmdDeployBlockSenderContract() {
	flagset := flag.NewFlagSet("deployBlockSenderContract", flag.ExitOnError)

	var (
		suaveRpc      = flagset.String("suave_rpc", "http://127.0.0.1:8545", "address of suave rpc")
		privKeyHex    = flagset.String("privkey", "", "private key as hex (for testing)")
		boostRelayUrl = flagset.String("relay_url", "http://127.0.0.1:8091", "address of boost relay that the contract will send blocks to")
		verbosity     = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
		privKey       *ecdsa.PrivateKey
	)

	flagset.Parse(os.Args[2:])

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	privKey, err := crypto.HexToECDSA(*privKeyHex)
	RequireNoErrorf(err, "-nodekeyhex: %v", err)
	/* shush linter */ privKey.Public()

	suaveClient, err := rpc.DialContext(context.TODO(), *suaveRpc)
	RequireNoErrorf(err, "could not connect to suave rpc: %v", err)

	genesis := core.DefaultSuaveGenesisBlock()

	suaveSigner := types.NewSuaveSigner(genesis.Config.ChainID)

	ethBlockBidSenderAddr, txHash, _ := sendBlockSenderCreationTx(suaveClient, suaveSigner, privKey, boostRelayUrl)

	// TODO: wait until tx is included and check receipt

	for i := 0; i < 10; i++ {
		time.Sleep(time.Duration(1+i/2) * time.Second)

		var receipt = make(map[string]interface{})
		err = suaveClient.Call(&receipt, "eth_getTransactionReceipt", txHash)
		if err == nil && receipt != nil {
			log.Info("All is good!", "receipt", receipt, "address", ethBlockBidSenderAddr)
			return
		}
	}

	utils.Fatalf("did not see the receipt succeed in time. hash: %s", txHash.String())
}

func sendBlockSenderCreationTx(suaveClient *rpc.Client, suaveSigner types.Signer, privKey *ecdsa.PrivateKey, boostRelayUrl *string) (*common.Address, *common.Hash, error) {
	var suaveAccNonceBytes hexutil.Uint64
	err := suaveClient.Call(&suaveAccNonceBytes, "eth_getTransactionCount", crypto.PubkeyToAddress(privKey.PublicKey), "latest")
	RequireNoErrorf(err, "could not call eth_getTransactionCount on suave: %v", err)
	suaveAccNonce := uint64(suaveAccNonceBytes)

	var suaveGp hexutil.Big
	err = suaveClient.Call(&suaveGp, "eth_gasPrice")
	RequireNoErrorf(err, "could not call eth_gasPrice on suave: %v", err)

	abiEncodedRelayUrl, err := ethBlockBidSenderAbi.Pack("", boostRelayUrl)
	RequireNoErrorf(err, "could not pack inputs to the constructor: %v", err)

	calldata := append(hexutil.MustDecode(blockSenderContractBytecode), abiEncodedRelayUrl...)
	ccTxData := &types.LegacyTx{
		Nonce:    suaveAccNonce,
		To:       nil, // contract creation
		Value:    big.NewInt(0),
		Gas:      10000000,
		GasPrice: (*big.Int)(&suaveGp),
		Data:     calldata,
	}

	tx, err := types.SignTx(types.NewTx(ccTxData), suaveSigner, privKey)
	RequireNoErrorf(err, "could not sign the deployment transaction: %v", err)

	from, _ := types.Sender(suaveSigner, tx)
	ethBlockBidSenderAddr := crypto.CreateAddress(from, tx.Nonce())
	log.Info("contract address will be", "addr", ethBlockBidSenderAddr)

	txBytes, err := tx.MarshalBinary()
	RequireNoErrorf(err, "could not marshal the deployment transaction: %v", err)

	var txHash common.Hash
	err = suaveClient.Call(&txHash, "eth_sendRawTransaction", hexutil.Encode(txBytes))
	RequireNoErrorf(err, "could not send the deployment transaction to suave node: %v", err)

	return &ethBlockBidSenderAddr, &txHash, nil
}
