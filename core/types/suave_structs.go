// Hash: c47d1edbb0c5bf09f52a6db80269fdd9cdd62ad7b06b95a4625de00d74ac316f
package types

import "github.com/ethereum/go-ethereum/common"

type BidId [16]byte

// Structs

type Bid struct {
	Id                  BidId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
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

type Withdrawal struct {
	Index     uint64
	Validator uint64
	Address   common.Address
	Amount    uint64
}
