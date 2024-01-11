package vm

import (
	"github.com/ethereum/go-ethereum/suave/offchain"
)

type System interface {
	Blocks() offchain.Blockstore
}
