package suave

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethgoAbi "github.com/umbracle/ethgo/abi"
)

type ExecResult struct {
	Logs []*types.Log
}

// Equal compares two ExecResult structs and returns true if they are equal.
// We need a special equal function because `types.Log` is a struct with metadata information
// that is not included (not necessary) during `EncodeABI`.
func (e *ExecResult) Equal(other *ExecResult) bool {
	if len(e.Logs) != len(other.Logs) {
		return false
	}

	for i, log := range e.Logs {
		if log.Address != other.Logs[i].Address {
			return false
		}
	}
	return true
}

var abiExecResult = ethgoAbi.MustNewType(`
tuple(tuple(address addr)[] logs)
`)

type execResultMarshal struct {
	Logs []logMarshal
}

type logMarshal struct {
	Addr common.Address
}

// EncodeABI encodes the ExecResult struct to an ABI byte array.
func (e *ExecResult) EncodeABI() ([]byte, error) {
	// Convert logs to ABI
	abiLogs := make([]logMarshal, len(e.Logs))
	for i, log := range e.Logs {
		abiLogs[i] = logMarshal{log.Address}
	}

	execRes := execResultMarshal{
		Logs: abiLogs,
	}
	data, err := abiExecResult.Encode(execRes)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DecodeABI decodes an ABI byte array to an ExecResult struct.
func (e *ExecResult) DecodeABI(data []byte) error {
	execRes := execResultMarshal{}
	if err := abiExecResult.DecodeStruct(data, &execRes); err != nil {
		return err
	}

	e.Logs = make([]*types.Log, len(execRes.Logs))
	for i, log := range execRes.Logs {
		e.Logs[i] = &types.Log{
			Address: log.Addr,
		}
	}

	return nil
}
