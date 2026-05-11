package bssci_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Implementations for Detach Propagate Complete Integration Tests ---
// Reuses patterns from attach_propagate_integration_test.go

// detPrpTestConn is a minimal mock connection
type detPrpTestConn struct {
	net.Conn
	mu        sync.Mutex
	sentCount int
}

func (m *detPrpTestConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentCount++
	return len(b), nil
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

// detPrpTestEndpointRepo is a minimal endpoint repository
type detPrpTestEndpointRepo struct {
	endpoints         map[uint64]*models.EndPoint
	getByEUICalls     int
	updateFieldsCalls int
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
func (r *detPrpTestEndpointRepo) Create(context.Context, *models.EndPoint) error { return nil }
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
func (r *detPrpTestEndpointRepo) Update(context.Context, *models.EndPoint) error { return nil }
func (r *detPrpTestEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
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
	return nil
}
func (r *detPrpTestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (r *detPrpTestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// detPrpCapturingStorage implements interfaces.Storage
type detPrpCapturingStorage struct {
	miotyMessages *capturingDetPrpMIOTYMessageRepo
	endpointRepo  *detPrpTestEndpointRepo
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
	return nil
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
		ID:                "test-detprp-integration",
		BaseStationEUI:    testBsEui,
		Conn:              mockConn,
		Encoding:          "msgpack",
		HandshakeComplete: true,
		ResolvedTenantID:  testTenantID,
		DbSessionID:       1, // Required for pending operation processing
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
	err := statusSvc.RecordPendingOperation(context.Background(), session, testOpId, pendingOp, session.DbSessionID)
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
		ID:                "test-detprp-no-pendingop",
		BaseStationEUI:    testBsEui,
		Conn:              mockConn,
		Encoding:          "msgpack",
		HandshakeComplete: true,
		ResolvedTenantID:  testTenantID,
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
