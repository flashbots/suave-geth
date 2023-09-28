package suave

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

var bidUuidSpace = uuid.UUID{0x42}
var emptyId [16]byte

func calculateBidId(bid types.Bid) (types.BidId, error) {
	copy(bid.Id[:], emptyId[:])

	body, err := json.Marshal(bid)
	if err != nil {
		return types.BidId{}, fmt.Errorf("could not marshal bid to calculate its id: %w", err)
	}

	uuidv5 := uuid.NewSHA1(bidUuidSpace, body)
	copy(bid.Id[:], uuidv5[:])

	return bid.Id, nil
}

func RandomBidId() types.BidId {
	return types.BidId(uuid.New())
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
