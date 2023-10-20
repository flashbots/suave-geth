package suave

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

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
