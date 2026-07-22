package bssci_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
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

// ============================================================================
// UL Data Deduplication Integration Tests (BSSCI §5.10, SCACI §3.8.1)
// Tests for multi-BS reception, deduplication, persistence, and window handling
// ============================================================================

// --- Test Constants ---

const (
	ulTestTenantID = int64(100)
	ulTestEpEui    = uint64(0x0000000012345678)
	ulTestBsEui1   = uint64(0x0000000087654321)
	ulTestBsEui2   = uint64(0x0000000087654322)
	ulTestBsEui3   = uint64(0x0000000087654323)
)

// --- Mock Implementations ---

// ulTestConn is a minimal mock connection for UL data integration tests
type ulTestConn struct {
	net.Conn
	mu        sync.Mutex
	sentCount int
}

func (m *ulTestConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentCount++
	return len(b), nil
}

func (m *ulTestConn) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentCount
}

func (m *ulTestConn) Close() error                       { return nil }
func (m *ulTestConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *ulTestConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *ulTestConn) SetDeadline(_ time.Time) error      { return nil }
func (m *ulTestConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *ulTestConn) SetWriteDeadline(_ time.Time) error { return nil }
func (m *ulTestConn) Read(_ []byte) (n int, err error)   { return 0, nil }

// updateBSParams captures parameters for UpdateULDataBaseStations calls
type updateBSParams struct {
	tenantID         int64
	epEui            uint64
	packetCnt        uint32
	rxTimeMin        int64
	baseStationsJSON []byte
}

// ulCapturingMIOTYMessageRepo captures UL data persistence calls
type ulCapturingMIOTYMessageRepo struct {
	mu                       sync.Mutex
	ulDataMessages           []*mioty.ULDataMessage
	createULDataCalls        int
	updateBaseStationsCalls  int
	updateBaseStationsParams []updateBSParams
}

func (r *ulCapturingMIOTYMessageRepo) CreateULDataMessage(_ context.Context, msg *mioty.ULDataMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createULDataCalls++
	// Deep copy to avoid mutation
	msgCopy := *msg
	if msg.BaseStations != nil {
		msgCopy.BaseStations = make([]mioty.BaseStationReception, len(msg.BaseStations))
		copy(msgCopy.BaseStations, msg.BaseStations)
	}
	r.ulDataMessages = append(r.ulDataMessages, &msgCopy)
	return nil
}

func (r *ulCapturingMIOTYMessageRepo) UpdateULDataBaseStations(_ context.Context, tenantID int64, epEui uint64, packetCnt uint32, rxTimeMin int64, baseStationsJSON []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateBaseStationsCalls++
	jsonCopy := make([]byte, len(baseStationsJSON))
	copy(jsonCopy, baseStationsJSON)
	r.updateBaseStationsParams = append(r.updateBaseStationsParams, updateBSParams{
		tenantID:         tenantID,
		epEui:            epEui,
		packetCnt:        packetCnt,
		rxTimeMin:        rxTimeMin,
		baseStationsJSON: jsonCopy,
	})
	return nil
}

// Helper getters (thread-safe)
func (r *ulCapturingMIOTYMessageRepo) GetCreateULDataCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createULDataCalls
}

func (r *ulCapturingMIOTYMessageRepo) GetUpdateBaseStationsCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.updateBaseStationsCalls
}

func (r *ulCapturingMIOTYMessageRepo) GetCapturedMessages() []*mioty.ULDataMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]*mioty.ULDataMessage, len(r.ulDataMessages))
	copy(result, r.ulDataMessages)
	return result
}

func (r *ulCapturingMIOTYMessageRepo) GetUpdateParams() []updateBSParams {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]updateBSParams, len(r.updateBaseStationsParams))
	copy(result, r.updateBaseStationsParams)
	return result
}

// Stub implementations for remaining MIOTYMessageRepository methods (17 total)
func (r *ulCapturingMIOTYMessageRepo) CreateDetachMessage(_ context.Context, _ *mioty.DetachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *ulCapturingMIOTYMessageRepo) CreateAttachMessage(_ context.Context, _ *mioty.AttachMessage, _ map[string]interface{}) error {
	return nil
}
func (r *ulCapturingMIOTYMessageRepo) CreateAttachPropagateMessage(_ context.Context, _ *mioty.AttachPropagateMessage) error {
	return nil
}
func (r *ulCapturingMIOTYMessageRepo) CreateDetachPropagateMessage(_ context.Context, _ *mioty.DetachPropagateMessage) error {
	return nil
}
func (r *ulCapturingMIOTYMessageRepo) GetULDataMessage(_ context.Context, _ string, _ int64) (*mioty.ULDataMessage, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetDetachMessage(_ context.Context, _ string, _ int64) (*mioty.DetachMessage, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) ListULDataMessages(_ context.Context, _ mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetExtendedMessageStatsByBaseStation(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetMessageStatsByEndpoint(_ context.Context, _ uint64, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetOverallStats(_ context.Context, _ int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetAnalyticsOverview(_ context.Context, _ int64, _, _ time.Time) (*mioty.AnalyticsOverviewStats, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetHourlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.HourlyActivity, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetDailyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.DailyActivity, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetTopEndpointsByActivity(_ context.Context, _ int64, _, _ time.Time, _ int) ([]mioty.EndpointActivity, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetSignalQualityStats(_ context.Context, _ int64, _, _ time.Time) (*mioty.SignalQualityStats, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetSignalQualityByBaseStation(_ context.Context, _ int64, _, _ time.Time) ([]mioty.BaseStationSignalQuality, error) {
	return nil, nil
}
func (r *ulCapturingMIOTYMessageRepo) GetBaseStationMessageStats(_ context.Context, _ int64, _ []byte, _, _ *time.Time) (*mioty.BaseStationMessageStats, error) {
	return &mioty.BaseStationMessageStats{}, nil
}

func (r *ulCapturingMIOTYMessageRepo) GetMessageCountsByEndpoint(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *ulCapturingMIOTYMessageRepo) GetMessageCountsByBaseStation(_ context.Context, _ int64, _, _ time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *ulCapturingMIOTYMessageRepo) GetWeeklyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.WeeklyActivity, error) {
	return nil, nil
}

func (r *ulCapturingMIOTYMessageRepo) GetMonthlyActivity(_ context.Context, _ int64, _, _ time.Time) ([]mioty.MonthlyActivity, error) {
	return nil, nil
}

// ulTestEndpointRepo is a minimal endpoint repository for UL data integration tests
type ulTestEndpointRepo struct {
	endpoints         map[uint64]*models.EndPoint
	getByEUICalls     int
	updateFieldsCalls int
}

func (r *ulTestEndpointRepo) GetByEUI(_ context.Context, _ int64, eui []byte) (*models.EndPoint, error) {
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

func (r *ulTestEndpointRepo) Get(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	if ep, ok := r.endpoints[eui.ToUint64()]; ok {
		return ep, nil
	}
	return nil, storage.ErrNotFound
}

func (r *ulTestEndpointRepo) UpdateFields(_ context.Context, _ int64, _ int64, _ map[string]interface{}) error {
	r.updateFieldsCalls++
	return nil
}

// Stub implementations for remaining EndpointRepository methods
func (r *ulTestEndpointRepo) Create(context.Context, *models.EndPoint) error { return nil }
func (r *ulTestEndpointRepo) GetByID(context.Context, int64, int64) (*models.EndPoint, error) {
	return nil, nil
}
func (r *ulTestEndpointRepo) GetByTenant(context.Context, int64) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *ulTestEndpointRepo) CountByTenant(context.Context, int64) (int64, error) { return 0, nil }
func (r *ulTestEndpointRepo) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *ulTestEndpointRepo) Update(context.Context, *models.EndPoint) error { return nil }
func (r *ulTestEndpointRepo) UpdateLastSeen(context.Context, int64, models.EUI, uint32) error {
	return nil
}
func (r *ulTestEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	return nil
}
func (r *ulTestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	return nil
}
func (r *ulTestEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
	return nil
}
func (r *ulTestEndpointRepo) StreamAllForPropagation(context.Context, int64, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (r *ulTestEndpointRepo) HasEndpointsSince(context.Context, time.Time) (bool, error) {
	return false, nil
}
func (r *ulTestEndpointRepo) GetEndpointWithKeysForDetachValidation(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}
func (r *ulTestEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}
func (r *ulTestEndpointRepo) DeleteByTenant(context.Context, int64, []byte) error {
	return nil
}
func (r *ulTestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (r *ulTestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// ulCapturingStorage implements interfaces.Storage with UL-capturing MIOTYMessages
type ulCapturingStorage struct {
	miotyMessages *ulCapturingMIOTYMessageRepo
	endpointRepo  *ulTestEndpointRepo
}

func (s *ulCapturingStorage) MIOTYMessages() interfaces.MIOTYMessageRepository {
	return s.miotyMessages
}
func (s *ulCapturingStorage) EndPoints() interfaces.EndpointRepository {
	if s.endpointRepo == nil {
		return nil
	}
	return s.endpointRepo
}
func (s *ulCapturingStorage) DownlinkQueue() interfaces.DownlinkQueueRepository { return nil }
func (s *ulCapturingStorage) BaseStationReceptions() interfaces.BaseStationReceptionRepository {
	return nil
}
func (s *ulCapturingStorage) EndPointSessions() interfaces.EndPointSessionRepository { return nil }
func (s *ulCapturingStorage) EndPointKeys() interfaces.EndPointKeyRepository         { return nil }
func (s *ulCapturingStorage) RoamingAgreements() interfaces.RoamingAgreementRepository {
	return nil
}
func (s *ulCapturingStorage) BaseStations() interfaces.BaseStationRepository { return nil }
func (s *ulCapturingStorage) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (s *ulCapturingStorage) DLRXStatus() interfaces.DLRXStatusRepository { return nil }
func (s *ulCapturingStorage) PendingOperations() interfaces.PendingOperationRepository {
	return nil
}
func (s *ulCapturingStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (s *ulCapturingStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *ulCapturingStorage) Users() interfaces.UserRepository                 { return nil }
func (s *ulCapturingStorage) APIKeys() interfaces.APIKeyRepository             { return nil }
func (s *ulCapturingStorage) Integrations() interfaces.IntegrationRepository   { return nil }
func (s *ulCapturingStorage) Manufacturers() interfaces.ManufacturerRepository { return nil }
func (s *ulCapturingStorage) DeviceModels() interfaces.DeviceModelRepository   { return nil }
func (s *ulCapturingStorage) Blueprints() interfaces.BlueprintRepository       { return nil }
func (s *ulCapturingStorage) Organizations() interfaces.OrganizationRepository { return nil }
func (s *ulCapturingStorage) GetSqlxDB() *sqlx.DB                              { return nil }
func (s *ulCapturingStorage) SystemEvents() interfaces.SystemEventStore        { return nil }
func (s *ulCapturingStorage) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (s *ulCapturingStorage) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}
func (s *ulCapturingStorage) DownlinkQueueReader() interfaces.DownlinkQueueReader { return nil }
func (s *ulCapturingStorage) BeginTx(_ context.Context) (interfaces.Transaction, error) {
	return nil, nil
}
func (s *ulCapturingStorage) Ping(_ context.Context) error { return nil }
func (s *ulCapturingStorage) Close() error                 { return nil }

// ulCapturingEventStore captures system events for UL data tests
type ulCapturingEventStore struct {
	mu     sync.Mutex
	events []*models.SystemEvent
}

func (s *ulCapturingEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *ulCapturingEventStore) GetEvents(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*models.SystemEvent, len(s.events))
	copy(result, s.events)
	return result, nil
}

func (s *ulCapturingEventStore) CapturedEvents() []*models.SystemEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*models.SystemEvent, len(s.events))
	copy(result, s.events)
	return result
}

// Stub implementations for remaining SystemEventStore methods
func (s *ulCapturingEventStore) GetEventsFiltered(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}
func (s *ulCapturingEventStore) GetActiveAlerts(_ context.Context, _ interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}
func (s *ulCapturingEventStore) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return nil, nil
}
func (s *ulCapturingEventStore) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}
func (s *ulCapturingEventStore) CountEvents(_ context.Context, _ interfaces.SystemEventFilter) (int64, error) {
	return 0, nil
}
func (s *ulCapturingEventStore) CountActiveAlerts(_ context.Context, _ interfaces.AlertFilter) (int64, error) {
	return 0, nil
}

// --- Helper Functions ---

// buildTestULDataPayload creates a valid UL data payload matching tenant_context_test.go pattern
func buildTestULDataPayload(epEui uint64, packetCnt int64, rxTime int64, snr, rssi float64, userData []byte) map[string]interface{} {
	return map[string]interface{}{
		"epEui":       epEui,
		"packetCnt":   packetCnt,
		"rxTime":      rxTime,
		"snr":         snr,
		"rssi":        rssi,
		"userData":    userData,
		"dlOpen":      false,
		"responseExp": false,
	}
}

// buildTestMessage creates a Message struct for handler invocation
func buildTestMessage(opId int64, data map[string]interface{}) *bssci.Message {
	return &bssci.Message{
		Command: mioty.CmdULData,
		OpId:    opId,
		Data:    data,
	}
}

// createTestEndpoint creates a test endpoint model with given EUI and tenant
func createTestEndpoint(epEui uint64, tenantID int64) *models.EndPoint {
	var euiBytes models.EUI
	binary.BigEndian.PutUint64(euiBytes[:], epEui)
	return &models.EndPoint{
		ID:       1001,
		EUI:      euiBytes,
		TenantID: tenantID,
	}
}

// createTestSession creates a test BSSCI session with given BS EUI
func createTestSession(bsEui uint64, tenantID int64, conn net.Conn) *bssci.Session {
	return &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-ul-dedup-session",
			BaseStationEUI:    bsEui,
			Encoding:          "msgpack",
			HandshakeComplete: true,
			ResolvedTenantID:  tenantID,
			DbSessionID:       1,
		},
		Conn: conn,
	}
}

// setupULTestServer creates a test server with all dependencies for UL dedup tests
func setupULTestServer(t *testing.T, msgRepo *ulCapturingMIOTYMessageRepo, epRepo *ulTestEndpointRepo) (*bssci.Server, *ulCapturingEventStore) {
	t.Helper()

	stor := &ulCapturingStorage{miotyMessages: msgRepo, endpointRepo: epRepo}
	eventStore := &ulCapturingEventStore{}

	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)

	server := bssci.NewTestServer(testLogger, stor, eventStore, ulTestTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	dedup := bssci.NewMessageDeduplicator(5 * time.Minute)
	server.SetDeduplicator(dedup)
	ingestSvc := bssciservices.NewUplinkIngestService(
		dedup,
		stor,
		nil,
		nil,
		epRepo,
		nil,
		nil,
		broadcaster,
		nil,
		testLogger,
		ulTestTenantID,
		0,
	)
	server.SetUplinkIngestService(ingestSvc)
	t.Cleanup(func() { dedup.Stop() })

	return server, eventStore
}

// --- Integration Tests ---

// TestULDataDeduplication_FirstReception_PersistsOnce verifies first UL reception
// creates exactly one message row via CreateULDataMessage
func TestULDataDeduplication_FirstReception_PersistsOnce(t *testing.T) {
	t.Parallel()

	// Setup
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn := &ulTestConn{}
	session := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn)
	server.RegisterSession(session)

	// Build UL data payload
	rxTime := time.Now().UnixNano()
	data := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, []byte{0x01, 0x02, 0x03})
	msg := buildTestMessage(1000, data)

	// Execute
	err := server.CallHandleULData(session, msg, data)
	require.NoError(t, err, "handleULData should succeed")

	// Assertions
	assert.Equal(t, 1, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called exactly once")
	assert.Equal(t, 0, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should NOT be called on first reception")

	// Verify captured message fields
	capturedMsgs := msgRepo.GetCapturedMessages()
	require.Len(t, capturedMsgs, 1, "Should have one captured message")
	capturedMsg := capturedMsgs[0]

	assert.Equal(t, mioty.CmdULData, capturedMsg.CommandType, "CommandType should be ulData")
	assert.Equal(t, ulTestEpEui, capturedMsg.EpEui, "EpEui should match")
	assert.Equal(t, uint32(42), capturedMsg.PacketCnt, "PacketCnt should match")
	require.Len(t, capturedMsg.BaseStations, 1, "Should have one base station")
	assert.Equal(t, ulTestBsEui1, capturedMsg.BaseStations[0].BsEui, "BaseStation EUI should match session")
	assert.Equal(t, 15.5, capturedMsg.BaseStations[0].Snr, "SNR should match payload")
	assert.Equal(t, -80.0, capturedMsg.BaseStations[0].Rssi, "RSSI should match payload")

	// Verify Duplicate flag
	require.NotNil(t, capturedMsg.Duplicate, "Duplicate pointer should be non-nil")
	assert.False(t, *capturedMsg.Duplicate, "Duplicate should be false on first reception")
}

// TestULDataDeduplication_DuplicateFromSecondBS_MergesBaseStations verifies that
// duplicate reception from a second BS calls UpdateULDataBaseStations with merged BS array
func TestULDataDeduplication_DuplicateFromSecondBS_MergesBaseStations(t *testing.T) {
	t.Parallel()

	// Setup
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn1 := &ulTestConn{}
	mockConn2 := &ulTestConn{}
	session1 := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn1)
	session1.ID = "test-ul-dedup-session-1"
	session2 := createTestSession(ulTestBsEui2, ulTestTenantID, mockConn2)
	session2.ID = "test-ul-dedup-session-2"
	server.RegisterSession(session1)
	server.RegisterSession(session2)

	rxTime := time.Now().UnixNano()
	userData := []byte{0x01, 0x02, 0x03}

	// First reception from BS1
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, userData)
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session1, msg1, data1)
	require.NoError(t, err, "First handleULData should succeed")

	// Second reception from BS2 with SAME epEui and packetCnt
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 12.0, -85.0, userData)
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session2, msg2, data2)
	require.NoError(t, err, "Second handleULData should succeed (duplicate)")

	// Assertions
	assert.Equal(t, 1, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called exactly once (first reception)")
	assert.Equal(t, 1, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should be called once (duplicate)")

	// Verify update params
	updateParams := msgRepo.GetUpdateParams()
	require.Len(t, updateParams, 1, "Should have one update call")
	assert.Equal(t, ulTestTenantID, updateParams[0].tenantID, "Update tenantID should match")
	assert.Equal(t, ulTestEpEui, updateParams[0].epEui, "Update epEui should match")
	assert.Equal(t, uint32(42), updateParams[0].packetCnt, "Update packetCnt should match")

	// Parse baseStationsJSON to verify both BS entries
	var baseStations []mioty.BaseStationReception
	err = json.Unmarshal(updateParams[0].baseStationsJSON, &baseStations)
	require.NoError(t, err, "Should be able to parse baseStationsJSON")
	require.Len(t, baseStations, 2, "Should have two base stations in merged array")

	// Verify both BS EUIs are present
	bsEuis := make(map[uint64]bool)
	for _, bs := range baseStations {
		bsEuis[bs.BsEui] = true
	}
	assert.True(t, bsEuis[ulTestBsEui1], "BS1 should be in baseStations")
	assert.True(t, bsEuis[ulTestBsEui2], "BS2 should be in baseStations")
}

// TestULDataDeduplication_DifferentPacketCnt_CreatesTwoMessages verifies that
// different packet counters create separate message rows
func TestULDataDeduplication_DifferentPacketCnt_CreatesTwoMessages(t *testing.T) {
	t.Parallel()

	// Setup
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn := &ulTestConn{}
	session := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn)
	server.RegisterSession(session)

	rxTime := time.Now().UnixNano()

	// First UL with packetCnt=42
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, []byte{0x01})
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session, msg1, data1)
	require.NoError(t, err)

	// Second UL with packetCnt=43 (different)
	data2 := buildTestULDataPayload(ulTestEpEui, 43, rxTime+1000, 15.5, -80.0, []byte{0x02})
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session, msg2, data2)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 2, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called twice")
	assert.Equal(t, 0, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should NOT be called")

	// Verify distinct PacketCnt values
	capturedMsgs := msgRepo.GetCapturedMessages()
	require.Len(t, capturedMsgs, 2)
	assert.Equal(t, uint32(42), capturedMsgs[0].PacketCnt)
	assert.Equal(t, uint32(43), capturedMsgs[1].PacketCnt)
}

// TestULDataDeduplication_SameBSTwice_StillOneMessage verifies that
// same BS sending duplicate doesn't create second row, but triggers update
func TestULDataDeduplication_SameBSTwice_StillOneMessage(t *testing.T) {
	t.Parallel()

	// Setup
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn := &ulTestConn{}
	session := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn)
	server.RegisterSession(session)

	rxTime := time.Now().UnixNano()
	userData := []byte{0x01, 0x02, 0x03}

	// First reception from BS1
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, userData)
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session, msg1, data1)
	require.NoError(t, err)

	// Second reception from SAME BS1 with same packetCnt
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 16.0, -79.0, userData)
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session, msg2, data2)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 1, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called once")
	assert.Equal(t, 1, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should be called once (duplicate)")

	// Parse baseStationsJSON - should still have only 1 BS entry
	updateParams := msgRepo.GetUpdateParams()
	require.Len(t, updateParams, 1)

	var baseStations []mioty.BaseStationReception
	err = json.Unmarshal(updateParams[0].baseStationsJSON, &baseStations)
	require.NoError(t, err)
	assert.Len(t, baseStations, 1, "Should have only one base station (same BS, updated values)")
	assert.Equal(t, ulTestBsEui1, baseStations[0].BsEui, "BS EUI should match")
}

// TestULDataDeduplication_ThreeBaseStations_AllTracked verifies that
// 3+ BS receptions are all tracked in the BaseStations array
func TestULDataDeduplication_ThreeBaseStations_AllTracked(t *testing.T) {
	t.Parallel()

	// Setup
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn1, mockConn2, mockConn3 := &ulTestConn{}, &ulTestConn{}, &ulTestConn{}
	session1 := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn1)
	session1.ID = "test-session-1"
	session2 := createTestSession(ulTestBsEui2, ulTestTenantID, mockConn2)
	session2.ID = "test-session-2"
	session3 := createTestSession(ulTestBsEui3, ulTestTenantID, mockConn3)
	session3.ID = "test-session-3"
	server.RegisterSession(session1)
	server.RegisterSession(session2)
	server.RegisterSession(session3)

	rxTime := time.Now().UnixNano()
	userData := []byte{0x01, 0x02, 0x03}

	// Reception from BS1
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, userData)
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session1, msg1, data1)
	require.NoError(t, err)

	// Reception from BS2
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 14.0, -82.0, userData)
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session2, msg2, data2)
	require.NoError(t, err)

	// Reception from BS3
	data3 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+2000, 13.0, -84.0, userData)
	msg3 := buildTestMessage(1002, data3)
	err = server.CallHandleULData(session3, msg3, data3)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 1, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called once")
	assert.Equal(t, 2, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should be called twice (for BS2 and BS3)")

	// Parse the LAST update call - should have all 3 BS entries
	updateParams := msgRepo.GetUpdateParams()
	require.Len(t, updateParams, 2)

	var baseStations []mioty.BaseStationReception
	err = json.Unmarshal(updateParams[1].baseStationsJSON, &baseStations)
	require.NoError(t, err)
	require.Len(t, baseStations, 3, "Should have three base stations in final array")

	// Verify all 3 BS EUIs are present
	bsEuis := make(map[uint64]bool)
	for _, bs := range baseStations {
		bsEuis[bs.BsEui] = true
		assert.NotZero(t, bs.Snr, "SNR should be non-zero")
		assert.NotZero(t, bs.Rssi, "RSSI should be non-zero")
	}
	assert.True(t, bsEuis[ulTestBsEui1], "BS1 should be in baseStations")
	assert.True(t, bsEuis[ulTestBsEui2], "BS2 should be in baseStations")
	assert.True(t, bsEuis[ulTestBsEui3], "BS3 should be in baseStations")
}

// TestULDataDeduplication_HashCollision_ReturnsError verifies that
// different userData with same packetCnt returns collision error
func TestULDataDeduplication_HashCollision_ReturnsError(t *testing.T) {
	t.Parallel()

	// Setup
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn := &ulTestConn{}
	session := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn)
	server.RegisterSession(session)

	rxTime := time.Now().UnixNano()

	// First UL with userData=[0x01]
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, []byte{0x01})
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session, msg1, data1)
	require.NoError(t, err, "First handleULData should succeed")

	// Second UL with same packetCnt but DIFFERENT userData=[0xFF]
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 15.5, -80.0, []byte{0xFF})
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session, msg2, data2)
	require.NoError(t, err, "handleULData should not propagate collision error; it writes a wire error response")

	// Collision triggers an error response written to the session, but persistence is skipped
	assert.GreaterOrEqual(t, mockConn.SentCount(), 2, "At least one response written for each handleULData call")
	assert.Equal(t, 1, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should only be called once")
}

// TestULDataDeduplication_OutsideWindow_CreatesNewMessage verifies that
// expired dedup window allows new message with same packetCnt
func TestULDataDeduplication_OutsideWindow_CreatesNewMessage(t *testing.T) {
	t.Parallel()

	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	stor := &ulCapturingStorage{miotyMessages: msgRepo, endpointRepo: epRepo}
	eventStore := &ulCapturingEventStore{}
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)
	server := bssci.NewTestServer(testLogger, stor, eventStore, ulTestTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	shortDedup := bssci.NewMessageDeduplicator(1 * time.Millisecond)
	t.Cleanup(func() { shortDedup.Stop() })
	server.SetDeduplicator(shortDedup)
	ingestSvc := bssciservices.NewUplinkIngestService(
		shortDedup, stor, nil, nil, epRepo, nil, nil, broadcaster, nil, testLogger, ulTestTenantID, 0,
	)
	server.SetUplinkIngestService(ingestSvc)

	mockConn := &ulTestConn{}
	session := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn)
	server.RegisterSession(session)

	rxTime := time.Now().UnixNano()
	userData := []byte{0x01, 0x02, 0x03}

	// First UL with packetCnt=42
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, userData)
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session, msg1, data1)
	require.NoError(t, err)

	// Wait for window to expire
	time.Sleep(5 * time.Millisecond)

	// Second UL with same packetCnt=42 (after window expired)
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000000, 15.5, -80.0, userData)
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session, msg2, data2)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 2, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called twice (window expired)")
	assert.Equal(t, 0, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should NOT be called")

	// Both messages should have same PacketCnt
	capturedMsgs := msgRepo.GetCapturedMessages()
	require.Len(t, capturedMsgs, 2)
	assert.Equal(t, uint32(42), capturedMsgs[0].PacketCnt)
	assert.Equal(t, uint32(42), capturedMsgs[1].PacketCnt)
}

// TestULDataDeduplication_EmptyUserData_DeduplicatesCorrectly verifies that
// empty userData (control messages) deduplicate correctly
func TestULDataDeduplication_EmptyUserData_DeduplicatesCorrectly(t *testing.T) {
	t.Parallel()

	// Setup
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn1, mockConn2 := &ulTestConn{}, &ulTestConn{}
	session1 := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn1)
	session1.ID = "test-session-1"
	session2 := createTestSession(ulTestBsEui2, ulTestTenantID, mockConn2)
	session2.ID = "test-session-2"
	server.RegisterSession(session1)
	server.RegisterSession(session2)

	rxTime := time.Now().UnixNano()

	// First UL with empty userData from BS1
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, []byte{})
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session1, msg1, data1)
	require.NoError(t, err)

	// Second UL with empty userData from BS2
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 14.0, -82.0, []byte{})
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session2, msg2, data2)
	require.NoError(t, err)

	// Assertions
	assert.Equal(t, 1, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called once")
	assert.Equal(t, 1, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should be called once (duplicate)")

	// Verify first message has empty userData
	capturedMsgs := msgRepo.GetCapturedMessages()
	require.Len(t, capturedMsgs, 1)
	assert.Empty(t, capturedMsgs[0].UserData, "UserData should be empty")
}

// --- SCACI Broadcast Verification ---

// ulCapturingSCACIBroadcaster captures BroadcastULData calls for verification
type ulCapturingSCACIBroadcaster struct {
	mu        sync.Mutex
	ulCalls   []*ulBroadcastCall
	dlResults []*mioty.DLDataResult
}

type ulBroadcastCall struct {
	tenantID int64
	msg      *mioty.ULDataMessage
}

func (b *ulCapturingSCACIBroadcaster) BroadcastULData(_ context.Context, tenantID int64, data *mioty.ULDataMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Deep copy to avoid mutation
	msgCopy := *data
	if data.BaseStations != nil {
		msgCopy.BaseStations = make([]mioty.BaseStationReception, len(data.BaseStations))
		copy(msgCopy.BaseStations, data.BaseStations)
	}
	if data.Duplicate != nil {
		dup := *data.Duplicate
		msgCopy.Duplicate = &dup
	}
	b.ulCalls = append(b.ulCalls, &ulBroadcastCall{tenantID: tenantID, msg: &msgCopy})
	return nil
}

func (b *ulCapturingSCACIBroadcaster) BroadcastDLDataResult(_ context.Context, _ int64, result *mioty.DLDataResult) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dlResults = append(b.dlResults, result)
	return nil
}

func (b *ulCapturingSCACIBroadcaster) GetULCalls() []*ulBroadcastCall {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]*ulBroadcastCall, len(b.ulCalls))
	copy(result, b.ulCalls)
	return result
}

// setupULTestServerWithBroadcaster creates a test server with a capturing SCACI broadcaster
func setupULTestServerWithBroadcaster(t *testing.T, msgRepo *ulCapturingMIOTYMessageRepo, epRepo *ulTestEndpointRepo) (*bssci.Server, *ulCapturingEventStore, *ulCapturingSCACIBroadcaster) {
	t.Helper()

	stor := &ulCapturingStorage{miotyMessages: msgRepo, endpointRepo: epRepo}
	eventStore := &ulCapturingEventStore{}
	broadcaster := &ulCapturingSCACIBroadcaster{}

	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)

	server := bssci.NewTestServer(testLogger, stor, eventStore, ulTestTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	dedup := bssci.NewMessageDeduplicator(5 * time.Minute)
	server.SetDeduplicator(dedup)
	ingestSvc := bssciservices.NewUplinkIngestService(
		dedup,
		stor,
		nil,
		nil,
		epRepo,
		nil,
		nil,
		broadcaster,
		nil,
		testLogger,
		ulTestTenantID,
		0,
	)
	server.SetUplinkIngestService(ingestSvc)
	t.Cleanup(func() { dedup.Stop() })

	return server, eventStore, broadcaster
}

// TestULDataDeduplication_DuplicateReception_BroadcastsCalled verifies that
// BroadcastULData is called for both first reception and duplicate, with correct BaseStations
func TestULDataDeduplication_DuplicateReception_BroadcastsCalled(t *testing.T) {
	t.Parallel()

	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _, broadcaster := setupULTestServerWithBroadcaster(t, msgRepo, epRepo)

	mockConn1 := &ulTestConn{}
	mockConn2 := &ulTestConn{}
	session1 := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn1)
	session1.ID = "test-broadcast-session-1"
	session2 := createTestSession(ulTestBsEui2, ulTestTenantID, mockConn2)
	session2.ID = "test-broadcast-session-2"
	server.RegisterSession(session1)
	server.RegisterSession(session2)

	rxTime := time.Now().UnixNano()
	userData := []byte{0x01, 0x02, 0x03}

	// First reception from BS1
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, userData)
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session1, msg1, data1)
	require.NoError(t, err, "First handleULData should succeed")

	// Assert broadcast called once with 1 BS and Duplicate=false
	calls := broadcaster.GetULCalls()
	require.Len(t, calls, 1, "BroadcastULData should be called once after first reception")
	require.Len(t, calls[0].msg.BaseStations, 1, "First broadcast should have 1 base station")
	assert.Equal(t, ulTestBsEui1, calls[0].msg.BaseStations[0].BsEui, "First BS EUI should match")
	require.NotNil(t, calls[0].msg.Duplicate, "Duplicate pointer should be non-nil")
	assert.False(t, *calls[0].msg.Duplicate, "First reception should not be marked as duplicate")

	// Second reception from BS2 (duplicate)
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 12.0, -85.0, userData)
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session2, msg2, data2)
	require.NoError(t, err, "Second handleULData should succeed (duplicate)")

	// Assert broadcast called again with 2 BS and Duplicate=true
	calls = broadcaster.GetULCalls()
	require.Len(t, calls, 2, "BroadcastULData should be called twice total")
	require.Len(t, calls[1].msg.BaseStations, 2, "Second broadcast should have 2 base stations")
	require.NotNil(t, calls[1].msg.Duplicate, "Duplicate pointer should be non-nil on second call")
	assert.True(t, *calls[1].msg.Duplicate, "Second reception should be marked as duplicate")

	// Verify both BS EUIs present in second broadcast
	bsEuis := make(map[uint64]bool)
	for _, bs := range calls[1].msg.BaseStations {
		bsEuis[bs.BsEui] = true
	}
	assert.True(t, bsEuis[ulTestBsEui1], "BS1 should be in second broadcast's BaseStations")
	assert.True(t, bsEuis[ulTestBsEui2], "BS2 should be in second broadcast's BaseStations")
}

// TestULDataDeduplication_MissingRxTime_NoStaleState verifies that sending UL data
// without rxTime returns an error AND does not poison the dedup cache
func TestULDataDeduplication_MissingRxTime_NoStaleState(t *testing.T) {
	t.Parallel()

	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn := &ulTestConn{}
	session := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn)
	server.RegisterSession(session)

	// Build UL data payload WITHOUT rxTime (missing mandatory field)
	data := map[string]interface{}{
		"epEui":       ulTestEpEui,
		"packetCnt":   int64(42),
		"snr":         15.5,
		"rssi":        -80.0,
		"userData":    []byte{0x01, 0x02, 0x03},
		"dlOpen":      false,
		"responseExp": false,
	}
	msg := buildTestMessage(1000, data)

	// Execute - should return error due to missing rxTime
	err := server.CallHandleULData(session, msg, data)
	require.NoError(t, err, "handleULData returns nil (error sent via sendError to BS)")

	// No message should be persisted
	assert.Equal(t, 0, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should NOT be called when rxTime is missing")

	// Now send a valid message with the same packetCnt - should be treated as first reception
	// because the invalid packet should NOT have been recorded in dedup
	validData := buildTestULDataPayload(ulTestEpEui, 42, time.Now().UnixNano(), 15.5, -80.0, []byte{0x01, 0x02, 0x03})
	validMsg := buildTestMessage(1001, validData)
	err = server.CallHandleULData(session, validMsg, validData)
	require.NoError(t, err, "Valid handleULData should succeed")

	// The valid message should be persisted as first reception (not duplicate)
	assert.Equal(t, 1, msgRepo.GetCreateULDataCallCount(), "CreateULDataMessage should be called once for the valid message")
	assert.Equal(t, 0, msgRepo.GetUpdateBaseStationsCallCount(), "UpdateULDataBaseStations should NOT be called (first reception)")
}

// TestULDataDeduplication_TenantID_SetCorrectly verifies that UL messages are persisted
// with the correct TenantID from the endpoint's owning tenant. This is a regression test
// for the tenant_id=0 bug fixed at server.go:3716-3717 (BSSCI §5.10.1 fix).
func TestULDataDeduplication_TenantID_SetCorrectly(t *testing.T) {
	t.Parallel()

	// Setup with explicit tenant ID
	const testTenantID = int64(12345) // Different from default to verify it's used
	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, testTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn := &ulTestConn{}
	session := createTestSession(ulTestBsEui1, testTenantID, mockConn)
	server.RegisterSession(session)

	// Build UL data payload
	rxTime := time.Now().UnixNano()
	data := buildTestULDataPayload(ulTestEpEui, 100, rxTime, 15.5, -80.0, []byte{0xAB, 0xCD})
	msg := buildTestMessage(2000, data)

	// Execute
	err := server.CallHandleULData(session, msg, data)
	require.NoError(t, err, "handleULData should succeed")

	// Verify TenantID is set correctly (regression test for tenant_id=0 bug)
	capturedMsgs := msgRepo.GetCapturedMessages()
	require.Len(t, capturedMsgs, 1, "Should have one captured message")
	capturedMsg := capturedMsgs[0]

	// CRITICAL assertion: TenantID must be the endpoint's tenant, not 0
	assert.Equal(t, testTenantID, capturedMsg.TenantID,
		"TenantID must match endpoint's owning tenant (regression test for tenant_id=0 fix)")
	assert.Greater(t, capturedMsg.TenantID, int64(0),
		"TenantID must never be 0 after UL data handling")

	// Also verify other key fields are set correctly
	assert.Equal(t, ulTestEpEui, capturedMsg.EpEui, "EpEui should match")
	assert.Equal(t, uint32(100), capturedMsg.PacketCnt, "PacketCnt should match")
}

// TestULDataDeduplication_DuplicateReception_SendsSingleAckPerCall verifies that each
// handleULData invocation sends exactly one ACK response to its originating BS connection,
// preventing double-ACK regressions in the duplicate path.
func TestULDataDeduplication_DuplicateReception_SendsSingleAckPerCall(t *testing.T) {
	t.Parallel()

	msgRepo := &ulCapturingMIOTYMessageRepo{}
	epRepo := &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, ulTestTenantID),
		},
	}
	server, _ := setupULTestServer(t, msgRepo, epRepo)

	mockConn1 := &ulTestConn{}
	mockConn2 := &ulTestConn{}
	session1 := createTestSession(ulTestBsEui1, ulTestTenantID, mockConn1)
	session1.ID = "test-ack-session-1"
	session2 := createTestSession(ulTestBsEui2, ulTestTenantID, mockConn2)
	session2.ID = "test-ack-session-2"
	server.RegisterSession(session1)
	server.RegisterSession(session2)

	rxTime := time.Now().UnixNano()
	userData := []byte{0x01, 0x02, 0x03}

	// sendMessage writes header + payload = 2 Conn.Write calls per response
	const writesPerResponse = 2

	// First UL from BS1
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, userData)
	msg1 := buildTestMessage(1000, data1)
	err := server.CallHandleULData(session1, msg1, data1)
	require.NoError(t, err, "First handleULData should succeed")

	// BS1 should have received exactly one response (header + payload)
	assert.Equal(t, writesPerResponse, mockConn1.SentCount(),
		"BS1 should receive exactly one response after first reception")

	bs1CountAfterFirst := mockConn1.SentCount()

	// Duplicate UL from BS2 (same epEui + packetCnt)
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 12.0, -85.0, userData)
	msg2 := buildTestMessage(1001, data2)
	err = server.CallHandleULData(session2, msg2, data2)
	require.NoError(t, err, "Duplicate handleULData should succeed")

	// BS2 should have received exactly one response (header + payload)
	assert.Equal(t, writesPerResponse, mockConn2.SentCount(),
		"BS2 should receive exactly one response after duplicate reception")

	// BS1 write count must not have changed after the duplicate path
	assert.Equal(t, bs1CountAfterFirst, mockConn1.SentCount(),
		"BS1 write count must remain unchanged after BS2 duplicate reception")
}
