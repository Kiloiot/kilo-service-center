// Package scaci provides tests for atomic opId persistence per SCACI §3.2.
//
// Coverage:
//   - persistOpIDs helper function
//   - Atomic persistence of AC and SC operation IDs
//   - Session ID validation (skip if <= 0)
//   - Error handling during persistence
//
// These tests verify that operation ID counters are persisted atomically to
// support session resume per SCACI §3.2 requirements.
package scaci

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// concurrentSettleDelay lets the concurrent persistence goroutines finish before assertions.
const concurrentSettleDelay = 100 * time.Millisecond

// ============================================================================
// Mock Session Repository
// ============================================================================

// mockSessionRepository implements interfaces.SCACISessionRepository for testing
type mockSessionRepository struct {
	mock.Mock
	updateCalls  []opIDsUpdate // Track calls for verification
	disconnected []int64       // Session IDs marked disconnected
	mu           sync.Mutex
}

// opIDsUpdate captures a call to UpdateOperationIDs
type opIDsUpdate struct {
	TenantID  int64
	SessionID int64
	AcOpID    int64
	ScOpID    int64
}

func (m *mockSessionRepository) UpdateOperationIDs(ctx context.Context, tenantID, sessionID, acOpId, scOpId int64) error {
	m.mu.Lock()
	m.updateCalls = append(m.updateCalls, opIDsUpdate{
		TenantID:  tenantID,
		SessionID: sessionID,
		AcOpID:    acOpId,
		ScOpID:    scOpId,
	})
	m.mu.Unlock()
	args := m.Called(ctx, tenantID, sessionID, acOpId, scOpId)
	return args.Error(0)
}

func (m *mockSessionRepository) MarkSessionDisconnected(_ context.Context, _, sessionID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnected = append(m.disconnected, sessionID)
	return nil
}

func (m *mockSessionRepository) disconnectedSessions() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]int64(nil), m.disconnected...)
}

// Implement remaining interface methods with no-op stubs
func (m *mockSessionRepository) CreateSession(_ context.Context, _ *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	return nil, nil
}

func (m *mockSessionRepository) GetSessionByID(_ context.Context, _, _ int64) (*models.SCACISession, error) {
	return nil, nil
}

func (m *mockSessionRepository) GetSessionByAcUUID(_ context.Context, _ models.SCACIApplicationCenter, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}

func (m *mockSessionRepository) GetSessionByScUUID(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}

func (m *mockSessionRepository) UpdateHeartbeat(_ context.Context, _, _ int64) error  { return nil }
func (m *mockSessionRepository) TerminateSession(_ context.Context, _, _ int64) error { return nil }
func (m *mockSessionRepository) ListSessions(_ context.Context, _ *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	return nil, 0, nil
}

func (m *mockSessionRepository) GetSessionStatistics(_ context.Context, _ int64) (*models.SCACISessionStatistics, error) {
	return nil, nil
}

func (m *mockSessionRepository) CheckSessionResumable(_ context.Context, _ models.SCACIApplicationCenter, _ [16]byte) (*models.SCACISessionResumptionInfo, error) {
	return nil, nil
}

// getUpdateCalls returns a copy of the update calls (thread-safe)
func (m *mockSessionRepository) getUpdateCalls() []opIDsUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]opIDsUpdate, len(m.updateCalls))
	copy(result, m.updateCalls)
	return result
}

// ============================================================================
// Test Server Factory
// ============================================================================

func newTestServerWithSessionRepo(repo sessionRowRepo) *Server {
	log := logger.NewNop()
	return &Server{
		registry:           newTestRegistryWithRows(nil, newHolderFake(), sessionRowsOver{repo: repo}),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		logger:             log,
		sessionPersistence: sessionRowsOver{repo: repo},
		config: &Config{
			LogPingOperations: true, // Enable for coverage
		},
	}
}

// ============================================================================
// persistOpIDs Tests
// ============================================================================

const (
	persistTestSessionID = int64(1)
	persistTestTenantID  = int64(100)
	persistTestAcOpID    = int64(42)
	persistTestScOpID    = int64(-10)
)

func persistTestSession(id int64) *Session {
	return withOpIDs(&Session{ID: id, TenantID: persistTestTenantID}, OpIDPair{AC: persistTestAcOpID, SC: persistTestScOpID})
}

// The counter pair is written as one snapshot of the session (SCACI §3.2).
func TestPersistOpIDs_WritesTheSessionSnapshot(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)
	mockRepo.On("UpdateOperationIDs", mock.Anything, persistTestTenantID, persistTestSessionID, persistTestAcOpID, persistTestScOpID).Return(nil)

	server.persistOpIDs(persistTestSession(persistTestSessionID))
	server.persistTasks.wg.Wait()

	mockRepo.AssertExpectations(t)
	assert.Len(t, mockRepo.getUpdateCalls(), 1)
}

func TestPersistOpIDs_SkipsUnpersistedSessions(t *testing.T) {
	for _, id := range []int64{0, -1} {
		mockRepo := new(mockSessionRepository)
		server := newTestServerWithSessionRepo(mockRepo)

		server.persistOpIDs(persistTestSession(id))
		server.persistTasks.wg.Wait()

		assert.Empty(t, mockRepo.getUpdateCalls(), "session %d has no row to update", id)
	}
}

func TestPersistOpIDs_ToleratesStoreFailure(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)
	mockRepo.On("UpdateOperationIDs", mock.Anything, persistTestTenantID, persistTestSessionID, persistTestAcOpID, persistTestScOpID).
		Return(errDatabaseConnectionLost)

	server.persistOpIDs(persistTestSession(persistTestSessionID))
	server.persistTasks.wg.Wait()

	mockRepo.AssertExpectations(t)
}

// syncConn is a connection double that accepts concurrent writers.
type syncConn struct {
	mockConn
	mu sync.Mutex
}

func (c *syncConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(b), nil
}

const concurrentOperations = 200

// The connection handler accepts AC operations while broadcasts start SC
// operations on other goroutines; both counters have one owner, the session,
// so no update is lost and none races.
func TestSessionCounters_ConcurrentAcAndScOperations(t *testing.T) {
	server := newBroadcastULDataServer(nil)
	server.config = &Config{}
	conn := &syncConn{}
	session := activeBroadcastSession(broadcastULDataTestTenant)
	session.ops.restore(OpIDPair{AC: initialOpIDCounter, SC: initialOpIDCounter})
	server.registry.sessions = map[net.Conn]*Session{conn: session}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for opID := int64(1); opID <= concurrentOperations; opID++ {
			sess := session
			assert.NoError(t, server.routeMessage(conn, &sess, nil, CmdPing, opID, nil))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < concurrentOperations; i++ {
			assert.NoError(t, server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, broadcastULDataFixture()))
		}
	}()
	wg.Wait()

	assert.Equal(t, OpIDPair{AC: concurrentOperations, SC: -concurrentOperations}, session.OpIDs())
}

// Sentinel errors returned by this package; callers match them with errors.Is.
var (
	errDatabaseConnectionLost = errors.New("database connection lost")
)

// TestStop_DrainsInFlightPersistence verifies shutdown waits for a detached
// session-state write instead of cancelling its context: the write launched
// just before Stop must complete.
func TestStop_DrainsInFlightPersistence(t *testing.T) {
	mockRepo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(mockRepo)
	server.shutdown = make(chan struct{})
	srvCtx, cancel := testutil.TestContextWithCancel()
	server.ctx = srvCtx
	server.cancel = cancel
	persistCtx, persistCancel := context.WithCancel(context.WithoutCancel(srvCtx))
	server.persistCtx = persistCtx
	server.persistCancel = persistCancel

	release := make(chan struct{})
	done := make(chan struct{})
	mockRepo.On("UpdateOperationIDs", mock.Anything, persistTestTenantID, persistTestSessionID, persistTestAcOpID, persistTestScOpID).
		Run(func(args mock.Arguments) {
			<-release
			// The persistence context must still be live even though Stop has
			// cancelled the main context by now.
			assert.NoError(t, args.Get(0).(context.Context).Err(),
				"the persistence context must survive until the drain finishes")
			close(done)
		}).Return(nil)

	server.persistOpIDs(persistTestSession(persistTestSessionID))
	go func() {
		time.Sleep(concurrentSettleDelay)
		close(release)
	}()

	require.NoError(t, server.Stop())

	select {
	case <-done:
	default:
		t.Fatal("Stop returned before the in-flight persistence completed")
	}
	assert.Len(t, mockRepo.getUpdateCalls(), 1)
}

const (
	lifecycleTestTenant    = int64(100)
	lifecycleTestAcEui     = uint64(0x70B3D59CD0000A01)
	lifecycleTestSessionID = int64(41)
	lifecycleTestNewID     = int64(42)
)

// Only the connection that still owns a session records its loss: a
// connection whose session a newer connection took over leaves it alone.
func TestReleaseConnection_OnlyTheOwningConnectionMarksTheSessionDisconnected(t *testing.T) {
	repo := new(mockSessionRepository)
	server := newTestServerWithSessionRepo(repo)
	oldConn, newConn := &mockConn{}, &mockConn{}
	server.registry.sessions = map[net.Conn]*Session{
		oldConn: {ID: lifecycleTestSessionID, TenantID: lifecycleTestTenant, AcEui: lifecycleTestAcEui, State: StateActive},
	}

	adopts(testutil.TestContext(), server.registry, newConn, &Session{ID: lifecycleTestNewID, TenantID: lifecycleTestTenant, AcEui: lifecycleTestAcEui, State: StateConnecting})
	server.registry.release(testutil.TestContext(), oldConn, SessionPersistTimeout)
	assert.Empty(t, repo.disconnectedSessions(), "the superseded connection no longer owns a session")

	server.registry.release(testutil.TestContext(), newConn, SessionPersistTimeout)
	assert.Equal(t, []int64{lifecycleTestNewID}, repo.disconnectedSessions())
	assert.Empty(t, server.registry.sessions)
}

func TestAdoptSession_KeepsOtherApplicationCentersAndTenants(t *testing.T) {
	server := newTestServerWithSessionRepo(new(mockSessionRepository))
	otherAC, otherTenant, newConn := &mockConn{}, &mockConn{}, &mockConn{}
	server.registry.sessions = map[net.Conn]*Session{
		otherAC:     {ID: lifecycleTestSessionID, TenantID: lifecycleTestTenant, AcEui: lifecycleTestAcEui + 1},
		otherTenant: {ID: lifecycleTestSessionID - 1, TenantID: lifecycleTestTenant + 1, AcEui: lifecycleTestAcEui},
	}

	adopts(testutil.TestContext(), server.registry, newConn, &Session{ID: lifecycleTestNewID, TenantID: lifecycleTestTenant, AcEui: lifecycleTestAcEui})

	assert.Len(t, server.registry.sessions, 3)
}
