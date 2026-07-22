package bssci

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// NewTestServer creates a Server instance for testing with access to unexported fields.
// This helper lives in package bssci (not bssci_test) so it can set internal fields.
// Uses interfaces.Storage for dependency inversion. StatusService is mandatory (panics if nil).

// staticTestVersionNegotiator mirrors the production negotiator semantics for
// the supported set {mioty.MIOTYProtocolVersion}: same-major requests select
// the canonical service center version; other majors and lower minors return
// the production catalog errors.
type staticTestVersionNegotiator struct{}

func (staticTestVersionNegotiator) Negotiate(_ context.Context, requested string) (string, error) {
	reqMajor, reqMinor, _, cerr := ParseVersion(requested)
	if cerr != nil {
		return "", cerr
	}
	scMajor, scMinor, _, _ := ParseVersion(mioty.MIOTYProtocolVersion)
	if reqMajor != scMajor {
		return "", NewCatalogError(ErrUnsupportedMajorVersion, POSIX_EPROTO)
	}
	if reqMinor < scMinor {
		return "", NewCatalogError(ErrUnsupportedMinorVersion, POSIX_EPROTO)
	}
	return mioty.MIOTYProtocolVersion, nil
}

func NewTestServer(
	log logger.Logger,
	storage interfaces.Storage,
	eventStore interfaces.SystemEventStore,
	tenantID int64,
	sessionSvc SessionService,
	downlinkSvc DownlinkService,
	statusSvc StatusService,
	connectionSvc ConnectionService,
	broadcaster SCACIBroadcaster,
	queueSerializer QueueSerializer,
	auditLogger AuditLogger,
	tenantResolver TenantResolver,
) *Server {
	// StatusService is required for all tests to persist pending operations
	if statusSvc == nil {
		panic("NewTestServer: statusSvc cannot be nil (use NewTestServerWithMemoryStatusService for automatic setup)")
	}

	s := &Server{
		logger:            log,
		storage:           storage,
		eventStore:        eventStore,
		tenantID:          tenantID,
		sessionSvc:        sessionSvc,
		downlinkSvc:       downlinkSvc,
		statusSvc:         statusSvc,
		connectionSvc:     connectionSvc,
		broadcaster:       broadcaster,
		queueSerializer:   queueSerializer,
		auditLogger:       auditLogger,
		tenantResolver:    tenantResolver,
		versionNegotiator: staticTestVersionNegotiator{},
		handlers:          make(map[string]HandlerFunc), // Initialize handlers map for tests
		sessions:          make(map[string]*Session),    // Initialize sessions map for tests
	}
	// Wire endpointRepo if storage is available (prevents nil panics in resolveEndpointTenantID)
	if storage != nil {
		s.endpointRepo = storage.EndPoints()
	}
	// Initialize broadcast hook (tests can override)
	s.broadcastFn = s.SendAttachPropagateToAll
	return s
}

// CallHandleDLRXStatus exposes the unexported handleDLRXStatus method for testing.
func (s *Server) CallHandleDLRXStatus(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDLRXStatus(s, session, msg, data)
}

// CallHandleDLDataResult exposes the unexported handleDLDataResult method for testing.
func (s *Server) CallHandleDLDataResult(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDLDataResult(s, session, msg, data)
}

// RegisterHandlers exposes the unexported registerHandlers method for testing.
func (s *Server) RegisterHandlers() {
	s.registerHandlers()
}

// GetDownlinkSvc exposes the downlinkSvc field for testing.
func (s *Server) GetDownlinkSvc() DownlinkService {
	return s.downlinkSvc
}

// Handlers exposes the unexported handlers map for testing.
// Returns the map as interface{} to avoid exposing unexported types.
func (s *Server) Handlers() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range s.handlers {
		result[k] = v
	}
	return result
}

// Int64Ptr returns a pointer to the given int64 value (helper for tests).
func Int64Ptr(v int64) *int64 {
	return &v
}

// Uint32Ptr returns a pointer to the given uint32 value (helper for tests).
func Uint32Ptr(v uint32) *uint32 {
	return &v
}

// CallValidateOperationID exposes the unexported validateOperationID method for testing.
func (s *Server) CallValidateOperationID(session *Session, opId int64, isBaseStationInitiated bool) string {
	return s.validateOperationID(session, opId, isBaseStationInitiated)
}

// CallHandleConnect exposes the unexported handleConnect method for testing.
func (s *Server) CallHandleConnect(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleConnect(s, session, msg, data)
}

// CallHandleConnectComplete exposes the unexported handleConnectComplete method for testing.
func (s *Server) CallHandleConnectComplete(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleConnectComplete(s, session, msg, data)
}

// SetConfig sets the server config for testing.
func (s *Server) SetConfig(config *Config) {
	s.config = config
}

// SetEndpointRepository sets the endpoint repository dependency for testing.
func (s *Server) SetEndpointRepository(repo interfaces.EndpointRepository) {
	s.endpointRepo = repo
}

// SetConnectionManager sets the connection manager for testing.
func (s *Server) SetConnectionManager(mgr *basestation.ConnectionManager) {
	s.connectionMgr = mgr
}

// CallHandleStatusResponse exposes the unexported handleStatusResponse method for testing.
func (s *Server) CallHandleStatusResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleStatusResponse(s, session, msg, data)
}

// LEGACY: Old PendingOperation helper removed after Issue #3 composite key implementation.
// Use s.pendingOps[makeSessionOpKey(session, opID)] directly in tests (requires session context).
// func (s *Server) PendingOperation(opID int64) *PendingOperation {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()
// 	if op, ok := s.pendingOps[opID]; ok {
// 		clone := *op
// 		return &clone
// 	}
// 	return nil
// }

// CallHandleAttachPropagateResponse exposes the unexported handleAttachPropagateResponse method for testing.
func (s *Server) CallHandleAttachPropagateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleAttachPropagateResponse(s, session, msg, data)
}

// CallHandleDetachPropagateResponse exposes the unexported handleDetachPropagateResponse method for testing.
func (s *Server) CallHandleDetachPropagateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDetachPropagateResponse(s, session, msg, data)
}

// CallHandlePingResponse exposes the unexported handlePingResponse method for testing.
func (s *Server) CallHandlePingResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handlePingResponse(s, session, msg, data)
}

// RegisterSession adds a session to the server's sessions map for testing.
func (s *Server) RegisterSession(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}

// GetSession retrieves a session from the server's sessions map for testing.
// Returns nil if the session is not found.
// This helper ensures thread-safe read access and helps prevent future race conditions.
func (s *Server) GetSession(sessionID string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[sessionID]
}

// CallHandleDLDataResultResponse exposes the unexported handleDLDataResultResponse method for testing.
func (s *Server) CallHandleDLDataResultResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDLDataResultResponse(s, session, msg, data)
}

// CallHandleDLRXStatusResponse exposes the unexported handleDLRXStatusResponse method for testing.
func (s *Server) CallHandleDLRXStatusResponse(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDLRXStatusResponse(s, session, msg, data)
}

// CallHandleMessage exposes the unexported handleMessage method for testing.
// Used to test sublayer error handling per BSSCI §4-01.
func (s *Server) CallHandleMessage(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleMessage(session, msg, data)
}

// CallNormalizePayload exposes normalizePayload for tests.
func CallNormalizePayload(ctx context.Context, log logger.Logger, command string, data map[string]interface{}) (map[string]interface{}, error) {
	return normalizePayload(ctx, log, command, data)
}

// CallHandleAttach exposes the unexported handleAttach method for testing.
func (s *Server) CallHandleAttach(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleAttach(s, session, msg, data)
}

// CallHandleDetach exposes the unexported handleDetach method for testing.
func (s *Server) CallHandleDetach(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDetach(s, session, msg, data)
}

// CallHandleDetachComplete exposes the unexported handleDetachComplete method for testing.
func (s *Server) CallHandleDetachComplete(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleDetachComplete(s, session, msg, data)
}

// CallHandleAttachComplete exposes the unexported handleAttachComplete method for testing.
func (s *Server) CallHandleAttachComplete(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleAttachComplete(s, session, msg, data)
}

// CallHandleULData exposes the unexported handleULData method for testing.
func (s *Server) CallHandleULData(session *Session, msg *Message, data map[string]interface{}) error {
	return s.handleULData(s, session, msg, data)
}

// SetDeduplicator sets the deduplicator for testing.
func (s *Server) SetDeduplicator(dedup *MessageDeduplicator) {
	s.deduplicator = dedup
}

// NewTestServerWithBaseStationRepo creates a Server instance for testing with a custom base station repository.
// This is useful for tests that need to verify tenant isolation by capturing repository call arguments.
// StatusService is mandatory (panics if nil).
func NewTestServerWithBaseStationRepo(
	log logger.Logger,
	storage interfaces.Storage,
	basestationRepo interfaces.BaseStationRepository,
	tenantID int64,
	sessionSvc SessionService,
	downlinkSvc DownlinkService,
	statusSvc StatusService,
	connectionSvc ConnectionService,
	broadcaster SCACIBroadcaster,
	queueSerializer QueueSerializer,
	auditLogger AuditLogger,
	tenantResolver TenantResolver,
) *Server {
	// StatusService is required for all tests to persist pending operations
	if statusSvc == nil {
		panic("NewTestServerWithBaseStationRepo: statusSvc cannot be nil")
	}

	s := &Server{
		logger:            log,
		storage:           storage,
		basestationRepo:   basestationRepo,
		tenantID:          tenantID,
		sessionSvc:        sessionSvc,
		downlinkSvc:       downlinkSvc,
		statusSvc:         statusSvc,
		connectionSvc:     connectionSvc,
		broadcaster:       broadcaster,
		queueSerializer:   queueSerializer,
		auditLogger:       auditLogger,
		tenantResolver:    tenantResolver,
		versionNegotiator: staticTestVersionNegotiator{},
		sessions:          make(map[string]*Session),
	}
	// Initialize broadcast hook (tests can override)
	s.broadcastFn = s.SendAttachPropagateToAll
	return s
}

// NewTestServerWithMemoryStatusService creates a Server instance for testing with all services automatically wired.
// This is the recommended helper for most tests - it creates a complete service stack with memory-backed StatusService.
// Convenience wrapper that ensures StatusService is always present.
//
// Usage:
//
//	server := bssci.NewTestServerWithMemoryStatusService(logger, mockStorage, nil, 1)
func NewTestServerWithMemoryStatusService(
	log logger.Logger,
	storage interfaces.Storage,
	eventStore interfaces.SystemEventStore,
	tenantID int64,
) *Server {
	// Use local CreateTestServices to avoid import cycle
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(log, eventStore)

	// Use provided storage if available, otherwise use mock
	if storage == nil {
		storage = mockStorage
	}

	return NewTestServer(
		log,
		storage,
		eventStore,
		tenantID,
		sessionSvc,
		downlinkSvc,
		statusSvc,
		connectionSvc,
		broadcaster,
		queueSerializer,
		auditLogger,
		tenantResolver,
	)
}

// Local test service implementations to avoid import cycle with internal/services/bssci

// mockSessionService implements SessionService for tests with stateful behavior
type mockSessionService struct {
	mu             sync.RWMutex
	sessions       map[string]*Session // Keyed by session ID
	sessionsByDBID map[int64]*Session  // Keyed by database session ID
	sessionsByUUID map[string]*Session // Keyed by uppercase hex(SessionUUID)
	lastPing       map[int64]time.Time // Keyed by database session ID
	dbIDCounter    int64               // Simulates DB auto-increment
}

// newMockSessionService creates a new mock session service with state
func newMockSessionService() *mockSessionService {
	return &mockSessionService{
		sessions:       make(map[string]*Session),
		sessionsByDBID: make(map[int64]*Session),
		sessionsByUUID: make(map[string]*Session),
		lastPing:       make(map[int64]time.Time),
		dbIDCounter:    1000, // Start at 1000 for test IDs
	}
}

func (m *mockSessionService) HandleResume(_ context.Context, _ *Session, bsUUID []byte, bsOpId, scOpId *int64, _ uint64) ResumeOutcome {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Look up session by UUID using uppercase hex (matches production)
	key := strings.ToUpper(hex.EncodeToString(bsUUID))
	if existingSession, exists := m.sessionsByUUID[key]; exists {
		// Mirror production constraint semantics: absent constraints pass;
		// required BS operation ID beyond known state or claimed SC
		// operation ID beyond issued state is an inconsistent resume
		if bsOpId != nil && *bsOpId > existingSession.LastBsOpId {
			return ResumeOutcome{Disposition: ResumeInconsistent, Previous: existingSession, Err: ErrResumeCounterMismatch}
		}
		if scOpId != nil && *scOpId < existingSession.LastScOpId {
			return ResumeOutcome{Disposition: ResumeInconsistent, Previous: existingSession, Err: ErrResumeCounterMismatch}
		}
		return ResumeOutcome{Disposition: ResumeCompatible, Previous: existingSession}
	}
	return ResumeOutcome{Disposition: ResumeNoMatch} // No valid resume
}

func (m *mockSessionService) PersistSession(_ context.Context, session *Session, _ *basestation.BaseStation, isResume bool, connectInfo json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Assign DB session ID if not resuming
	if !isResume && session.DbSessionID == 0 {
		m.dbIDCounter++
		session.DbSessionID = m.dbIDCounter
	}

	// Persist ConnectInfo for resume validation
	if connectInfo != nil {
		session.ConnectInfo = connectInfo
	}

	// Store in sessions map by session ID
	if session.ID != "" {
		m.sessions[session.ID] = session
	}

	// Store by database ID for lookups
	if session.DbSessionID != 0 {
		m.sessionsByDBID[session.DbSessionID] = session
	}

	// Store by UUID if present (via StoreSessionByUUID which will be called separately)
	// The HandleResume test flow relies on explicit StoreSessionByUUID calls

	return nil
}

func (m *mockSessionService) StoreSessionByUUID(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session.SessionUUID != nil {
		key := strings.ToUpper(hex.EncodeToString(session.SessionUUID))
		m.sessionsByUUID[key] = session
	}
}

func (m *mockSessionService) MarkHandshakeComplete(session *Session) {
	session.HandshakeComplete = true
}

func (m *mockSessionService) RemoveSession(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, session.ID)
	if session.DbSessionID != 0 {
		delete(m.sessionsByDBID, session.DbSessionID)
		delete(m.lastPing, session.DbSessionID)
	}
	if session.SessionUUID != nil {
		key := strings.ToUpper(hex.EncodeToString(session.SessionUUID))
		delete(m.sessionsByUUID, key)
	}
}

func (m *mockSessionService) MarkDisconnected(_ context.Context, session *Session) error {
	if session.DbSessionID == 0 {
		return fmt.Errorf("cannot mark session disconnected: not persisted (DbSessionID=0)")
	}
	return nil
}

func (m *mockSessionService) TerminateSession(_ context.Context, session *Session) error {
	m.RemoveSession(session)
	return nil
}

func (m *mockSessionService) UpdateEncoding(_ context.Context, _ int64, sessionID int64, encoding string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find session by DB ID and update encoding
	for _, sess := range m.sessions {
		if sess.DbSessionID == sessionID {
			sess.Encoding = encoding
			return nil
		}
	}
	return nil
}

func (m *mockSessionService) UpdateSessionCounters(_ context.Context, _ *Session) error {
	// In a real implementation, this would persist to DB
	// For tests, the counters are already updated in the session object
	return nil
}

func (m *mockSessionService) UpdatePingTimestamp(_ context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if session.DbSessionID != 0 {
		m.lastPing[session.DbSessionID] = now
		// Also update the in-memory session if found
		if sess, exists := m.sessionsByDBID[session.DbSessionID]; exists {
			sess.LastSeen = now
		}
	}
	return nil
}

// GetLastPing returns the last ping timestamp for a session (for testing)
func (m *mockSessionService) GetLastPing(dbSessionID int64) (time.Time, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, exists := m.lastPing[dbSessionID]
	return t, exists
}

// memoryStatusService implements StatusService with in-memory storage for tests
type memoryStatusService struct {
	pendingOps *map[SessionOpKey]*PendingOperation
	mu         *sync.RWMutex
	logger     logger.Logger
	// recordErr, when set, is returned by Record* calls to exercise the
	// persist-failure abort path (no wire frame, counter gap only).
	recordErr error
}

func (m *memoryStatusService) RecordPendingOperation(_ context.Context, session *Session, opId int64, op *PendingOperation, _ int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordErr != nil {
		return m.recordErr
	}
	key := makeSessionOpKey(session, opId)
	(*m.pendingOps)[key] = op
	return nil
}

func (m *memoryStatusService) RecordPendingOperations(_ context.Context, session *Session, ops []*PendingOperation, _ int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordErr != nil {
		return m.recordErr
	}
	for _, op := range ops {
		key := makeSessionOpKey(session, op.OperationID)
		(*m.pendingOps)[key] = op
	}
	return nil
}

func (m *memoryStatusService) RestorePendingOperation(session *Session, opId int64, op *PendingOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := makeSessionOpKey(session, opId)
	(*m.pendingOps)[key] = op
}

func (m *memoryStatusService) GetPendingOperation(session *Session, opId int64) (*PendingOperation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := makeSessionOpKey(session, opId)
	op, exists := (*m.pendingOps)[key]
	if !exists {
		return nil, fmt.Errorf("pending operation not found")
	}
	return op, nil
}

func (m *memoryStatusService) RemovePendingOperation(_ context.Context, session *Session, opId int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := makeSessionOpKey(session, opId)
	delete(*m.pendingOps, key)
	return nil
}

func (m *memoryStatusService) ExtractQueueMetadata(session *Session, opId int64) (endpointEUI uint64, queueID int64, tenantID string) {
	op, err := m.GetPendingOperation(session, opId)
	if err != nil {
		return 0, 0, ""
	}
	if qid, ok := op.Metadata["queId"].(uint64); ok {
		queueID = int64(qid)
	}
	if tid, ok := op.Metadata["tenantId"].(string); ok {
		tenantID = tid
	}
	return
}

func (m *memoryStatusService) ProcessOperationStatusUpdate(_ context.Context, _ *Session, _ int64, _ string) error {
	return nil
}

// Canonical test mocks defined in test_mocks_test.go (same package, test-only compilation).
// Mocks remain in pkg/bssci to avoid import cycle (mocks implement bssci interfaces).
// The _test.go suffix ensures test-only compilation, following Go best practices.

// CreateTestServices creates all service dependencies for tests
// Minimal local implementation to avoid internal/services/bssci import cycle.
// Returns nil for most services - tests needing real implementations should provide their own.
func CreateTestServices(log logger.Logger, _ interfaces.SystemEventStore) (
	SessionService,
	DownlinkService,
	StatusService,
	ConnectionService,
	SCACIBroadcaster,
	QueueSerializer,
	AuditLogger,
	TenantResolver,
	interfaces.Storage,
) {
	// Create StatusService with in-memory map (most tests only need this)
	ops := make(map[SessionOpKey]*PendingOperation)
	var mu sync.RWMutex
	statusSvc := &memoryStatusService{
		pendingOps: &ops,
		mu:         &mu,
		logger:     log,
	}

	// Create stateful mock SessionService for VM tests
	sessionSvc := newMockSessionService()

	// Create canonical mock TenantResolver (from test_mocks_test.go)
	tenantResolver := NewMockTenantResolver(log)

	// Create canonical mock QueueSerializer (from test_mocks_test.go)
	queueSerializer := NewMockQueueSerializer()

	// Tests needing storage or other dependencies should inject their own implementations
	// This avoids ad-hoc fakes that break dependency boundaries
	return sessionSvc, nil, statusSvc, nil, nil, queueSerializer, nil, tenantResolver, nil
}
