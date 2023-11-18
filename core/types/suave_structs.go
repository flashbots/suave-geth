// Code generated by suave/gen. DO NOT EDIT.
// Hash: 1533cfa728d1e455a6e09acc62aaa8dadc4f44ae630eeac37107ad0555f925c7
package types

import "github.com/ethereum/go-ethereum/common"

type BidId [16]byte

// Structs

type Bid struct {
	Id                  BidId
	Salt                BidId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
}

type BuildBlockArgs struct {
	Slot           uint64
	ProposerPubkey []byte
	Parent         common.Hash
	Timestamp      uint64
	FeeRecipient   common.Address
	GasLimit       uint64
	Random         common.Hash
	Withdrawals    []*Withdrawal
}

type SimulateTransactionResult struct {
	Egp  uint64
	Logs []*SimulatedLog
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
