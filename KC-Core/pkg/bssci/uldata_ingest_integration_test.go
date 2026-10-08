package bssci_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
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
// UL Data ingest integration tests (BSSCI §5.10, SCACI §3.8.1)
// The handler runs the real ingest service against a fake uplink store that
// classifies receptions the way the PostgreSQL classifier does.
// ============================================================================

const (
	ulTestWindow   = 5 * time.Minute
	ulTestTenantID = int64(100)
	ulTestEpEui    = uint64(0x0000000012345678)
	ulTestBsEui1   = uint64(0x0000000087654321)
	ulTestBsEui2   = uint64(0x0000000087654322)
	ulTestBsEui3   = uint64(0x0000000087654323)

	// sendMessage writes each response as one frame
	ulWritesPerResponse = 1

	// ulCounterKeyFormat keys the fake classifier by tenant, endpoint and counter.
	ulCounterKeyFormat = "%d/%s/%d"
)

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

// ulFakeUplinkStore mirrors the classifier contract: the first reception of a
// counter is new, later receptions with the same payload are duplicates, a
// different payload behind the same counter is a collision, and a counter the
// message store cannot hold is refused.
type ulFakeUplinkStore struct {
	mu       sync.Mutex
	requests []models.UplinkPersistRequest
	payloads map[string][]byte
	messages map[string]*mioty.ULDataMessage
}

func newULFakeUplinkStore() *ulFakeUplinkStore {
	return &ulFakeUplinkStore{payloads: map[string][]byte{}, messages: map[string]*mioty.ULDataMessage{}}
}

func ulCounterKey(msg *mioty.ULDataMessage) string {
	return fmt.Sprintf(ulCounterKeyFormat, msg.TenantID, mioty.FormatEUI64(msg.EpEui), msg.PacketCnt)
}

func (s *ulFakeUplinkStore) Persist(_ context.Context, req models.UplinkPersistRequest) (models.UplinkPersistOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)
	key := ulCounterKey(req.Message)
	if seen, ok := s.payloads[key]; ok {
		if !bytes.Equal(seen, req.Message.UserData) {
			return models.UplinkPersistOutcome{}, storage.ErrPacketCounterCollision
		}
		first := s.messages[key]
		first.BaseStations = append(first.BaseStations, req.Message.BaseStations...)
		return models.UplinkPersistOutcome{
			Classification: models.UplinkDuplicate,
			MessageID:      first.ID,
			BaseStations:   first.BaseStations,
			DuplicateCount: len(first.BaseStations) - 1,
		}, nil
	}
	s.payloads[key] = append([]byte(nil), req.Message.UserData...)
	stored := *req.Message
	stored.BaseStations = append([]mioty.BaseStationReception(nil), req.Message.BaseStations...)
	s.messages[key] = &stored
	return models.UplinkPersistOutcome{
		Classification: models.UplinkNew,
		MessageID:      stored.ID,
		BaseStations:   stored.BaseStations,
	}, nil
}

func (s *ulFakeUplinkStore) Requests() []models.UplinkPersistRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]models.UplinkPersistRequest(nil), s.requests...)
}

func (s *ulFakeUplinkStore) StoredMessages() []*mioty.ULDataMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*mioty.ULDataMessage, 0, len(s.messages))
	for _, msg := range s.messages {
		out = append(out, msg)
	}
	return out
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

func (r *ulTestEndpointRepo) EndpointRegistrationUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointRegistrationParams) error {
	r.updateFieldsCalls++
	return nil
}

func (r *ulTestEndpointRepo) EndpointAttachmentStateUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointAttachmentStateParams) error {
	r.updateFieldsCalls++
	return nil
}

func (r *ulTestEndpointRepo) EndpointAttachSessionUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointAttachSessionParams) error {
	r.updateFieldsCalls++
	return nil
}

func (r *ulTestEndpointRepo) EndpointDetachStateUpdate(_ context.Context, _ int64, _ int64, _ models.EndpointDetachStateParams) error {
	r.updateFieldsCalls++
	return nil
}

func (r *ulTestEndpointRepo) TransitionEndpointStatus(context.Context, int64, int64, string) (bool, error) {
	return false, nil
}

func (r *ulTestEndpointRepo) RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error) {
	return r.TransitionEndpointStatus(ctx, tenantID, endpointID, status)
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

func (r *ulTestEndpointRepo) UpdateRadioMetricsSelective(context.Context, int64, models.EUI, models.RadioMetricsUpdate) error {
	return nil
}

func (r *ulTestEndpointRepo) GetPreferredBsEui(context.Context, int64, []byte) (*uint64, bool, error) {
	return nil, false, nil
}

func (*ulTestEndpointRepo) RestartPacketCounter(context.Context, int64, int64) error { return nil }

func (*ulTestEndpointRepo) LockAttachCounter(context.Context, int64, int64) (*uint32, error) {
	return nil, nil
}

func (*ulTestEndpointRepo) GetByAttachmentChangedSince(context.Context, int64, string, *time.Time) ([]*models.EndPoint, error) {
	return nil, nil
}

func (r *ulTestEndpointRepo) CreateWithStatus(ctx context.Context, ep *models.EndPoint, status string) error {
	ep.EpStatus = status
	return r.Create(ctx, ep)
}

func (r *ulTestEndpointRepo) DeleteByTenant(context.Context, int64, []byte) (int64, error) {
	return 0, nil
}

func (r *ulTestEndpointRepo) UpdateWithEUI(_ context.Context, _ int64, _ []byte, ep *models.EndPoint) (*models.EndPoint, error) {
	return ep, nil
}

func (r *ulTestEndpointRepo) CheckEUIUnique(_ context.Context, _ []byte) error {
	return nil
}

// ulCapturingStorage implements interfaces.Storage with UL-capturing MIOTYMessages
type ulCapturingStorage struct {
	endpointRepo *ulTestEndpointRepo
}

func (s *ulCapturingStorage) MIOTYMessages() interfaces.MIOTYMessageRepository { return nil }

func (s *ulCapturingStorage) EndPoints() interfaces.EndpointRepository {
	if s.endpointRepo == nil {
		return nil
	}
	return s.endpointRepo
}
func (s *ulCapturingStorage) EndPointSessions() interfaces.EndPointSessionRepository { return nil }
func (s *ulCapturingStorage) BaseStations() interfaces.BaseStationRepository         { return nil }
func (s *ulCapturingStorage) BaseStationSessions() interfaces.BaseStationSessionRepository {
	return nil
}
func (s *ulCapturingStorage) DLRXStatus() interfaces.DLRXStatusRepository        { return nil }
func (s *ulCapturingStorage) MIOTYDownlinks() interfaces.MIOTYDownlinkRepository { return nil }
func (s *ulCapturingStorage) MIOTYBaseStationStatus() interfaces.MIOTYBaseStationStatusRepository {
	return nil
}
func (s *ulCapturingStorage) APIKeys() interfaces.APIKeyRepository             { return nil }
func (s *ulCapturingStorage) DeviceModels() interfaces.DeviceModelRepository   { return nil }
func (s *ulCapturingStorage) Blueprints() interfaces.BlueprintRepository       { return nil }
func (s *ulCapturingStorage) Organizations() interfaces.OrganizationRepository { return nil }
func (s *ulCapturingStorage) GetSqlxDB() *sqlx.DB                              { return nil }
func (s *ulCapturingStorage) SystemEvents() interfaces.SystemEventStore        { return nil }
func (s *ulCapturingStorage) SCACISessions() interfaces.SCACISessionRepository { return nil }
func (s *ulCapturingStorage) SCACIOperations() interfaces.SCACIOperationRepository {
	return nil
}

func (s *ulCapturingStorage) BeginTx(_ context.Context) (bssci.AttachTx, error) {
	return ulAttachTx{endpoints: s.endpointRepo}, nil
}

// ulAttachTx commits an attach or attach propagate against the endpoint fake,
// for an endpoint without an active session.
type ulAttachTx struct {
	endpoints *ulTestEndpointRepo
}

func (t ulAttachTx) EndPoints() interfaces.EndpointRepository { return t.endpoints }
func (ulAttachTx) EndPointSessions() interfaces.EndPointSessionRepository {
	return attPrpNoActiveSession{}
}
func (ulAttachTx) Commit() error                           { return nil }
func (ulAttachTx) Rollback() error                         { return nil }
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

func (s *ulCapturingEventStore) GetEvents(_ context.Context, _ models.SystemEventFilter) ([]*models.SystemEvent, error) {
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
func (s *ulCapturingEventStore) GetEventsFiltered(_ context.Context, _ models.SystemEventFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (s *ulCapturingEventStore) GetActiveAlerts(_ context.Context, _ models.AlertFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (s *ulCapturingEventStore) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return nil, nil
}

func (s *ulCapturingEventStore) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}

func (s *ulCapturingEventStore) CountEvents(_ context.Context, _ models.SystemEventFilter) (int64, error) {
	return 0, nil
}

func (s *ulCapturingEventStore) CountActiveAlerts(_ context.Context, _ models.AlertFilter) (int64, error) {
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
		ID:            1001,
		EUI:           euiBytes,
		TenantID:      tenantID,
		OwnerTenantID: tenantID,
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

// setupULTestServer creates a test server whose ulData handler runs the real
// ingest service against the fake uplink store.
func setupULTestServer(t *testing.T, store *ulFakeUplinkStore, epRepo *ulTestEndpointRepo) (*bssci.Server, *ulCapturingEventStore) {
	t.Helper()

	stor := &ulCapturingStorage{endpointRepo: epRepo}
	eventStore := &ulCapturingEventStore{}

	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(testLogger, eventStore)

	server := bssci.NewTestServer(testLogger, stor, eventStore, ulTestTenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	endpointOwners, err := bssciservices.NewEndpointOwnerResolver(epRepo)
	require.NoError(t, err)
	ingestSvc, err := bssciservices.NewUplinkIngestService(
		store,
		bssciservices.UplinkWindows{Duplicate: ulTestWindow},
		[]models.DeliveryChannel{models.DeliveryChannelSCACI},
		nil, // no DL RX metrics in these scenarios
		nil,
		nil,
		epRepo,
		endpointOwners,
		nil,
		nil,
		ulIgnoredAcks{},
		testLogger,
		ulTestTenantID,
		0,
	)
	require.NoError(t, err)
	server.SetUplinkIngestService(ingestSvc)

	return server, eventStore
}

// ulIgnoredAcks accepts every endpoint acknowledgement; these scenarios carry none.
type ulIgnoredAcks struct{}

func (ulIgnoredAcks) RecordEndpointAck(context.Context, int64, uint64, uint32, string) error {
	return nil
}

func registeredEndpoint(tenantID int64) *ulTestEndpointRepo {
	return &ulTestEndpointRepo{
		endpoints: map[uint64]*models.EndPoint{
			ulTestEpEui: createTestEndpoint(ulTestEpEui, tenantID),
		},
	}
}

func openSession(server *bssci.Server, bsEui uint64, tenantID int64, id string) (*bssci.Session, *ulTestConn) {
	conn := &ulTestConn{}
	session := createTestSession(bsEui, tenantID, conn)
	session.ID = id
	server.RegisterSession(session)
	return session, conn
}

// --- Integration Tests ---

func TestULDataIngest_FirstReception_PersistsOnce(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, conn := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-first")

	rxTime := time.Now().UnixNano()
	data := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, []byte{0x01, 0x02, 0x03})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1000, data), data))

	requests := store.Requests()
	require.Len(t, requests, 1, "one reception is exactly one persist")
	req := requests[0]
	assert.Equal(t, ulTestWindow, req.Window)
	assert.Equal(t, []models.DeliveryChannel{models.DeliveryChannelSCACI}, req.Channels)
	msg := req.Message
	assert.Equal(t, mioty.CmdULData, msg.CommandType)
	assert.Equal(t, ulTestEpEui, msg.EpEui)
	assert.Equal(t, uint32(42), msg.PacketCnt)
	assert.Equal(t, rxTime, msg.RxTime)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, msg.UserData)
	require.Len(t, msg.BaseStations, 1)
	assert.Equal(t, ulTestBsEui1, msg.BaseStations[0].BsEui)
	assert.Equal(t, 15.5, msg.BaseStations[0].Snr)
	assert.Equal(t, -80.0, msg.BaseStations[0].Rssi)
	assert.Equal(t, ulWritesPerResponse, conn.SentCount(), "exactly one response to the base station")
}

func TestULDataIngest_DuplicateFromSecondBS_MergesInStore(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session1, conn1 := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-dup-1")
	session2, conn2 := openSession(server, ulTestBsEui2, ulTestTenantID, "ul-dup-2")

	rxTime := time.Now().UnixNano()
	userData := []byte{0x01, 0x02, 0x03}
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, userData)
	require.NoError(t, server.CallHandleULData(session1, buildTestMessage(1000, data1), data1))
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 12.0, -85.0, userData)
	require.NoError(t, server.CallHandleULData(session2, buildTestMessage(1001, data2), data2))

	require.Len(t, store.Requests(), 2, "every reception reaches the classifier")
	stored := store.StoredMessages()
	require.Len(t, stored, 1, "a duplicate merges into the first message")
	require.Len(t, stored[0].BaseStations, 2)
	assert.Equal(t, ulTestBsEui1, stored[0].BaseStations[0].BsEui)
	assert.Equal(t, ulTestBsEui2, stored[0].BaseStations[1].BsEui)
	assert.Equal(t, ulWritesPerResponse, conn1.SentCount(), "BS1 gets exactly one response")
	assert.Equal(t, ulWritesPerResponse, conn2.SentCount(), "BS2 gets exactly one response for its duplicate")
}

func TestULDataIngest_ThreeBaseStations_AllTracked(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	rxTime := time.Now().UnixNano()
	userData := []byte{0xAA}
	for i, bsEui := range []uint64{ulTestBsEui1, ulTestBsEui2, ulTestBsEui3} {
		session, _ := openSession(server, bsEui, ulTestTenantID, fmt.Sprintf("ul-three-%d", i))
		data := buildTestULDataPayload(ulTestEpEui, 7, rxTime+int64(i), 10.0, -90.0, userData)
		require.NoError(t, server.CallHandleULData(session, buildTestMessage(int64(2000+i), data), data))
	}

	stored := store.StoredMessages()
	require.Len(t, stored, 1)
	require.Len(t, stored[0].BaseStations, 3)
}

func TestULDataIngest_DifferentPacketCnt_CreatesTwoMessages(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, _ := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-two-counters")

	rxTime := time.Now().UnixNano()
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, []byte{0x01})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1000, data1), data1))
	data2 := buildTestULDataPayload(ulTestEpEui, 43, rxTime+1000, 15.5, -80.0, []byte{0x01})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1001, data2), data2))

	assert.Len(t, store.StoredMessages(), 2)
}

func TestULDataIngest_HashCollision_AnswersErrorAndKeepsFirst(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, conn := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-collision")

	rxTime := time.Now().UnixNano()
	data1 := buildTestULDataPayload(ulTestEpEui, 42, rxTime, 15.5, -80.0, []byte{0x01})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1000, data1), data1))
	data2 := buildTestULDataPayload(ulTestEpEui, 42, rxTime+1000, 15.5, -80.0, []byte{0xFF})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1001, data2), data2),
		"a collision is answered on the wire, not propagated")

	assert.Equal(t, 2*ulWritesPerResponse, conn.SentCount(), "an ack for the first and an error for the collision")
	stored := store.StoredMessages()
	require.Len(t, stored, 1)
	assert.Equal(t, []byte{0x01}, stored[0].UserData, "the first payload behind the counter stays authoritative")
}

func TestULDataIngest_EmptyUserData_DeduplicatesCorrectly(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session1, _ := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-empty-1")
	session2, _ := openSession(server, ulTestBsEui2, ulTestTenantID, "ul-empty-2")

	rxTime := time.Now().UnixNano()
	data1 := buildTestULDataPayload(ulTestEpEui, 5, rxTime, 15.5, -80.0, []byte{})
	require.NoError(t, server.CallHandleULData(session1, buildTestMessage(1000, data1), data1))
	data2 := buildTestULDataPayload(ulTestEpEui, 5, rxTime+1, 15.5, -80.0, []byte{})
	require.NoError(t, server.CallHandleULData(session2, buildTestMessage(1001, data2), data2))

	stored := store.StoredMessages()
	require.Len(t, stored, 1)
	assert.Len(t, stored[0].BaseStations, 2)
}

func TestULDataIngest_MissingRxTime_NeverReachesStore(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, _ := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-no-rxtime")

	data := map[string]interface{}{
		"epEui":       ulTestEpEui,
		"packetCnt":   int64(42),
		"snr":         15.5,
		"rssi":        -80.0,
		"userData":    []byte{0x01, 0x02, 0x03},
		"dlOpen":      false,
		"responseExp": false,
	}
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1000, data), data),
		"the validation error goes to the base station, not the caller")
	assert.Empty(t, store.Requests(), "an invalid frame must not touch the classifier")

	valid := buildTestULDataPayload(ulTestEpEui, 42, time.Now().UnixNano(), 15.5, -80.0, []byte{0x01, 0x02, 0x03})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1001, valid), valid))
	require.Len(t, store.Requests(), 1)
	assert.Len(t, store.StoredMessages(), 1, "the valid frame is a first reception")
}

func TestULDataIngest_TenantID_SetFromOwningEndpoint(t *testing.T) {
	t.Parallel()
	const ownerTenantID = int64(12345)
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ownerTenantID))
	session, _ := openSession(server, ulTestBsEui1, ownerTenantID, "ul-tenant")

	data := buildTestULDataPayload(ulTestEpEui, 100, time.Now().UnixNano(), 15.5, -80.0, []byte{0xAB, 0xCD})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(2000, data), data))

	requests := store.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, ownerTenantID, requests[0].Message.TenantID, "the message belongs to the endpoint's tenant")
	assert.Equal(t, uint32(100), requests[0].Message.PacketCnt)
}

func TestULDataIngest_UnknownEndpoint_NeverReachesStore(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, &ulTestEndpointRepo{endpoints: map[uint64]*models.EndPoint{}})
	session, conn := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-unknown")

	data := buildTestULDataPayload(ulTestEpEui, 1, time.Now().UnixNano(), 15.5, -80.0, []byte{0x01})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1000, data), data))

	assert.Empty(t, store.Requests())
	assert.Equal(t, ulWritesPerResponse, conn.SentCount(), "the base station is answered with an error")
}

// openCapturingSession registers a session whose connection decodes every
// frame, so a test can read the error code and message the station received.
func openCapturingSession(server *bssci.Server, bsEui uint64, tenantID int64, id string) (*bssci.Session, *testutil.TestConn) {
	conn := &testutil.TestConn{Encoding: "msgpack"}
	session := createTestSession(bsEui, tenantID, conn)
	session.ID = id
	server.RegisterSession(session)
	return session, conn
}

func TestULDataIngest_StoresTheFrameOpID(t *testing.T) {
	t.Parallel()
	const frameOpID = int64(4711)
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, _ := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-op-id")

	data := buildTestULDataPayload(ulTestEpEui, 9, time.Now().UnixNano(), 15.5, -80.0, []byte{0x09})
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(frameOpID, data), data))

	requests := store.Requests()
	require.Len(t, requests, 1)
	assert.Equal(t, frameOpID, requests[0].Message.OpId, "the stored message keeps the base station's ulData opId")
}

func TestULDataIngest_CounterRefusalsAnswerFromTheCatalog(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		packetCnt int64
		payloads  [][]byte
		posix     int
		token     string
	}{
		{"collision", 42, [][]byte{{0x01}, {0xFF}}, bssci.POSIX_EEXIST, bssci.ErrTokenPacketCounterCollision},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newULFakeUplinkStore()
			server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
			session, conn := openCapturingSession(server, ulTestBsEui1, ulTestTenantID, "ul-refusal-"+tc.name)

			rxTime := time.Now().UnixNano()
			for i, payload := range tc.payloads {
				data := buildTestULDataPayload(ulTestEpEui, tc.packetCnt, rxTime+int64(i), 15.5, -80.0, payload)
				require.NoError(t, server.CallHandleULData(session, buildTestMessage(int64(3000+i), data), data),
					"a refusal is answered on the wire, not propagated")
			}

			code, message := conn.LastError()
			assert.Equal(t, tc.posix, code)
			assert.Equal(t, bssci.ResolveErrorMessage(tc.token), message)
		})
	}
}

func TestULDataIngest_CounterRefusalsCiteTheULDataOperation(t *testing.T) {
	t.Parallel()
	for _, token := range []string{bssci.ErrTokenPacketCounterCollision} {
		assert.Equal(t, "§3.10", bssci.GetErrorDefinition(token).SpecSection, "%s is a ulData refusal (BSSCI §3.10)", token)
	}
}

func TestULDataIngest_FractionalEqSnrIsStoredForEveryConsumer(t *testing.T) {
	t.Parallel()
	const fractionalEqSnr = 7.5
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, _ := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-eq-snr")

	data := buildTestULDataPayload(ulTestEpEui, 7, time.Now().UnixNano(), 15.5, -80.0, []byte{0x07})
	data["eqSnr"] = fractionalEqSnr
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1100, data), data))

	requests := store.Requests()
	require.Len(t, requests, 1)
	msg := requests[0].Message
	require.NotNil(t, msg.EqSnr, "a fractional eqSnr is stored with the message")
	assert.Equal(t, fractionalEqSnr, *msg.EqSnr)
	require.Len(t, msg.BaseStations, 1)
	require.NotNil(t, msg.BaseStations[0].EqSnr, "the reception carries eqSnr into SCACI baseStations[]")
	assert.Equal(t, fractionalEqSnr, *msg.BaseStations[0].EqSnr)
}

// ulTestHardwareSubpackets is the subpackets object shape a field base station
// reports: single-precision measurements and frequencies with sub-Hz fractions.
func ulTestHardwareSubpackets() map[string]interface{} {
	return map[string]interface{}{
		"snr":       []interface{}{29.971159, 29.49518},
		"rssi":      []interface{}{-43.88186, -43.882126},
		"frequency": []interface{}{8.680587252265625e+08, 8.680968111640625e+08},
	}
}

func TestULDataIngest_SubpacketFrequenciesWithSubHertzFractionsAreKept(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, _ := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-subpacket-frequency")

	data := buildTestULDataPayload(ulTestEpEui, 8, time.Now().UnixNano(), 15.5, -80.0, []byte{0x08})
	data["subpackets"] = ulTestHardwareSubpackets()
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1101, data), data))

	requests := store.Requests()
	require.Len(t, requests, 1)
	msg := requests[0].Message
	require.NotNil(t, msg.Subpackets, "the subpackets object is stored")
	assert.Equal(t, []int64{868058725, 868096811}, msg.Subpackets.Frequency,
		"fractional frequencies are kept at the canonical whole-Hz resolution")
	assert.Equal(t, []float64{29.971159, 29.49518}, msg.Subpackets.SNR)
	require.NotNil(t, msg.BaseStations[0].Subpackets)
	assert.Equal(t, msg.Subpackets.Frequency, msg.BaseStations[0].Subpackets.Frequency)
}

func TestULDataIngest_MalformedSubpacketsAreAProtocolError(t *testing.T) {
	t.Parallel()
	withField := func(key string, value interface{}) map[string]interface{} {
		subpackets := ulTestHardwareSubpackets()
		subpackets[key] = value
		return subpackets
	}
	without := func(key string) map[string]interface{} {
		subpackets := ulTestHardwareSubpackets()
		delete(subpackets, key)
		return subpackets
	}
	cases := []struct {
		name       string
		subpackets interface{}
	}{
		{"unequal array lengths", withField("rssi", []interface{}{-43.88186})},
		{"phase of another length", withField("phase", []interface{}{12.5})},
		{"non-numeric element", withField("snr", []interface{}{29.971159, "high"})},
		{"missing frequency", without("frequency")},
		{"empty arrays", map[string]interface{}{"snr": []interface{}{}, "rssi": []interface{}{}, "frequency": []interface{}{}}},
		{"not an object", []interface{}{29.971159}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newULFakeUplinkStore()
			server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
			session, conn := openCapturingSession(server, ulTestBsEui1, ulTestTenantID, "ul-subpackets-"+tc.name)

			data := buildTestULDataPayload(ulTestEpEui, 9, time.Now().UnixNano(), 15.5, -80.0, []byte{0x09})
			data["subpackets"] = tc.subpackets
			require.NoError(t, server.CallHandleULData(session, buildTestMessage(1102, data), data))

			assert.Empty(t, store.Requests(), "a malformed ulData is not stored")
			code, message := conn.LastError()
			assert.Equal(t, bssci.POSIX_EPROTO, code)
			assert.Equal(t, bssci.ResolveErrorMessage(bssci.ErrTokenInvalidSubpackets), message)
		})
	}
}

func TestULDataIngest_PhaseOfTheSameLengthIsAccepted(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	session, _ := openSession(server, ulTestBsEui1, ulTestTenantID, "ul-subpackets-phase")

	subpackets := ulTestHardwareSubpackets()
	subpackets["phase"] = []interface{}{12.5, -170.25}
	data := buildTestULDataPayload(ulTestEpEui, 10, time.Now().UnixNano(), 15.5, -80.0, []byte{0x0A})
	data["subpackets"] = subpackets
	require.NoError(t, server.CallHandleULData(session, buildTestMessage(1103, data), data))

	requests := store.Requests()
	require.Len(t, requests, 1)
	require.NotNil(t, requests[0].Message.Subpackets)
	assert.Equal(t, []float64{12.5, -170.25}, requests[0].Message.Subpackets.Phase)
}

func TestULDataIngest_UserDataBeyondTheRadioLimitIsAProtocolError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		length   int
		accepted bool
	}{
		{"at the limit", mioty.MaxULUserDataBytes, true},
		{"one byte over", mioty.MaxULUserDataBytes + 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newULFakeUplinkStore()
			server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
			session, conn := openCapturingSession(server, ulTestBsEui1, ulTestTenantID, "ul-user-data-"+tc.name)

			data := buildTestULDataPayload(ulTestEpEui, 11, time.Now().UnixNano(), 15.5, -80.0, make([]byte, tc.length))
			require.NoError(t, server.CallHandleULData(session, buildTestMessage(1104, data), data))

			if tc.accepted {
				assert.Len(t, store.Requests(), 1)
				assert.True(t, conn.SeenCommand(mioty.CmdULDataResponse))
				return
			}
			assert.Empty(t, store.Requests(), "an uplink payload no radio can carry is not stored")
			code, message := conn.LastError()
			assert.Equal(t, bssci.POSIX_EPROTO, code)
			assert.Equal(t, bssci.ResolveErrorMessage(bssci.ErrTokenULUserDataTooLong), message)
		})
	}
}
