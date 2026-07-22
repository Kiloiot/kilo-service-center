package bssci_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// --- Mock Implementations for Detach Propagate Complete Integration Tests ---
// Reuses patterns from attach_propagate_integration_test.go

// detPrpTestConn is a minimal mock connection. It captures every Write so tests can
// decode the msgpack payload frame written by sendMessage (header then payload).
type detPrpTestConn struct {
	net.Conn
	mu     sync.Mutex
	frames [][]byte
}

func (m *detPrpTestConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := make([]byte, len(b))
	copy(clone, b)
	m.frames = append(m.frames, clone)
	return len(b), nil
}

func (m *detPrpTestConn) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.frames)
}

func (m *detPrpTestConn) Frames() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([][]byte, len(m.frames))
	for i, f := range m.frames {
		dup := make([]byte, len(f))
		copy(dup, f)
		out[i] = dup
	}
	return out
}

func (m *detPrpTestConn) Close() error                       { return nil }
func (m *detPrpTestConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *detPrpTestConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *detPrpTestConn) SetDeadline(_ time.Time) error      { return nil }
func (m *detPrpTestConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *detPrpTestConn) SetWriteDeadline(_ time.Time) error { return nil }
func (m *detPrpTestConn) Read(_ []byte) (n int, err error)   { return 0, nil }

// capturingDetPrpMIOTYMessageRepo captures CreateDetachPropagateMessage calls
type capturingDetPrpMIOTYMessageRepo struct {
	mu                      sync.Mutex
	detachPropagateMessages []*mioty.DetachPropagateMessage
	createDetPrpCalls       int
}

func (r *capturingDetPrpMIOTYMessageRepo) CreateDetachPropagateMessage(_ context.Context, msg *mioty.DetachPropagateMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createDetPrpCalls++
	r.detachPropagateMessages = append(r.detachPropagateMessages, msg)
	return nil
}

// GetDetachMessages returns captured detach propagate messages (thread-safe)
func (r *capturingDetPrpMIOTYMessageRepo) GetDetachMessages() []*mioty.DetachPropagateMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]*mioty.DetachPropagateMessage, len(r.detachPropagateMessages))
	copy(result, r.detachPropagateMessages)
	return result
}

// Stub implementations for other MIOTYMessageRepository methods
func (r *capturingDetPrpMIOTYMessageRepo) CreateAttachPropagateMessage(_ context.Context, _ *mioty.AttachPropagateMessage) error {
	return nil
}
func (r *capturingDetPrpMIOTYMessageRepo) CreateULDataMessage(_ context.Context, _ *mioty.ULDataMessage) error {
	return nil
}
func (r *capturingDetPrpMIOTYMessageRepo) CreateDetachMessage(_ context.Context, _ *mioty.DetachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *capturingDetPrpMIOTYMessageRepo) CreateAttachMessage(_ context.Context, _ *mioty.AttachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetULDataMessage(_ context.Context, _ string, _ int64) (*mioty.ULDataMessage, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetDetachMessage(_ context.Context, _ string, _ int64) (*mioty.DetachMessage, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) ListULDataMessages(_ context.Context, _ mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) UpdateULDataBaseStations(_ context.Context, _ int64, _ uint64, _ uint32, _ int64, _ []byte) error {
	return nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetExtendedMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetMessageStatsByEndpoint(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetOverallStats(_ context.Context, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetAnalyticsOverview(_ context.Context, _ int64, _, _ time.Time) (*mioty.AnalyticsOverviewStats, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetHourlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.HourlyActivity, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetDailyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.DailyActivity, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetTopEndpointsByActivity(_ context.Context, _ int64, _, _ time.Time, _ int) ([]mioty.EndpointActivity, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetSignalQualityStats(_ context.Context, _ int64, _, _ time.Time) (*mioty.SignalQualityStats, error) {
	return nil, nil
}
func (r *capturingDetPrpMIOTYMessageRepo) GetSignalQualityByBaseStation(_ context.Context, _ int64, _, _ time.Time) ([]mioty.BaseStationSignalQuality, error) {
	return nil, nil
}

func (r *capturingDetPrpMIOTYMessageRepo) GetBaseStationMessageStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}

func (r *capturingDetPrpMIOTYMessageRepo) GetMessageCountsByEndpoint(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *capturingDetPrpMIOTYMessageRepo) GetMessageCountsByBaseStation(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *capturingDetPrpMIOTYMessageRepo) GetWeeklyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.WeeklyActivity, error) {
	return nil, nil
}

func (r *capturingDetPrpMIOTYMessageRepo) GetMonthlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.MonthlyActivity, error) {
	return nil, nil
}

// detPrpTestEndpointRepo is a minimal endpoint repository. Every mutation method
// increments its own tally so rejected-propagate tests can assert zero state changes.
type detPrpTestEndpointRepo struct {
	endpoints map[uint64]*models.EndPoint

	getByEUICalls                    int
	createCalls                      int
	updateCalls                      int
	updateFieldsCalls                int
	updateLastSeenCalls              int
	updateRadioMetricsCalls          int
	updateRadioMetricsSelectiveCalls int
	updateDetachMetricsCalls         int
	updateWithEUICalls               int
	deleteByTenantCalls              int
}

func (r *detPrpTestEndpointRepo) GetByEUI(_ context.Context, _ int64, eui []byte) (*models.EndPoint, error) {
	r.getByEUICalls++
	if len(eui) != 8 {
		return nil, storage.ErrNotFound
	}
	key := binary.BigEndian.Uint64(eui)
	if ep, ok := r.endpoints[key]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

func (r *detPrpTestEndpointRepo) Get(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	if ep, ok := r.endpoints[eui.ToUint64()]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

func (r *detPrpTestEndpointRepo) UpdateFields(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	r.updateFieldsCalls++
	return nil
}

// Stub implementations for remaining EndpointRepository methods
func (r *detPrpTestEndpointRepo) Create(context.Context, *models.EndPoint) error {
	r.createCalls++
	return nil
}
func (r *detPrpTestEndpointRepo) GetByID(context.Context, int64, int64) (*models.EndPoint, error) {
	return nil, nil
}
func (r *detPrpTestEndpointRepo) GetByTenant(context.Context, int64) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *detPrpTestEndpointRepo) CountByTenant(context.Context, int64) (int64, error) { return 0, nil }
func (r *detPrpTestEndpointRepo) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *detPrpTestEndpointRepo) Update(context.Context, *models.EndPoint) error {
	r.updateCalls++
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	r.updateLastSeenCalls++
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	r.updateRadioMetricsCalls++
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	r.updateRadioMetricsSelectiveCalls++
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
	r.updateDetachMetricsCalls++
	return nil
}
func (r *detPrpTestEndpointRepo) StreamAllForPropagation(context.Context, int64, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *detPrpTestEndpointRepo) HasEndpointsSince(context.Context, time.Time) (bool, error) {
	return false, nil
}
func (r *detPrpTestEndpointRepo) GetEndpointWithKeysForDetachValidation(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}
func (r *detPrpTestEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}
func (r *detPrpTestEndpointRepo) DeleteByTenant(context.Context, int64, []byte) error {
	r.deleteByTenantCalls++
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	r.updateWithEUICalls++
	return ep, nil
}
func (r *detPrpTestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// detPrpCapturingStorage implements interfaces.Storage
type detPrpCapturingStorage struct {
	miotyMessages *capturingDetPrpMIOTYMessageRepo
	endpointRepo  *detPrpTestEndpointRepo
	pendingOps    *detPrpCapturingPendingOps
}

func (s *detPrpCapturingStorage) MIOTYMessages() interfaces.MIOTYMessageRepository {
	return s.miotyMessages
}
func (s *detPrpCapturingStorage) EndPoints() interfaces.EndpointRepository {
	if s.endpointRepo == nil {
		return nil
	}
	return s.endpointRepo
}
func (s *detPrpCapturingStorage) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (s *detPrpCapturingStorage) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (s *detPrpCapturingStorage) EndPointSessions() interfaces.EndPointSessionRepository { return nil }
func (s *detPrpCapturingStorage) EndPointKeys() interfaces.EndPointKeyRepository         { return nil }
func (s *detPrpCapturingStorage) RoamingAgreements() interfaces.RoamingAgreementRepository {
	return nil
}
func (s *detPrpCapturingStorage) BaseStations() interfaces.BaseStationRepository { return nil }
func (s *detPrpCapturingStorage) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (s *detPrpCapturingStorage) DLRXStatus() interfaces.DLRXStatusRepository { return nil }
func (s *detPrpCapturingStorage) PendingOperations() interfaces.PendingOperationRepository {
	if s.pendingOps == nil {
		// Lazily create so tests that don't inspect pending ops still work.
		s.pendingOps = &detPrpCapturingPendingOps{}
	}
	return s.pendingOps
}

// detPrpUpdateMetadataCall captures a single UpdateMetadata invocation.
type detPrpUpdateMetadataCall struct {
	SessionID   int64
	OperationID int64
	Metadata    json.RawMessage
}

// detPrpCapturingPendingOps is a PendingOperationRepository that captures UpdateMetadata
// calls so tests can assert on the failure metadata persisted by handlePropagateResponseFailure.
type detPrpCapturingPendingOps struct {
	mu      sync.Mutex
	updates []detPrpUpdateMetadataCall
}

func (r *detPrpCapturingPendingOps) Create(_ context.Context, _ *interfaces.PendingOperationRequest) error {
	return nil
}

func (r *detPrpCapturingPendingOps) CreateBatch(_ context.Context, _ []*interfaces.PendingOperationRequest) error {
	return nil
}
func (r *detPrpCapturingPendingOps) UpdateMetadata(_ context.Context, sessionID, operationID int64, metadata json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dup := make(json.RawMessage, len(metadata))
	copy(dup, metadata)
	r.updates = append(r.updates, detPrpUpdateMetadataCall{SessionID: sessionID, OperationID: operationID, Metadata: dup})
	return nil
}
func (r *detPrpCapturingPendingOps) DeleteBySessionAndOperation(_ context.Context, _ int64, _ int64) error {
	return nil
}
func (r *detPrpCapturingPendingOps) DeleteByOperation(_ context.Context, _ int64) error { return nil }
func (r *detPrpCapturingPendingOps) DeleteBySession(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}
func (r *detPrpCapturingPendingOps) GetBySession(_ context.Context, _ int64) ([]*interfaces.PendingOperation, error) {
	return nil, nil
}

// LastUpdate returns the most recent UpdateMetadata call, or nil if none captured.
func (r *detPrpCapturingPendingOps) LastUpdate() *detPrpUpdateMetadataCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.updates) == 0 {
		return nil
	}
	call := r.updates[len(r.updates)-1]
	return &call
}

func (r *detPrpCapturingPendingOps) UpdateCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.updates)
}
func (s *detPrpCapturingStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (s *detPrpCapturingStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *detPrpCapturingStorage) Users() interfaces.UserRepository                 { return nil }
func (s *detPrpCapturingStorage) APIKeys() interfaces.APIKeyRepository             { return nil }
func (s *detPrpCapturingStorage) Integrations() interfaces.IntegrationRepository   { return nil }
func (s *detPrpCapturingStorage) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (s *detPrpCapturingStorage) DeviceModels() interfaces.DeviceModelRepository   { return nil }
func (s *detPrpCapturingStorage) Blueprints() interfaces.BlueprintRepository       { return nil }
func (s *detPrpCapturingStorage) Organizations() interfaces.OrganizationRepository { return nil }
func (s *detPrpCapturingStorage) GetSqlxDB() *sqlx.DB                              { return nil }
func (s *detPrpCapturingStorage) SystemEvents() interfaces.SystemEventStore        { return nil }
func (s *detPrpCapturingStorage) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (s *detPrpCapturingStorage) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (s *detPrpCapturingStorage) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }
func (s *detPrpCapturingStorage) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	return nil, nil
}
func (s *detPrpCapturingStorage) Ping(_ context.Context) error { return nil }
func (s *detPrpCapturingStorage) Close() error                 { return nil }

// detPrpCapturingEventStore captures CreateEvent calls
type detPrpCapturingEventStore struct {
	mu     sync.Mutex
	events []*models.SystemEvent
}

func (s *detPrpCapturingEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *detPrpCapturingEventStore) GetEvents(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*models.SystemEvent, len(s.events))
	copy(result, s.events)
	return result, nil
}

func (s *detPrpCapturingEventStore) CapturedEvents() []*models.SystemEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*models.SystemEvent, len(s.events))
	copy(result, s.events)
	return result
}

// Stub implementations for other SystemEventStore methods
func (s *detPrpCapturingEventStore) GetEventsFiltered(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}
func (s *detPrpCapturingEventStore) GetActiveAlerts(_ context.Context, _ interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}
func (s *detPrpCapturingEventStore) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return nil, nil
}
func (s *detPrpCapturingEventStore) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}
func (s *detPrpCapturingEventStore) CountEvents(_ context.Context, _ interfaces.SystemEventFilter) (int64, error) {
	return 0, nil
}
func (s *detPrpCapturingEventStore) CountActiveAlerts(_ context.Context, _ interfaces.AlertFilter) (int64, error) {
	return 0, nil
}

// --- Integration Tests ---

// TestDetachPropagateCompletionIntegration_MessagePersistence verifies that when detPrpRsp
// is received with result=0 (success), the handler:
// 1. Persists a detPrpCmp message to the messages table
// 2. Emits a detach_propagate_complete system event
// 3. Has correct field values (OpId as int64, EpEui, TenantID, BasestationEui)
func TestDetachPropagateCompletionIntegration_MessagePersistence(t *testing.T) {
	t.Parallel()

	const (
		testTenantID = int64(100)
		testOpId     = int64(-67890) // Negative for SC-initiated operations
		// Use smaller EUI values to avoid float64 precision loss
		testBsEui = uint64(0x0000000087654321)
		testEpEui = uint64(0x0000000012345678)
	)

	// Create test endpoint for lookup
	var epEUIBytes models.EUI
	binary.BigEndian.PutUint64(epEUIBytes[:], testEpEui)
	testEndpoint := &models.EndPoint{
		ID:       1001,
		EUI:      epEUIBytes,
		TenantID: testTenantID,
	}

	// Create capturing storage with endpoint repo
	msgRepo := &capturingDetPrpMIOTYMessageRepo{}
	endpointRepo := &detPrpTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			testEpEui: testEndpoint,
		},
	}
	storageImpl := &detPrpCapturingStorage{miotyMessages: msgRepo, endpointRepo: endpointRepo}

	// Create capturing event store
	eventStore := &detPrpCapturingEventStore{}

	// Create server using test infrastructure
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)

	server := bssci.NewTestServer(testLogger, storageImpl, eventStore, testTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	// Create mock connection
	mockConn := &detPrpTestConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-detprp-integration",
			BaseStationEUI:    testBsEui,
			Encoding:          "msgpack",
			HandshakeComplete: true,
			ResolvedTenantID:  testTenantID,
			DbSessionID:       1,
		},
		Conn: mockConn,
		// Required for pending operation processing
	}
	server.RegisterSession(session)

	// Register pending operation with metadata (using float64 for numbers as JSON deserializes)
	pendingOp := &bssci.PendingOperation{
		OperationType: mioty.CmdDetachPropagate,
		OperationID:   testOpId,
		CreatedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"endpointEUI": float64(testEpEui),
			"tenantId":    float64(testTenantID),
		},
	}
	err := statusSvc.RecordPendingOperation(testutil.TestContextWithTenant(testTenantID), session, testOpId, pendingOp, session.DbSessionID)
	require.NoError(t, err, "RecordPendingOperation should succeed")

	// Verify pending op was recorded
	retrievedOp, getErr := statusSvc.GetPendingOperation(session, testOpId)
	require.NoError(t, getErr, "GetPendingOperation should succeed")
	require.NotNil(t, retrievedOp, "Retrieved pending op should not be nil")

	// Simulate detPrpRsp with result=0 (success)
	// This triggers handleDetachPropagateResponse which calls handleDetachPropagateComplete
	data := map[string]interface{}{
		"command": mioty.CmdDetachPropagateResponse,
		"opId":    testOpId,
		"result":  int64(0), // Success
	}
	msg := &bssci.Message{
		Command: mioty.CmdDetachPropagateResponse,
		OpId:    testOpId,
		Data:    data,
	}

	// Call the real handler
	err = server.CallHandleDetachPropagateResponse(session, msg, data)
	require.NoError(t, err, "CallHandleDetachPropagateResponse should succeed")

	// Get captured events
	capturedEvents := eventStore.CapturedEvents()

	// Assert message persistence
	persistedMsgs := msgRepo.GetDetachMessages()
	require.Len(t, persistedMsgs, 1, "CreateDetachPropagateMessage should be called exactly once")
	persistedMsg := persistedMsgs[0]

	// Verify message fields
	assert.Equal(t, mioty.CmdDetachPropagateComplete, persistedMsg.CommandType, "CommandType should be detPrpCmp")
	assert.Equal(t, testOpId, persistedMsg.OpId, "OpId should match (int64)")
	assert.Equal(t, testEpEui, persistedMsg.EpEui, "EpEui should match metadata")
	assert.NotZero(t, persistedMsg.EpEui, "EpEui should be non-zero")
	assert.Equal(t, testTenantID, persistedMsg.TenantID, "TenantID should match")

	// Verify BaseStationEui
	var expectedBsEUIBytes [8]byte
	binary.BigEndian.PutUint64(expectedBsEUIBytes[:], testBsEui)
	assert.Equal(t, expectedBsEUIBytes[:], persistedMsg.BasestationEui, "BasestationEui should match session")

	// Verify metadata fields
	assert.Equal(t, mioty.MessageTypeDetachPropagate, persistedMsg.MessageType, "MessageType should be detach_propagate")
	assert.Equal(t, mioty.DirectionDownlink, persistedMsg.Direction, "Direction should be downlink")
	assert.Equal(t, mioty.InterfaceBSSCI, persistedMsg.InterfaceType, "InterfaceType should be bssci")

	// Assert event creation
	require.GreaterOrEqual(t, len(capturedEvents), 1, "CreateEvent should be called at least once")

	// Find the completion event
	var foundCompletionEvent bool
	expectedBsEuiStr := fmt.Sprintf("%016X", testBsEui)
	for _, evt := range capturedEvents {
		if evt.EventType == models.EventTypeDetachPropagateCompleted {
			foundCompletionEvent = true
			assert.Equal(t, mioty.CategoryEndpoint, evt.Category, "Event category should be endpoint")
			assert.NotEmpty(t, evt.SourceName, "SourceName should not be empty")
			// One event has endpoint source, one has basestation source
			if evt.SourceType == mioty.SourceTypeBaseStation {
				assert.Contains(t, evt.SourceName, expectedBsEuiStr, "BS event SourceName should contain BS EUI")
			}
			break
		}
	}
	assert.True(t, foundCompletionEvent, "Should have detach_propagate_complete event")
}

// TestDetachPropagateCompletionIntegration_NoPendingOp verifies that when detPrpRsp
// is received with result=0 but NO pendingOp exists, the handler:
// 1. Does NOT persist a detPrpCmp message (no valid epEui)
// 2. Does NOT crash (graceful handling)
func TestDetachPropagateCompletionIntegration_NoPendingOp(t *testing.T) {
	t.Parallel()

	const (
		testTenantID = int64(100)
		testOpId     = int64(-99999)
		testBsEui    = uint64(0x1122334455667788)
	)

	// Create capturing storage (no endpoint repo needed - we expect no persistence)
	msgRepo := &capturingDetPrpMIOTYMessageRepo{}
	storageImpl := &detPrpCapturingStorage{miotyMessages: msgRepo}

	// Create capturing event store
	eventStore := &detPrpCapturingEventStore{}

	// Create server using test infrastructure
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)

	server := bssci.NewTestServer(testLogger, storageImpl, eventStore, testTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	// Create mock connection
	mockConn := &detPrpTestConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-detprp-no-pendingop",
			BaseStationEUI:    testBsEui,
			Encoding:          "msgpack",
			HandshakeComplete: true,
			ResolvedTenantID:  testTenantID,
		},
		Conn: mockConn,
	}
	server.RegisterSession(session)

	// NOTE: Do NOT register pendingOp - test the negative path

	// Simulate detPrpRsp with result=0 (success)
	data := map[string]interface{}{
		"command": mioty.CmdDetachPropagateResponse,
		"opId":    testOpId,
		"result":  int64(0), // Success
	}
	msg := &bssci.Message{
		Command: mioty.CmdDetachPropagateResponse,
		OpId:    testOpId,
		Data:    data,
	}

	// Call the real handler - should NOT crash
	err := server.CallHandleDetachPropagateResponse(session, msg, data)
	require.NoError(t, err, "CallHandleDetachPropagateResponse should succeed even without pendingOp")

	// Assert NO message persistence (no epEui available)
	persistedMsgs := msgRepo.GetDetachMessages()
	assert.Len(t, persistedMsgs, 0, "CreateDetachPropagateMessage should NOT be called without pendingOp")
}

// TestHandleDetachPropagateResponse_Rejected verifies that when detPrpRsp arrives
// with a non-zero result, handlePropagateResponseFailure emits the cataloged event
// type (EventTypeEndpointDetachFailed) and uses TitleDetachPropagateFailedForEndpointOnBS
// rather than the pre-Fix-A pendingOp-dependent fallback strings.
func TestHandleDetachPropagateResponse_Rejected(t *testing.T) {
	t.Parallel()

	const (
		testTenantID = int64(100)
		testOpId     = int64(-12345)
		testEpEui    = uint64(0x12345678)
		testBsEui    = uint64(0x1122334455667788)
		rejectCode   = 42
	)

	msgRepo := &capturingDetPrpMIOTYMessageRepo{}
	endpointRepo := &detPrpTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{testEpEui: {ID: 1, TenantID: testTenantID}},
	}
	pendingOps := &detPrpCapturingPendingOps{}
	storageImpl := &detPrpCapturingStorage{
		miotyMessages: msgRepo,
		endpointRepo:  endpointRepo,
		pendingOps:    pendingOps,
	}
	eventStore := &detPrpCapturingEventStore{}
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)
	server := bssci.NewTestServer(testLogger, storageImpl, eventStore, testTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	mockConn := &detPrpTestConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-detprp-rejected",
			BaseStationEUI:    testBsEui,
			Encoding:          "msgpack",
			HandshakeComplete: true,
			ResolvedTenantID:  testTenantID,
			DbSessionID:       1,
		},
		Conn: mockConn,
	}
	server.RegisterSession(session)

	pendingOp := &bssci.PendingOperation{
		OperationType: mioty.CmdDetachPropagate,
		OperationID:   testOpId,
		CreatedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"endpointEUI": float64(testEpEui),
			"tenantId":    float64(testTenantID),
		},
	}
	require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContextWithTenant(testTenantID), session, testOpId, pendingOp, session.DbSessionID))

	data := map[string]interface{}{
		"command": mioty.CmdDetachPropagateResponse,
		"opId":    testOpId,
		"result":  int64(rejectCode),
	}
	msg := &bssci.Message{
		Command: mioty.CmdDetachPropagateResponse,
		OpId:    testOpId,
		Data:    data,
	}

	require.NoError(t, server.CallHandleDetachPropagateResponse(session, msg, data),
		"failure path must keep session alive (handler returns nil after sendError)")

	// Wire response: sendMessage writes header + msgpack payload (two Write calls).
	frames := mockConn.Frames()
	require.GreaterOrEqual(t, len(frames), 2, "sendMessage must write header + payload")
	assert.Len(t, frames[0], 12, "header is 8-byte magic + 4-byte length")

	var wireResp map[string]interface{}
	require.NoError(t, msgpack.Unmarshal(frames[1], &wireResp), "payload must be msgpack-decodable")
	assert.Equal(t, mioty.CmdError, wireResp["command"], "rejected propagate must reply with mioty.CmdError")
	assert.Equal(t, testOpId, testInt64(t, wireResp["opId"]), "response opId must echo the request")
	assert.Equal(t, int64(bssci.POSIX_EPROTO), testInt64(t, wireResp["code"]), "POSIX code must be EPROTO per BSSCI-4-01")
	assert.Equal(t, bssci.ResolveErrorMessage(bssci.ErrDetachPropagateFailed), wireResp["message"],
		"wire message must be cataloged via ErrDetachPropagateFailed")

	// Pending-op metadata persistence.
	update := pendingOps.LastUpdate()
	require.NotNil(t, update, "UpdateMetadata must be called once on the rejected path")
	require.Equal(t, 1, pendingOps.UpdateCalls(), "exactly one UpdateMetadata call")
	var metadata map[string]interface{}
	require.NoError(t, json.Unmarshal(update.Metadata, &metadata))
	assert.Equal(t, true, metadata["failed"], "metadata.failed must be true")
	assert.Equal(t, int64(rejectCode), testInt64(t, metadata["failureCode"]), "metadata.failureCode must match the BS-reported code")
	assert.Equal(t, fmt.Sprintf(bssci.PropagateFailureReasonFormat, rejectCode), metadata["failureReason"],
		"metadata.failureReason must use the cataloged PropagateFailureReasonFormat")

	// Event creation: both endpoint- and BS-scoped events, cataloged event type + title.
	captured := eventStore.CapturedEvents()
	require.GreaterOrEqual(t, len(captured), 2, "expected endpoint + base-station failure events")
	var endpointEvt, baseStationEvt *models.SystemEvent
	for _, evt := range captured {
		if evt.EventType != bssci.EventTypeEndpointDetachFailed {
			continue
		}
		if evt.SourceType == mioty.SourceTypeEndpoint && endpointEvt == nil {
			endpointEvt = evt
		}
		if evt.SourceType == mioty.SourceTypeBaseStation && baseStationEvt == nil {
			baseStationEvt = evt
		}
	}
	require.NotNil(t, endpointEvt, "endpoint-scoped failure event missing")
	require.NotNil(t, baseStationEvt, "base-station-scoped failure event missing")

	epStr := fmt.Sprintf("%016X", testEpEui)
	bsStr := fmt.Sprintf("%016X", testBsEui)
	assert.Equal(t, fmt.Sprintf(bssci.TitleDetachPropagateFailedForEndpointOnBS, epStr, bsStr), endpointEvt.Title,
		"endpoint event title must match cataloged TitleDetachPropagateFailedForEndpointOnBS format")
	assert.Equal(t, epStr, endpointEvt.SourceName, "endpoint SourceName must be the EUI")
	assert.Equal(t, bsStr, baseStationEvt.SourceName, "base-station SourceName must be the BS EUI")

	// Endpoint must not be mutated on the rejected path.
	assert.Equal(t, 0, endpointRepo.createCalls, "no Create on rejected propagate")
	assert.Equal(t, 0, endpointRepo.updateCalls, "no Update on rejected propagate")
	assert.Equal(t, 0, endpointRepo.updateFieldsCalls, "no UpdateFields on rejected propagate")
	assert.Equal(t, 0, endpointRepo.updateLastSeenCalls, "no UpdateLastSeen on rejected propagate")
	assert.Equal(t, 0, endpointRepo.updateRadioMetricsCalls, "no UpdateRadioMetrics on rejected propagate")
	assert.Equal(t, 0, endpointRepo.updateRadioMetricsSelectiveCalls, "no UpdateRadioMetricsSelective on rejected propagate")
	assert.Equal(t, 0, endpointRepo.updateDetachMetricsCalls, "no UpdateDetachMetrics on rejected propagate")
	assert.Equal(t, 0, endpointRepo.updateWithEUICalls, "no UpdateWithEUI on rejected propagate")
	assert.Equal(t, 0, endpointRepo.deleteByTenantCalls, "no DeleteByTenant on rejected propagate")
}

// testInt64 coerces any numeric value decoded from msgpack/JSON to int64.
// MessagePack picks the narrowest integer type that fits; JSON decodes numbers
// as float64. Tests assert through this helper to stay deterministic.
func testInt64(t *testing.T, v interface{}) int64 {
	t.Helper()
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		t.Fatalf("not a numeric type: %T (%v)", v, v)
		return 0
	}
}

// detPrpFailingWriteConn returns a hard write error so that sendMessage fails.
type detPrpFailingWriteConn struct {
	detPrpTestConn
}

func (c *detPrpFailingWriteConn) Write(_ []byte) (int, error) {
	return 0, fmt.Errorf("simulated write failure")
}

// TestSendDetachPropagateComplete_SendFailure exercises the path where the
// three-way handshake completion message fails to send. The handler must propagate
// the sendMessage error to its caller after logging LogBSSCIFailedToSendDetachPropagateComplete.
func TestSendDetachPropagateComplete_SendFailure(t *testing.T) {
	t.Parallel()

	const (
		testTenantID = int64(100)
		testOpId     = int64(-67890)
		testEpEui    = uint64(0x12345678)
		testBsEui    = uint64(0x1122334455667788)
	)

	msgRepo := &capturingDetPrpMIOTYMessageRepo{}
	endpointRepo := &detPrpTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{testEpEui: {ID: 1, EUI: models.EUI{0x00, 0x00, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78}, TenantID: testTenantID}},
	}
	pendingOps := &detPrpCapturingPendingOps{}
	storageImpl := &detPrpCapturingStorage{miotyMessages: msgRepo, endpointRepo: endpointRepo, pendingOps: pendingOps}
	eventStore := &detPrpCapturingEventStore{}

	// Zap observer logger so the test can assert the failure log token.
	core, recorded := observer.New(zapcore.DebugLevel)
	testLogger := logger.FromZap(zap.New(core))

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)
	server := bssci.NewTestServer(testLogger, storageImpl, eventStore, testTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	mockConn := &detPrpFailingWriteConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-detprp-send-failure",
			BaseStationEUI:    testBsEui,
			Encoding:          "msgpack",
			HandshakeComplete: true,
			ResolvedTenantID:  testTenantID,
			DbSessionID:       1,
		},
		Conn: mockConn,
	}
	server.RegisterSession(session)

	pendingOp := &bssci.PendingOperation{
		OperationType: mioty.CmdDetachPropagate,
		OperationID:   testOpId,
		CreatedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"endpointEUI": float64(testEpEui),
			"tenantId":    float64(testTenantID),
		},
	}
	require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContextWithTenant(testTenantID), session, testOpId, pendingOp, session.DbSessionID))

	data := map[string]interface{}{
		"command": mioty.CmdDetachPropagateResponse,
		"opId":    testOpId,
		"result":  int64(0),
	}
	msg := &bssci.Message{
		Command: mioty.CmdDetachPropagateResponse,
		OpId:    testOpId,
		Data:    data,
	}

	err := server.CallHandleDetachPropagateResponse(session, msg, data)
	require.Error(t, err, "send-complete failure must propagate as a returned error")
	assert.Contains(t, err.Error(), "simulated write failure", "the underlying connection error must be surfaced")

	// Log assertion: the handler must emit LogBSSCIFailedToSendDetachPropagateComplete at error level.
	var found bool
	for _, entry := range recorded.All() {
		if entry.Level == zapcore.ErrorLevel && entry.Message == bssci.LogBSSCIFailedToSendDetachPropagateComplete {
			found = true
			break
		}
	}
	assert.True(t, found, "expected error-level log %q to be emitted", bssci.LogBSSCIFailedToSendDetachPropagateComplete)
}
