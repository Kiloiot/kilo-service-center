package bssci

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/google/uuid"
)

// NewServerForTesting creates a minimal Server instance for external tests.
// FOR TESTS ONLY - provides lightweight Server construction without full initialization.
func NewServerForTesting(log logger.Logger) *Server {
	s := &Server{
		logger:   log,
		sessions: make(map[string]*Session),
		// mu relies on zero-value initialization (valid for sync.RWMutex)
	}
	return s
}

// AddSessionForTesting registers a session on the server for external tests.
func (s *Server) AddSessionForTesting(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}

// NopUplinkIngestService is a no-op UplinkIngestService for tests that need handleULData
// to proceed without real ingest logic.
type NopUplinkIngestService struct{}

// Ingest returns a minimal successful result.
func (n *NopUplinkIngestService) Ingest(_ context.Context, _ *UplinkPayload, _ UplinkIngestOptions) (*IngestResult, error) {
	return &IngestResult{
		OwnerTenantID: 0,
		OwnerOrgUUID:  uuid.Nil,
		MessageID:     "test-msg-id",
	}, nil
}
