package bssci

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	require.NoError(t, env.server.handleDetach(env.session, msg, payload))
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"CmdDetachResponse must be sent to the base station")

	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "detach must store pending op for detCmp")
	require.NotNil(t, pending, "pending operation should exist")
	assert.Equal(t, mioty.CmdDetach, pending.OperationType)
	assert.Equal(t, endpoint.ID, pending.Metadata["endpointID"])

	require.NoError(t, env.server.handleDetachComplete(env.session, &Message{
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["eqSnr"] = float64(23.5)
	payload["profile"] = "eu1"

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	t.Logf("Before handleDetach: session.ID=%q, session.DbSessionID=%d, opID=%d",
		env.session.ID, env.session.DbSessionID, opID)

	err := env.server.handleDetach(env.session, msg, payload)

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

// TestDetachTelemetryUpdates verifies the detach-state update receives detach telemetry.
func TestDetachTelemetryUpdates(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(303)
		epEui = uint64(0x001020304050607)
	)

	endpoint := buildTestEndpoint(epEui, 1)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

	require.NotEmpty(t, env.repo.detachStateCalls, "detach must update endpoint telemetry")
	update := env.repo.detachStateCalls[0]
	require.NotNil(t, update.LastDetachTime, "detach must set last_detach_time")
	require.NotNil(t, update.LastDetachPacketCnt, "detach must set last_detach_packet_cnt")
	require.NotNil(t, update.PropagateStatus, "detach must set propagate_status")
	assert.Equal(t, PropagateStatusDetachReceived, *update.PropagateStatus)
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Step 1: Handle detach request
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detach must send detachRsp to base station")

	// Step 2: Complete three-way handshake with detachCmp
	// Audit event is created during handleDetachComplete per server.go:3003
	require.NoError(t, env.server.handleDetachComplete(env.session, &Message{
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
	conn    *bsscitest.TestConn
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
	server.detachValidator = &fakeDetachSignatureValidator{
		endpoints: server.endpointRepo.(*fakeEndpointRepo).endpoints,
	}
	server.SetStorageForTest(storage)
	server.orgResolver = &fakeOrgResolver{
		tenantToOrg: make(map[int64]uuid.UUID),
		orgToTenant: make(map[uuid.UUID]int64),
	}
	// Wire UplinkIngestService for handleULData tests
	server.SetUplinkIngestService(NewMockUplinkIngestSvc())

	if server.endpointRepo == nil {
		t.Fatalf("endpoint repository not configured")
	}

	conn := &bsscitest.TestConn{Encoding: "json"}
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

// detachableEndpointRepo holds one attached endpoint a detach completion can
// detach, so that completion is the one announcing the detachment.
func detachableEndpointRepo(epEui uint64, tenant int64, endpointID int64) *fakeEndpointRepo {
	ep := buildTestEndpoint(epEui, tenant)
	ep.ID = endpointID
	ep.EpStatus = EndpointStatusAttached
	return newFakeEndpointRepo(ep)
}

func uuidBytes() []byte {
	id := uuid.New()
	return id[:]
}

// fakeDetachSignatureValidator is the injected authoritative validator for
// enabled-mode tests: it compares the detach signature against the seeded
// endpoint's stored attach signature and returns owner tenant metadata.
type fakeDetachSignatureValidator struct {
	endpoints map[uint64]*models.EndPoint
}

func (f *fakeDetachSignatureValidator) ValidateDetachSignature(_ context.Context, epEUI uint64, detachSign []byte) (*DetachValidationResult, error) {
	ep, ok := f.endpoints[epEUI]
	if !ok {
		return nil, ErrDetachValidationEndpointNotFound
	}
	if !bytes.Equal(detachSign, ep.Sign) {
		return &DetachValidationResult{
			Valid:            false,
			TenantID:         ep.TenantID,
			OwnerTenantID:    ep.TenantID,
			ValidationStatus: ValidationStatusInvalidSignature,
		}, ErrDetachSignatureInvalid
	}
	return &DetachValidationResult{
		Valid:            true,
		TenantID:         ep.TenantID,
		OwnerTenantID:    ep.TenantID,
		ValidationStatus: ValidationStatusValidated,
	}, nil
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
	mu                   sync.Mutex
	endpoints            map[uint64]*models.EndPoint
	attachmentStateCalls []models.EndpointAttachmentStateParams
	detachStateCalls     []models.EndpointDetachStateParams
	detachStateTenants   []int64
	radioMetricsTenants  []int64
	radioMetricsUpdates  []models.RadioMetricsUpdate
	// lookupErr, when set, is returned by GetByEUI and Get to simulate a
	// repository failure that must fail the detach closed.
	lookupErr error
	// detachStateErr, when set, is returned by EndpointDetachStateUpdate.
	detachStateErr error
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
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
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if ep, ok := f.endpoints[eui.ToUint64()]; ok {
		clone := *ep
		return &clone, nil
	}
	return nil, storage.ErrNotFound
}

func (f *fakeEndpointRepo) GetByID(_ context.Context, id int64, tenantID int64) (*models.EndPoint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ep := range f.endpoints {
		if ep.ID == id && ep.TenantID == tenantID {
			clone := *ep
			return &clone, nil
		}
	}
	return nil, nil
}

func (f *fakeEndpointRepo) EndpointRegistrationUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointRegistrationParams) error {
	return nil
}

func (f *fakeEndpointRepo) EndpointAttachmentStateUpdate(_ context.Context, _ int64, _ int64, p models.EndpointAttachmentStateParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attachmentStateCalls = append(f.attachmentStateCalls, p)
	return nil
}

func (f *fakeEndpointRepo) EndpointAttachSessionUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointAttachSessionParams) error {
	return nil
}

func (f *fakeEndpointRepo) EndpointDetachStateUpdate(_ context.Context, tenantID int64, _ int64, p models.EndpointDetachStateParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.detachStateCalls = append(f.detachStateCalls, p)
	f.detachStateTenants = append(f.detachStateTenants, tenantID)
	return f.detachStateErr
}

func (f *fakeEndpointRepo) TransitionEndpointStatus(_ context.Context, tenantID, endpointID int64, status string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ep := range f.endpoints {
		if ep.ID == endpointID && ep.TenantID == tenantID {
			changed := ep.EpStatus != status
			ep.EpStatus = status
			return changed, nil
		}
	}
	return false, nil
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

func (f *fakeEndpointRepo) UpdateRadioMetricsSelective(_ context.Context, tenantID int64, _ models.EUI, update models.RadioMetricsUpdate) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.radioMetricsTenants = append(f.radioMetricsTenants, tenantID)
	f.radioMetricsUpdates = append(f.radioMetricsUpdates, update)
	return nil
}

func (f *fakeEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil // No preference in tests
}

func (f *fakeEndpointRepo) DeleteByTenant(context.Context, int64, []byte) (int64, error) {
	return 0, nil
}

func (f *fakeEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (f *fakeEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// stubStorage minimally satisfies interfaces.Storage for these tests.
type stubStorage struct {
	msgRepo *stubMIOTYMessageRepo
}

func newStubStorage() *stubStorage {
	return &stubStorage{
		msgRepo: &stubMIOTYMessageRepo{
			detachs:          make([]*mioty.DetachMessage, 0),
			detachPropagates: make([]*mioty.DetachPropagateMessage, 0),
		},
	}
}

func (s *stubStorage) EndPoints() interfaces.EndpointRepository                     { return nil }
func (s *stubStorage) EndPointSessions() interfaces.EndPointSessionRepository       { return nil }
func (s *stubStorage) BaseStations() interfaces.BaseStationRepository               { return nil }
func (s *stubStorage) BaseStationSessions() interfaces.BaseStationSessionRepository { return nil }
func (s *stubStorage) DLRXStatus() interfaces.DLRXStatusRepository                  { return nil }

func (s *stubStorage) MIOTYMessages() interfaces.MIOTYMessageRepository   { return s.msgRepo }
func (s *stubStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (s *stubStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *stubStorage) APIKeys() interfaces.APIKeyRepository           { return nil }
func (s *stubStorage) DeviceModels() interfaces.DeviceModelRepository { return nil } // Blueprint catalog
func (s *stubStorage) Blueprints() interfaces.BlueprintRepository     { return nil } // Blueprint catalog
func (s *stubStorage) BeginTx(context.Context) (AttachTx, error)      { return nil, nil }
func (s *stubStorage) Ping(context.Context) error                     { return nil }
func (s *stubStorage) Close() error                                   { return nil }

// Additional accessors for Storage interface
func (s *stubStorage) Organizations() interfaces.OrganizationRepository     { return nil }
func (s *stubStorage) GetSqlxDB() *sqlx.DB                                  { return nil }
func (s *stubStorage) SystemEvents() interfaces.SystemEventStore            { return nil }
func (s *stubStorage) SCACISessions() interfaces.SCACISessionRepository     { return nil }
func (s *stubStorage) SCACIOperations() interfaces.SCACIOperationRepository { return nil }

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

func (r *stubMIOTYMessageRepo) ListULDataMessages(context.Context, mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	return nil, 0, nil
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
		DetachSignatureValidationEnabled: detachSigValidationOn, // DET-01: validation enabled
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{10.0, 20.0, 30.0, 40.0} // Matching signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	err := env.server.handleDetach(env.session, msg, payload)

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
		DetachSignatureValidationEnabled: detachSigValidationOn, // DET-01: validation enabled
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{99.0, 88.0, 77.0, 66.0} // Mismatched signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	err := env.server.handleDetach(env.session, msg, payload)

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
		DetachSignatureValidationEnabled: detachSigValidationOff, // DET-01: validation disabled
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{99.0, 88.0, 77.0, 66.0} // Mismatched signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	err := env.server.handleDetach(env.session, msg, payload)

	require.NoError(t, err, "detach must succeed when validation disabled")
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent even with signature mismatch (validation disabled)")

	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op stored when validation disabled")
	require.NotNil(t, pending, "pending op stored when validation disabled")
}

// TestDetach_EnabledValidationFailure_NoPersistenceBeforeReject proves the
// known-endpoint signature check runs before any persistence: a forged detach in
// enabled mode is rejected with no pending operation, no detach message, and no
// telemetry write.
func TestDetach_EnabledValidationFailure_NoPersistenceBeforeReject(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(910)
		epEui = uint64(0x00AABBCCDDEE01)
	)

	endpoint := buildTestEndpoint(epEui, 1)
	endpoint.Sign = []byte{10, 20, 30, 40} // Stored attach signature

	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOn,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{99.0, 88.0, 77.0, 66.0} // Mismatched signature

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.session, msg, payload),
		"handler sends the error frame and returns nil")

	assert.False(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"no detach response on a rejected signature")

	pending, perr := env.server.statusSvc.GetPendingOperation(env.session, opID)
	if perr == nil {
		assert.Nil(t, pending, "no pending operation persisted before a rejected signature")
	}

	msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
	msgRepo.mu.Lock()
	detachCount := len(msgRepo.detachs)
	msgRepo.mu.Unlock()
	assert.Zero(t, detachCount, "no detach message persisted before a rejected signature")

	assert.Empty(t, env.repo.detachStateCalls, "no telemetry write before a rejected signature")
}

// TestDetach_ValidationStatusWrittenToDetachRow proves the resolved validation
// provenance is carried onto the persisted detach message: "validated" for an
// enabled+validated known endpoint, "unverified" when validation is disabled.
func TestDetach_ValidationStatusWrittenToDetachRow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		enabled  bool
		opID     int64
		epEui    uint64
		expected string
	}{
		{"enabled_validated", detachSigValidationOn, 921, 0x00AABBCCDDEE02, ValidationStatusValidated},
		{"disabled_unverified", detachSigValidationOff, 922, 0x00AABBCCDDEE03, ValidationStatusUnverified},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			endpoint := buildTestEndpoint(tc.epEui, 1)
			endpoint.Sign = []byte{10, 20, 30, 40}

			env := newDetachTestEnv(t, &Config{
				DetachSignatureValidationEnabled: tc.enabled,
				MessageEncoding:                  EncodingJSON,
			}, endpoint)

			payload := buildDetachPayload(tc.epEui)
			payload["sign"] = []interface{}{10.0, 20.0, 30.0, 40.0} // Matching signature

			msg := &Message{Command: mioty.CmdDetach, OpId: tc.opID, Data: payload}
			require.NoError(t, env.server.handleDetach(env.session, msg, payload))
			require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse))

			msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
			msgRepo.mu.Lock()
			defer msgRepo.mu.Unlock()
			require.Len(t, msgRepo.detachs, 1, "exactly one detach message must be persisted")
			assert.Equal(t, tc.expected, msgRepo.detachs[0].ValidationStatus,
				"persisted detach message must carry the resolved validation status")
		})
	}
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
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

	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

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
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// DET-02: Session from different tenant (roaming)
	env.session.ResolvedTenantID = sessionTenant

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	err := env.server.handleDetach(env.session, msg, payload)
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

// TestDetachCompleteUsesOwnerTenant verifies the detCmp leg targets the
// endpoint owner recorded in the pending-operation metadata: under roaming the
// serving session belongs to a different tenant, and the owner's endpoint
// state, radio metrics, and system event must land under the owner tenant.
func TestDetachCompleteUsesOwnerTenant(t *testing.T) {
	t.Parallel()

	const (
		opID          = int64(507)
		epEui         = uint64(0x004444555566667)
		ownerTenant   = int64(200) // Endpoint owned by tenant 200
		servingTenant = int64(300) // Serving base station belongs to tenant 300
	)

	endpoint := buildTestEndpoint(epEui, ownerTenant)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)
	env.session.ResolvedTenantID = servingTenant

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

	require.NoError(t, env.server.handleDetachComplete(env.session, &Message{
		Command: mioty.CmdDetachComplete,
		OpId:    opID,
	}, map[string]interface{}{}))

	env.repo.mu.Lock()
	detachTenants := append([]int64(nil), env.repo.detachStateTenants...)
	radioTenants := append([]int64(nil), env.repo.radioMetricsTenants...)
	env.repo.mu.Unlock()

	require.NotEmpty(t, detachTenants, "detach state must be written")
	for _, tid := range detachTenants {
		assert.Equal(t, ownerTenant, tid,
			"every detach state write must target the endpoint owner, not the serving tenant")
	}
	require.NotEmpty(t, radioTenants, "radio metrics must be written on detCmp")
	for _, tid := range radioTenants {
		assert.Equal(t, ownerTenant, tid,
			"radio metrics must target the endpoint owner, not the serving tenant")
	}

	env.events.mu.Lock()
	events := append([]*models.SystemEvent(nil), env.events.created...)
	env.events.mu.Unlock()
	var sawDetached bool
	for _, ev := range events {
		if ev.EventType == models.EventTypeEndpointDetached {
			sawDetached = true
			assert.Equal(t, fmt.Sprintf("%d", ownerTenant), ev.TenantID,
				"the detach event must be filed under the endpoint owner tenant")
		}
	}
	assert.True(t, sawDetached, "an endpoint_detached event must be recorded")
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
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
		"sign": []interface{}{
			json.Number("1"), json.Number("2"),
			json.Number("3"), json.Number("4"),
		}, // Will be normalized to []byte
	}

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

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
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// Force session/base station tenant to differ from endpoint owner to simulate roaming.
	env.server.tenantID = basestationTenant
	env.session.ResolvedTenantID = basestationTenant

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

	// FIX-1: Verify message stored with endpoint owner tenant using Background context
	msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil = unknown endpoint

	// newDetachTestEnv sets session tenant to 1 when endpoint is nil (line 178)
	expectedTenant := int64(1)

	// Detach from unknown EUI
	payload := buildDetachPayload(unknownEpEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Execute detach for unknown endpoint
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))
	env.conn.Mu.Lock()
	commandsSnapshot := append([]map[string]interface{}(nil), env.conn.SentMessages...)
	env.conn.Mu.Unlock()
	require.Truef(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"unknown endpoint detach must return detRsp, commands seen: %#v", commandsSnapshot)

	// FIX-3: Verify message persisted with session tenant
	msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
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

	ctx := testutil.TestContext()
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	// FIX-4: Configure org resolver to fail
	env.server.orgResolver = &fakeOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{}, // Empty - simulates org not found
	}

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// FIX-4: Handler should succeed despite org resolver failure
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

	// Verify message persisted with tenant but nil org UUID
	msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
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
		DetachSignatureValidationEnabled: detachSigValidationOff,
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
	msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
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
// An endpoint deleted while its detach propagate is being sent leaves no row
// to record the propagate on; that is the expected end of a delete, not a
// failure, and the propagate still goes out.
func TestSendDetachPropagate_EndpointDeletedMeanwhile_IsNotAnError(t *testing.T) {
	t.Parallel()

	endpoint := buildTestEndpoint(TestEpEuiTenant01, 100)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)
	env.server.endpointRepo.(*fakeEndpointRepo).detachStateErr = storage.ErrNotFound
	log := newRecordingLogger()
	env.server.logger = log
	env.session.HandshakeComplete = true
	env.session.LastScOpId = -1
	env.session.Bidirectional = true
	env.server.RegisterSession(env.session)

	require.NoError(t, env.server.SendDetachPropagate(env.session.ID, TestEpEuiTenant01))

	assert.Empty(t, entriesWithMessage(log, LogBSSCIFailedToUpdateEndpointWithDetachInfo))
	gone := entriesWithMessage(log, LogBSSCIEndpointNotFoundInDatabaseForDetachPropagate)
	require.Len(t, gone, 1)
	assert.Equal(t, "DEBUG", gone[0].level)
	msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
	msgRepo.mu.Lock()
	defer msgRepo.mu.Unlock()
	assert.Len(t, msgRepo.detachPropagates, 1)
}

func TestSendDetachPropagateUnknownEndpoint(t *testing.T) {
	t.Parallel()

	const (
		unknownEpEui      = TestEpEuiUnknown // Unknown endpoint EUI
		basestationTenant = int64(200)       // Session/base station tenant
	)

	// Setup: NO endpoint registered (nil endpoint = unknown device scenario)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
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
	msgRepo := env.server.protocolMessages.(*stubMIOTYMessageRepo)
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
		DetachSignatureValidationEnabled: detachSigValidationOff, // Not used for unknown endpoints
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
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

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
		DetachSignatureValidationEnabled: detachSigValidationOn, // enabled: the injected validator decides
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
		}, errors.New(errFmtSignatureValidationFailed)
	})
	env.server.SetDetachValidator(mockValidator)

	payload := buildDetachPayload(unknownEpEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Execute detach for unknown endpoint with invalid signature
	require.NoError(t, env.server.handleDetach(env.session, msg, payload),
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
		DetachSignatureValidationEnabled: detachSigValidationOff, // Not used for unknown endpoints
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil = unknown endpoint

	// NO validator configured (env.server.detachValidator is nil)

	payload := buildDetachPayload(unknownEpEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	// Execute detach for unknown endpoint without validator
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

	// Verify detachRsp was sent (validation skipped, logged warning)
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent even without validator (logs warning)")

	// Verify pending operation stored (validation skipped)
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored when validator not configured")
	require.NotNil(t, pending, "pending op must exist when validation skipped")
}

// TestDetachValidatorDecidesForKnownEndpoints verifies the injected validator
// is the single authority in enabled mode: it is called for known endpoints
// too, and its verdict decides the detach.
func TestDetachValidatorDecidesForKnownEndpoints(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(704)
		epEui = uint64(0x006666777788889)
	)

	// Setup: Known endpoint with signature
	endpoint := buildTestEndpoint(epEui, 1)
	endpoint.Sign = []byte{10, 20, 30, 40}

	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOn,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	validatorCalled := false
	mockValidator := NewMockDetachSignatureValidator(func(_ context.Context, gotEUI uint64, gotSign []byte) (*DetachValidationResult, error) {
		validatorCalled = true
		assert.Equal(t, epEui, gotEUI)
		assert.Len(t, gotSign, 4)
		return &DetachValidationResult{
			Valid:            true,
			TenantID:         1,
			OwnerTenantID:    1,
			ValidationStatus: ValidationStatusValidated,
		}, nil
	})
	env.server.SetDetachValidator(mockValidator)

	payload := buildDetachPayload(epEui)
	payload["sign"] = []interface{}{10.0, 20.0, 30.0, 40.0}

	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}
	require.NoError(t, env.server.handleDetach(env.session, msg, payload))

	assert.True(t, validatorCalled, "the injected validator must decide known-endpoint detaches in enabled mode")

	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"detRsp must be sent for known endpoint")

	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "pending op must be stored for known endpoint")
	require.NotNil(t, pending, "pending op must exist for known endpoint")
}

// Error format strings shared by this package's failure paths; verbs are filled at the point of failure.
const (
	errFmtSignatureValidationFailed = "signature validation failed"
)

// fakeRoamingErrService fails DetectAndValidateRoaming to exercise the
// roaming-validation-failure path; the other methods are inert.
type fakeRoamingErrService struct{ detectErr error }

func (f *fakeRoamingErrService) DetectAndValidateRoaming(_ context.Context, _ []byte, servingTenantID int64) (bool, int64, error) {
	return false, servingTenantID, f.detectErr
}

func (f *fakeRoamingErrService) RecordAttach(context.Context, []byte, []byte, int64) error {
	return nil
}

func (f *fakeRoamingErrService) RecordDetach(context.Context, []byte, []byte, int64) error {
	return nil
}

func (f *fakeRoamingErrService) UpdateSessionRoaming(context.Context, int64, []byte, bool, int64) error {
	return nil
}

// TestDetach_OwnerLookupFailure_FailsClosed proves a repository failure during
// owner resolution aborts the detach instead of treating a known endpoint as
// unknown: no pending operation is persisted and no detach response is sent.
func TestDetach_OwnerLookupFailure_FailsClosed(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(707)
		epEui = uint64(0x1234567890A1)
	)

	endpoint := buildTestEndpoint(epEui, 100)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)
	env.repo.lookupErr = errors.New("connection refused")

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	err := env.server.handleDetach(env.session, msg, payload)
	require.NoError(t, err, "handler returns nil after sending the error frame")

	assert.False(t, env.conn.SeenCommand(mioty.CmdDetachResponse),
		"no detach response on a failed owner lookup")
	pending, perr := env.server.statusSvc.GetPendingOperation(env.session, opID)
	if perr == nil {
		assert.Nil(t, pending, "no pending operation persisted on a failed owner lookup")
	}
	assert.Empty(t, env.repo.detachStateCalls, "no telemetry write on a failed owner lookup")
}

// TestDetach_RoamingFailure_OwnerNotReassigned proves that when roaming
// validation fails for a known endpoint detaching through another tenant's
// base station, the detach is still recorded under the true owner tenant and
// never reassigned to the serving tenant.
func TestDetach_RoamingFailure_OwnerNotReassigned(t *testing.T) {
	t.Parallel()

	const (
		opID          = int64(808)
		epEui         = uint64(0x1234567890A2)
		ownerTenant   = int64(100)
		servingTenant = int64(200)
	)

	endpoint := buildTestEndpoint(epEui, ownerTenant)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: detachSigValidationOff,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)
	env.session.ResolvedTenantID = servingTenant
	env.server.SetRoamingService(&fakeRoamingErrService{detectErr: errors.New("roaming backend down")})

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: mioty.CmdDetach, OpId: opID, Data: payload}

	err := env.server.handleDetach(env.session, msg, payload)
	require.NoError(t, err)
	require.True(t, env.conn.SeenCommand(mioty.CmdDetachResponse))

	require.NotEmpty(t, env.repo.detachStateTenants, "a telemetry write must occur for the known endpoint")
	for _, tid := range env.repo.detachStateTenants {
		assert.Equal(t, ownerTenant, tid,
			"detach must be recorded under the owner tenant, never the serving tenant")
	}
}
