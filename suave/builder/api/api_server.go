package api

import (
	"context"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core/types"
)

// sessionManager is the backend that manages the session state of the builder API.
type sessionManager interface {
	NewSession() (string, error)
	AddTransaction(sessionId string, tx *types.Transaction) error
	Finalize(sessionId string) (*engine.ExecutionPayloadEnvelope, error)
}

func NewServer(s sessionManager) *Server {
	api := &Server{
		sessionMngr: s,
	}
	return api
}

type Server struct {
	sessionMngr sessionManager
}

func (s *Server) NewSession(ctx context.Context) (string, error) {
	return s.sessionMngr.NewSession()
}

func (s *Server) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) error {
	return s.sessionMngr.AddTransaction(sessionId, tx)
}

func (s *Server) Finalize(ctx context.Context, sessionId string) (*engine.ExecutionPayloadEnvelope, error) {
	return s.sessionMngr.Finalize(sessionId)
}
