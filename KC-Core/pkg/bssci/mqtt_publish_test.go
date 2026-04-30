package bssci

import (
	"context"
	"crypto/x509"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/bssci/testutil"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMQTTEventPublisher records calls to MQTTEventPublisher methods.
type mockMQTTEventPublisher struct {
	mu        sync.Mutex
	uplinks   []mockUplinkCall
	attaches  []mockAttachCall
	detaches  []mockDetachCall
	dlResults []mockDLResultCall
	returnErr error
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

type mockAttachCall struct {
	OrgUUID string
	EpEUI   uint64
	BsEUI   uint64
}

type mockDetachCall struct {
	OrgUUID string
	EpEUI   uint64
	BsEUI   uint64
}

type mockDLResultCall struct {
	OrgUUID string
	EpEUI   uint64
	QueID   uint64
	Result  string
}

func (m *mockMQTTEventPublisher) PublishUplink(_ context.Context, orgUUID string, epEUI uint64, bsEUI uint64,
	rssi float64, snr float64, rxTime int64, packetCnt uint32, userData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uplinks = append(m.uplinks, mockUplinkCall{
		OrgUUID: orgUUID, EpEUI: epEUI, BsEUI: bsEUI,
		Rssi: rssi, Snr: snr, RxTime: rxTime, PacketCnt: packetCnt, UserData: userData,
	})
	return m.returnErr
}

func (m *mockMQTTEventPublisher) PublishAttach(_ context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attaches = append(m.attaches, mockAttachCall{OrgUUID: orgUUID, EpEUI: epEUI, BsEUI: bsEUI})
	return m.returnErr
}

func (m *mockMQTTEventPublisher) PublishDetach(_ context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.detaches = append(m.detaches, mockDetachCall{OrgUUID: orgUUID, EpEUI: epEUI, BsEUI: bsEUI})
	return m.returnErr
}

func (m *mockMQTTEventPublisher) PublishDownlinkResult(_ context.Context, orgUUID string, epEUI uint64, queID uint64, result string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dlResults = append(m.dlResults, mockDLResultCall{OrgUUID: orgUUID, EpEUI: epEUI, QueID: queID, Result: result})
	return m.returnErr
}

func (m *mockMQTTEventPublisher) uplinkCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.uplinks)
}

func (m *mockMQTTEventPublisher) attachCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.attaches)
}

func (m *mockMQTTEventPublisher) detachCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.detaches)
}

func (m *mockMQTTEventPublisher) dlResultCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.dlResults)
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

func (m *syncMockMQTTEventPublisher) PublishUplink(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64,
	rssi float64, snr float64, rxTime int64, packetCnt uint32, userData []byte) error {
	defer m.wg.Done()
	return m.mockMQTTEventPublisher.PublishUplink(ctx, orgUUID, epEUI, bsEUI, rssi, snr, rxTime, packetCnt, userData)
}

func (m *syncMockMQTTEventPublisher) PublishAttach(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error {
	defer m.wg.Done()
	return m.mockMQTTEventPublisher.PublishAttach(ctx, orgUUID, epEUI, bsEUI)
}

func (m *syncMockMQTTEventPublisher) PublishDetach(ctx context.Context, orgUUID string, epEUI uint64, bsEUI uint64) error {
	defer m.wg.Done()
	return m.mockMQTTEventPublisher.PublishDetach(ctx, orgUUID, epEUI, bsEUI)
}

func (m *syncMockMQTTEventPublisher) PublishDownlinkResult(ctx context.Context, orgUUID string, epEUI uint64, queID uint64, result string) error {
	defer m.wg.Done()
	return m.mockMQTTEventPublisher.PublishDownlinkResult(ctx, orgUUID, epEUI, queID, result)
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

func (d *mqttTestDownlinkService) EnqueueDownlink(_ context.Context, _ uint64, _ []byte, _ float32, _ int64) (int64, error) {
	return 0, nil
}

func (d *mqttTestDownlinkService) UpdateDownlinkStatus(_ context.Context, _ uint64, _ string, _ string) error {
	return nil
}

func (d *mqttTestDownlinkService) ProcessDLDataResult(_ context.Context, _ *Session, result *mioty.DLDataResult) (map[string]interface{}, error) {
	return map[string]interface{}{
		"command": mioty.CmdDLDataResultResponse,
		"opId":    result.OpId,
	}, nil
}

func (d *mqttTestDownlinkService) ProcessRevokeResponse(_ context.Context, _ *Session, opId int64, _ int64, _ uint64) (map[string]interface{}, error) {
	return map[string]interface{}{
		"command": "dlDataRevRsp",
		"opId":    opId,
	}, nil
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

func TestHandleAttachComplete_NilMQTTPublisher_NoPanic(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 42)

	// mqttPublisher is nil (default)
	session := &Session{
		ID:               "test-nil-mqtt-attach",
		BaseStationEUI:   0xABCD,
		ResolvedTenantID: 42,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
	}

	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   1001,
		OperationType: mioty.CmdAttach,
		Metadata: map[string]interface{}{
			"epEui": int64(0x70B3D59CD00009E6),
		},
	}
	err := server.statusSvc.RecordPendingOperation(context.Background(), session, 1001, pendingOp, 42)
	require.NoError(t, err)

	msg := &Message{Command: mioty.CmdAttachComplete, OpId: 1001}
	assert.NotPanics(t, func() {
		_ = server.CallHandleAttachComplete(session, msg, nil)
	})
}

func TestHandleDetachComplete_NilMQTTPublisher_NoPanic(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 42)

	session := &Session{
		ID:               "test-nil-mqtt-detach",
		BaseStationEUI:   0xABCD,
		ResolvedTenantID: 42,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
	}

	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   2001,
		OperationType: mioty.CmdDetach,
		Metadata: map[string]interface{}{
			"epEui": int64(0x70B3D59CD00009E6),
		},
	}
	err := server.statusSvc.RecordPendingOperation(context.Background(), session, 2001, pendingOp, 42)
	require.NoError(t, err)

	msg := &Message{Command: mioty.CmdDetachComplete, OpId: 2001}
	assert.NotPanics(t, func() {
		_ = server.CallHandleDetachComplete(session, msg, nil)
	})
}

// --- Attach Publish Tests ---

func TestHandleAttachComplete_PublishesMQTTWithOwnerOrg(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 42)

	epOwnerOrg := uuid.New()
	syncMock := newSyncMockMQTTEventPublisher()
	server.SetMQTTPublisher(syncMock)
	server.orgResolver = &mqttTestOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{99: epOwnerOrg},
	}

	session := &Session{
		ID:               "test-attach-mqtt",
		BaseStationEUI:   0xABCDEF1234567890,
		ResolvedTenantID: 42,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
	}

	epEUI := int64(0x70B3D59CD00009E6)
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   1001,
		OperationType: mioty.CmdAttach,
		Metadata: map[string]interface{}{
			"epEui":            epEUI,
			"endpointTenantID": int64(99),
		},
	}
	err := server.statusSvc.RecordPendingOperation(context.Background(), session, 1001, pendingOp, 42)
	require.NoError(t, err)

	syncMock.ExpectCall(1)

	msg := &Message{Command: mioty.CmdAttachComplete, OpId: 1001}
	err = server.CallHandleAttachComplete(session, msg, nil)
	require.NoError(t, err)

	syncMock.Wait()

	assert.Equal(t, 1, syncMock.attachCount())
	syncMock.mu.Lock()
	call := syncMock.attaches[0]
	syncMock.mu.Unlock()
	assert.Equal(t, epOwnerOrg.String(), call.OrgUUID)
	assert.Equal(t, uint64(epEUI), call.EpEUI)
	assert.Equal(t, uint64(0xABCDEF1234567890), call.BsEUI)
}

func TestHandleAttachComplete_OrgUnresolved_SkipsPublish(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 42)

	mock := &mockMQTTEventPublisher{}
	server.SetMQTTPublisher(mock)
	// orgResolver returns uuid.Nil for unknown tenants
	server.orgResolver = &mqttTestOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{},
	}

	session := &Session{
		ID:               "test-attach-no-org",
		BaseStationEUI:   0xABCD,
		ResolvedTenantID: 42,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
	}

	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   1002,
		OperationType: mioty.CmdAttach,
		Metadata: map[string]interface{}{
			"epEui":            int64(0x1234),
			"endpointTenantID": int64(999), // no mapping for this tenant
		},
	}
	err := server.statusSvc.RecordPendingOperation(context.Background(), session, 1002, pendingOp, 42)
	require.NoError(t, err)

	msg := &Message{Command: mioty.CmdAttachComplete, OpId: 1002}
	_ = server.CallHandleAttachComplete(session, msg, nil)

	// Publish should be skipped (org unresolved → uuid.Nil → empty string)
	assert.Equal(t, 0, mock.attachCount())
}

// --- Detach Publish Tests ---

func TestHandleDetachComplete_PublishesMQTTWithTypedMetaOrg(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 42)

	ownerOrg := uuid.New()
	syncMock := newSyncMockMQTTEventPublisher()
	server.SetMQTTPublisher(syncMock)

	session := &Session{
		ID:               "test-detach-mqtt",
		BaseStationEUI:   0xABCDEF1234567890,
		ResolvedTenantID: 42,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
	}

	epEUI := uint64(0x70B3D59CD00009E6)
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   2001,
		OperationType: mioty.CmdDetach,
		Metadata: map[string]interface{}{
			"epEui":   int64(epEUI),
			"orgUuid": ownerOrg.String(),
		},
	}
	err := server.statusSvc.RecordPendingOperation(context.Background(), session, 2001, pendingOp, 42)
	require.NoError(t, err)

	syncMock.ExpectCall(1)

	msg := &Message{Command: mioty.CmdDetachComplete, OpId: 2001}
	err = server.CallHandleDetachComplete(session, msg, nil)
	require.NoError(t, err)

	syncMock.Wait()

	assert.Equal(t, 1, syncMock.detachCount())
	syncMock.mu.Lock()
	call := syncMock.detaches[0]
	syncMock.mu.Unlock()
	assert.Equal(t, ownerOrg.String(), call.OrgUUID)
	assert.Equal(t, epEUI, call.EpEUI)
}

func TestHandleDetachComplete_OrgUnresolved_SkipsPublish(t *testing.T) {
	t.Parallel()
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, 42)

	mock := &mockMQTTEventPublisher{}
	server.SetMQTTPublisher(mock)

	session := &Session{
		ID:               "test-detach-no-org",
		BaseStationEUI:   0xABCD,
		ResolvedTenantID: 42,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
	}

	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   2002,
		OperationType: mioty.CmdDetach,
		Metadata: map[string]interface{}{
			"epEui": int64(0x1234),
			// No orgUuid in metadata
		},
	}
	err := server.statusSvc.RecordPendingOperation(context.Background(), session, 2002, pendingOp, 42)
	require.NoError(t, err)

	msg := &Message{Command: mioty.CmdDetachComplete, OpId: 2002}
	_ = server.CallHandleDetachComplete(session, msg, nil)

	assert.Equal(t, 0, mock.detachCount())
}

// --- Uplink MQTT Publish Tests ---

// mqttNoopIngestService satisfies UplinkIngestService for unit tests.
type mqttNoopIngestService struct{}

func (s *mqttNoopIngestService) Ingest(_ context.Context, _ *UplinkPayload, _ UplinkIngestOptions) (*IngestResult, error) {
	return &IngestResult{}, nil
}

func TestHandleULData_PublishesMQTTUplink(t *testing.T) {
	t.Parallel()

	const tenantID int64 = 42
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, tenantID)

	// Wire deduplicator (required by handleULData)
	dedup := NewMessageDeduplicator(5 * time.Minute)
	defer dedup.Stop()
	server.deduplicator = dedup
	server.uplinkIngestSvc = &mqttNoopIngestService{}

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
		ID:               "test-uldata-mqtt",
		BaseStationEUI:   0xABCDEF1234567890,
		ResolvedTenantID: tenantID,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
		Conn:             &testutil.TestConn{Encoding: "json"},
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

	// Wire deduplicator
	dedup := NewMessageDeduplicator(5 * time.Minute)
	defer dedup.Stop()
	server.uplinkIngestSvc = &mqttNoopIngestService{}
	server.deduplicator = dedup

	// orgResolver returns uuid.Nil for server tenant (no org mapping)
	server.orgResolver = &mqttTestOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{},
	}

	// Wire non-sync mock (no WaitGroup since publish should be skipped)
	mock := &mockMQTTEventPublisher{}
	server.SetMQTTPublisher(mock)

	session := &Session{
		ID:               "test-uldata-no-org",
		BaseStationEUI:   0xABCD,
		ResolvedTenantID: tenantID,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
		Conn:             &testutil.TestConn{Encoding: "json"},
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

	// Publish should be skipped (org unresolved → uuid.Nil → empty string)
	assert.Equal(t, 0, mock.uplinkCount())
}

// --- DL Result MQTT Publish Tests ---

func TestHandleDLDataResult_PublishesMQTTDLResult(t *testing.T) {
	t.Parallel()

	const tenantID int64 = 42
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, tenantID)

	// Wire mock DownlinkService (required by handleDLDataResult)
	server.downlinkSvc = &mqttTestDownlinkService{}

	// Wire tenantResolver to return known tenant for queId
	server.tenantResolver.RegisterQueueTenant(int64(999), strconv.FormatInt(tenantID, 10))

	// Wire orgResolver to map tenant → org
	ownerOrg := uuid.New()
	server.orgResolver = &mqttTestOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{tenantID: ownerOrg},
	}

	// Wire sync MQTT publisher
	syncMock := newSyncMockMQTTEventPublisher()
	server.SetMQTTPublisher(syncMock)
	syncMock.ExpectCall(1)

	// Session with TestConn for sendMessage
	session := &Session{
		ID:               "test-dlresult-mqtt",
		BaseStationEUI:   0xABCDEF1234567890,
		ResolvedTenantID: tenantID,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
		Conn:             &testutil.TestConn{Encoding: "json"},
	}

	// Build DL result data map (handleDLDataResult uses type assertions directly)
	data := map[string]interface{}{
		"epEui":  uint64(0x70B3D59CD00009E6),
		"queId":  uint64(999),
		"result": "sent",
	}

	msg := &Message{Command: mioty.CmdDLDataResult, OpId: 6001}
	err := server.CallHandleDLDataResult(session, msg, data)
	require.NoError(t, err)

	syncMock.Wait()

	assert.Equal(t, 1, syncMock.dlResultCount())
	syncMock.mu.Lock()
	call := syncMock.dlResults[0]
	syncMock.mu.Unlock()
	assert.Equal(t, ownerOrg.String(), call.OrgUUID)
	assert.Equal(t, uint64(0x70B3D59CD00009E6), call.EpEUI)
	assert.Equal(t, uint64(999), call.QueID)
	assert.Equal(t, "sent", call.Result)
}

func TestHandleDLDataResult_OrgUnresolved_SkipsPublish(t *testing.T) {
	t.Parallel()

	const tenantID int64 = 42
	testLogger := logger.NewNop()
	server := NewTestServerWithMemoryStatusService(testLogger, nil, nil, tenantID)

	// Wire mock DownlinkService
	server.downlinkSvc = &mqttTestDownlinkService{}

	// tenantResolver wired but orgResolver returns uuid.Nil
	server.tenantResolver.RegisterQueueTenant(int64(888), strconv.FormatInt(tenantID, 10))
	server.orgResolver = &mqttTestOrgResolver{
		tenantToOrg: map[int64]uuid.UUID{}, // no mapping → uuid.Nil
	}

	// Wire non-sync mock (publish should be skipped)
	mock := &mockMQTTEventPublisher{}
	server.SetMQTTPublisher(mock)

	session := &Session{
		ID:               "test-dlresult-no-org",
		BaseStationEUI:   0xABCD,
		ResolvedTenantID: tenantID,
		DbSessionID:      1,
		Encoding:         EncodingJSON,
		Conn:             &testutil.TestConn{Encoding: "json"},
	}

	data := map[string]interface{}{
		"epEui":  uint64(0x1234),
		"queId":  uint64(888),
		"result": "expired",
	}

	msg := &Message{Command: mioty.CmdDLDataResult, OpId: 6002}
	_ = server.CallHandleDLDataResult(session, msg, data)

	// Publish should be skipped (org unresolved → empty mqttOrgStr)
	assert.Equal(t, 0, mock.dlResultCount())
}
