// Package scaci handler ping heartbeat tests
//
// SCACI §3.4 Heartbeat Persistence Tests
//
// These tests verify that ping handlers correctly call PersistHeartbeat
// to persist heartbeat timestamps to the database. Tests invoke REAL handlers
// with mock dependencies to verify wiring, not just mock behavior.
//
// Spec Reference: SCACI §3.4 (Ping)
package scaci

import (
	"context"
	"sync"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Synchronizing Mock for Async Operation Verification
// ============================================================================

// syncMockSessionRepository wraps mockSessionRepository with completion signaling
// for deterministic testing of async persistOpIDs calls.
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

// heartbeatFake records heartbeat writes by session ID. Unlike a testify
// mock it never formats the live session the handler keeps using, and it can
// hold a write open until released.
type heartbeatFake struct {
	mu       sync.Mutex
	sessions []int64
	entered  chan struct{}
	release  chan struct{}
	counters sessionRowRepo
}

func (f *heartbeatFake) PersistHeartbeat(_ context.Context, session *Session) error {
	f.mu.Lock()
	f.sessions = append(f.sessions, session.ID)
	f.mu.Unlock()
	if f.release != nil {
		close(f.entered)
		<-f.release
	}
	return nil
}

// PersistOpIDs writes to the counter repository when the test observes it.
func (f *heartbeatFake) PersistOpIDs(ctx context.Context, session *Session, ids OpIDPair) error {
	if f.counters == nil {
		return nil
	}
	return sessionRowsOver{repo: f.counters}.PersistOpIDs(ctx, session, ids)
}

func (f *heartbeatFake) PersistConnectSync(context.Context, *Session, string, string, string, string, string, string) (int64, error) {
	return 0, nil
}

func (f *heartbeatFake) heartbeats() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int64(nil), f.sessions...)
}

// ============================================================================
// Handler-Level Heartbeat Persistence Tests
// ============================================================================

// TestHandlePing_HeartbeatPersisted validates that handlePing invokes
// PersistHeartbeat when session.ID > 0.
func TestHandlePing_HeartbeatPersisted(t *testing.T) {
	heartbeats := &heartbeatFake{}

	s := &Server{
		registry:           newTestRegistry(nil, nil),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: heartbeats,
	}

	session := &Session{
		ID:       123,
		TenantID: 42,
	}

	conn := &mockConn{}

	// Call the ACTUAL handler
	err := s.handlePing(conn, session, 1)

	require.NoError(t, err, "handlePing should complete without error")
	s.persistTasks.wg.Wait()
	assert.Equal(t, []int64{session.ID}, heartbeats.heartbeats())
}

// TestHandlePingResponse_HeartbeatPersisted validates that handlePingResponse
// invokes PersistHeartbeat when session.ID > 0.
func TestHandlePingResponse_HeartbeatPersisted(t *testing.T) {
	heartbeats := &heartbeatFake{}

	s := &Server{
		registry:           newTestRegistry(nil, nil),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: heartbeats,
	}

	session := &Session{
		ID:       456,
		TenantID: 99,
	}

	conn := &mockConn{}

	err := s.handlePingResponse(conn, session, -1)

	require.NoError(t, err, "handlePingResponse should complete without error")
	s.persistTasks.wg.Wait()
	assert.Equal(t, []int64{session.ID}, heartbeats.heartbeats())
}

// TestHandlePingComplete_HeartbeatPersisted validates that handlePingComplete
// invokes PersistHeartbeat when session.ID > 0.
func TestHandlePingComplete_HeartbeatPersisted(t *testing.T) {
	heartbeats := &heartbeatFake{}

	s := &Server{
		registry:           newTestRegistry(nil, nil),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: heartbeats,
	}

	session := &Session{
		ID:       789,
		TenantID: 1,
	}

	conn := &mockConn{}

	err := s.handlePingComplete(conn, session, 1)

	require.NoError(t, err, "handlePingComplete should complete without error")
	s.persistTasks.wg.Wait()
	assert.Equal(t, []int64{session.ID}, heartbeats.heartbeats())
}

// TestInitiatePing_HeartbeatPersisted validates that initiatePing invokes
// PersistHeartbeat after a successful send, and persistOpIDs updates opIds.
func TestInitiatePing_HeartbeatPersisted(t *testing.T) {
	mockSessionRepo := newSyncMockSessionRepository() // Use sync variant for deterministic wait
	heartbeats := &heartbeatFake{counters: mockSessionRepo}

	s := &Server{
		registry:           newTestRegistry(nil, nil),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: heartbeats, // persistOpIDs writes the counters through it when session.ID > 0
	}

	session := &Session{
		ID:       123,
		TenantID: 42,
	}

	// persistOpIDs is called async; mock it to verify it runs
	mockSessionRepo.On("UpdateOperationIDs", mock.Anything, int64(42), int64(123), mock.AnythingOfType("int64"), mock.AnythingOfType("int64")).Return(nil)

	conn := &mockConn{}

	err := s.initiatePing(conn, session)

	require.NoError(t, err, "initiatePing should complete without error")
	s.persistTasks.wg.Wait()
	assert.Equal(t, []int64{session.ID}, heartbeats.heartbeats())

	// Wait for async persistOpIDs goroutine to complete
	mockSessionRepo.waitForCompletion()
	mockSessionRepo.AssertExpectations(t)
	mockSessionRepo.AssertNumberOfCalls(t, "UpdateOperationIDs", 1)
}

// ============================================================================
// Negative Cases - Guard Clause Tests
// ============================================================================

// TestHandlePing_SkipsWhenSessionIDZero validates that PersistHeartbeat
// is NOT called when session.ID == 0 (session not yet persisted).
func TestHandlePing_SkipsWhenSessionIDZero(t *testing.T) {
	heartbeats := &heartbeatFake{}

	s := &Server{
		registry:           newTestRegistry(nil, nil),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: heartbeats,
	}

	session := &Session{
		ID:       0, // Not yet persisted
		TenantID: 42,
	}

	// Do NOT set expectation - mock should NOT be called

	conn := &mockConn{}

	err := s.handlePing(conn, session, 1)

	require.NoError(t, err, "handlePing should complete without error")
	s.persistTasks.wg.Wait()
	assert.Empty(t, heartbeats.heartbeats())
}

// TestHandlePing_SkipsWhenPersistenceNil validates that handlers don't panic
// when sessionPersistence is nil.
func TestHandlePing_SkipsWhenPersistenceNil(t *testing.T) {
	s := &Server{
		registry:           newTestRegistry(nil, nil),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
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
// skips PersistHeartbeat when session.ID == 0.
func TestInitiatePing_SkipsWhenSessionIDZero(t *testing.T) {
	heartbeats := &heartbeatFake{}

	s := &Server{
		registry:           newTestRegistry(nil, nil),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		config:             &Config{LogPingOperations: false},
		logger:             logger.NewNop(),
		sessionPersistence: heartbeats,
	}

	session := &Session{
		ID:       0, // Not yet persisted
		TenantID: 42,
	}

	conn := &mockConn{}

	err := s.initiatePing(conn, session)

	require.NoError(t, err, "initiatePing should complete without error")
	s.persistTasks.wg.Wait()
	assert.Empty(t, heartbeats.heartbeats())
}
