// Package scaci handler ping heartbeat tests
//
// SCACI §3.4 Heartbeat Persistence Tests
//
// These tests verify that ping handlers correctly call PersistHeartbeatAsync
// to persist heartbeat timestamps to the database. Tests invoke REAL handlers
// with mock dependencies to verify wiring, not just mock behavior.
//
// Spec Reference: SCACI §3.4 (Ping)
package scaci

import (
	"context"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Synchronizing Mock for Async Operation Verification
// ============================================================================

// syncMockSessionRepository wraps mockSessionRepository with completion signaling
// for deterministic testing of async persistOpIDsPair calls.
type syncMockSessionRepository struct {
	mockSessionRepository
	completionCh chan struct{}
}

func newSyncMockSessionRepository() *syncMockSessionRepository {
	return &syncMockSessionRepository{
		completionCh: make(chan struct{}, 1), // Buffered to avoid blocking
	}
}

func (m *syncMockSessionRepository) UpdateOperationIDs(ctx context.Context, tenantID, sessionID, acOpId, scOpId int64) error {
	err := m.mockSessionRepository.UpdateOperationIDs(ctx, tenantID, sessionID, acOpId, scOpId)
	m.completionCh <- struct{}{} // Signal completion
	return err
}

func (m *syncMockSessionRepository) waitForCompletion() {
	<-m.completionCh
}

// ============================================================================
// Handler-Level Heartbeat Persistence Tests
// ============================================================================

// TestHandlePing_HeartbeatPersisted validates that handlePing invokes
// PersistHeartbeatAsync when session.ID > 0.
func TestHandlePing_HeartbeatPersisted(t *testing.T) {
	mockPersistence := new(MockSessionPersistence)

	s := &Server{
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: mockPersistence,
	}

	session := &Session{
		ID:       123,
		TenantID: 42,
	}

	// Expect PersistHeartbeatAsync called once
	mockPersistence.On("PersistHeartbeatAsync", mock.Anything, session).Return()

	conn := &mockConn{}

	// Call the ACTUAL handler
	err := s.handlePing(conn, session, 1)

	require.NoError(t, err, "handlePing should complete without error")
	mockPersistence.AssertExpectations(t)
	mockPersistence.AssertNumberOfCalls(t, "PersistHeartbeatAsync", 1)
}

// TestHandlePingResponse_HeartbeatPersisted validates that handlePingResponse
// invokes PersistHeartbeatAsync when session.ID > 0.
func TestHandlePingResponse_HeartbeatPersisted(t *testing.T) {
	mockPersistence := new(MockSessionPersistence)

	s := &Server{
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: mockPersistence,
	}

	session := &Session{
		ID:       456,
		TenantID: 99,
	}

	mockPersistence.On("PersistHeartbeatAsync", mock.Anything, session).Return()

	conn := &mockConn{}

	err := s.handlePingResponse(conn, session, -1)

	require.NoError(t, err, "handlePingResponse should complete without error")
	mockPersistence.AssertExpectations(t)
	mockPersistence.AssertNumberOfCalls(t, "PersistHeartbeatAsync", 1)
}

// TestHandlePingComplete_HeartbeatPersisted validates that handlePingComplete
// invokes PersistHeartbeatAsync when session.ID > 0.
func TestHandlePingComplete_HeartbeatPersisted(t *testing.T) {
	mockPersistence := new(MockSessionPersistence)

	s := &Server{
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: mockPersistence,
	}

	session := &Session{
		ID:       789,
		TenantID: 1,
	}

	mockPersistence.On("PersistHeartbeatAsync", mock.Anything, session).Return()

	conn := &mockConn{}

	err := s.handlePingComplete(conn, session, 1)

	require.NoError(t, err, "handlePingComplete should complete without error")
	mockPersistence.AssertExpectations(t)
	mockPersistence.AssertNumberOfCalls(t, "PersistHeartbeatAsync", 1)
}

// TestInitiatePing_HeartbeatPersisted validates that initiatePing invokes
// PersistHeartbeatAsync after a successful send, and persistOpIDsPair updates opIds.
func TestInitiatePing_HeartbeatPersisted(t *testing.T) {
	mockPersistence := new(MockSessionPersistence)
	mockSessionRepo := newSyncMockSessionRepository() // Use sync variant for deterministic wait

	s := &Server{
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: mockPersistence,
		sessionRepo:        mockSessionRepo, // Required for persistOpIDsPair when session.ID > 0
	}

	session := &Session{
		ID:            123,
		TenantID:      42,
		ScOpIdCounter: 0, // Will decrement to -1 via NextScOpId()
	}

	mockPersistence.On("PersistHeartbeatAsync", mock.Anything, session).Return()
	// persistOpIDsPair is called async; mock it to verify it runs
	mockSessionRepo.On("UpdateOperationIDs", mock.Anything, int64(42), int64(123), mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).Return(nil)

	conn := &mockConn{}

	err := s.initiatePing(conn, session)

	require.NoError(t, err, "initiatePing should complete without error")
	mockPersistence.AssertExpectations(t)
	mockPersistence.AssertNumberOfCalls(t, "PersistHeartbeatAsync", 1)

	// Wait for async persistOpIDsPair goroutine to complete
	mockSessionRepo.waitForCompletion()
	mockSessionRepo.AssertExpectations(t)
	mockSessionRepo.AssertNumberOfCalls(t, "UpdateOperationIDs", 1)
}

// ============================================================================
// Negative Cases - Guard Clause Tests
// ============================================================================

// TestHandlePing_SkipsWhenSessionIDZero validates that PersistHeartbeatAsync
// is NOT called when session.ID == 0 (session not yet persisted).
func TestHandlePing_SkipsWhenSessionIDZero(t *testing.T) {
	mockPersistence := new(MockSessionPersistence)

	s := &Server{
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: mockPersistence,
	}

	session := &Session{
		ID:       0, // Not yet persisted
		TenantID: 42,
	}

	// Do NOT set expectation - mock should NOT be called

	conn := &mockConn{}

	err := s.handlePing(conn, session, 1)

	require.NoError(t, err, "handlePing should complete without error")
	mockPersistence.AssertNotCalled(t, "PersistHeartbeatAsync", mock.Anything, mock.Anything)
}

// TestHandlePing_SkipsWhenPersistenceNil validates that handlers don't panic
// when sessionPersistence is nil.
func TestHandlePing_SkipsWhenPersistenceNil(t *testing.T) {
	s := &Server{
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: nil, // Explicitly nil
	}

	session := &Session{
		ID:       123,
		TenantID: 42,
	}

	conn := &mockConn{}

	// Should not panic
	err := s.handlePing(conn, session, 1)

	require.NoError(t, err, "handlePing should complete without error even with nil persistence")
}

// TestInitiatePing_SkipsWhenSessionIDZero validates that initiatePing
// skips PersistHeartbeatAsync when session.ID == 0.
func TestInitiatePing_SkipsWhenSessionIDZero(t *testing.T) {
	mockPersistence := new(MockSessionPersistence)

	s := &Server{
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: mockPersistence,
	}

	session := &Session{
		ID:            0, // Not yet persisted
		TenantID:      42,
		ScOpIdCounter: 0,
	}

	conn := &mockConn{}

	err := s.initiatePing(conn, session)

	require.NoError(t, err, "initiatePing should complete without error")
	mockPersistence.AssertNotCalled(t, "PersistHeartbeatAsync", mock.Anything, mock.Anything)
}
