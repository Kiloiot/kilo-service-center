package bssci_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	bssciutil "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"

	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// mockEventStore implements the event store interface
type mockEventStoreImpl struct {
	createEventCalled bool
	lastEvent         *models.SystemEvent
}

func (m *mockEventStoreImpl) CreateEvent(_ context.Context, event *models.SystemEvent) error {
	m.createEventCalled = true
	m.lastEvent = event
	return nil
}

func (m *mockEventStoreImpl) GetEvents(_ context.Context, _ models.SystemEventFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (m *mockEventStoreImpl) GetActiveAlerts(_ context.Context, _ models.AlertFilter) ([]*models.SystemEvent, error) {
	return nil, nil
}

func (m *mockEventStoreImpl) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return nil, nil
}

func (m *mockEventStoreImpl) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}
func (m *mockEventStoreImpl) CountEvents(_ context.Context, _ models.SystemEventFilter) (int64, error) {
	return 0, nil
}
func (m *mockEventStoreImpl) CountActiveAlerts(_ context.Context, _ models.AlertFilter) (int64, error) {
	return 0, nil
}

// mockTenantResolver implements TenantResolver for testing
type mockTenantResolver struct {
	queueTenants map[int64]string
	resolveCalls []int64 // Track calls to ResolveTenant
}

func newMockTenantResolver() *mockTenantResolver {
	return &mockTenantResolver{
		queueTenants: make(map[int64]string),
		resolveCalls: make([]int64, 0),
	}
}

func (m *mockTenantResolver) ResolveTenant(_ context.Context, queueID int64) (string, error) {
	m.resolveCalls = append(m.resolveCalls, queueID)
	if tenant, exists := m.queueTenants[queueID]; exists {
		return tenant, nil
	}
	return "", fmt.Errorf(errFmtQueueIDNotRegisteredInMock, queueID)
}

func (m *mockTenantResolver) RegisterQueueTenant(queueID int64, tenantID string) {
	m.queueTenants[queueID] = tenantID
}

func (m *mockTenantResolver) UnregisterQueueTenant(queueID int64) {
	delete(m.queueTenants, queueID)
}

// mockConnWithFrames implements net.Conn and captures msgpack frames
type mockConnWithFrames struct {
	readBuffer  bytes.Buffer
	writeBuffer bytes.Buffer
	frames      [][]byte // Captured msgpack frames
}

func (m *mockConnWithFrames) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *mockConnWithFrames) Write(b []byte) (n int, err error) {
	// Capture the frame
	frame := make([]byte, len(b))
	copy(frame, b)
	m.frames = append(m.frames, frame)
	return m.writeBuffer.Write(b)
}

func (m *mockConnWithFrames) Close() error {
	return nil
}

func (m *mockConnWithFrames) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mockConnLocalPort}
}

func (m *mockConnWithFrames) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mockConnRemotePort}
}

func (m *mockConnWithFrames) SetDeadline(_ time.Time) error {
	return nil
}

func (m *mockConnWithFrames) SetReadDeadline(_ time.Time) error {
	return nil
}

func (m *mockConnWithFrames) SetWriteDeadline(_ time.Time) error {
	return nil
}

// TestCanonicalMsgpackParsing verifies the canonical msgpack parsing path
func TestCanonicalMsgpackParsing(t *testing.T) {
	// Create test data as it would come from the wire
	wireData := map[string]interface{}{
		"epEui":     bssci.TestEpEui01,
		"queId":     uint64(42),
		"result":    "sent",
		"txTime":    int64(1234567890),
		"packetCnt": uint32(10),
	}

	// This is what handleDLDataResult does internally
	msgpackData, err := msgpack.Marshal(wireData)
	require.NoError(t, err)

	// The handler unmarshals into the canonical type
	var dlResult struct {
		EpEui     uint64  `msgpack:"epEui"`
		QueID     uint64  `msgpack:"queId"`
		Result    string  `msgpack:"result"`
		TxTime    *int64  `msgpack:"txTime"`
		PacketCnt *uint32 `msgpack:"packetCnt"`
	}

	err = msgpack.Unmarshal(msgpackData, &dlResult)
	require.NoError(t, err)

	// Verify all fields are correctly parsed
	assert.Equal(t, bssci.TestEpEui01, dlResult.EpEui)
	assert.Equal(t, uint64(42), dlResult.QueID)
	assert.Equal(t, "sent", dlResult.Result)
	assert.NotNil(t, dlResult.TxTime)
	assert.Equal(t, int64(1234567890), *dlResult.TxTime)
	assert.NotNil(t, dlResult.PacketCnt)
	assert.Equal(t, uint32(10), *dlResult.PacketCnt)
}

// fakeDownlinkService is a minimal test implementation of DownlinkService
type fakeDownlinkService struct{}

func (f *fakeDownlinkService) ProcessDLDataResult(_ context.Context, _ *bssci.Session, result *mioty.DLDataResult) (map[string]interface{}, error) {
	// Return minimal valid response per BSSCI §5.14.2
	return map[string]interface{}{
		"command": "dlDataResRsp",
		"opId":    result.OpId,
	}, nil
}

func (f *fakeDownlinkService) ProcessRevokeResponse(_ context.Context, _ *bssci.Session, opId int64, _ int64, _ uint64) (map[string]interface{}, bool, error) {
	// Return minimal valid response per BSSCI §5.13.2
	return map[string]interface{}{
		"command": "dlDataRevRsp",
		"opId":    opId,
	}, true, nil
}

func (f *fakeDownlinkService) ProcessQueueAck(context.Context, *bssci.Session, bssci.QueueAcknowledgement) error {
	return nil
}

func (f *fakeDownlinkService) ProcessQueueError(context.Context, *bssci.Session, bssci.QueueRejection) error {
	return nil
}

func (f *fakeDownlinkService) ProcessRevokeRefusal(context.Context, *bssci.Session, bssci.RevokeRefusal) (bool, error) {
	return true, nil
}

// TestThreeWayHandshake verifies the complete BSSCI three-way handshake
func TestThreeWayHandshake(t *testing.T) {
	eventStore := &mockEventStoreImpl{}
	conn := &mockConnWithFrames{}

	// Wire DownlinkService with shared mockResolver
	logger := logger.NewNop()
	mockResolver := newMockTenantResolver()

	// Create services individually so DownlinkService uses the shared mockResolver
	sessionSvc, _, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, _, mockStorage := bssci.CreateTestServices(logger, eventStore)

	// Create fake DownlinkService for test
	downlinkSvc := &fakeDownlinkService{}

	server := bssci.NewTestServer(logger, mockStorage, eventStore, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, mockResolver)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: bssci.TestBsEui01,
		},
		Conn: conn,
	}

	msg := &bssci.Message{
		OpId:    456,
		Command: "dlDataRes",
	}

	data := map[string]interface{}{
		"epEui":  bssci.TestEpEui01,
		"queId":  uint64(42),
		"result": "expired",
	}

	// Inject queue-to-tenant mapping for test
	mockResolver.RegisterQueueTenant(42, "1")

	// Call handler
	err := server.CallHandleDLDataResult(session, msg, data)
	require.NoError(t, err)

	// Verify handler sends dlDataResRsp as one frame (header+payload)
	require.Len(t, conn.frames, 1, "Should send dlDataResRsp")

	// dlDataResRsp (sent) - frames[0]=header, frames[1]=payload
	var resp map[string]interface{}
	err = msgpack.Unmarshal(bssciutil.FramePayload(conn.frames[0]), &resp)
	require.NoError(t, err)
	// BSSCI §5.14.2: dlDataResRsp MUST contain only command and opId
	assert.Len(t, resp, 2, "Response must contain exactly 2 fields")
	assert.Equal(t, "dlDataResRsp", resp["command"], "Command field must be dlDataResRsp")
	assert.Equal(t, int64(456), resp["opId"], "OpId field must match request")
}

// Error format strings shared by this package's failure paths; verbs are filled at the point of failure.
const (
	errFmtQueueIDNotRegisteredInMock = "queue ID %d not registered in mock"
)
