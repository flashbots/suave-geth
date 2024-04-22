package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

var Moss MossIntrospection

type MossIntrospection interface {
	AddTransaction(txn *types.Transaction) (*types.Receipt, error)
	SendBundleToPool(caller common.Address, bundle []byte) error
}

func (r *suaveRuntime) mossAddTransaction(txnRaw []byte) ([]types.SimulatedLog, bool, string, error) {
	var txn types.Transaction
	if err := rlp.DecodeBytes(txnRaw, &txn); err != nil {
		return nil, false, "", fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	receipt, err := Moss.AddTransaction(&txn)
	if err != nil {
		return nil, false, "", fmt.Errorf("failed to add transaction: %w", err)
	}

	// convert to simulated logs
	var logs []types.SimulatedLog
	for _, log := range receipt.Logs {
		logs = append(logs, types.SimulatedLog{
			Addr:   log.Address,
			Topics: log.Topics,
			Data:   log.Data,
		})
	}

	return logs, true, "", nil
}

func (r *suaveRuntime) mossSendBundle(bundle []byte) error {
	fmt.Println("__ SEND THE BUNDLE __")

	// Pick the second last caller in the stack because the last caller is the address
	// of this precompile
	to := r.suaveContext.CallerStack[len(r.suaveContext.CallerStack)-2]

	if err := Moss.SendBundleToPool(*to, bundle); err != nil {
		panic(err)
	}
	return nil
}
