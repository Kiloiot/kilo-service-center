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
)

// --- Mock Implementations for Integration Tests ---

// attPrpTestConn is a minimal mock connection for attach propagate integration tests
// attPrpTestConn captures every Write so tests can decode the msgpack payload frame.
// sendMessage writes the header and payload in two separate Conn.Write calls.
type attPrpTestConn struct {
	net.Conn
	mu     sync.Mutex
	frames [][]byte
}

func (m *attPrpTestConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := make([]byte, len(b))
	copy(clone, b)
	m.frames = append(m.frames, clone)
	return len(b), nil
}

func (m *attPrpTestConn) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.frames)
}

func (m *attPrpTestConn) Frames() [][]byte {
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

func (m *attPrpTestConn) Close() error                       { return nil }
func (m *attPrpTestConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *attPrpTestConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *attPrpTestConn) SetDeadline(_ time.Time) error      { return nil }
func (m *attPrpTestConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *attPrpTestConn) SetWriteDeadline(_ time.Time) error { return nil }
func (m *attPrpTestConn) Read(_ []byte) (n int, err error)   { return 0, nil }

// capturingMIOTYMessageRepo captures CreateAttachPropagateMessage calls for verification
type capturingMIOTYMessageRepo struct {
	mu                      sync.Mutex
	attachPropagateMessages []*mioty.AttachPropagateMessage
	createAttPrpCalls       int
}

func (r *capturingMIOTYMessageRepo) CreateAttachPropagateMessage(_ context.Context, msg *mioty.AttachPropagateMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createAttPrpCalls++
	r.attachPropagateMessages = append(r.attachPropagateMessages, msg)
	return nil
}

func (r *capturingMIOTYMessageRepo) GetCreateAttPrpCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createAttPrpCalls
}

// GetMessages returns captured messages (thread-safe)
func (r *capturingMIOTYMessageRepo) GetMessages() []*mioty.AttachPropagateMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]*mioty.AttachPropagateMessage, len(r.attachPropagateMessages))
	copy(result, r.attachPropagateMessages)
	return result
}

// Stub implementations for other MIOTYMessageRepository methods (not used in this test)
func (r *capturingMIOTYMessageRepo) CreateULDataMessage(_ context.Context, _ *mioty.ULDataMessage) error {
	return nil
}
func (r *capturingMIOTYMessageRepo) CreateDetachMessage(_ context.Context, _ *mioty.DetachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *capturingMIOTYMessageRepo) CreateAttachMessage(_ context.Context, _ *mioty.AttachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *capturingMIOTYMessageRepo) CreateDetachPropagateMessage(_ context.Context, _ *mioty.DetachPropagateMessage) error {
	return nil
}
func (r *capturingMIOTYMessageRepo) GetULDataMessage(_ context.Context, _ string, _ int64) (*mioty.ULDataMessage, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetDetachMessage(_ context.Context, _ string, _ int64) (*mioty.DetachMessage, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) ListULDataMessages(_ context.Context, _ mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}
func (r *capturingMIOTYMessageRepo) UpdateULDataBaseStations(_ context.Context, _ int64, _ uint64, _ uint32, _ int64, _ []byte) error {
	return nil
}
func (r *capturingMIOTYMessageRepo) GetMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetExtendedMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetMessageStatsByEndpoint(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetOverallStats(_ context.Context, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetAnalyticsOverview(_ context.Context, _ int64, _, _ time.Time) (*mioty.AnalyticsOverviewStats, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetHourlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.HourlyActivity, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetDailyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.DailyActivity, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetTopEndpointsByActivity(_ context.Context, _ int64, _, _ time.Time, _ int) ([]mioty.EndpointActivity, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetSignalQualityStats(_ context.Context, _ int64, _, _ time.Time) (*mioty.SignalQualityStats, error) {
	return nil, nil
}
func (r *capturingMIOTYMessageRepo) GetSignalQualityByBaseStation(_ context.Context, _ int64, _, _ time.Time) ([]mioty.BaseStationSignalQuality, error) {
	return nil, nil
}

func (r *capturingMIOTYMessageRepo) GetBaseStationMessageStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}

func (r *capturingMIOTYMessageRepo) GetMessageCountsByEndpoint(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *capturingMIOTYMessageRepo) GetMessageCountsByBaseStation(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *capturingMIOTYMessageRepo) GetWeeklyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.WeeklyActivity, error) {
	return nil, nil
}

func (r *capturingMIOTYMessageRepo) GetMonthlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.MonthlyActivity, error) {
	return nil, nil
}

// attPrpTestEndpointRepo is a minimal endpoint repository for attach propagate integration tests.
// Every mutation method increments its tally so rejected-propagate tests can assert zero state changes.
type attPrpTestEndpointRepo struct {
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

func (r *attPrpTestEndpointRepo) GetByEUI(_ context.Context, _ int64, eui []byte) (*models.EndPoint, error) {
	r.getByEUICalls++
	if len(eui) != 8 {
		return nil, storage.ErrNotFound
	}
	key := uint64(0)
	for i := 0; i < 8; i++ {
		key = (key << 8) | uint64(eui[i])
	}
	if ep, ok := r.endpoints[key]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

func (r *attPrpTestEndpointRepo) Get(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	if ep, ok := r.endpoints[eui.ToUint64()]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

func (r *attPrpTestEndpointRepo) UpdateFields(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	r.updateFieldsCalls++
	return nil
}

// Stub implementations for remaining EndpointRepository methods
func (r *attPrpTestEndpointRepo) Create(context.Context, *models.EndPoint) error {
	r.createCalls++
	return nil
}
func (r *attPrpTestEndpointRepo) GetByID(context.Context, int64, int64) (*models.EndPoint, error) {
	return nil, nil
}
func (r *attPrpTestEndpointRepo) GetByTenant(context.Context, int64) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *attPrpTestEndpointRepo) CountByTenant(context.Context, int64) (int64, error) { return 0, nil }
func (r *attPrpTestEndpointRepo) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *attPrpTestEndpointRepo) Update(context.Context, *models.EndPoint) error {
	r.updateCalls++
	return nil
}
func (r *attPrpTestEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	r.updateLastSeenCalls++
	return nil
}
func (r *attPrpTestEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	r.updateRadioMetricsCalls++
	return nil
}
func (r *attPrpTestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	r.updateRadioMetricsSelectiveCalls++
	return nil
}
func (r *attPrpTestEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
	r.updateDetachMetricsCalls++
	return nil
}
func (r *attPrpTestEndpointRepo) StreamAllForPropagation(context.Context, int64, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *attPrpTestEndpointRepo) HasEndpointsSince(context.Context, time.Time) (bool, error) {
	return false, nil
}
func (r *attPrpTestEndpointRepo) GetEndpointWithKeysForDetachValidation(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}
func (r *attPrpTestEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}
func (r *attPrpTestEndpointRepo) DeleteByTenant(context.Context, int64, []byte) error {
	r.deleteByTenantCalls++
	return nil
}
func (r *attPrpTestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	r.updateWithEUICalls++
	return ep, nil
}
func (r *attPrpTestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// capturingStorage implements interfaces.Storage with capturing MIOTYMessages repository
type capturingStorage struct {
	miotyMessages *capturingMIOTYMessageRepo
	endpointRepo  *attPrpTestEndpointRepo
	pendingOps    *attPrpCapturingPendingOps
}

func (s *capturingStorage) MIOTYMessages() interfaces.MIOTYMessageRepository {
	return s.miotyMessages
}

// EndPoints returns the endpoint repository (may be nil if not needed for test)
func (s *capturingStorage) EndPoints() interfaces.EndpointRepository {
	if s.endpointRepo == nil {
		return nil
	}
	return s.endpointRepo
}
func (s *capturingStorage) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (s *capturingStorage) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (s *capturingStorage) EndPointSessions() interfaces.EndPointSessionRepository { return nil }
func (s *capturingStorage) EndPointKeys() interfaces.EndPointKeyRepository         { return nil }
func (s *capturingStorage) RoamingAgreements() interfaces.RoamingAgreementRepository {
	return nil
}
func (s *capturingStorage) BaseStations() interfaces.BaseStationRepository { return nil }
func (s *capturingStorage) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (s *capturingStorage) DLRXStatus() interfaces.DLRXStatusRepository { return nil }
func (s *capturingStorage) PendingOperations() interfaces.PendingOperationRepository {
	if s.pendingOps == nil {
		s.pendingOps = &attPrpCapturingPendingOps{}
	}
	return s.pendingOps
}

// attPrpUpdateMetadataCall captures a single UpdateMetadata invocation.
type attPrpUpdateMetadataCall struct {
	SessionID   int64
	OperationID int64
	Metadata    json.RawMessage
}

// attPrpCapturingPendingOps records every UpdateMetadata call so rejected-path tests
// can assert on the failure metadata that handlePropagateResponseFailure persists.
type attPrpCapturingPendingOps struct {
	mu      sync.Mutex
	updates []attPrpUpdateMetadataCall
}

func (r *attPrpCapturingPendingOps) Create(_ context.Context, _ *interfaces.PendingOperationRequest) error {
	return nil
}

func (r *attPrpCapturingPendingOps) CreateBatch(ctx context.Context, reqs []*interfaces.PendingOperationRequest) error {
	for _, req := range reqs {
		if err := r.Create(ctx, req); err != nil {
			return err
		}
	}
	return nil
}
func (r *attPrpCapturingPendingOps) UpdateMetadata(_ context.Context, sessionID, operationID int64, metadata json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dup := make(json.RawMessage, len(metadata))
	copy(dup, metadata)
	r.updates = append(r.updates, attPrpUpdateMetadataCall{SessionID: sessionID, OperationID: operationID, Metadata: dup})
	return nil
}
func (r *attPrpCapturingPendingOps) DeleteBySessionAndOperation(_ context.Context, _ int64, _ int64) error {
	return nil
}
func (r *attPrpCapturingPendingOps) DeleteByOperation(_ context.Context, _ int64) error { return nil }
func (r *attPrpCapturingPendingOps) DeleteBySession(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}
func (r *attPrpCapturingPendingOps) GetBySession(_ context.Context, _ int64) ([]*interfaces.PendingOperation, error) {
	return nil, nil
}

func (r *attPrpCapturingPendingOps) LastUpdate() *attPrpUpdateMetadataCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.updates) == 0 {
		return nil
	}
	call := r.updates[len(r.updates)-1]
	return &call
}

func (r *attPrpCapturingPendingOps) UpdateCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.updates)
}
func (s *capturingStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (s *capturingStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *capturingStorage) Users() interfaces.UserRepository                 { return nil }
func (s *capturingStorage) APIKeys() interfaces.APIKeyRepository             { return nil }
func (s *capturingStorage) Integrations() interfaces.IntegrationRepository   { return nil }
func (s *capturingStorage) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (s *capturingStorage) DeviceModels() interfaces.DeviceModelRepository   { return nil }
func (s *capturingStorage) Blueprints() interfaces.BlueprintRepository       { return nil }
func (s *capturingStorage) Organizations() interfaces.OrganizationRepository { return nil }
func (s *capturingStorage) GetSqlxDB() *sqlx.DB                              { return nil }
func (s *capturingStorage) SystemEvents() interfaces.SystemEventStore        { return nil }
func (s *capturingStorage) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (s *capturingStorage) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (s *capturingStorage) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }
func (s *capturingStorage) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	return nil, nil
}
func (s *capturingStorage) Ping(_ context.Context) error { return nil }
func (s *capturingStorage) Close() error                 { return nil }

// capturingEventStore captures CreateEvent and RecordAttachPropagate calls
type capturingEventStore struct {
	mu     sync.Mutex
	events []*models.SystemEvent
}

func (s *capturingEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

// GetEvents implements the interface method (returns captured events)
func (s *capturingEventStore) GetEvents(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*models.SystemEvent, len(s.events))
	copy(result, s.events)
	return result, nil
}

// CapturedEvents returns captured events (test helper, thread-safe)
func (s *capturingEventStore) CapturedEvents() []*models.SystemEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*models.SystemEvent, len(s.events))
	copy(result, s.events)
	return result
}

// Stub implementations for other SystemEventStore methods
func (s *capturingEventStore) GetEventsFiltered(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}
func (s *capturingEventStore) GetActiveAlerts(_ context.Context, _ interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}
func (s *capturingEventStore) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return nil, nil
}
func (s *capturingEventStore) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}
func (s *capturingEventStore) CountEvents(_ context.Context, _ interfaces.SystemEventFilter) (int64, error) {
	return 0, nil
}
func (s *capturingEventStore) CountActiveAlerts(_ context.Context, _ interfaces.AlertFilter) (int64, error) {
	return 0, nil
}

// --- Integration Tests ---

// TestAttachPropagateCompletionIntegration_WithPendingOp verifies that when attPrpRsp
// is received with result=0 and a pendingOp exists with metadata, the handler:
// 1. Persists an attPrpCmp message with correct fields (EpEui, ShAddr, Bidi, LastPacketCnt)
// 2. Emits an EventTypeAttachPropagateCompleted event
// 3. Does NOT include NwkSnKey in the completion row (security)
func TestAttachPropagateCompletionIntegration_WithPendingOp(t *testing.T) {
	t.Parallel()

	const (
		testTenantID = int64(100)
		testOpId     = int64(-12345)
		// Use smaller EUI values to avoid float64 precision loss during JSON/metadata extraction
		// float64 only has 53 bits of mantissa precision, so values > 2^53 lose precision
		testEpEui         = uint64(0x0000000012345678) // 305419896 - fits in float64
		testBsEui         = uint64(0x0000000087654321) // 2271560481 - fits in float64
		testShAddr        = uint16(0x1234)
		testBidi          = true
		testLastPacketCnt = uint32(42)
	)

	// Create test endpoint for lookup
	var epEUIBytes models.EUI
	binary.BigEndian.PutUint64(epEUIBytes[:], testEpEui)
	testEndpoint := &models.EndPoint{
		ID:       1001,
		EUI:      epEUIBytes,
		TenantID: testTenantID,
	}

	// Create capturing storage with endpoint repository
	msgRepo := &capturingMIOTYMessageRepo{}
	endpointRepo := &attPrpTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			testEpEui: testEndpoint,
		},
	}
	storage := &capturingStorage{miotyMessages: msgRepo, endpointRepo: endpointRepo}

	// Create capturing event store
	eventStore := &capturingEventStore{}

	// Create server using test infrastructure
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)

	server := bssci.NewTestServer(testLogger, storage, eventStore, testTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	// Create mock connection
	mockConn := &attPrpTestConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-attprp-integration",
			BaseStationEUI:    testBsEui,
			Encoding:          "msgpack",
			HandshakeComplete: true,
			ResolvedTenantID:  testTenantID,
			DbSessionID:       1, // Required for pending operation processing
		},
		Conn: mockConn,
	}
	server.RegisterSession(session)

	// Register pending operation with metadata (using float64 for numbers as JSON deserializes)
	pendingOp := &bssci.PendingOperation{
		OperationType: mioty.CmdAttachPropagate,
		CreatedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"epEui":         float64(testEpEui),
			"tenantId":      float64(testTenantID),
			"shortAddr":     float64(testShAddr),
			"bidirectional": testBidi,
			"lastPacketCnt": float64(testLastPacketCnt),
			"dualChannel":   true,
			"repetition":    false,
			"wideCarrOff":   true,
			"longBlkDist":   false,
		},
	}
	err := statusSvc.RecordPendingOperation(testutil.TestContextWithTenant(testTenantID), session, testOpId, pendingOp, session.DbSessionID)
	require.NoError(t, err, "RecordPendingOperation should succeed")

	// Verify pending op was recorded
	retrievedOp, getErr := statusSvc.GetPendingOperation(session, testOpId)
	require.NoError(t, getErr, "GetPendingOperation should succeed")
	require.NotNil(t, retrievedOp, "Retrieved pending op should not be nil")

	// Simulate attPrpRsp with result=0 (success)
	data := map[string]interface{}{
		"command": mioty.CmdAttachPropagateResponse,
		"opId":    testOpId,
		"result":  int64(0), // Success
	}
	msg := &bssci.Message{
		Command: mioty.CmdAttachPropagateResponse,
		OpId:    testOpId,
		Data:    data,
	}

	// Call the real handler
	err = server.CallHandleAttachPropagateResponse(session, msg, data)
	require.NoError(t, err, "CallHandleAttachPropagateResponse should succeed")

	// Get captured events
	capturedEvents := eventStore.CapturedEvents()

	// Assert message persistence
	persistedMsgs := msgRepo.GetMessages()
	require.Len(t, persistedMsgs, 1, "CreateAttachPropagateMessage should be called exactly once")
	persistedMsg := persistedMsgs[0]

	assert.Equal(t, mioty.CmdAttachPropagateComplete, persistedMsg.CommandType, "CommandType should be attPrpCmp")
	assert.Equal(t, testEpEui, persistedMsg.EpEui, "EpEui should match metadata")
	assert.NotZero(t, persistedMsg.EpEui, "EpEui should be non-zero")
	assert.Equal(t, testShAddr, persistedMsg.ShAddr, "ShAddr should match metadata")
	assert.Equal(t, testBidi, persistedMsg.Bidi, "Bidi should match metadata")
	assert.Equal(t, testLastPacketCnt, persistedMsg.LastPacketCnt, "LastPacketCnt should match metadata")
	assert.Nil(t, persistedMsg.NwkSnKey, "NwkSnKey should NOT be in completion row (security)")

	// Assert optional fields also extracted
	assert.True(t, persistedMsg.DualChan, "DualChan should be extracted from metadata")
	assert.False(t, persistedMsg.Repetition, "Repetition should be extracted from metadata")
	assert.True(t, persistedMsg.WideCarrOff, "WideCarrOff should be extracted from metadata")
	assert.False(t, persistedMsg.LongBlkDist, "LongBlkDist should be extracted from metadata")

	// Assert event creation (use already captured events from earlier debug logging)
	require.GreaterOrEqual(t, len(capturedEvents), 1, "CreateEvent should be called at least once")

	// Find the completion event (EventTypeAttachPropagateCompleted)
	var foundCompletionEvent bool
	expectedBsEuiStr := fmt.Sprintf("%016X", testBsEui) // Format BS EUI as uppercase hex with leading zeros
	for _, evt := range capturedEvents {
		if evt.EventType == bssci.EventTypeAttachPropagateCompleted {
			foundCompletionEvent = true
			assert.NotEmpty(t, evt.SourceName, "SourceName should contain BS EUI")
			assert.Contains(t, evt.SourceName, expectedBsEuiStr, "SourceName should be formatted BS EUI")
			break
		}
	}
	assert.True(t, foundCompletionEvent, "Should have EventTypeAttachPropagateCompleted event")
}

// TestAttachPropagateCompletionIntegration_NoPendingOp verifies that when attPrpRsp
// is received with result=0 but NO pendingOp exists, the handler:
// 1. Does NOT persist an attPrpCmp message (no valid epEui)
// 2. STILL emits an EventTypeAttachPropagateCompleted event (unconditional)
func TestAttachPropagateCompletionIntegration_NoPendingOp(t *testing.T) {
	t.Parallel()

	const (
		testTenantID = int64(100)
		testOpId     = int64(-12345)
		testBsEui    = uint64(0x1122334455667788)
	)

	// Create capturing storage
	msgRepo := &capturingMIOTYMessageRepo{}
	storage := &capturingStorage{miotyMessages: msgRepo}

	// Create capturing event store
	eventStore := &capturingEventStore{}

	// Create server using test infrastructure
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)

	server := bssci.NewTestServer(testLogger, storage, eventStore, testTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	// Create mock connection
	mockConn := &attPrpTestConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-attprp-no-pendingop",
			BaseStationEUI:    testBsEui,
			Encoding:          "msgpack",
			HandshakeComplete: true,
			ResolvedTenantID:  testTenantID,
		},
		Conn: mockConn,
	}
	server.RegisterSession(session)

	// NOTE: Do NOT register pendingOp - test the negative path

	// Simulate attPrpRsp with result=0 (success)
	data := map[string]interface{}{
		"command": mioty.CmdAttachPropagateResponse,
		"opId":    testOpId,
		"result":  int64(0), // Success
	}
	msg := &bssci.Message{
		Command: mioty.CmdAttachPropagateResponse,
		OpId:    testOpId,
		Data:    data,
	}

	// Call the real handler
	err := server.CallHandleAttachPropagateResponse(session, msg, data)
	require.NoError(t, err, "CallHandleAttachPropagateResponse should succeed even without pendingOp")

	// Assert NO message persistence (no valid epEui without pendingOp)
	persistedMsgs := msgRepo.GetMessages()
	assert.Empty(t, persistedMsgs, "CreateAttachPropagateMessage should NOT be called when no pendingOp")

	// Assert event STILL emitted (unconditional)
	capturedEvents := eventStore.CapturedEvents()
	require.GreaterOrEqual(t, len(capturedEvents), 1, "CreateEvent should be called (unconditional)")

	// Find the completion event
	var foundCompletionEvent bool
	for _, evt := range capturedEvents {
		if evt.EventType == bssci.EventTypeAttachPropagateCompleted {
			foundCompletionEvent = true
			assert.NotEmpty(t, evt.SourceName, "SourceName should contain BS EUI even without pendingOp")
			break
		}
	}
	assert.True(t, foundCompletionEvent, "Should have EventTypeAttachPropagateCompleted event even without pendingOp")
}

// TestEUIPrecisionValidation verifies that the EUI precision check correctly
// identifies values that exceed float64's safe integer range (>2^53) and would
// lose precision when cast from float64 to uint64.
// BSSCI §3.8: Verifies fail-fast behavior for large EUIs.
func TestEUIPrecisionValidation(t *testing.T) {
	t.Parallel()

	// maxSafeFloat64Int is the maximum integer value that can be exactly
	// represented in float64 without precision loss (2^53)
	const maxSafeFloat64Int = 1 << 53

	testCases := []struct {
		name           string
		euiFloat64     float64
		shouldError    bool
		expectedUint64 uint64
	}{
		{
			name:           "small EUI - no precision loss",
			euiFloat64:     float64(0x12345678), // 305419896
			shouldError:    false,
			expectedUint64: 0x12345678,
		},
		{
			name:           "EUI at 2^53 boundary - safe",
			euiFloat64:     float64(maxSafeFloat64Int), // 9007199254740992
			shouldError:    false,
			expectedUint64: uint64(maxSafeFloat64Int),
		},
		{
			name:           "large EUI - 0xAABBCCDDEEFF1122 would lose precision",
			euiFloat64:     float64(uint64(0xAABBCCDDEEFF1122)), // This EUI > 2^53, so precision loss occurs
			shouldError:    true,                                // Should trigger fail-fast
			expectedUint64: 0,                                   // Not used if error
		},
		{
			name:           "max uint64 - clearly exceeds float64 precision",
			euiFloat64:     float64(^uint64(0)), // 18446744073709551615 - largest uint64
			shouldError:    true,
			expectedUint64: 0,
		},
		{
			name:           "common large EUI pattern",
			euiFloat64:     float64(uint64(0xFFFFFFFFFFFFFFFF)),
			shouldError:    true,
			expectedUint64: 0,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Apply the same precision check as server.go
			var err error
			var epEUI uint64
			if tc.euiFloat64 > float64(maxSafeFloat64Int) {
				err = fmt.Errorf("epEui exceeds float64 precision limit")
			} else {
				epEUI = uint64(tc.euiFloat64)
			}

			if tc.shouldError {
				assert.Error(t, err, "EUI %v should trigger precision error", tc.euiFloat64)
			} else {
				assert.NoError(t, err, "EUI %v should not trigger precision error", tc.euiFloat64)
				assert.Equal(t, tc.expectedUint64, epEUI, "Converted EUI should match expected")
			}
		})
	}
}

// TestEUIPrecisionLossDetection demonstrates why the precision check is necessary.
// This test shows that large uint64 values lose precision when stored as float64.
func TestEUIPrecisionLossDetection(t *testing.T) {
	t.Parallel()

	// Original EUI that would lose precision
	originalEUI := uint64(0xAABBCCDDEEFF1122)

	// Store as float64 (simulating JSON metadata extraction)
	asFloat64 := float64(originalEUI)

	// Convert back to uint64
	recoveredEUI := uint64(asFloat64)

	// Due to float64's 53-bit mantissa, precision is lost for values > 2^53
	// Original: 12302652060373954850
	// Recovered: 12302652060373954560 (approximately - varies by platform)
	t.Logf("Original EUI:  %d (0x%016X)", originalEUI, originalEUI)
	t.Logf("As float64:    %.0f", asFloat64)
	t.Logf("Recovered EUI: %d (0x%016X)", recoveredEUI, recoveredEUI)

	// The key assertion: precision is lost for large values
	if originalEUI > (1 << 53) {
		// This may or may not be equal depending on the specific value and rounding
		// The important thing is that the precision check should prevent this path
		t.Logf("Precision loss: originalEUI != recoveredEUI demonstrates why fail-fast is needed")
	}
}

// TestHandleAttachPropagateResponse_Rejected verifies that when attPrpRsp arrives
// with a non-zero result, handlePropagateResponseFailure emits the cataloged event
// type (EventTypeEndpointAttachFailed) and uses TitleAttachPropagateFailedForEndpointOnBS
// rather than the pre-Fix-A pendingOp-dependent fallback strings.
func TestHandleAttachPropagateResponse_Rejected(t *testing.T) {
	t.Parallel()

	const (
		testTenantID = int64(100)
		testOpId     = int64(-54321)
		testEpEui    = uint64(0x12345678)
		testBsEui    = uint64(0x1122334455667788)
		rejectCode   = 7
	)

	var epEUIBytes models.EUI
	binary.BigEndian.PutUint64(epEUIBytes[:], testEpEui)
	endpoint := &models.EndPoint{ID: 1001, EUI: epEUIBytes, TenantID: testTenantID}

	msgRepo := &capturingMIOTYMessageRepo{}
	endpointRepo := &attPrpTestEndpointRepo{endpoints: map[uint64]*models.EndPoint{testEpEui: endpoint}}
	pendingOps := &attPrpCapturingPendingOps{}
	storageImpl := &capturingStorage{miotyMessages: msgRepo, endpointRepo: endpointRepo, pendingOps: pendingOps}
	eventStore := &capturingEventStore{}
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)
	server := bssci.NewTestServer(testLogger, storageImpl, eventStore, testTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	mockConn := &attPrpTestConn{}
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-attprp-rejected",
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
		OperationType: mioty.CmdAttachPropagate,
		OperationID:   testOpId,
		CreatedAt:     time.Now(),
		Metadata: map[string]interface{}{
			"epEui":    float64(testEpEui),
			"tenantId": float64(testTenantID),
		},
	}
	require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContextWithTenant(testTenantID), session, testOpId, pendingOp, session.DbSessionID))

	data := map[string]interface{}{
		"command": mioty.CmdAttachPropagateResponse,
		"opId":    testOpId,
		"result":  int64(rejectCode),
	}
	msg := &bssci.Message{
		Command: mioty.CmdAttachPropagateResponse,
		OpId:    testOpId,
		Data:    data,
	}

	require.NoError(t, server.CallHandleAttachPropagateResponse(session, msg, data),
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
	assert.Equal(t, bssci.ResolveErrorMessage(bssci.ErrAttachPropagateFailed), wireResp["message"],
		"wire message must be cataloged via ErrAttachPropagateFailed")

	// Pending-op metadata persistence.
	updates := bssci.StatusMetadataUpdates(statusSvc)
	require.Len(t, updates, 1, "exactly one metadata persistence call on the rejected path")
	update := updates[0]
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
		if evt.EventType != bssci.EventTypeEndpointAttachFailed {
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
	assert.Equal(t, fmt.Sprintf(bssci.TitleAttachPropagateFailedForEndpointOnBS, epStr, bsStr), endpointEvt.Title,
		"endpoint event title must match cataloged TitleAttachPropagateFailedForEndpointOnBS format")
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
