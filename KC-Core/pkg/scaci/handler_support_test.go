// Package scaci provides handler-level integration tests for SCACI operations.
//
// Coverage:
//   - handleULDataTransmit missing userData returns POSIX_EINVAL (§2.4)
//   - handleULDataTransmit format field defaults to 0 when absent (§3.9.1)
package scaci

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/nettransport"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// asyncSettleDelay lets goroutines spawned by an asynchronous handler finish.
const asyncSettleDelay = 50 * time.Millisecond

// testWriteTimeout bounds the frame writes of servers the tests build directly.
const testWriteTimeout = 5 * time.Second

// testFrameCodec is the codec NewServer builds, for servers built directly.
var testFrameCodec = mustFrameCodec(testWriteTimeout)

func mustFrameCodec(writeTimeout time.Duration) nettransport.FrameCodec {
	codec, err := newFrameCodec(writeTimeout)
	if err != nil {
		panic(err)
	}
	return codec
}

// noRowWrites is the lifecycle store of registries whose tests do not observe
// the session rows.
type noRowWrites struct{}

func (noRowWrites) PersistResume(context.Context, *Session, string, string) error { return nil }

func (noRowWrites) PersistDisconnect(context.Context, *Session) error { return nil }

// newTestRegistry is a registry over the given connections and holder; nil
// stands for none and for a holder fake.
// adopts is SessionRegistry.adopt for tests that do not look at the sessions
// it supersedes.
func adopts(ctx context.Context, r *SessionRegistry, conn net.Conn, session *Session) bool {
	adopted, _ := r.adopt(ctx, conn, session)
	return adopted
}

func newTestRegistry(sessions map[net.Conn]*Session, holder ResumeHolder) *SessionRegistry {
	return newTestRegistryWithRows(sessions, holder, noRowWrites{})
}

func newTestRegistryWithRows(sessions map[net.Conn]*Session, holder ResumeHolder, rows SessionLifecycleStore) *SessionRegistry {
	if sessions == nil {
		sessions = make(map[net.Conn]*Session)
	}
	if holder == nil {
		holder = newHolderFake()
	}
	return &SessionRegistry{sessions: sessions, holder: holder, rows: rows, logger: testLogger()}
}

// sessionRowRepo is the part of a session repository double the tests
// observe: the operation ID counters and the loss of the connection.
type sessionRowRepo interface {
	UpdateOperationIDs(ctx context.Context, tenantID, sessionID, acOpId, scOpId int64) error
	MarkSessionDisconnected(ctx context.Context, tenantID, sessionID int64) error
}

// sessionRowsOver writes the session rows to a repository double the way the
// production owner (scaciservices.SessionRows) does.
type sessionRowsOver struct {
	repo sessionRowRepo
}

func (r sessionRowsOver) PersistOpIDs(ctx context.Context, session *Session, ids OpIDPair) error {
	return r.repo.UpdateOperationIDs(ctx, session.TenantID, session.ID, ids.AC, ids.SC)
}

func (r sessionRowsOver) PersistDisconnect(ctx context.Context, session *Session) error {
	return r.repo.MarkSessionDisconnected(ctx, session.TenantID, session.ID)
}

func (sessionRowsOver) PersistResume(context.Context, *Session, string, string) error { return nil }

func (sessionRowsOver) PersistConnectSync(context.Context, *Session, string, string, string, string, string, string) (int64, error) {
	return 0, nil
}

func (sessionRowsOver) PersistHeartbeat(context.Context, *Session) error { return nil }

// withOpIDs sets the operation ID counters of a session built as a literal.
func withOpIDs(session *Session, ids OpIDPair) *Session {
	session.ops.restore(ids)
	return session
}

// createTestSession creates a valid active session for testing.
func createTestSession(t *testing.T) *Session {
	t.Helper()
	return &Session{
		ID:       0, // ID=0 skips operation recording
		TenantID: 1,
		State:    StateActive,
		AcEui:    0xFEDCBA0987654321,
	}
}

// decodeResponse decodes msgpack response, skipping MIOTYA01 frame header.
// Frame format: "MIOTYA01" (8 bytes) + size (4 bytes LE) + payload
func decodeResponse(data []byte, v interface{}) error {
	// Check for MIOTYA01 header
	if len(data) > 12 && string(data[:8]) == "MIOTYA01" {
		// Read payload size from bytes 8-11 (little-endian)
		payloadSize := binary.LittleEndian.Uint32(data[8:12])
		if int(payloadSize)+12 <= len(data) {
			data = data[12 : 12+payloadSize]
		}
	}
	return msgpack.Unmarshal(data, v)
}

// assertErrorToken verifies that the error response contains the expected message
// from the error catalog. ErrorToken is internal-only (msgpack:"-") and not on wire,
// so we verify by comparing the Message field against the catalog definition.
func assertErrorToken(t *testing.T, errorResp Error, expectedToken string) {
	t.Helper()
	expectedDef := GetErrorDefinition(expectedToken)
	assert.Equal(t, expectedDef.Message, errorResp.Message,
		"Error message should match catalog for token %s", expectedToken)
}

// =============================================================================
// §2.4 Mandatory Field Validation Tests
// =============================================================================

// =============================================================================
// §3.9.1 Optional Field Default Tests
// =============================================================================

// =============================================================================
// §3.9 UL Data Transmit Tests - Error Mapping, Tenant Isolation, Operation Recording
// =============================================================================

// createTestServerFull creates a Server with all mock dependencies for comprehensive testing.
// Use this when testing operation recording, state transitions, or tenant verification.
type testServerOpts struct {
	ulSvc     *MockULService
	recorder  *MockOperationRecorder
	opRepo    *MockSCACIOperationRepository
	statusSvc *MockStatusService
}

func createTestServerFull(t *testing.T, opts testServerOpts) *Server {
	t.Helper()
	return &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		ulSvc:             opts.ulSvc,
		operationRecorder: opts.recorder,
		operationRepo:     opts.opRepo, // CRITICAL: non-nil to hit UpdateOperationState
		statusSvc:         opts.statusSvc,
		config:            &Config{},
	}
}

// createTestSessionWithID creates a session with ID > 0 for operation recording tests.
func createTestSessionWithID(t *testing.T) *Session {
	t.Helper()
	return &Session{
		ID:       123, // Non-zero to enable operation recording
		TenantID: 1,
		State:    StateActive,
		AcEui:    0xFEDCBA0987654321,
	}
}

// =============================================================================
// §3.9 Error Mapping Tests (POSIX Code Verification)
// =============================================================================

// =============================================================================
// §3.9 Tenant Isolation Tests (Explicit bsEui Verification)
// =============================================================================

// =============================================================================
// §3.9 Operation Recording Tests
// =============================================================================

// =============================================================================
// §3.9 Three-Way Handshake Completion Tests
// =============================================================================

// =============================================================================
// §3.7 Deregister Handler Tests - epEui Caching Flow
// =============================================================================

// =============================================================================
// §3.6 Register Handler Tests - POSIX Mapping and Session Guard
// =============================================================================

// =============================================================================
// §3.6 Register Handler Tests - Decode Failure Coverage
// =============================================================================

// =============================================================================
// §3.7 Deregister Handler Tests - Send Ordering and Failure Handling
// =============================================================================

// failingMockConn implements net.Conn and always returns an error on Write.
// Used to test send failure → failed state transitions.
type failingMockConn struct {
	mockConn       // Embed mockConn for Read/Close/etc defaults
	writeErr error // Error to return on Write
}

func (m *failingMockConn) Write(_ []byte) (n int, err error) {
	return 0, m.writeErr
}

// trackingMockConn implements net.Conn and calls onWrite callback on Write.
type trackingMockConn struct {
	mockConn
	onWrite func()
}

func (m *trackingMockConn) Write(b []byte) (n int, err error) {
	if m.onWrite != nil {
		m.onWrite()
	}
	m.written = append(m.written, b...)
	return len(b), nil
}

// =============================================================================
// §3.10 DLDataQueue Handler Tests - Single-Payload Constraint Validation
// =============================================================================

// =============================================================================
// §3.11 DL Data Revoke Handler Tests
// =============================================================================

// ============================================================================
// Inbound Error Validation Tests (Fix 3: SCACI §3.14.1)
// ============================================================================

// =============================================================================
// §3.6.3 Register Complete - Pre-Attach Propagation Tests
// =============================================================================

// mockPropagationService mocks propagation.Service for testing pre-attach propagation
type mockPropagationService struct {
	mock.Mock
}

func (m *mockPropagationService) TriggerEndpointPropagate(ctx context.Context, endpointID int64, activeSessions []propagation.BaseStationSession) error {
	args := m.Called(ctx, endpointID, activeSessions)
	return args.Error(0)
}

func (m *mockPropagationService) ReconcileBaseStation(ctx context.Context, session propagation.BaseStationSession, bs *models.BaseStation) error {
	args := m.Called(ctx, session, bs)
	return args.Error(0)
}

// mockSessionSnapshotProvider mocks propagation.SessionSnapshotProvider
type mockSessionSnapshotProvider struct {
	sessions []propagation.BaseStationSession
}

func (m *mockSessionSnapshotProvider) ConnectedSessionsSnapshot() []propagation.BaseStationSession {
	return m.sessions
}

// mustTestCommandRegistry indexes the production command table for tests.
func mustTestCommandRegistry() *commandRegistry {
	registry, err := newCommandRegistry(commandTable)
	if err != nil {
		panic(err)
	}
	return registry
}

// testInitiator resolves a command's initiator from the production table.
func testInitiator(t *testing.T, command string) CommandInitiator {
	t.Helper()
	spec, ok := mustTestCommandRegistry().lookup(command)
	require.True(t, ok, "command %s must be in the command table", command)
	return spec.Initiator
}

// testReplayable reports whether the command table replays the command.
func testReplayable(command string) bool {
	spec, ok := mustTestCommandRegistry().lookup(command)
	return ok && spec.Replayable
}

// testNonReplayReason returns the command table's non-replay reason.
func testNonReplayReason(command string) (string, bool) {
	spec, ok := mustTestCommandRegistry().lookup(command)
	if !ok || spec.NonReplayReason == "" {
		return "", false
	}
	return spec.NonReplayReason, true
}
