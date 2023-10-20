package types

import "github.com/ethereum/go-ethereum/common"

type EngineBid struct {
	Id                  BidId
	Salt                BidId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
	CreationTx          *Transaction
	Signature           []byte
}

func (b *EngineBid) ToInnerBid() Bid {
	return Bid{
		Id:                  b.Id,
		Salt:                b.Salt,
		DecryptionCondition: b.DecryptionCondition,
		AllowedPeekers:      b.AllowedPeekers,
		AllowedStores:       b.AllowedStores,
		Version:             b.Version,
	}
}
