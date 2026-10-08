package scaci

import (
	"encoding/binary"
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

// TestHandleULDataTransmit_MissingEpEui_ReturnsError verifies that missing
// mandatory epEui field (zero value) returns POSIX_EINVAL.
//
// Spec: SCACI §2.4 - mandatory field validation
// Spec: SCACI §3.9.1 - epEui is mandatory field for ULDataTransmit
func TestHandleULDataTransmit_MissingEpEui_ReturnsError(t *testing.T) {
	mockULSvc := new(MockULService)
	mockRecorder := new(MockOperationRecorder)

	server := createTestServerFull(t, testServerOpts{ulSvc: mockULSvc, recorder: mockRecorder})
	conn := &mockConn{}
	session := createTestSession(t)

	// Create ULDataTx payload with zero epEui (missing mandatory field)
	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 42},
		EpEui:       0, // Missing mandatory field (zero value)
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler
	handlerErr := server.handleULDataTransmit(conn, session, 42, payload)

	// Should NOT return Go error
	assert.NoError(t, handlerErr)

	// Verify Error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// Assert POSIX_EINVAL and error token
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code)
	assertErrorToken(t, errorResp, errEpEuiZero)

	// Verify services not called
	mockRecorder.AssertNotCalled(t, "Record")
	mockULSvc.AssertNotCalled(t, "ScheduleULTransmit")
}

// TestHandleULDataTransmit_FormatDefaultApplied verifies that format field
// defaults to 0 when absent from the request.
//
// Spec: SCACI §2.4 - optional field default handling
// Spec: SCACI §3.9.1 - format field is optional, default 0
func TestHandleULDataTransmit_FormatDefaultApplied(t *testing.T) {
	// Setup mock services with capture
	var capturedReq *mioty.ULDataTransmit
	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(req *mioty.ULDataTransmit) bool {
			capturedReq = req
			return true
		}),
		mock.AnythingOfType("int64"),
	).Return(int64(123), uint64(0x70B3D59CD00009E6), "")

	mockRecorder := new(MockOperationRecorder)
	// Record is not called because session.ID=0 and operationRepo=nil

	server := createTestServerFull(t, testServerOpts{ulSvc: mockULSvc, recorder: mockRecorder})
	conn := &mockConn{}
	session := createTestSession(t)

	// Create ULDataTx WITHOUT format field (should default to 0)
	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 42},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Format:      nil, // Omit format - should default to 0
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler
	handlerErr := server.handleULDataTransmit(conn, session, 42, payload)
	assert.NoError(t, handlerErr)

	// Verify ULService was called
	mockULSvc.AssertExpectations(t)

	// CRITICAL: Verify format was defaulted to 0 in the captured request
	require.NotNil(t, capturedReq, "ULService should have been called with request")
	require.NotNil(t, capturedReq.Format, "Format should NOT be nil after defaulting")
	assert.Equal(t, uint8(0), *capturedReq.Format, "Format should default to 0 per §3.9.1")
}

// TestHandleULDataTransmit_FormatPreserved verifies that explicitly provided
// format field is preserved (not overwritten by default).
//
// Spec: SCACI §3.9.1 - format field optional, explicit value preserved
func TestHandleULDataTransmit_FormatPreserved(t *testing.T) {
	// Setup mock services with capture
	var capturedReq *mioty.ULDataTransmit
	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(req *mioty.ULDataTransmit) bool {
			capturedReq = req
			return true
		}),
		mock.AnythingOfType("int64"),
	).Return(int64(123), uint64(0x70B3D59CD00009E6), "")

	mockRecorder := new(MockOperationRecorder)

	server := createTestServerFull(t, testServerOpts{ulSvc: mockULSvc, recorder: mockRecorder})
	conn := &mockConn{}
	session := createTestSession(t)

	// Create ULDataTx WITH explicit format=1
	explicitFormat := uint8(1)
	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 42},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Format:      &explicitFormat, // Explicitly set format=1
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler
	handlerErr := server.handleULDataTransmit(conn, session, 42, payload)
	assert.NoError(t, handlerErr)

	// Verify ULService was called
	mockULSvc.AssertExpectations(t)

	// Verify format was preserved as 1
	require.NotNil(t, capturedReq, "ULService should have been called with request")
	require.NotNil(t, capturedReq.Format, "Format should NOT be nil")
	assert.Equal(t, uint8(1), *capturedReq.Format, "Format should be preserved as 1")
}

// TestHandleULDataTransmit_BaseStationUnavailable_ReturnsPOSIX_EAGAIN verifies that
// scheduler resource exhaustion returns POSIX_EAGAIN per SCACI §3.9.2 error mapping.
//
// Spec: SCACI §3.9.2 - UL Data Transmit Response error handling
func TestHandleULDataTransmit_BaseStationUnavailable_ReturnsPOSIX_EAGAIN(t *testing.T) {
	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything, // Handler creates its own context via sessionContext()
		mock.MatchedBy(func(_ *mioty.ULDataTransmit) bool { return true }),
		int64(1), // tenantID
	).Return(int64(0), uint64(0), ErrBaseStationUnavailable) // Exported constant

	mockRecorder := new(MockOperationRecorder)
	server := createTestServerFull(t, testServerOpts{ulSvc: mockULSvc, recorder: mockRecorder})
	conn := &mockConn{}
	session := createTestSession(t)

	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 42},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 42, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// CRITICAL: Verify POSIX_EAGAIN for temporary resource unavailability
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EAGAIN, errorResp.Code, "BS unavailable must return POSIX_EAGAIN")
	assertErrorToken(t, errorResp, ErrBaseStationUnavailable)

	mockULSvc.AssertExpectations(t)
}

// TestHandleULDataTransmit_BaseStationNotFound_ReturnsPOSIX_ENOENT verifies that
// missing base station returns POSIX_ENOENT per SCACI §3.9.2 error mapping.
//
// Spec: SCACI §3.9.2 - UL Data Transmit Response error handling
func TestHandleULDataTransmit_BaseStationNotFound_ReturnsPOSIX_ENOENT(t *testing.T) {
	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(_ *mioty.ULDataTransmit) bool { return true }),
		int64(1),
	).Return(int64(0), uint64(0), ErrBaseStationNotFound)

	mockRecorder := new(MockOperationRecorder)
	server := createTestServerFull(t, testServerOpts{ulSvc: mockULSvc, recorder: mockRecorder})
	conn := &mockConn{}
	session := createTestSession(t)

	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 43},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 43, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_ENOENT, errorResp.Code, "BS not found must return POSIX_ENOENT")
	assertErrorToken(t, errorResp, ErrBaseStationNotFound)

	mockULSvc.AssertExpectations(t)
}

// TestHandleULDataTransmit_ULTransmitNotSupported_ReturnsPOSIX_ENOTSUP verifies that
// feature not supported returns POSIX_ENOTSUP per SCACI §3.9.2 error mapping.
//
// Spec: SCACI §3.9.2 - UL Data Transmit Response error handling
func TestHandleULDataTransmit_ULTransmitNotSupported_ReturnsPOSIX_ENOTSUP(t *testing.T) {
	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(_ *mioty.ULDataTransmit) bool { return true }),
		int64(1),
	).Return(int64(0), uint64(0), ErrULTransmitNotSupported)

	mockRecorder := new(MockOperationRecorder)
	server := createTestServerFull(t, testServerOpts{ulSvc: mockULSvc, recorder: mockRecorder})
	conn := &mockConn{}
	session := createTestSession(t)

	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 44},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 44, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_ENOTSUP, errorResp.Code, "UL transmit not supported must return POSIX_ENOTSUP")
	assertErrorToken(t, errorResp, ErrULTransmitNotSupported)

	mockULSvc.AssertExpectations(t)
}

// TestHandleULDataTransmit_ExplicitBsEui_TenantMismatch_ReturnsNotFound verifies that
// when explicit bsEui is provided and not found for tenant, POSIX_ENOENT is returned.
//
// Spec: SCACI §3.9.1 - BS tenant verification before scheduling
func TestHandleULDataTransmit_ExplicitBsEui_TenantMismatch_ReturnsNotFound(t *testing.T) {
	// Mock statusSvc to return ErrNotFound (BS not found for tenant)
	mockStatusSvc := new(MockStatusService)
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, 0x70B3D59CD00009E6)
	mockStatusSvc.On(
		"GetBaseStation",
		mock.Anything,
		int64(1), // tenantID
		bsEuiBytes,
	).Return(nil, storage.ErrNotFound)

	// ULSvc should NOT be called - validation fails before scheduling
	mockULSvc := new(MockULService)
	mockRecorder := new(MockOperationRecorder)

	server := createTestServerFull(t, testServerOpts{
		ulSvc:     mockULSvc,
		recorder:  mockRecorder,
		statusSvc: mockStatusSvc,
	})
	conn := &mockConn{}
	session := createTestSession(t)

	// Create request with explicit bsEui
	explicitBsEui := uint64(0x70B3D59CD00009E6)
	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 45},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
		BsEui:       &explicitBsEui,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 45, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_ENOENT, errorResp.Code, "BS not found for tenant must return POSIX_ENOENT")
	assertErrorToken(t, errorResp, ErrBaseStationNotFound)

	// CRITICAL: Verify ULService was NOT called (validation failed before scheduling)
	mockULSvc.AssertNotCalled(t, "ScheduleULTransmit")
	mockStatusSvc.AssertExpectations(t)
}

// TestHandleULDataTransmit_ExplicitBsEui_LookupFailed_ReturnsPOSIX_EIO verifies that
// when explicit bsEui lookup fails with generic error, POSIX_EIO is returned.
//
// Spec: SCACI §3.9.1 - BS lookup failure handling
func TestHandleULDataTransmit_ExplicitBsEui_LookupFailed_ReturnsPOSIX_EIO(t *testing.T) {
	mockStatusSvc := new(MockStatusService)
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, 0x70B3D59CD00009E6)
	mockStatusSvc.On(
		"GetBaseStation",
		mock.Anything,
		int64(1),
		bsEuiBytes,
	).Return(nil, assert.AnError) // Generic error (not ErrNotFound)

	mockULSvc := new(MockULService)
	mockRecorder := new(MockOperationRecorder)

	server := createTestServerFull(t, testServerOpts{
		ulSvc:     mockULSvc,
		recorder:  mockRecorder,
		statusSvc: mockStatusSvc,
	})
	conn := &mockConn{}
	session := createTestSession(t)

	explicitBsEui := uint64(0x70B3D59CD00009E6)
	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 46},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
		BsEui:       &explicitBsEui,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 46, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EIO, errorResp.Code, "BS lookup failure must return POSIX_EIO")
	assertErrorToken(t, errorResp, errFailedVerifyBS) // Internal constant (not exported)

	mockULSvc.AssertNotCalled(t, "ScheduleULTransmit")
	mockStatusSvc.AssertExpectations(t)
}

// TestHandleULDataTransmit_ExplicitBsEui_Success_VerifiesTenant verifies that
// when explicit bsEui is found, both statusSvc and ulSvc are called with correct tenant.
//
// Spec: SCACI §3.9.1 - BS tenant verification success path
func TestHandleULDataTransmit_ExplicitBsEui_Success_VerifiesTenant(t *testing.T) {
	mockStatusSvc := new(MockStatusService)
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, 0x70B3D59CD00009E6)
	var bsEuiArray models.EUI
	copy(bsEuiArray[:], bsEuiBytes)
	mockStatusSvc.On(
		"GetBaseStation",
		mock.Anything,
		int64(1), // tenantID
		bsEuiBytes,
	).Return(&models.BaseStation{EUI: bsEuiArray}, nil) // BS found

	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(req *mioty.ULDataTransmit) bool {
			return req.BsEui != nil && *req.BsEui == 0x70B3D59CD00009E6
		}),
		int64(1), // tenantID must match
	).Return(int64(456), uint64(0x70B3D59CD00009E6), "")

	mockRecorder := new(MockOperationRecorder)

	server := createTestServerFull(t, testServerOpts{
		ulSvc:     mockULSvc,
		recorder:  mockRecorder,
		statusSvc: mockStatusSvc,
	})
	conn := &mockConn{}
	session := createTestSession(t)

	explicitBsEui := uint64(0x70B3D59CD00009E6)
	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 47},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
		BsEui:       &explicitBsEui,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 47, payload)
	assert.NoError(t, handlerErr)

	// Verify success response (not error)
	var resp ULDataTransmitResponse
	require.NoError(t, decodeResponse(conn.written, &resp))
	assert.Equal(t, CmdULDataTransmitResponse, resp.CommandType)
	assert.Equal(t, int64(47), resp.OpId)

	// Verify both services were called with correct tenant
	mockStatusSvc.AssertExpectations(t)
	mockULSvc.AssertExpectations(t)
}

// TestHandleULDataTransmit_OperationLogging_RecordsCalled verifies that operation
// recording is called with correct fields and excludes nwkSnKey for security.
//
// Spec: SCACI §3.9.1 - Operation recording for resume safety
func TestHandleULDataTransmit_OperationLogging_RecordsCalled(t *testing.T) {
	var capturedRequestData map[string]interface{}
	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On(
		"Record",
		mock.Anything,
		mock.MatchedBy(func(s *Session) bool { return s.ID == 123 }),
		int64(48),         // opId
		CmdULDataTransmit, // command
		models.OperationDirectionInbound,
		mock.MatchedBy(func(data map[string]interface{}) bool {
			capturedRequestData = data
			return true
		}),
	).Return(nil)

	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.Anything,
		int64(1),
	).Return(int64(789), uint64(0x70B3D59CD00009E6), "")

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123), // sessionID
		int64(48),  // opId
		models.OperationStateAcknowledged,
		mock.Anything,
	).Return(nil)

	server := createTestServerFull(t, testServerOpts{
		ulSvc:    mockULSvc,
		recorder: mockRecorder,
		opRepo:   mockOpRepo,
	})
	conn := &mockConn{}
	session := createTestSessionWithID(t)

	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 48},
		EpEui:       0x70B3D59CD00008C1,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 48, payload)
	assert.NoError(t, handlerErr)

	// Verify Record was called
	mockRecorder.AssertExpectations(t)

	// Verify requestData fields
	require.NotNil(t, capturedRequestData, "Record should have been called with requestData")
	assert.Contains(t, capturedRequestData, "epEui")
	assert.Contains(t, capturedRequestData, "shAddr")
	assert.Contains(t, capturedRequestData, "packetCnt")
	assert.Contains(t, capturedRequestData, "format")
	assert.Contains(t, capturedRequestData, "userData")

	// CRITICAL: nwkSnKey must NOT be in requestData (security)
	_, hasNwkSnKey := capturedRequestData["nwkSnKey"]
	assert.False(t, hasNwkSnKey, "nwkSnKey must NOT be in requestData for security")
}

// TestHandleULDataTransmit_SchedulingFailed_MarksFailed verifies that when
// scheduling fails, operation is marked as failed with error metadata.
//
// Spec: SCACI §3.9.2 - Failed operation state tracking
func TestHandleULDataTransmit_SchedulingFailed_MarksFailed(t *testing.T) {
	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.Anything,
		int64(1),
	).Return(int64(0), uint64(0), ErrBaseStationUnavailable)

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On(
		"Record",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123), // sessionID
		int64(49),  // opId
		models.OperationStateFailed,
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			errorToken, hasToken := meta["errorToken"].(string)
			errorDetail, hasDetail := meta["errorDetail"].(string)
			return hasToken && errorToken == ErrBaseStationUnavailable &&
				hasDetail && errorDetail == "UL transmit scheduling failed"
		}),
	).Return(nil)

	server := createTestServerFull(t, testServerOpts{
		ulSvc:    mockULSvc,
		recorder: mockRecorder,
		opRepo:   mockOpRepo,
	})
	conn := &mockConn{}
	session := createTestSessionWithID(t)

	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 49},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 49, payload)
	assert.NoError(t, handlerErr)

	// Verify UpdateOperationState was called with StateFailed
	mockOpRepo.AssertExpectations(t)
	// Verify NOT called with Acknowledged
	mockOpRepo.AssertNotCalled(t, "UpdateOperationState",
		mock.Anything, mock.Anything, mock.Anything, models.OperationStateAcknowledged, mock.Anything)
}

// TestHandleULDataTransmit_Success_MarksAcknowledged verifies that successful
// scheduling marks operation as acknowledged with response metadata.
//
// Spec: SCACI §3.9.2 - Acknowledged operation state tracking
func TestHandleULDataTransmit_Success_MarksAcknowledged(t *testing.T) {
	mockULSvc := new(MockULService)
	mockULSvc.On(
		"ScheduleULTransmit",
		mock.Anything,
		mock.Anything,
		int64(1),
	).Return(int64(789), uint64(0x70B3D59CD00009E6), "") // Success

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On(
		"Record",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123), // sessionID
		int64(50),  // opId
		models.OperationStateAcknowledged,
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			bssciOpID, hasOpID := meta["bssciOpID"]
			bsEui, hasBsEui := meta["bsEui"]
			return hasOpID && bssciOpID == int64(789) &&
				hasBsEui && bsEui == "70B3D59CD00009E6"
		}),
	).Return(nil)

	server := createTestServerFull(t, testServerOpts{
		ulSvc:    mockULSvc,
		recorder: mockRecorder,
		opRepo:   mockOpRepo,
	})
	conn := &mockConn{}
	session := createTestSessionWithID(t)

	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 50},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleULDataTransmit(conn, session, 50, payload)
	assert.NoError(t, handlerErr)

	// Verify response was sent
	var resp ULDataTransmitResponse
	require.NoError(t, decodeResponse(conn.written, &resp))
	assert.Equal(t, CmdULDataTransmitResponse, resp.CommandType)

	// Verify UpdateOperationState was called with StateAcknowledged
	mockOpRepo.AssertExpectations(t)
}

// TestHandleULDataTransmitComplete_MarksCompleted verifies that handshake
// completion marks operation as completed with timestamp and sends no response.
//
// Spec: SCACI §3.9.3 - UL Data Transmit Complete processing
func TestHandleULDataTransmitComplete_MarksCompleted(t *testing.T) {
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123), // sessionID
		int64(51),  // opId
		models.OperationStateCompleted,
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			completedAt, hasCompleted := meta["completedAt"].(string)
			return hasCompleted && completedAt != ""
		}),
	).Return(nil)

	server := createTestServerFull(t, testServerOpts{
		opRepo: mockOpRepo,
	})
	conn := &mockConn{}
	session := createTestSessionWithID(t)

	handlerErr := server.handleULDataTransmitComplete(conn, session, 51)
	assert.NoError(t, handlerErr)

	// Verify UpdateOperationState was called with StateCompleted
	mockOpRepo.AssertExpectations(t)

	// CRITICAL: Verify NO response was written (per SCACI §3.9.3)
	assert.Empty(t, conn.written, "handleULDataTransmitComplete must send no response per §3.9.3")
}

// TestHandleULDataTransmitComplete_NilSession_ReturnsGracefully verifies that
// nil session is handled gracefully.
//
// Spec: SCACI §3.9.3 - Session validation
func TestHandleULDataTransmitComplete_NilSession_ReturnsGracefully(t *testing.T) {
	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		config:   &Config{},
	}

	conn := &mockConn{}

	// Execute handler with nil session
	handlerErr := server.handleULDataTransmitComplete(conn, nil, 52)
	// Handler should NOT return error - it sends error response via sendErrorWithCatalog
	assert.NoError(t, handlerErr)

	// Verify error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code)
	assertErrorToken(t, errorResp, errNoActiveSession)
}
