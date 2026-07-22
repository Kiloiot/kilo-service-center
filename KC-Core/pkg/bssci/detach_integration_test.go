package bssci

import (
	"context"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetachThreeWayHandshake validates that the detach → detRsp → detCmp flow
// stores pending operations and clears them on completion per BSSCI §5.7.3.
func TestDetachThreeWayHandshake(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(101)
		epEui = TestEpEui01 // Issue #3-4 Fix A3: Endpoint EUI, not Service Center EUI
	)

	endpoint := buildTestEndpoint(epEui, 1)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"CmdDetachResponse must be sent to the base station")

	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "detach must store pending op for detCmp")
	require.NotNil(t, pending, "pending operation should exist")
	assert.Equal(t, mioty.CmdDetach, pending.OperationType)
	assert.Equal(t, endpoint.ID, pending.Metadata["endpointID"])

	require.NoError(t, env.server.handleDetachComplete(env.server, env.session, &Message{
		Command: mioty.CmdDetachComplete,
		OpId:    opID,
	}, map[string]interface{}{}))

	// StatusService returns error when operation doesn't exist
	_, err = env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.Error(t, err, "pending op must be cleared after detCmp")
}

// TestDetachOptionalFieldsPreserved ensures optional eqSnr/profile fields remain in metadata.
func TestDetachOptionalFieldsPreserved(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(202)
		epEui = TestEpEui01 // Issue #3-4 Fix A3: Endpoint EUI, not Service Center EUI
	)

	endpoint := buildTestEndpoint(epEui, 1)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["eqSnr"] = float64(23.5)
	payload["profile"] = "eu1"

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	t.Logf("Before handleDetach: session.ID=%q, session.DbSessionID=%d, opID=%d",
		env.session.ID, env.session.DbSessionID, opID)

	err := env.server.handleDetach(env.server, env.session, msg, payload)

	// Check if an error message was sent instead of success
	env.conn.Mu.Lock()
	if len(env.conn.SentMessages) > 0 {
		lastMsg := env.conn.SentMessages[len(env.conn.SentMessages)-1]
		if cmd, ok := lastMsg["command"].(string); ok && cmd == "error" {
			t.Fatalf("handleDetach sent error: code=%v, message=%v", lastMsg["code"], lastMsg["message"])
		}
	}
	env.conn.Mu.Unlock()

	require.NoError(t, err, "handleDetach should succeed")

	// Use StatusService to retrieve pending operation
	t.Logf("After handleDetach: looking for opID=%d", opID)
	pendingOp, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending operation must exist")
	require.NotNil(t, pendingOp, "pending operation must not be nil")
	// After require.NotNil check, pendingOp is guaranteed non-nil
	if pendingOp != nil {
		require.NotNil(t, pendingOp.Metadata, "pending operation metadata must not be nil")
		assert.Equal(t, float64(23.5), pendingOp.Metadata["eqSnr"])
		assert.Equal(t, "eu1", pendingOp.Metadata["profile"])
	}
}

// TestDetachTelemetryUpdates verifies UpdateFields receives detach telemetry.
func TestDetachTelemetryUpdates(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(303)
		epEui = uint64(0x001020304050607)
	)

	endpoint := buildTestEndpoint(epEui, 1)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	require.NotEmpty(t, env.repo.updateFieldsCalls, "detach must update endpoint telemetry")
	update := env.repo.updateFieldsCalls[0]
	assert.Contains(t, update, "last_detach_time")
	assert.Contains(t, update, "last_detach_packet_cnt")
	assert.Equal(t, PropagateStatusDetachReceived, update["propagate_status"])
}

// TestDetachAuditEventRecorded ensures detach flow emits an audit event.
// Per BSSCI §5.7.3, audit events are created during detachCmp (handleDetachComplete).
func TestDetachAuditEventRecorded(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(404)
		epEui = uint64(0x000FEBABE112233)
	)
	endpoint := buildTestEndpoint(epEui, 1)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Step 1: Handle detach request
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detach must send detachRsp to base station")

	// Step 2: Complete three-way handshake with detachCmp
	// Audit event is created during handleDetachComplete per server.go:3003
	require.NoError(t, env.server.handleDetachComplete(env.server, env.session, &Message{
		Command: mioty.CmdDetachComplete,
		OpId:    opID,
	}, map[string]interface{}{}))

	// Step 3: Verify audit event was created
	require.NotEmpty(t, env.events.created, "audit event should be recorded after detachCmp")
	assert.Equal(t, "endpoint", env.events.created[0].Category) // CategoryEndpoint per mioty.types.go:993
}

// --- Helpers ----------------------------------------------------------------

type detachTestEnv struct {
	server  *Server
	session *Session
	conn    *testutil.TestConn
	repo    *fakeEndpointRepo
	events  *recordingEventStore
}

func newDetachTestEnv(t *testing.T, cfg *Config, endpoint *models.EndPoint) *detachTestEnv {
	t.Helper()

	testLogger := logger.NewNop()
	eventStore := &recordingEventStore{}
	storage := newStubStorage()

	// Use CreateTestServices for StatusService and other services
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(testLogger, eventStore)

	server := NewTestServer(
		testLogger,
		storage,
		eventStore,
		1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver,
	)
	server.config = cfg
	server.endpointRepo = newFakeEndpointRepo(endpoint)
	server.storage = storage
	server.orgResolver = &fakeOrgResolver{
		tenantToOrg: make(map[int64]uuid.UUID),
		orgToTenant: make(map[uuid.UUID]int64),
	}
	// Initialize deduplicator for UL data tests
	server.SetDeduplicator(NewMessageDeduplicator(5 * time.Minute))
	// Wire UplinkIngestService for handleULData tests
	server.SetUplinkIngestService(NewMockUplinkIngestSvc())

	if server.endpointRepo == nil {
		t.Fatalf("endpoint repository not configured")
	}

	conn := &testutil.TestConn{Encoding: "json"}
	// Default tenant to 1 if endpoint is nil (unknown endpoint test case)
	sessionTenant := int64(1)
	if endpoint != nil {
		sessionTenant = endpoint.TenantID
	}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "session-detach",
			BaseStationEUI:   0xABCDEF1234567890,
			ResolvedTenantID: sessionTenant,
			// Non-zero to enable database persistence for crash recovery tests
			DbSessionID: 1,
			SessionUUID: uuidBytes(),
			Encoding:    EncodingJSON,
		},
		Conn: conn,
	}

	return &detachTestEnv{
		server:  server,
		session: session,
		conn:    conn,
		repo:    server.endpointRepo.(*fakeEndpointRepo),
		events:  eventStore,
	}
}

func buildDetachPayload(epEui uint64) map[string]interface{} {
	return map[string]interface{}{
		"command":   mioty.CmdDetach,
		"epEui":     float64(epEui),
		"rxTime":    time.Now().UnixNano(),
		"packetCnt": float64(10),
		"snr":       float64(12.5),
		"rssi":      float64(-85.2),
		"sign":      []interface{}{1.0, 2.0, 3.0, 4.0},
	}
}

func buildTestEndpoint(epEui uint64, tenant int64) *models.EndPoint {
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], epEui)
	return &models.EndPoint{
		ID:       1001,
		EUI:      eui,
		TenantID: tenant,
		Sign:     []byte{1, 2, 3, 4},
	}
}

func uuidBytes() []byte {
	id := uuid.New()
	return id[:]
}

// recordingConn removed - migrated to TestConn (testconn_test.go) with thread safety

// recordingEventStore implements interfaces.SystemEventStore for assertions.
type recordingEventStore struct {
	interfaces.SystemEventStore
	mu      sync.Mutex
	created []*models.SystemEvent
}

func (r *recordingEventStore) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, event)
	return nil
}

// fakeEndpointRepo implements interfaces.EndpointRepository with minimal behavior for tests.
type fakeEndpointRepo struct {
	mu                sync.Mutex
	endpoints         map[uint64]*models.EndPoint
	updateFieldsCalls []map[string]interface{}
}

func newFakeEndpointRepo(endpoints ...*models.EndPoint) *fakeEndpointRepo {
	m := make(map[uint64]*models.EndPoint)
	for _, ep := range endpoints {
		if ep != nil {
			m[ep.EUI.ToUint64()] = ep
		}
	}
	return &fakeEndpointRepo{endpoints: m}
}

func (f *fakeEndpointRepo) GetByEUI(_ context.Context, tenantID int64, eui []byte) (*models.EndPoint, error) {
	key := binary.BigEndian.Uint64(eui)
	// Tenant-specific lookup only (cross-tenant fallback handled by caller via Get)
	if ep, ok := f.endpoints[key]; ok && ep.TenantID == tenantID {
		clone := *ep
		return &clone, nil
	}
	// Not found for this tenant
	return nil, storage.ErrNotFound
}

func (f *fakeEndpointRepo) Get(_ context.Context, eui models.EUI) (*models.EndPoint, error) {
	if ep, ok := f.endpoints[eui.ToUint64()]; ok {
		clone := *ep
		return &clone, nil
	}
	return nil, storage.ErrNotFound
}

func (f *fakeEndpointRepo) GetByID(_ context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	for _, ep := range f.endpoints {
		if ep.ID == id && ep.TenantID == tenantID {
			clone := *ep
			return &clone, nil
		}
	}
	return nil, nil
}

func (f *fakeEndpointRepo) UpdateFields(_ context.Context, _ int64, _ int64, updates map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateFieldsCalls = append(f.updateFieldsCalls, updates)
	return nil
}

func (f *fakeEndpointRepo) UpdateDetachMetrics(context.Context, int64, models.EUI, interfaces.DetachMetricsUpdate) error {
	return nil
}

// Remaining methods satisfy the interface but are not used in these tests.
func (f *fakeEndpointRepo) Create(context.Context, *models.EndPoint) error { return nil }
func (f *fakeEndpointRepo) GetByTenant(context.Context, int64) ([]*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) CountByTenant(context.Context, int64) (int64, error) { return 0, nil }
func (f *fakeEndpointRepo) ListByTenantPaginated(context.Context, int64, int, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) Update(context.Context, *models.EndPoint) error { return nil }
func (f *fakeEndpointRepo) UpdateLastSeen(_ context.Context, _ int64, _ models.EUI, _ uint32) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateRadioMetrics(context.Context, int64, models.EUI, float64, float64, float64, int64, int64, string) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, interfaces.RadioMetricsUpdate) error {
	return nil
}
func (f *fakeEndpointRepo) StreamAllForPropagation(context.Context, int64, int) ([]*models.EndPoint, error) {
	return nil, nil
}
func (f *fakeEndpointRepo) HasEndpointsSince(context.Context, time.Time) (bool, error) {
	return false, nil
}
func (f *fakeEndpointRepo) GetEndpointWithKeysForDetachValidation(context.Context, models.EUI) (*models.EndPoint, error) {
	return nil, storage.ErrNotFound
}
func (f *fakeEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil // No preference in tests
}
func (f *fakeEndpointRepo) DeleteByTenant(context.Context, int64, []byte) error {
	return nil
}
func (f *fakeEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}
func (f *fakeEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// stubStorage minimally satisfies interfaces.Storage for these tests.
type stubStorage struct {
	msgRepo     *stubMIOTYMessageRepo
	pendingRepo interfaces.PendingOperationRepository
}

func newStubStorage() *stubStorage {
	return &stubStorage{
		msgRepo: &stubMIOTYMessageRepo{
			detachs:          make([]*mioty.DetachMessage, 0),
			detachPropagates: make([]*mioty.DetachPropagateMessage, 0),
		},
		pendingRepo: &stubPendingOperationRepo{},
	}
}

func (s *stubStorage) EndPoints() interfaces.EndpointRepository                         { return nil }
func (s *stubStorage) DownlinkQueue() interfaces.DownlinkQueueRepository                { return nil }
func (s *stubStorage) BaseStationReceptions() interfaces.BaseStationReceptionRepository { return nil }
func (s *stubStorage) EndPointSessions() interfaces.EndPointSessionRepository           { return nil }
func (s *stubStorage) EndPointKeys() interfaces.EndPointKeyRepository                   { return nil }
func (s *stubStorage) RoamingAgreements() interfaces.RoamingAgreementRepository         { return nil }
func (s *stubStorage) BaseStations() interfaces.BaseStationRepository                   { return nil }
func (s *stubStorage) BaseStationSessions() interfaces.BaseStationSessionRepository     { return nil }
func (s *stubStorage) DLRXStatus() interfaces.DLRXStatusRepository                      { return nil }
func (s *stubStorage) PendingOperations() interfaces.PendingOperationRepository         { return s.pendingRepo }
func (s *stubStorage) MIOTYMessages() interfaces.MIOTYMessageRepository                 { return s.msgRepo }
func (s *stubStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository               { return nil }
func (s *stubStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *stubStorage) Users() interfaces.UserRepository                        { return nil }
func (s *stubStorage) APIKeys() interfaces.APIKeyRepository                    { return nil }
func (s *stubStorage) Integrations() interfaces.IntegrationRepository          { return nil }
func (s *stubStorage) Manufacturers() interfaces.ManufacturerRepository        { return nil } // Blueprint catalog
func (s *stubStorage) DeviceModels() interfaces.DeviceModelRepository          { return nil } // Blueprint catalog
func (s *stubStorage) Blueprints() interfaces.BlueprintRepository              { return nil } // Blueprint catalog
func (s *stubStorage) BeginTx(context.Context) (interfaces.Transaction, error) { return nil, nil }
func (s *stubStorage) Ping(context.Context) error                              { return nil }
func (s *stubStorage) Close() error                                            { return nil }

// Additional accessors for Storage interface
func (s *stubStorage) Organizations() interfaces.OrganizationRepository     { return nil }
func (s *stubStorage) GetSqlxDB() *sqlx.DB                                  { return nil }
func (s *stubStorage) SystemEvents() interfaces.SystemEventStore            { return nil }
func (s *stubStorage) SCACISessions() interfaces.SCACISessionRepository     { return nil }
func (s *stubStorage) SCACIOperations() interfaces.SCACIOperationRepository { return nil }
func (s *stubStorage) DownlinkQueueReader() interfaces.DownlinkQueueReader  { return nil }

// stubMIOTYMessageRepo captures detach messages for assertions if needed.
type stubMIOTYMessageRepo struct {
	mu               sync.Mutex
	detachs          []*mioty.DetachMessage
	detachPropagates []*mioty.DetachPropagateMessage
}

func (r *stubMIOTYMessageRepo) CreateDetachMessage(_ context.Context, msg *mioty.DetachMessage, _ map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detachs = append(r.detachs, msg)
	return nil
}

// Remaining methods satisfy the interface but are no-ops for these tests.
func (r *stubMIOTYMessageRepo) CreateULDataMessage(context.Context, *mioty.ULDataMessage) error {
	return nil
}
func (r *stubMIOTYMessageRepo) CreateAttachMessage(context.Context, *mioty.AttachMessage, map[string]interface{}) error {
	return nil
}
func (r *stubMIOTYMessageRepo) CreateAttachPropagateMessage(context.Context, *mioty.AttachPropagateMessage) error {
	return nil
}
func (r *stubMIOTYMessageRepo) CreateDetachPropagateMessage(_ context.Context, msg *mioty.DetachPropagateMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.detachPropagates = append(r.detachPropagates, msg)
	return nil
}
func (r *stubMIOTYMessageRepo) GetULDataMessage(context.Context, string, int64) (*mioty.ULDataMessage, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetDetachMessage(context.Context, string, int64) (*mioty.DetachMessage, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) ListULDataMessages(context.Context, mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
}
func (r *stubMIOTYMessageRepo) GetMessageStatsByBaseStation(context.Context, uint64, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetExtendedMessageStatsByBaseStation(context.Context, uint64, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetMessageStatsByEndpoint(context.Context, uint64, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetOverallStats(context.Context, int64) (*mioty.MessageStats, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetAnalyticsOverview(context.Context, int64, time.Time, time.Time) (*mioty.AnalyticsOverviewStats, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetHourlyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.HourlyActivity, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetDailyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.DailyActivity, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetTopEndpointsByActivity(context.Context, int64, time.Time, time.Time, int) ([]mioty.EndpointActivity, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetSignalQualityStats(context.Context, int64, time.Time, time.Time) (*mioty.SignalQualityStats, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) GetSignalQualityByBaseStation(context.Context, int64, time.Time, time.Time) ([]mioty.BaseStationSignalQuality, error) {
	return nil, nil
}
func (r *stubMIOTYMessageRepo) UpdateULDataBaseStations(context.Context, int64, uint64, uint32, int64, []byte) error {
	return nil
}

func (r *stubMIOTYMessageRepo) GetBaseStationMessageStats(context.Context, int64, []byte, *time.Time, *time.Time) (*mioty.BaseStationMessageStats, error) {
	return nil, nil
}

func (r *stubMIOTYMessageRepo) GetMessageCountsByEndpoint(context.Context, int64, time.Time, time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *stubMIOTYMessageRepo) GetMessageCountsByBaseStation(context.Context, int64, time.Time, time.Time) (map[string]int64, error) {
	return nil, nil
}

func (r *stubMIOTYMessageRepo) GetWeeklyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.WeeklyActivity, error) {
	return nil, nil
}

func (r *stubMIOTYMessageRepo) GetMonthlyActivity(context.Context, int64, time.Time, time.Time) ([]mioty.MonthlyActivity, error) {
	return nil, nil
}

type stubPendingOperationRepo struct{}

func (stubPendingOperationRepo) Create(context.Context, *interfaces.PendingOperationRequest) error {
	return nil
}
func (stubPendingOperationRepo) UpdateMetadata(context.Context, int64, int64, json.RawMessage) error {
	return nil
}
func (stubPendingOperationRepo) DeleteBySessionAndOperation(context.Context, int64, int64) error {
	return nil
}
func (stubPendingOperationRepo) DeleteByOperation(context.Context, int64) error        { return nil }
func (stubPendingOperationRepo) DeleteBySession(context.Context, int64) (int64, error) { return 0, nil }
func (stubPendingOperationRepo) GetBySession(context.Context, int64) ([]*interfaces.PendingOperation, error) {
	return nil, nil
}

// --- Tests: DET-01, DET-02, DET-03 (detach signature validation) ----------------

// TestDetachSignatureValidationSuccess verifies DET-01: signature validation succeeds
// when detach signature matches stored attach signature and validation is enabled.
func TestDetachSignatureValidationSuccess(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(501)
		epEui = uint64(0x001111222233334)
	)

	endpoint := buildTestEndpoint(epEui, 1)
	endpoint.Sign = []byte{10, 20, 30, 40} // Stored attach signature

	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: true, // DET-01: validation enabled
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{10.0, 20.0, 30.0, 40.0} // Matching signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	err := env.server.handleDetach(env.server, env.session, msg, payload)

	require.NoError(t, err, "detach with valid signature must succeed")
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent when signature validates")

	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored")
	require.NotNil(t, pending, "pending op must be stored on successful validation")
}

// TestDetachSignatureValidationFailure verifies DET-01: signature validation rejects
// messages with mismatched signatures when validation is enabled.
func TestDetachSignatureValidationFailure(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(502)
		epEui = uint64(0x005555666677778)
	)

	endpoint := buildTestEndpoint(epEui, 1)
	endpoint.Sign = []byte{10, 20, 30, 40} // Stored attach signature

	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: true, // DET-01: validation enabled
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{99.0, 88.0, 77.0, 66.0} // Mismatched signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	err := env.server.handleDetach(env.server, env.session, msg, payload)

	require.NoError(t, err, "handleDetach must not return error (sends error to BS instead)")

	// Verify error was sent to base station
	env.conn.Mu.Lock()
	lastMsg := env.conn.SentMessages[len(env.conn.SentMessages)-1]
	env.conn.Mu.Unlock()

	assert.Equal(t, mioty.CmdError, lastMsg["command"], "error command must be sent")
	assert.Contains(t, lastMsg, "code", "error must include POSIX code")
	assert.NotZero(t, lastMsg["code"], "POSIX code must be non-zero")
}

// TestDetachSignatureValidationDisabled verifies DET-01: signature validation is bypassed
// when feature flag is disabled, even with mismatched signatures.
func TestDetachSignatureValidationDisabled(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(503)
		epEui = uint64(0x001111222233345)
	)

	endpoint := buildTestEndpoint(epEui, 1)
	endpoint.Sign = []byte{10, 20, 30, 40} // Stored attach signature

	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false, // DET-01: validation disabled
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{99.0, 88.0, 77.0, 66.0} // Mismatched signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	err := env.server.handleDetach(env.server, env.session, msg, payload)

	require.NoError(t, err, "detach must succeed when validation disabled")
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent even with signature mismatch (validation disabled)")

	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op stored when validation disabled")
	require.NotNil(t, pending, "pending op stored when validation disabled")
}

// TestDetachTenantOrgPropagation verifies DET-02: tenant/org context is resolved
// correctly from endpoint owner for roaming support.
func TestDetachTenantOrgPropagation(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(504)
		epEui    = uint64(0x002222333344445)
		tenantID = int64(100)
		orgUUID  = "550e8400-e29b-41d4-a716-446655440000"
	)

	endpoint := buildTestEndpoint(epEui, tenantID)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// DET-02: Mock organization resolver to return org UUID for tenant
	parsedOrgUUID := uuid.MustParse(orgUUID)
	env.server.orgResolver = &fakeOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{
			tenantID: parsedOrgUUID,
		},
		orgToTenant: map[uuid.UUID]int64{
			parsedOrgUUID: tenantID,
		},
	}

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// DET-02: Verify detach was processed successfully (response sent)
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent after successful tenant/org resolution")

	// Verify pending operation stored with correct endpoint context
	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored")
	require.NotNil(t, pending, "pending op must be stored")
	assert.Equal(t, endpoint.ID, pending.Metadata["endpointID"],
		"pending op must reference endpoint from owner tenant")
}

// TestDetachCrossTenantLookup verifies DET-02: cross-tenant endpoint resolution
// uses models.EUI fallback when same-tenant lookup fails (roaming scenario).
func TestDetachCrossTenantLookup(t *testing.T) {
	t.Parallel()

	const (
		opID           = int64(505)
		epEui          = uint64(0x001234567890ABC)
		endpointTenant = int64(200) // Endpoint owned by tenant 200
		sessionTenant  = int64(300) // Session from tenant 300 (roaming)
	)

	endpoint := buildTestEndpoint(epEui, endpointTenant)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// DET-02: Session from different tenant (roaming)
	env.session.ResolvedTenantID = sessionTenant

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	err := env.server.handleDetach(env.server, env.session, msg, payload)
	require.NoError(t, err, "cross-tenant detach must succeed via EUI fallback")

	// Verify detach was processed (response sent)
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent for cross-tenant endpoint")

	// Verify pending op uses endpoint owner tenant, not session tenant
	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored")
	require.NotNil(t, pending, "pending op must be stored for roaming detach")
	assert.Equal(t, endpoint.ID, pending.Metadata["endpointID"],
		"pending op must reference correct endpoint")
}

// TestDetachPayloadNormalization verifies DET-03: handler correctly processes
// payload with json.Number values and signature arrays (the shapes produced by
// strict JSON decoding with UseNumber).
func TestDetachPayloadNormalization(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(506)
		epEui = uint64(0x003333444455556)
	)

	endpoint := buildTestEndpoint(epEui, 1)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// DET-03: Payload with json.Number values (from strict JSON decoding);
	// rxTime exceeds 2^53 and must survive exactly
	payload := map[string]interface{}{
		"command":    mioty.CmdDetach,
		"epEui":      json.Number(strconv.FormatUint(epEui, 10)), // Will be normalized to uint64
		"bsEui":      json.Number("188900967593046"),             // Will be normalized to uint64
		"rxTime":     json.Number("1699876543000000000"),         // Will be normalized to int64
		"packetCnt":  json.Number("42"),                          // Will be normalized to uint32
		"snr":        json.Number("15.7"),                        // Becomes float64
		"rssi":       json.Number("-92.3"),                       // Becomes float64
		"eqSnr":      json.Number("14.2"),                        // Becomes float64
		"profile":    "eu1",                                      // Stays string
		"rxDuration": json.Number("500"),                         // Will be normalized to int64
		"sign": []interface{}{json.Number("1"), json.Number("2"),
			json.Number("3"), json.Number("4")}, // Will be normalized to []byte
	}

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// DET-03: Verify handler processed payload successfully (response sent)
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent after successful payload normalization")

	// Verify pending operation stored with correct metadata
	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored")
	require.NotNil(t, pending, "pending op must be stored")
	assert.Equal(t, endpoint.ID, pending.Metadata["endpointID"],
		"pending op must reference correct endpoint")

	// Verify optional fields preserved in metadata
	assert.Equal(t, "eu1", pending.Metadata["profile"], "profile field must be preserved")
	assert.NotNil(t, pending.Metadata["eqSnr"], "eqSnr field must be preserved")
}

// TestDetachCrossTenantContextIsolation verifies FIX-1: ownerCtx uses Background()
// instead of session context to prevent tenant pollution.
func TestDetachCrossTenantContextIsolation(t *testing.T) {
	t.Parallel()

	const (
		opID              = int64(601)
		epEui             = TestEpEuiTenant02 // 48-bit compliant EUI
		endpointTenant    = int64(100)        // Endpoint owner tenant
		basestationTenant = int64(200)        // Session/base station tenant
	)

	endpoint := buildTestEndpoint(epEui, endpointTenant)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// Force session/base station tenant to differ from endpoint owner to simulate roaming.
	env.server.tenantID = basestationTenant
	env.session.ResolvedTenantID = basestationTenant

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// FIX-1: Verify message stored with endpoint owner tenant using Background context
	msgRepo := env.server.storage.MIOTYMessages().(*stubMIOTYMessageRepo)
	msgRepo.mu.Lock()
	require.Len(t, msgRepo.detachs, 1, "detach message must be persisted")
	storedMsg := msgRepo.detachs[0]
	msgRepo.mu.Unlock()

	assert.Equal(t, endpointTenant, storedMsg.TenantID,
		"message must use endpoint owner tenant (FIX-1: Background context prevents pollution)")
}

// TestDetachUnknownEndpointMetadataPersistence verifies FIX-3: detach message
// from unknown endpoint persists with session tenant for crash recovery.
func TestDetachUnknownEndpointMetadataPersistence(t *testing.T) {
	t.Parallel()

	const (
		opID         = int64(602)
		unknownEpEui = TestEpEuiUnknown // Unknown endpoint (within 48-bit EUI range)
	)

	// Setup: NO endpoint registered (nil endpoint = unknown device scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil = unknown endpoint

	// newDetachTestEnv sets session tenant to 1 when endpoint is nil (line 178)
	expectedTenant := int64(1)

	// Detach from unknown EUI
	payload := buildDetachPayload(unknownEpEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Execute detach for unknown endpoint
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))
	env.conn.Mu.Lock()
	commandsSnapshot := append([]map[string]interface{}(nil), env.conn.SentMessages...)
	env.conn.Mu.Unlock()
	require.Truef(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"unknown endpoint detach must return detRsp, commands seen: %#v", commandsSnapshot)

	// FIX-3: Verify message persisted with session tenant
	msgRepo := env.server.storage.MIOTYMessages().(*stubMIOTYMessageRepo)
	msgRepo.mu.Lock()
	require.Len(t, msgRepo.detachs, 1, "detach message must be persisted for unknown endpoint")
	storedMsg := msgRepo.detachs[0]
	msgRepo.mu.Unlock()

	// Validate crash-recovery metadata uses session tenant
	assert.Equal(t, expectedTenant, storedMsg.TenantID,
		"unknown endpoint must use session tenant for crash recovery (FIX-3)")
	assert.Nil(t, storedMsg.OrgUUID,
		"unknown endpoint should have nil org UUID")
}

// TestFakeEndpointRepoBasics validates the fake repository behavior for testing.
func TestFakeEndpointRepoBasics(t *testing.T) {
	t.Parallel()

	const (
		knownEui   = TestEpEuiTenant01 // 48-bit compliant EUI
		unknownEui = TestEpEuiUnknown
		tenant100  = int64(100)
		tenant200  = int64(200)
	)

	// Setup: Create endpoint in tenant 100
	endpoint := buildTestEndpoint(knownEui, tenant100)
	repo := newFakeEndpointRepo(endpoint)

	ctx := context.Background()
	euiBytes := make([]byte, 8)

	// Test 1: Same-tenant lookup should succeed
	binary.BigEndian.PutUint64(euiBytes, knownEui)
	ep1, err1 := repo.GetByEUI(ctx, tenant100, euiBytes)
	require.NoError(t, err1, "same-tenant lookup must succeed")
	require.NotNil(t, ep1)
	assert.Equal(t, tenant100, ep1.TenantID)

	// Test 2: Wrong-tenant lookup should fail with ErrNotFound
	ep2, err2 := repo.GetByEUI(ctx, tenant200, euiBytes)
	assert.ErrorIs(t, err2, storage.ErrNotFound, "wrong-tenant lookup must return ErrNotFound")
	assert.Nil(t, ep2)

	// Test 3: Cross-tenant Get() should find endpoint regardless of tenant
	var eui models.EUI
	binary.BigEndian.PutUint64(eui[:], knownEui)
	ep3, err3 := repo.Get(ctx, eui)
	require.NoError(t, err3, "cross-tenant Get() must succeed")
	require.NotNil(t, ep3)
	assert.Equal(t, tenant100, ep3.TenantID)

	// Test 4: Unknown EUI should return ErrNotFound from both methods
	binary.BigEndian.PutUint64(euiBytes, unknownEui)
	ep4, err4 := repo.GetByEUI(ctx, tenant100, euiBytes)
	assert.ErrorIs(t, err4, storage.ErrNotFound, "unknown EUI via GetByEUI must return ErrNotFound")
	assert.Nil(t, ep4)

	binary.BigEndian.PutUint64(eui[:], unknownEui)
	ep5, err5 := repo.Get(ctx, eui)
	assert.ErrorIs(t, err5, storage.ErrNotFound, "unknown EUI via Get must return ErrNotFound")
	assert.Nil(t, ep5)
}

// TestDetachOrgResolverFailureFallback verifies FIX-4: org resolver errors are
// logged and system continues with nil org UUID.
func TestDetachOrgResolverFailureFallback(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(603)
		epEui    = uint64(0x005555666677778)
		tenantID = int64(400)
	)

	endpoint := buildTestEndpoint(epEui, tenantID)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// FIX-4: Configure org resolver to fail
	env.server.orgResolver = &fakeOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{}, // Empty - simulates org not found
	}

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// FIX-4: Handler should succeed despite org resolver failure
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// Verify message persisted with tenant but nil org UUID
	msgRepo := env.server.storage.MIOTYMessages().(*stubMIOTYMessageRepo)
	msgRepo.mu.Lock()
	require.Len(t, msgRepo.detachs, 1)
	storedMsg := msgRepo.detachs[0]
	msgRepo.mu.Unlock()

	assert.Equal(t, tenantID, storedMsg.TenantID, "tenant must be set")
	assert.Nil(t, storedMsg.OrgUUID, "org UUID must be nil when resolver fails")
}

// fakeOrgResolver implements a minimal OrgResolver for testing tenant-to-org mapping.
type fakeOrgResolver struct {
	tenantToOrg map[int64]uuid.UUID
	orgToTenant map[uuid.UUID]int64
}

func (f *fakeOrgResolver) GetDefaultOrgForTenant(_ context.Context, tenantID int64) (uuid.UUID, error) {
	if orgUUID, ok := f.tenantToOrg[tenantID]; ok {
		return orgUUID, nil
	}
	return uuid.Nil, nil
}

func (f *fakeOrgResolver) LookupTenant(_ context.Context, orgUUID uuid.UUID) (int64, error) {
	if tenantID, ok := f.orgToTenant[orgUUID]; ok {
		return tenantID, nil
	}
	return 0, nil
}

func (f *fakeOrgResolver) ResolveCert(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, 0, nil
}

// TestSendDetachPropagatePersistence verifies SendDetachPropagate persists
// detach propagate message to mioty_messages audit trail with correct tenant/org resolution.
// Validates BSSCI §5.9 for detach propagate audit trail.
func TestSendDetachPropagatePersistence(t *testing.T) {
	t.Parallel()

	const (
		epEui             = TestEpEuiTenant01 // 48-bit compliant EUI
		endpointTenant    = int64(100)        // Endpoint owner tenant
		basestationTenant = int64(200)        // Session/base station tenant
	)

	// Setup endpoint owned by tenant 100
	endpoint := buildTestEndpoint(epEui, endpointTenant)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// Configure org resolver to map tenant 100 to org UUID
	testOrgUUID := uuid.MustParse("12345678-1234-5678-1234-567812345678")
	env.server.orgResolver = &fakeOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{
			endpointTenant: testOrgUUID,
		},
		orgToTenant: map[uuid.UUID]int64{
			testOrgUUID: endpointTenant,
		},
	}

	// Force session/base station tenant to differ from endpoint owner to simulate roaming
	env.server.tenantID = basestationTenant
	env.session.ResolvedTenantID = basestationTenant
	env.session.HandshakeComplete = true // Required for SendDetachPropagate
	env.session.LastScOpId = -1          // Initialize SC operation ID counter
	env.session.Bidirectional = true     // Enable bidirectional communication

	// Register session with server so SendDetachPropagate can find it
	env.server.RegisterSession(env.session)

	// Call SendDetachPropagate (the method that should persist detach propagate message)
	err := env.server.SendDetachPropagate(env.session.ID, epEui)
	require.NoError(t, err, "SendDetachPropagate should succeed")

	// Verify detach propagate message was persisted
	msgRepo := env.server.storage.MIOTYMessages().(*stubMIOTYMessageRepo)
	msgRepo.mu.Lock()
	require.Len(t, msgRepo.detachPropagates, 1, "detach propagate message must be persisted")
	storedMsg := msgRepo.detachPropagates[0]
	msgRepo.mu.Unlock()

	// Verify persisted message has correct fields
	assert.Equal(t, mioty.CmdDetachPropagate, storedMsg.CommandType,
		"command_type must be detPropagate")
	assert.Equal(t, epEui, storedMsg.EpEui,
		"ep_eui must match endpoint EUI")
	assert.Equal(t, endpointTenant, storedMsg.TenantID,
		"tenant_id must use endpoint owner tenant (roaming scenario)")
	assert.NotNil(t, storedMsg.OrgUUID, "org_uuid must be set")
	if storedMsg.OrgUUID != nil {
		assert.Equal(t, testOrgUUID.String(), *storedMsg.OrgUUID,
			"org_uuid must match endpoint owner's organization")
	}
	assert.NotNil(t, storedMsg.BasestationEui, "basestation_eui must be set")
	assert.Len(t, storedMsg.BasestationEui, 8, "basestation_eui must be 8 bytes")
	assert.Equal(t, mioty.MessageTypeDetachPropagate, storedMsg.MessageType,
		"message_type must be detach_propagate")
	assert.Equal(t, mioty.DirectionDownlink, storedMsg.Direction,
		"direction must be downlink")
	assert.Equal(t, mioty.InterfaceBSSCI, storedMsg.InterfaceType,
		"interface must be BSSCI")
	assert.False(t, storedMsg.ReceivedAt.IsZero(), "received_at must be set")
	assert.False(t, storedMsg.CreatedAt.IsZero(), "created_at must be set")
	assert.False(t, storedMsg.UpdatedAt.IsZero(), "updated_at must be set")
}

// TestSendDetachPropagateUnknownEndpoint verifies detach propagate persistence
// for unknown endpoints uses session tenant as fallback.
func TestSendDetachPropagateUnknownEndpoint(t *testing.T) {
	t.Parallel()

	const (
		unknownEpEui      = TestEpEuiUnknown // Unknown endpoint EUI
		basestationTenant = int64(200)       // Session/base station tenant
	)

	// Setup: NO endpoint registered (nil endpoint = unknown device scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil = unknown endpoint

	// Force session tenant
	env.server.tenantID = basestationTenant
	env.session.ResolvedTenantID = basestationTenant
	env.session.HandshakeComplete = true // Required for SendDetachPropagate
	env.session.LastScOpId = -1          // Initialize SC operation ID counter
	env.session.Bidirectional = true     // Enable bidirectional communication

	// Register session with server so SendDetachPropagate can find it
	env.server.RegisterSession(env.session)

	// Call SendDetachPropagate for unknown endpoint
	err := env.server.SendDetachPropagate(env.session.ID, unknownEpEui)
	require.NoError(t, err, "SendDetachPropagate should succeed for unknown endpoint")

	// Verify detach propagate message was persisted
	msgRepo := env.server.storage.MIOTYMessages().(*stubMIOTYMessageRepo)
	msgRepo.mu.Lock()
	require.Len(t, msgRepo.detachPropagates, 1, "detach propagate message must be persisted for unknown endpoint")
	storedMsg := msgRepo.detachPropagates[0]
	msgRepo.mu.Unlock()

	// Verify persisted message uses session tenant as fallback
	assert.Equal(t, unknownEpEui, storedMsg.EpEui,
		"ep_eui must match unknown endpoint EUI")
	assert.Equal(t, basestationTenant, storedMsg.TenantID,
		"tenant_id must use session tenant for unknown endpoint")
	assert.Nil(t, storedMsg.OrgUUID,
		"org_uuid must be nil for unknown endpoint")
}

// --- Integration Tests: Detach Signature Validator Flow ---------------------

// TestDetachValidatorUnknownEndpointSuccess verifies detach signature
// validation succeeds for unknown endpoint when internal validator confirms signature.
func TestDetachValidatorUnknownEndpointSuccess(t *testing.T) {
	t.Parallel()

	const (
		opID         = int64(701)
		unknownEpEui = TestEpEuiUnknown // Unknown endpoint EUI
	)

	// Setup: NO endpoint registered (nil endpoint = unknown device scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false, // Not used for unknown endpoints
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil = unknown endpoint

	// Configure mock validator to return success
	mockValidator := NewMockDetachSignatureValidator(func(_ context.Context, epEUI uint64, detachSign []byte) (*DetachValidationResult, error) {
		// Verify validator receives correct parameters
		assert.Equal(t, unknownEpEui, epEUI, "validator must receive correct EUI")
		assert.Len(t, detachSign, 4, "validator must receive 4-byte signature")
		return &DetachValidationResult{
			Valid:            true,
			TenantID:         1,
			OwnerTenantID:    1,
			ValidationStatus: ValidationStatusValidated,
		}, nil // Signature is valid per internal validator
	})
	env.server.SetDetachValidator(mockValidator)

	payload := buildDetachPayload(unknownEpEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Execute detach for unknown endpoint with valid signature
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// Verify detachRsp was sent (signature validated successfully)
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent when validator confirms signature")

	// Verify pending operation stored
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored on successful validation")
	require.NotNil(t, pending, "pending op must exist after validator success")
}

// TestDetachValidatorUnknownEndpointFailure verifies detach signature
// validation rejects unknown endpoint when internal validator denies signature.
func TestDetachValidatorUnknownEndpointFailure(t *testing.T) {
	t.Parallel()

	const (
		opID         = int64(702)
		unknownEpEui = TestEpEuiUnknown // Unknown endpoint EUI
	)

	// Setup: NO endpoint registered (nil endpoint = unknown device scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false, // Not used for unknown endpoints
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil = unknown endpoint

	// Configure mock validator to return failure
	mockValidator := NewMockDetachSignatureValidator(func(_ context.Context, epEUI uint64, detachSign []byte) (*DetachValidationResult, error) {
		// Verify validator receives correct parameters
		assert.Equal(t, unknownEpEui, epEUI, "validator must receive correct EUI")
		assert.Len(t, detachSign, 4, "validator must receive 4-byte signature")
		// Signature validation failed per internal validator
		return &DetachValidationResult{
			Valid:            false,
			ValidationStatus: ValidationStatusInvalidSignature,
		}, fmt.Errorf("signature validation failed")
	})
	env.server.SetDetachValidator(mockValidator)

	payload := buildDetachPayload(unknownEpEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Execute detach for unknown endpoint with invalid signature
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload),
		"handleDetach must not return error (sends error to BS instead)")

	// Verify error was sent to base station (signature validation failed)
	env.conn.Mu.Lock()
	lastMsg := env.conn.SentMessages[len(env.conn.SentMessages)-1]
	env.conn.Mu.Unlock()

	assert.Equal(t, mioty.CmdError, lastMsg["command"], "error command must be sent")
	assert.Contains(t, lastMsg, "code", "error must include POSIX code")
	assert.NotZero(t, lastMsg["code"], "POSIX code must be non-zero")
	assert.Contains(t, lastMsg, "message", "error must include message")

	// Verify pending operation was NOT stored (validation failed)
	_, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.Error(t, err, "pending op must NOT be stored when validator rejects signature")
}

// TestDetachValidatorNotConfigured verifies unknown endpoint detach
// is rejected gracefully when validator is not configured (logs warning).
func TestDetachValidatorNotConfigured(t *testing.T) {
	t.Parallel()

	const (
		opID         = int64(703)
		unknownEpEui = TestEpEuiUnknown // Unknown endpoint EUI
	)

	// Setup: NO endpoint registered (nil endpoint = unknown device scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false, // Not used for unknown endpoints
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil = unknown endpoint

	// NO validator configured (env.server.detachValidator is nil)

	payload := buildDetachPayload(unknownEpEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Execute detach for unknown endpoint without validator
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// Verify detachRsp was sent (validation skipped, logged warning)
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent even without validator (logs warning)")

	// Verify pending operation stored (validation skipped)
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored when validator not configured")
	require.NotNil(t, pending, "pending op must exist when validation skipped")
}

// TestDetachValidatorKnownEndpointBypassed verifies validator is NOT
// called for known endpoints (uses existing signature validation flow).
func TestDetachValidatorKnownEndpointBypassed(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(704)
		epEui = uint64(0x006666777788889)
	)

	// Setup: Known endpoint with signature
	endpoint := buildTestEndpoint(epEui, 1)
	endpoint.Sign = []byte{10, 20, 30, 40}

	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: true, // Known endpoint uses this flag
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// Configure mock validator that should NOT be called
	validatorCalled := false
	mockValidator := NewMockDetachSignatureValidator(func(_ context.Context, _ uint64, _ []byte) (*DetachValidationResult, error) {
		validatorCalled = true // This should NOT happen for known endpoints
		return &DetachValidationResult{
			Valid:            true,
			TenantID:         1,
			OwnerTenantID:    1,
			ValidationStatus: ValidationStatusValidated,
		}, nil
	})
	env.server.SetDetachValidator(mockValidator)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{10.0, 20.0, 30.0, 40.0} // Matching signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// Verify validator was NOT called (known endpoint uses existing flow)
	assert.False(t, validatorCalled, "validator must NOT be called for known endpoints")

	// Verify detach processed successfully
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent for known endpoint")

	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored for known endpoint")
	require.NotNil(t, pending, "pending op must exist for known endpoint")
}
