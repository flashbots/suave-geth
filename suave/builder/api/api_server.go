package api

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
)

// sessionManager is the backend that manages the session state of the builder API.
type sessionManager interface {
	NewSession() (string, error)
	AddTransaction(sessionId string, tx *types.Transaction) (*types.Receipt, error)
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
	fmt.Println("__ NEW SESSION __")
	return s.sessionMngr.NewSession()
}

func (s *Server) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.Receipt, error) {
	return s.sessionMngr.AddTransaction(sessionId, tx)
}

type MockServer struct {
}

func (s *MockServer) NewSession(ctx context.Context) (string, error) {
	fmt.Println("_ NEW SESSION 2 _")
	return "", nil
}

func (s *MockServer) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.Receipt, error) {
	return &types.Receipt{}, nil
}
