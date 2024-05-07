package types

import "github.com/ethereum/go-ethereum/common"

type MossBundle struct {
	To             common.Address
	Data           []byte
	BlockNumber    uint64
	MaxBlockNumber uint64
}

func (m *MossBundle) Copy() *MossBundle {
	return &MossBundle{
		To:             m.To,
		Data:           append([]byte(nil), m.Data...),
		BlockNumber:    m.BlockNumber,
		MaxBlockNumber: m.MaxBlockNumber,
	}
}
