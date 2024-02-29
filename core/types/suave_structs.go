// Code generated by suave/gen. DO NOT EDIT.
// Hash: 294bd11c203e11658343c4d6b0dbb41cfd95cbce67e2eacf3d18fe165d929e68
package types

import "github.com/ethereum/go-ethereum/common"

type DataId [16]byte

// Structs

type BuildBlockArgs struct {
	Slot           uint64
	ProposerPubkey []byte
	Parent         common.Hash
	Timestamp      uint64
	FeeRecipient   common.Address
	GasLimit       uint64
	Random         common.Hash
	Withdrawals    []*Withdrawal
	Extra          []byte
	BeaconRoot     common.Hash
	FillPending    bool
}

type DataRecord struct {
	Id                  DataId
	Salt                DataId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
}

type HttpRequest struct {
	Url                    string
	Method                 string
	Headers                []string
	Body                   []byte
	WithFlashbotsSignature bool
}

type SimulateTransactionResult struct {
	Egp     uint64
	Logs    []*SimulatedLog
	Success bool
	Error   string
}

type SimulatedLog struct {
	Data   []byte
	Addr   common.Address
	Topics []common.Hash
}

type Withdrawal struct {
	Index     uint64
	Validator uint64
	Address   common.Address
	Amount    uint64
}

type CryptoSignature uint8

const (
	CryptoSignature_SECP256 CryptoSignature = 0

	CryptoSignature_BLS CryptoSignature = 1
)
