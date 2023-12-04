package builder

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

// blockchain is the minimum interface to the blockchain
// required to build a block
type blockchain interface {
	// Header returns the current tip of the chain
	Header() *types.Header

	// StateAt returns the state at the given root
	StateAt(root common.Hash) (*state.StateDB, error)
}

type Config struct {
	GasCeil uint64
}

type SessionManager struct {
	sessions     map[string]*builder
	sessionsLock sync.RWMutex
	blockchain   blockchain
	config       *Config
}

func NewSessionManager(blockchain blockchain, config *Config) *SessionManager {
	if config.GasCeil == 0 {
		config.GasCeil = 1000000000000000000
	}

	s := &SessionManager{
		sessions:   make(map[string]*builder),
		blockchain: blockchain,
		config:     config,
	}
	return s
}

// NewSession creates a new builder session and returns the session id
func (s *SessionManager) NewSession() (string, error) {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	parent := s.blockchain.Header()

	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit, s.config.GasCeil),
		Time:       1000,             // TODO: fix this
		Coinbase:   common.Address{}, // TODO: fix this
	}

	stateRef, err := s.blockchain.StateAt(parent.Root)
	if err != nil {
		return "", err
	}

	cfg := &builderConfig{
		preState: stateRef,
		header:   header,
	}

	id := uuid.New().String()[:7]
	s.sessions[id] = newBuilder(cfg)

	return id, nil
}

func (s *SessionManager) getSession(sessionId string) (*builder, error) {
	s.sessionsLock.RLock()
	defer s.sessionsLock.RUnlock()

	session, ok := s.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionId)
	}
	return session, nil
}

func (s *SessionManager) AddTransaction(sessionId string, tx *types.Transaction) error {
	builder, err := s.getSession(sessionId)
	if err != nil {
		return err
	}
	return builder.AddTransaction(tx)
}

func (s *SessionManager) Finalize(sessionId string) (*engine.ExecutionPayloadEnvelope, error) {
	builder, err := s.getSession(sessionId)
	if err != nil {
		return nil, err
	}

	block, err := builder.Finalize()
	if err != nil {
		return nil, err
	}
	data := &engine.ExecutableData{
		ParentHash:    block.ParentHash(),
		Number:        block.Number().Uint64(),
		GasLimit:      block.GasLimit(),
		GasUsed:       block.GasUsed(),
		LogsBloom:     block.Bloom().Bytes(),
		ReceiptsRoot:  block.ReceiptHash(),
		BlockHash:     block.Hash(),
		StateRoot:     block.Root(),
		Timestamp:     block.Time(),
		ExtraData:     block.Extra(),
		BaseFeePerGas: &big.Int{}, // TODO
		Transactions:  [][]byte{},
	}

	// convert transactions to bytes
	for _, txn := range block.Transactions() {
		txnData, err := txn.MarshalBinary()
		if err != nil {
			return nil, err
		}
		data.Transactions = append(data.Transactions, txnData)
	}

	payload := &engine.ExecutionPayloadEnvelope{
		BlockValue:       big.NewInt(0), // TODO
		ExecutionPayload: data,
	}
	return payload, nil
}
