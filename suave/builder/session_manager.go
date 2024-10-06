package builder

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/uuid"
)

// blockchain is the minimum interface to the blockchain
// required to build a block
type blockchain interface {
	core.ChainContext

	// Header returns the current tip of the chain
	CurrentHeader() *types.Header

	// StateAt returns the state at the given root
	StateAt(root common.Hash) (*state.StateDB, error)

	// Config returns the chain config
	Config() *params.ChainConfig

	// SubscribeChainHeadEvent to subscribe to ChainHeadEvent
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

type Config struct {
	GasCeil               uint64
	SessionIdleTimeout    time.Duration
	MaxConcurrentSessions int
}

type SessionManager struct {
	sem           chan struct{}
	sessions      map[string]*builder
	sessionTimers map[string]*time.Timer
	sessionsLock  sync.RWMutex
	blockchain    blockchain
	config        *Config
	subscription  event.Subscription
	chainHeadChan chan core.ChainHeadEvent
	exitCh        chan struct{}
	closed        bool
	closeMu       sync.RWMutex
}

func NewSessionManager(blockchain blockchain, config *Config) *SessionManager {
	if config.GasCeil == 0 {
		config.GasCeil = 1000000000000000000
	}
	if config.SessionIdleTimeout == 0 {
		config.SessionIdleTimeout = 5 * time.Second
	}
	if config.MaxConcurrentSessions <= 0 {
		config.MaxConcurrentSessions = 16 // chosen arbitrarily
	}

	sem := make(chan struct{}, config.MaxConcurrentSessions)
	for len(sem) < cap(sem) {
		sem <- struct{}{} // fill 'er up
	}

	s := &SessionManager{
		sem:           sem,
		sessions:      make(map[string]*builder),
		sessionTimers: make(map[string]*time.Timer),
		blockchain:    blockchain,
		config:        config,
		exitCh:        make(chan struct{}),
	}

	s.chainHeadChan = make(chan core.ChainHeadEvent, 100)
	s.subscription = s.blockchain.SubscribeChainHeadEvent(s.chainHeadChan)
	go s.listenForChainHeadEvents()

	return s
}

// NewSession creates a new builder session and returns the session id
func (s *SessionManager) NewSession(ctx context.Context) (string, error) {
	s.closeMu.RLock()
	if s.closed {
		s.closeMu.RUnlock()
		return "", fmt.Errorf("session manager is closed")
	}
	s.closeMu.RUnlock()

	// Wait for session to become available
	select {
	case <-s.sem:
		s.sessionsLock.Lock()
		defer s.sessionsLock.Unlock()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	parent := s.blockchain.CurrentHeader()
	chainConfig := s.blockchain.Config()

	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit, s.config.GasCeil),
		Time:       1000,             // TODO: fix this
		Coinbase:   common.Address{}, // TODO: fix this
		Difficulty: big.NewInt(1),
	}

	// Set baseFee and GasLimit if we are on an EIP-1559 chain
	if chainConfig.IsLondon(header.Number) {
		header.BaseFee = misc.CalcBaseFee(chainConfig, parent)
		if !chainConfig.IsLondon(parent.Number) {
			parentGasLimit := parent.GasLimit * chainConfig.ElasticityMultiplier()
			header.GasLimit = core.CalcGasLimit(parentGasLimit, s.config.GasCeil)
		}
	}

	stateRef, err := s.blockchain.StateAt(parent.Root)
	if err != nil {
		return "", err
	}

	cfg := &builderConfig{
		preState: stateRef,
		header:   header,
		config:   s.blockchain.Config(),
		context:  s.blockchain,
	}

	id := uuid.New().String()[:7]
	s.sessions[id] = newBuilder(cfg)

	// start session timer
	s.sessionTimers[id] = time.AfterFunc(s.config.SessionIdleTimeout, func() {
		s.sessionsLock.Lock()
		defer s.sessionsLock.Unlock()

		delete(s.sessions, id)
		delete(s.sessionTimers, id)

		// Technically, we are certain that there is an open slot in the semaphore
		// channel, but let's be defensive and panic if the invariant is violated.
		select {
		case s.sem <- struct{}{}:
		default:
			panic("released more sessions than are open") // unreachable
		}
	})

	return id, nil
}

func (s *SessionManager) getSession(sessionId string) (*builder, error) {
	s.sessionsLock.RLock()
	defer s.sessionsLock.RUnlock()

	session, ok := s.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionId)
	}

	// reset session timer
	s.sessionTimers[sessionId].Reset(s.config.SessionIdleTimeout)

	return session, nil
}

func (s *SessionManager) AddTransaction(sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error) {
	builder, err := s.getSession(sessionId)
	if err != nil {
		return nil, err
	}
	return builder.AddTransaction(tx)
}

func (s *SessionManager) listenForChainHeadEvents() {
	for {
		select {
		case _, ok := <-s.chainHeadChan:
			if !ok {
				return
			}
			s.terminateAllSessions()
		case <-s.exitCh:
			return
		}
	}
}

func (s *SessionManager) terminateAllSessions() {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()

	for id, session := range s.sessions {
		session.Terminate()

		delete(s.sessions, id)

		if timer, exists := s.sessionTimers[id]; exists {
			timer.Stop()
			delete(s.sessionTimers, id)
		}

		select {
		case s.sem <- struct{}{}:
		default:
			panic("released more sessions than are open")
		}
	}
}

func (s *SessionManager) Close() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closed {
		return
	}

	close(s.exitCh)

	if s.subscription != nil {
		s.subscription.Unsubscribe()
	}

	s.terminateAllSessions()
	s.closed = true
}
