package backends

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type KeystoreDASigner struct {
	Keystore Keystore
}

type Keystore interface {
	SignHash(a accounts.Account, hash []byte) ([]byte, error)
}

func (w *KeystoreDASigner) Sign(account common.Address, hash []byte) ([]byte, error) {
	return w.Keystore.SignHash(accounts.Account{Address: account}, hash)
}

func (w *KeystoreDASigner) Sender(hash []byte, signature []byte) (common.Address, error) {
	recoveredPubkey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return common.Address{}, err
	}

	recoveredAcc := crypto.PubkeyToAddress(*recoveredPubkey)

	return recoveredAcc, nil
}
