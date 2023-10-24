package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// Simplified Share Bundle Type for PoC

type SBundle struct {
	// BlockNumber is superseded by DecryptionCondition
	Txs             Transactions  `json:"txs"`
	RevertingHashes []common.Hash `json:"revertingHashes,omitempty"`
	RefundPercent   int           `json:"percent,omitempty"`
	MatchId         BidId         `json:"MatchId,omitempty"`
}
