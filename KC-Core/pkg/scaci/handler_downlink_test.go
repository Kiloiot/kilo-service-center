package scaci

import (
	"net"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// TestHandleDLDataQueue_NonCntDependMultiPayload_ReturnsEINVAL verifies that when
// cntDepend=false but UserData has more than one entry, the handler rejects the
// request with errNonCntDependMultiPayload and POSIX_EINVAL.
//
// Spec: SCACI §3.10.1 - "single user data entry if cntDepend is false"
// Gap: Missing validation for multi-payload with non-counter-dependent queue
func TestHandleDLDataQueue_NonCntDependMultiPayload_ReturnsEINVAL(t *testing.T) {
	mockRecorder := new(MockOperationRecorder)
	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		operationRecorder: mockRecorder,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 123, TenantID: 1, State: StateActive}

	req := DLDataQueue{
		BaseMessage: mioty.BaseMessage{CommandType: CmdDLDataQueue, OpId: 100},
		EpEui:       0x1234567890ABCDEF,
		QueId:       42,
		CntDepend:   false,
		UserData:    [][]byte{{0x01, 0x02, 0x03}, {0x04, 0x05, 0x06}}, // Multiple entries violates spec
	}

	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleDLDataQueue(conn, session, 100, payload)
	assert.NoError(t, handlerErr)

	// Decode response - should be an error response
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// Verify error details
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code)
	assertErrorToken(t, errorResp, errNonCntDependMultiPayload)

	// Verify no DB write occurred (operation not persisted)
	mockRecorder.AssertNotCalled(t, "Record")
}

// TestHandleDLDataRevoke_Success_SendsRevRsp verifies that a valid dlDataRev
// payload triggers lookup, revoke, and response with matching opId.
//
// Spec: SCACI §3.11.1-3.11.2 - DL Data Revoke request/response flow
func TestHandleDLDataRevoke_Success_SendsRevRsp(t *testing.T) {
	// Mock DLService for GetDownlinksByPacketCnt lookup
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(1, 0x1234567890ABCDEF, 100)).
		Return([]*storage.DownlinkMessage{{QueID: 42}}, nil)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(1, 0x1234567890ABCDEF, 42)).
		Return(uint64(0x70B3D59CD00009E6), "")

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On(
		"Record",
		mock.Anything, mock.Anything, int64(10), CmdDLDataRevoke,
		models.OperationDirectionInbound, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything, int64(123), int64(10),
		models.OperationStateAcknowledged, mock.Anything,
	).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		dlSvc:             mockDL,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 123, TenantID: 1, State: StateActive}

	req := DLDataRevoke{
		BaseMessage: BaseMessage{Command: CmdDLDataRevoke, OpId: 10},
		EpEui:       0x1234567890ABCDEF,
		PacketCnt:   100,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleDLDataRevoke(conn, session, 10, payload)
	assert.NoError(t, handlerErr)

	// Decode response
	var resp DLDataRevokeResponse
	require.NoError(t, decodeResponse(conn.written, &resp))

	assert.Equal(t, CmdDLDataRevokeResponse, resp.Command)
	assert.Equal(t, int64(10), resp.OpId)

	mockDL.AssertExpectations(t)
	mockRecorder.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// TestHandleDLDataRevoke_NothingScheduledCompletesTheRevoke: when no downlink
// is scheduled for the packet counter, what the Application Center asks for
// already holds, so the revoke completes with dlDataRevRsp and revokes nothing
// (SCACI §3.11.1-3.11.2), instead of an error the Application Center would retry.
func TestHandleDLDataRevoke_NothingScheduledCompletesTheRevoke(t *testing.T) {
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(1, 0x1234567890ABCDEF, 999)).
		Return([]*storage.DownlinkMessage{}, nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything, int64(123), int64(12),
		models.OperationStateAcknowledged, mock.MatchedBy(func(meta map[string]interface{}) bool {
			queIDs, ok := meta["queIds"].([]string)
			return ok && len(queIDs) == 0
		}),
	).Return(nil)

	server := &Server{
		registry:      newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:         testFrameCodec,
		commands:      mustTestCommandRegistry(),
		clock:         clock.SystemClock{},
		logger:        testLogger(),
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 123, TenantID: 1, State: StateActive}

	req := DLDataRevoke{
		BaseMessage: BaseMessage{Command: CmdDLDataRevoke, OpId: 12},
		EpEui:       0x1234567890ABCDEF,
		PacketCnt:   999,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	require.NoError(t, server.handleDLDataRevoke(conn, session, 12, payload))

	var resp DLDataRevokeResponse
	require.NoError(t, decodeResponse(conn.written, &resp))
	assert.Equal(t, CmdDLDataRevokeResponse, resp.Command)
	assert.Equal(t, int64(12), resp.OpId)
	mockDL.AssertNotCalled(t, "RevokeDownlink", mock.Anything, mock.Anything)
	mockDL.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// TestHandleDLDataRevoke_SchedulerUnavailable_ReturnsENOTSUP verifies that when
// RevokeDownlink returns errSchedulerUnavailable, POSIX_ENOTSUP is returned.
//
// Spec: SCACI §3.11 - Scheduler unavailable error mapping
func TestHandleDLDataRevoke_SchedulerUnavailable_ReturnsENOTSUP(t *testing.T) {
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(1, 0x1234567890ABCDEF, 100)).
		Return([]*storage.DownlinkMessage{{QueID: 42}}, nil)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(1, 0x1234567890ABCDEF, 42)).
		Return(uint64(0), ErrSchedulerUnavailable)

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On(
		"Record",
		mock.Anything, mock.Anything, int64(13), CmdDLDataRevoke,
		models.OperationDirectionInbound, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything, int64(123), int64(13),
		models.OperationStateFailed, mock.MatchedBy(func(meta map[string]interface{}) bool {
			return meta["errorToken"] == ErrSchedulerUnavailable
		}),
	).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		dlSvc:             mockDL,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 123, TenantID: 1, State: StateActive}

	req := DLDataRevoke{
		BaseMessage: BaseMessage{Command: CmdDLDataRevoke, OpId: 13},
		EpEui:       0x1234567890ABCDEF,
		PacketCnt:   100,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleDLDataRevoke(conn, session, 13, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_ENOTSUP, errorResp.Code)
	assertErrorToken(t, errorResp, ErrSchedulerUnavailable)

	mockDL.AssertExpectations(t)
	mockRecorder.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// TestHandleDLDataRevokeComplete_MarksCompleted_NoResponse verifies that
// handleDLDataRevokeComplete marks the operation as completed and sends no response.
//
// Spec: SCACI §3.11.3 - DL Data Revoke Complete processing, no response sent
func TestHandleDLDataRevokeComplete_MarksCompleted_NoResponse(t *testing.T) {
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123), // sessionID
		int64(14),  // opId
		models.OperationStateCompleted,
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			completedAt, hasCompleted := meta["completedAt"].(string)
			status, hasStatus := meta["status"].(string)
			return hasCompleted && completedAt != "" && hasStatus && status == ResultRevoked
		}),
	).Return(nil)

	server := &Server{
		registry:      newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:         testFrameCodec,
		commands:      mustTestCommandRegistry(),
		clock:         clock.SystemClock{},
		logger:        testLogger(),
		operationRepo: mockOpRepo,
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 123, TenantID: 1, State: StateActive}

	handlerErr := server.handleDLDataRevokeComplete(conn, session, 14)
	assert.NoError(t, handlerErr)

	// CRITICAL: Verify NO response was written (per SCACI §3.11.3)
	assert.Empty(t, conn.written, "handleDLDataRevokeComplete must send no response per §3.11.3")

	mockOpRepo.AssertExpectations(t)
}

// revokeHandlerServer serves one persisted session of tenant 1 for the dlDataRev tests.
func revokeHandlerServer(dl DLService, recorder OperationRecorder, repo *MockSCACIOperationRepository) *Server {
	return &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		dlSvc:             dl,
		operationRecorder: recorder,
		operationRepo:     repo,
		config:            &Config{},
	}
}

func revokePayload(t *testing.T, opID int64, packetCnt uint32) []byte {
	t.Helper()
	payload, err := msgpack.Marshal(&DLDataRevoke{
		BaseMessage: BaseMessage{Command: CmdDLDataRevoke, OpId: opID},
		EpEui:       0x1234567890ABCDEF,
		PacketCnt:   packetCnt,
	})
	require.NoError(t, err)
	return payload
}

// TestHandleDLDataRevoke_RevokesEveryDownlinkScheduledForTheCounter pins
// SCACI §3.11.1: dlDataRev names the scheduled data by packet counter, so
// every downlink scheduled for that counter is revoked, not only the newest.
func TestHandleDLDataRevoke_RevokesEveryDownlinkScheduledForTheCounter(t *testing.T) {
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(1, 0x1234567890ABCDEF, 100)).
		Return([]*storage.DownlinkMessage{{QueID: 42}, {QueID: 43}}, nil)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(1, 0x1234567890ABCDEF, 42)).Return(uint64(0x70B3D59CD00009E6), "")
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(1, 0x1234567890ABCDEF, 43)).Return(uint64(0), "")
	recorder := new(MockOperationRecorder)
	recorder.On("Record", mock.Anything, mock.Anything, int64(14), CmdDLDataRevoke,
		models.OperationDirectionInbound, mock.Anything).Return(nil)
	opRepo := new(MockSCACIOperationRepository)
	opRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(14),
		models.OperationStateAcknowledged, mock.MatchedBy(func(meta map[string]interface{}) bool {
			return assert.ObjectsAreEqual([]string{"42", "43"}, meta["queIds"])
		})).Return(nil)
	server := revokeHandlerServer(mockDL, recorder, opRepo)
	conn := &mockConn{}

	require.NoError(t, server.handleDLDataRevoke(conn, &Session{ID: 123, TenantID: 1, State: StateActive}, 14, revokePayload(t, 14, 100)))

	var resp DLDataRevokeResponse
	require.NoError(t, decodeResponse(conn.written, &resp))
	assert.Equal(t, CmdDLDataRevokeResponse, resp.Command)
	mockDL.AssertExpectations(t)
	opRepo.AssertExpectations(t)
}

// TestHandleDLDataRevoke_PartialFailureRevokesTheRestAndReportsIt: one
// downlink that cannot be revoked does not keep the others scheduled, and
// the Application Center learns the revoke did not complete.
func TestHandleDLDataRevoke_PartialFailureRevokesTheRestAndReportsIt(t *testing.T) {
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(1, 0x1234567890ABCDEF, 100)).
		Return([]*storage.DownlinkMessage{{QueID: 42}, {QueID: 43}}, nil)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(1, 0x1234567890ABCDEF, 42)).Return(uint64(0), ErrDownlinkNotFound)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(1, 0x1234567890ABCDEF, 43)).Return(uint64(0), "")
	server := revokeHandlerServer(mockDL, nil, nil)
	conn := &mockConn{}

	require.NoError(t, server.handleDLDataRevoke(conn, &Session{TenantID: 1, State: StateActive}, 15, revokePayload(t, 15, 100)))

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))
	assert.Equal(t, POSIX_ENOENT, errorResp.Code)
	assertErrorToken(t, errorResp, errDownlinkNotFound)
	mockDL.AssertExpectations(t)
}
