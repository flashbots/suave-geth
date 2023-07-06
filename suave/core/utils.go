package suave

import (
	"encoding/json"
)

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
