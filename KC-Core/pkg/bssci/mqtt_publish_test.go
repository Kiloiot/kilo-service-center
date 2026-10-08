package bssci

import (
	"context"
	"crypto/x509"
	"sync"
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noPublishWait is how long assertNoPublish listens before declaring silence.
const noPublishWait = 200 * time.Millisecond

// mockMQTTEventPublisher records calls to MQTTEventPublisher methods.
type mockMQTTEventPublisher struct {
	mu        sync.Mutex
	uplinks   []mockUplinkCall
	returnErr error
	published chan struct{}
}

// signalPublish notifies a waiting assertNoPublish; safe when no channel is set.
func (m *mockMQTTEventPublisher) signalPublish() {
	if m.published != nil {
		select {
		case m.published <- struct{}{}:
		default:
		}
	}
}

// assertNoPublish fails the test when any publish lands within the window.
// The publishes under test run in goroutines, so an immediate counter check
// can pass before the goroutine executes; waiting on the signal channel makes
// the negative assertion deterministic.
func (m *mockMQTTEventPublisher) assertNoPublish(t *testing.T) {
	t.Helper()
	select {
	case <-m.published:
		t.Fatal("unexpected MQTT publish for unresolved organization")
	case <-time.After(noPublishWait):
	}
}

// newSilentMockMQTTEventPublisher builds a mock armed for assertNoPublish.
func newSilentMockMQTTEventPublisher() *mockMQTTEventPublisher {
	return &mockMQTTEventPublisher{published: make(chan struct{}, 16)}
}

type mockUplinkCall struct {
	OrgUUID   string
	EpEUI     uint64
	BsEUI     uint64
	Rssi      float64
	Snr       float64
	RxTime    int64
	PacketCnt uint32
	UserData  []byte
}

func (m *mockMQTTEventPublisher) PublishUplink(_ context.Context, orgUUID string, msg *mioty.ULDataMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uplinks = append(m.uplinks, mockUplinkCall{
		OrgUUID: orgUUID, EpEUI: msg.EpEui, BsEUI: msg.BsEui,
		Rssi: msg.RSSI, Snr: msg.SNR, RxTime: msg.RxTime, PacketCnt: msg.PacketCnt, UserData: msg.UserData,
	})
	m.signalPublish()
	return m.returnErr
}

func (m *mockMQTTEventPublisher) uplinkCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.uplinks)
}

// syncMockMQTTEventPublisher wraps the basic mock with WaitGroup for goroutine tests.
type syncMockMQTTEventPublisher struct {
	*mockMQTTEventPublisher
	wg sync.WaitGroup
}

func newSyncMockMQTTEventPublisher() *syncMockMQTTEventPublisher {
	return &syncMockMQTTEventPublisher{
		mockMQTTEventPublisher: &mockMQTTEventPublisher{},
	}
}

func (m *syncMockMQTTEventPublisher) PublishUplink(ctx context.Context, orgUUID string, msg *mioty.ULDataMessage) error {
	defer m.wg.Done()
	return m.mockMQTTEventPublisher.PublishUplink(ctx, orgUUID, msg)
}

func (m *syncMockMQTTEventPublisher) ExpectCall(n int) {
	m.wg.Add(n)
}

func (m *syncMockMQTTEventPublisher) Wait() {
	m.wg.Wait()
}

// Compile-time interface assertions
var _ MQTTEventPublisher = (*mockMQTTEventPublisher)(nil)
var _ MQTTEventPublisher = (*syncMockMQTTEventPublisher)(nil)

// mqttTestOrgResolver implements org.Resolver for MQTT publish tests.
type mqttTestOrgResolver struct {
	tenantToOrg map[int64]uuid.UUID
	orgToTenant map[uuid.UUID]int64
}

func (f *mqttTestOrgResolver) GetDefaultOrgForTenant(_ context.Context, tenantID int64) (uuid.UUID, error) {
	if orgUUID, ok := f.tenantToOrg[tenantID]; ok {
		return orgUUID, nil
	}
	return uuid.Nil, nil
}

func (f *mqttTestOrgResolver) LookupTenant(_ context.Context, orgUUID uuid.UUID) (int64, error) {
	if tenantID, ok := f.orgToTenant[orgUUID]; ok {
		return tenantID, nil
	}
	return 0, nil
}

func (f *mqttTestOrgResolver) ResolveCert(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, 0, nil
}

// mqttTestDownlinkService implements DownlinkService for MQTT publish tests.
type mqttTestDownlinkService struct{}

func (d *mqttTestDownlinkService) ProcessDLDataResult(_ context.Context, _ *Session, result *mioty.DLDataResult) (map[string]interface{}, error) {
	return map[string]interface{}{
		"command": mioty.CmdDLDataResultResponse,
		"opId":    result.OpId,
	}, nil
}

func (d *mqttTestDownlinkService) ProcessRevokeResponse(_ context.Context, _ *Session, opId int64, _ int64, _ uint64) (map[string]interface{}, bool, error) {
	return map[string]interface{}{
		"command": "dlDataRevRsp",
		"opId":    opId,
	}, true, nil
}

func (d *mqttTestDownlinkService) ProcessQueueAck(context.Context, *Session, QueueAcknowledgement) error {
	return nil
}

func (d *mqttTestDownlinkService) ProcessQueueError(context.Context, *Session, QueueRejection) error {
	return nil
}

func (d *mqttTestDownlinkService) ProcessRevokeRefusal(context.Context, *Session, RevokeRefusal) (bool, error) {
	return true, nil
}

// --- SetMQTTPublisher Tests ---

func TestSetMQTTPublisher_StoresPublisher(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 1)

	mock := &mockMQTTEventPublisher{}
	server.SetMQTTPublisher(mock)

	assert.NotNil(t, server.mqttPublisher)
}

func TestSetMQTTPublisher_NilDoesNotPanic(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 1)

	assert.NotPanics(t, func() {
		server.SetMQTTPublisher(nil)
	})
}

// --- Nil Guard Tests ---

// --- Attach Publish Tests ---

// --- Detach Publish Tests ---

// --- Uplink MQTT Publish Tests ---

// mqttPublishingIngestService drives the same MQTT publish path the real ingest exercises,
// reading orgResolver/mqttPublisher/tenantID from the server. This keeps the test in
// package bssci while avoiding the pkg/bssci ↔ internal/services/bssci import cycle that
// would arise from constructing the real internal ingest here.
type mqttPublishingIngestService struct {
	server *Server
}

func (s *mqttPublishingIngestService) Ingest(ctx context.Context, payload *UplinkPayload, _ UplinkIngestOptions) (*IngestResult, error) {
	var ownerOrgUUID uuid.UUID
	if s.server.orgResolver != nil {
		ownerOrgUUID, _ = s.server.orgResolver.GetDefaultOrgForTenant(ctx, s.server.tenantID)
	}
	if s.server.mqttPublisher != nil && ownerOrgUUID != uuid.Nil {
		_ = s.server.mqttPublisher.PublishUplink(ctx, ownerOrgUUID.String(), &mioty.ULDataMessage{
			EpEui: payload.EpEUI, BsEui: payload.BsEUI, RSSI: payload.RSSI, SNR: payload.SNR,
			RxTime: payload.RxTime, PacketCnt: payload.PacketCnt, UserData: payload.UserData,
		})
	}
	return &IngestResult{OwnerTenantID: s.server.tenantID, OwnerOrgUUID: ownerOrgUUID}, nil
}

func TestHandleULData_PublishesMQTTUplink(t *testing.T) {
	t.Parallel()

	const tenantID int64 = 42
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, tenantID)

	server.uplinkIngestSvc = &mqttPublishingIngestService{server: server}

	// Wire org resolver to map server tenant → known org UUID
	ownerOrg := uuid.New()
	server.orgResolver = &mqttTestOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{tenantID: ownerOrg},
	}

	// Wire sync MQTT publisher
	syncMock := newSyncMockMQTTEventPublisher()
	server.SetMQTTPublisher(syncMock)
	syncMock.ExpectCall(1)

	// Create session with TestConn to absorb sendMessage writes
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-uldata-mqtt",
			BaseStationEUI:   0xABCDEF1234567890,
			ResolvedTenantID: tenantID,
			DbSessionID:      1,
			Encoding:         EncodingJSON,
		},
		Conn: &bsscitest.TestConn{Encoding: "json"},
	}

	// Build UL data map with required fields
	epEUI := int64(0x70B3D59CD00009E6)
	rxTime := time.Now().UnixNano()
	data := map[string]interface{}{
		"epEui":     epEUI,
		"packetCnt": int64(100),
		"userData":  []byte{0x01, 0x02, 0x03},
		"rssi":      float64(-85.5),
		"snr":       float64(12.3),
		"rxTime":    rxTime,
	}

	msg := &Message{Command: mioty.CmdULData, OpId: 5001}
	err := server.CallHandleULData(session, msg, data)
	require.NoError(t, err)

	syncMock.Wait()

	assert.Equal(t, 1, syncMock.uplinkCount())
	syncMock.mu.Lock()
	call := syncMock.uplinks[0]
	syncMock.mu.Unlock()
	assert.Equal(t, ownerOrg.String(), call.OrgUUID)
	assert.Equal(t, uint64(epEUI), call.EpEUI)
	assert.Equal(t, uint64(0xABCDEF1234567890), call.BsEUI)
	assert.Equal(t, float64(-85.5), call.Rssi)
	assert.Equal(t, float64(12.3), call.Snr)
	assert.Equal(t, rxTime, call.RxTime)
	assert.Equal(t, uint32(100), call.PacketCnt)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, call.UserData)
}

func TestHandleULData_OrgUnresolved_SkipsPublish(t *testing.T) {
	t.Parallel()

	const tenantID int64 = 42
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, tenantID)

	server.uplinkIngestSvc = &mqttPublishingIngestService{server: server}

	// orgResolver returns uuid.Nil for server tenant (no org mapping)
	server.orgResolver = &mqttTestOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{},
	}

	mock := newSilentMockMQTTEventPublisher()
	server.SetMQTTPublisher(mock)

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-uldata-no-org",
			BaseStationEUI:   0xABCD,
			ResolvedTenantID: tenantID,
			DbSessionID:      1,
			Encoding:         EncodingJSON,
		},
		Conn: &bsscitest.TestConn{Encoding: "json"},
	}

	data := map[string]interface{}{
		"epEui":     int64(0x70B3D59CD00009E6),
		"packetCnt": int64(200),
		"userData":  []byte{0xAA},
		"rssi":      float64(-90.0),
		"snr":       float64(8.0),
		"rxTime":    time.Now().UnixNano(),
	}

	msg := &Message{Command: mioty.CmdULData, OpId: 5002}
	_ = server.CallHandleULData(session, msg, data)

	// Publish must be skipped (org unresolved → uuid.Nil)
	mock.assertNoPublish(t)
	assert.Equal(t, 0, mock.uplinkCount())
}
