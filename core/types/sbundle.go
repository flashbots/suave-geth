package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// Simplified Share Bundle Type for PoC

type SBundle struct {
	Txs             Transactions  `json:"txs"`
	RevertingHashes []common.Hash `json:"revertingHashes,omitempty"`
	RefundPercent   int           `json:"percent,omitempty"`
	MatchId         [16]byte      `json:"MatchId,omitempty"`
}
