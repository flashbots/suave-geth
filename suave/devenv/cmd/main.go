package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	_ "embed"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/suave/e2e"
	"github.com/ethereum/go-ethereum/suave/sdk"
)

var (
	exNodeEthAddr = common.HexToAddress("b5feafbdd752ad52afb7e1bd2e40432a485bbb7f")
	exNodeNetAddr = "http://localhost:8545"

	// This account is funded in both devnev networks
	// address: 0xBE69d72ca5f88aCba033a063dF5DBe43a4148De0
	fundedAccount = newPrivKeyFromHex("91ab9a7e53c220e6210460b65a7a3bb2ca181412a8a7b43ff336b3df1737ce12")
)

var (
	bundleBidContract = e2e.BundleBidContract
	mevShareArtifact  = e2e.MevShareBidContract
)

func main() {
	rpc, _ := rpc.Dial(exNodeNetAddr)
	mevmClt := sdk.NewClient(rpc, fundedAccount.priv, exNodeEthAddr)

	var mevShareContract *sdk.Contract
	_ = mevShareContract

	var (
		testAddr1 *privKey
		testAddr2 *privKey
	)

	var (
		ethTxn1       *types.Transaction
		ethTxnBackrun *types.Transaction
	)

	fundBalance := big.NewInt(100000000)
	var bidId [16]byte

	steps := []step{
		{
			name: "Create and fund test accounts",
			action: func() error {
				testAddr1 = generatePrivKey()
				testAddr2 = generatePrivKey()

				if err := fundAccount(mevmClt, testAddr1.Address(), fundBalance); err != nil {
					return err
				}
				fmt.Printf("- Funded test account: %s (%s)\n", testAddr1.Address().Hex(), fundBalance.String())

				// craft mev transactions

				// we use the sdk.Client for the Sign function though we only
				// want to sign simple ethereum transactions and not compute requests
				cltAcct1 := sdk.NewClient(rpc, testAddr1.priv, common.Address{})
				cltAcct2 := sdk.NewClient(rpc, testAddr2.priv, common.Address{})

				targeAddr := testAddr1.Address()

				ethTxn1, _ = cltAcct1.SignTxn(&types.LegacyTx{
					To:       &targeAddr,
					Value:    big.NewInt(1000),
					Gas:      21000,
					GasPrice: big.NewInt(13),
				})

				ethTxnBackrun, _ = cltAcct2.SignTxn(&types.LegacyTx{
					To:       &targeAddr,
					Value:    big.NewInt(1000),
					Gas:      21420,
					GasPrice: big.NewInt(13),
				})
				return nil
			},
		},
		{
			name: "Deploy mev-share contract",
			action: func() error {
				txnResult, err := sdk.DeployContract(mevShareArtifact.Code, mevmClt)
				if err != nil {
					return err
				}
				receipt, err := txnResult.Wait()
				if err != nil {
					return err
				}
				if receipt.Status == 0 {
					return fmt.Errorf("failed to deploy contract")
				}

				fmt.Printf("- Mev share contract deployed: %s\n", receipt.ContractAddress)
				mevShareContract = sdk.GetContract(receipt.ContractAddress, mevShareArtifact.Abi, mevmClt)
				return nil
			},
		},
		{
			name: "Send bid",
			action: func() error {
				refundPercent := 10
				bundle := &types.SBundle{
					Txs:             types.Transactions{ethTxn1},
					RevertingHashes: []common.Hash{},
					RefundPercent:   &refundPercent,
				}
				bundleBytes, _ := json.Marshal(bundle)

				// new bid inputs
				targetBlock := uint64(1)
				allowedPeekers := []common.Address{mevShareContract.Address()}

				confidentialDataBytes, _ := bundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(bundleBytes)

				txnResult, err := mevShareContract.SendTransaction("newBid", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{}}, confidentialDataBytes)
				if err != nil {
					return err
				}
				receipt, err := txnResult.Wait()
				if err != nil {
					return err
				}
				if receipt.Status == 0 {
					return fmt.Errorf("failed to send bid")
				}

				bidEvent := &BidEvent{}
				if err := bidEvent.Unpack(receipt.Logs[0]); err != nil {
					return err
				}
				hintEvent := &HintEvent{}
				if err := hintEvent.Unpack(receipt.Logs[1]); err != nil {
					return err
				}
				bidId = bidEvent.BidId

				fmt.Printf("- Bid sent at txn: %s\n", receipt.TxHash.Hex())
				fmt.Printf("- Bid id: %x\n", bidEvent.BidId)

				return nil
			},
		},
		{
			name: "Send backrun",
			action: func() error {
				backRunBundle := &types.SBundle{
					Txs:             types.Transactions{ethTxnBackrun},
					RevertingHashes: []common.Hash{},
				}
				backRunBundleBytes, _ := json.Marshal(backRunBundle)

				confidentialDataMatchBytes, _ := bundleBidContract.Abi.Methods["fetchBidConfidentialBundleData"].Outputs.Pack(backRunBundleBytes)

				// backrun inputs
				targetBlock := uint64(1)
				allowedPeekers := []common.Address{mevShareContract.Address()}

				txnResult, err := mevShareContract.SendTransaction("newMatch", []interface{}{targetBlock + 1, allowedPeekers, []common.Address{}, bidId}, confidentialDataMatchBytes)
				if err != nil {
					return err
				}
				receipt, err := txnResult.Wait()
				if err != nil {
					return err
				}
				if receipt.Status == 0 {
					return fmt.Errorf("failed to send bid")
				}

				bidEvent := &BidEvent{}
				if err := bidEvent.Unpack(receipt.Logs[0]); err != nil {
					return err
				}

				fmt.Printf("- Backrun sent at txn: %s\n", receipt.TxHash.Hex())
				fmt.Printf("- Backrun bid id: %x\n", bidEvent.BidId)

				return nil
			},
		},
	}

	for indx, step := range steps {
		fmt.Printf("Step %d: %s\n", indx, step.name)
		if err := step.action(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func fundAccount(clt *sdk.Client, to common.Address, value *big.Int) error {
	txn := &types.LegacyTx{
		Value: value,
		To:    &to,
	}
	result, err := clt.SendTransaction(txn)
	if err != nil {
		return err
	}
	_, err = result.Wait()
	if err != nil {
		return err
	}
	// check balance
	balance, err := clt.RPC().BalanceAt(context.Background(), to, nil)
	if err != nil {
		return err
	}
	if balance.Cmp(value) != 0 {
		return fmt.Errorf("failed to fund account")
	}
	return nil
}

type step struct {
	name   string
	action func() error
}

type privKey struct {
	priv *ecdsa.PrivateKey
}

func (p *privKey) Address() common.Address {
	return crypto.PubkeyToAddress(p.priv.PublicKey)
}

func (p *privKey) MarshalPrivKey() []byte {
	return crypto.FromECDSA(p.priv)
}

func newPrivKeyFromHex(hex string) *privKey {
	key, err := crypto.HexToECDSA(hex)
	if err != nil {
		panic(fmt.Sprintf("failed to parse private key: %v", err))
	}
	return &privKey{priv: key}
}

func generatePrivKey() *privKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate private key: %v", err))
	}
	return &privKey{priv: key}
}

type HintEvent struct {
	BidId [16]byte
	Hint  []byte
}

func (h *HintEvent) Unpack(log *types.Log) error {
	unpacked, err := mevShareArtifact.Abi.Events["HintEvent"].Inputs.Unpack(log.Data)
	if err != nil {
		return err
	}
	h.BidId = unpacked[0].([16]byte)
	h.Hint = unpacked[1].([]byte)
	return nil
}

type BidEvent struct {
	BidId               [16]byte
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
}

func (b *BidEvent) Unpack(log *types.Log) error {
	unpacked, err := bundleBidContract.Abi.Events["BidEvent"].Inputs.Unpack(log.Data)
	if err != nil {
		return err
	}
	b.BidId = unpacked[0].([16]byte)
	b.DecryptionCondition = unpacked[1].(uint64)
	b.AllowedPeekers = unpacked[2].([]common.Address)
	return nil
}
