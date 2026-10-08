package scaci

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var scaciClockTestNow = time.Date(2026, 9, 2, 10, 30, 0, 123456789, time.UTC)

// Every timestamp the SCACI handlers persist comes from the server clock, so
// a fake clock shows up verbatim in the operation metadata.
func TestHandleDLDataRevoke_StampsMetadataFromTheServerClock(t *testing.T) {
	const (
		tenantID int64 = 1
		opID     int64 = 10
		queID    int64 = 555
	)
	want := scaciClockTestNow.UTC().Format(time.RFC3339Nano)

	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(tenantID, 0x1234567890ABCDEF, 100)).
		Return([]*storage.DownlinkMessage{{QueID: queID}}, nil)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(tenantID, 0x1234567890ABCDEF, uint64(queID))).Return(uint64(0x1122334455667788), "")

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record", mock.Anything, mock.Anything, opID, CmdDLDataRevoke, models.OperationDirectionInbound,
		mock.MatchedBy(func(data map[string]interface{}) bool { return data["receivedAt"] == want })).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), opID, models.OperationStateAcknowledged,
		mock.MatchedBy(func(data map[string]interface{}) bool { return data["acknowledgedAt"] == want })).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             testutil.NewFakeClock(scaciClockTestNow),
		logger:            testLogger(),
		dlSvc:             mockDL,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		config:            &Config{},
	}
	session := &Session{ID: 123, TenantID: tenantID, State: StateActive}
	payload, err := msgpack.Marshal(&DLDataRevoke{
		BaseMessage: BaseMessage{Command: CmdDLDataRevoke, OpId: opID},
		EpEui:       0x1234567890ABCDEF,
		PacketCnt:   100,
	})
	require.NoError(t, err)

	assert.NoError(t, server.handleDLDataRevoke(&mockConn{}, session, opID, payload))
	mockRecorder.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// The status response reports the service center time from the server clock.
func TestHandleStatus_ReportsTheServerClockTime(t *testing.T) {
	const opID int64 = 7
	statusSvc := new(MockStatusService)
	statusSvc.On("GetUptime").Return(int64(90))
	statusSvc.On("GetBaseStations", mock.Anything, int64(1)).Return([]*models.BaseStation{}, nil)

	server := &Server{
		registry:  newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:     testFrameCodec,
		commands:  mustTestCommandRegistry(),
		clock:     testutil.NewFakeClock(scaciClockTestNow),
		logger:    testLogger(),
		statusSvc: statusSvc,
		config:    &Config{},
	}
	conn := &mockConn{}
	session := &Session{TenantID: 1, State: StateActive}

	require.NoError(t, server.handleStatus(conn, session, opID))
	var resp StatusResponse
	require.NoError(t, decodeResponse(conn.written, &resp))
	assert.Equal(t, scaciClockTestNow.UnixNano(), resp.Time)
	assert.True(t, session.LastSeen.Equal(scaciClockTestNow), "last seen is stamped from the server clock")
}
