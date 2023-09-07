package suave

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

var bidUuidSpace = uuid.UUID{0x42}

func calculateBidId(bid types.Bid) (BidId, error) {
	var emptyId [32]byte
	if bid.Id == emptyId {
		randomUuid := uuid.New()
		copy(bid.Id[:], randomUuid[:])
	}

	// clear second half
	for i := 16; i < 32; i++ {
		bid.Id[i] = 0
	}
	body, err := json.Marshal(bid)
	if err != nil {
		return BidId{}, fmt.Errorf("could not marshal bid to calculate its id: %w", err)
	}

	// hash part
	uuidv5 := uuid.NewSHA1(bidUuidSpace, body)

	copy(bid.Id[16:], uuidv5[:])
	return bid.Id, nil
}

func RandomBidId() BidId {
	var bidId [32]byte

	randomUuid := uuid.New()
	copy(bidId[:], randomUuid[:])
	return bidId
}

func MustEncode[T any](data T) []byte {
	res, err := json.Marshal(data)
	if err != nil {
		panic(err.Error())
	}
	return res
}

func MustDecode[T any](data []byte) T {
	var t T
	if err := json.Unmarshal(data, &t); err != nil {
		panic(err.Error())
	}
	return t
}
