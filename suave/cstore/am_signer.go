package cstore

import (
	"github.com/flashbots/suave-geth/accounts"
	"github.com/flashbots/suave-geth/common"
	"github.com/flashbots/suave-geth/crypto"
)

type AccountManagerDASigner struct {
	Manager *accounts.Manager
}

func (w *AccountManagerDASigner) Sign(account common.Address, data []byte) ([]byte, error) {
	keystoreAcc := accounts.Account{Address: account}
	wallet, err := w.Manager.Find(keystoreAcc)
	if err != nil {
		return nil, err
	}
	return wallet.SignData(keystoreAcc, "", data)
}

func (w *AccountManagerDASigner) Sender(data []byte, signature []byte) (common.Address, error) {
	hash := crypto.Keccak256(data)
	recoveredPubkey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return common.Address{}, err
	}

	recoveredAcc := crypto.PubkeyToAddress(*recoveredPubkey)

	return recoveredAcc, nil
}

func (w *AccountManagerDASigner) LocalAddresses() []common.Address {
	return w.Manager.Accounts()
}
