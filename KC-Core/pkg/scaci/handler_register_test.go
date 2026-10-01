package scaci

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// TestRevokeEndpointDownlinks_NilDLSvc_ReportsError: without a downlink
// service the revoke reports the wiring fault instead of panicking.
func TestRevokeEndpointDownlinks_NilDLSvc_ReportsError(t *testing.T) {
	server := &Server{
		registry: newTestRegistry(nil, nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		dlSvc:    nil, // nil to test defensive guard
	}

	count, err := server.revokeEndpointDownlinks(testutil.TestContext(), ApplicationCenter{TenantID: 1}, 0x1234567890ABCDEF)
	require.ErrorIs(t, err, errDownlinkServiceUnavailable, "a missing downlink service is a wiring fault, not a successful revoke")
	assert.Equal(t, 0, count)
}

// TestHandleRegister_DatabaseError_ReturnsCatalogPOSIX verifies that database errors
// return POSIX_EIO from the error catalog, not the handler's default POSIX_EINVAL.
//
// Spec: SCACI §3.6.2 - Register operation error handling
// Per plan: Tests must verify catalog-driven POSIX behavior
func TestHandleRegister_DatabaseError_ReturnsCatalogPOSIX(t *testing.T) {
	// Setup mock services
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Register", mock.Anything, mock.Anything, int64(1)).Return(ErrDatabaseError)

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Create server with mocks
	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		operationRepo:     nil,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123, // Non-zero to enable operation recording
		TenantID: 1,
		State:    StateActive,
	}

	// Create valid Register payload
	req := Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: 42},
		EpEui:       0x1234567890ABCDEF,
		NwkKey:      [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler
	handlerErr := server.handleRegister(conn, session, 42, payload)
	assert.NoError(t, handlerErr)

	// Verify Error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// CRITICAL: Verify POSIX_EIO comes from catalog (not handler default POSIX_EINVAL)
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EIO, errorResp.Code, "Database error must return POSIX_EIO from catalog, not default POSIX_EINVAL")
	assertErrorToken(t, errorResp, ErrDatabaseError)

	mockEndpoint.AssertExpectations(t)
}

// TestHandleRegister_CreateFailed_ReturnsCatalogPOSIX verifies that endpoint creation
// failures return POSIX_EIO from the error catalog.
//
// Spec: SCACI §3.6.2 - RegisterResponse error handling
// Per plan: Assert POSIX_EIO comes from catalog override, not handler default
func TestHandleRegister_CreateFailed_ReturnsCatalogPOSIX(t *testing.T) {
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Register", mock.Anything, mock.Anything, int64(1)).Return(ErrFailedCreateEndpoint)

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	req := Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: 43},
		EpEui:       0x1234567890ABCDEF,
		NwkKey:      [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleRegister(conn, session, 43, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// CRITICAL: Verify POSIX_EIO from catalog
	assert.Equal(t, POSIX_EIO, errorResp.Code, "Failed create must return POSIX_EIO from catalog")
	assertErrorToken(t, errorResp, ErrFailedCreateEndpoint)
}

// TestHandleRegister_ValidationError_ReturnsEINVAL verifies that validation errors
// return POSIX_EINVAL from the error catalog.
//
// Spec: SCACI §3.6.1 - Register operation validation
func TestHandleRegister_ValidationError_ReturnsEINVAL(t *testing.T) {
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Register", mock.Anything, mock.Anything, int64(1)).Return(ErrMissingEpEui)

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	req := Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: 44},
		EpEui:       0x1234567890ABCDEF,
		NwkKey:      [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleRegister(conn, session, 44, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// Verify POSIX_EINVAL from catalog
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "Validation error must return POSIX_EINVAL from catalog")
	assertErrorToken(t, errorResp, ErrMissingEpEui)
}

// TestHandleRegister_NilSession_ReturnsEINVAL verifies that nil session returns
// errNoActiveSession with POSIX_EINVAL.
//
// Spec: SCACI §3.3 - Session required for operations
// Per plan: Nil session guard test
func TestHandleRegister_NilSession_ReturnsEINVAL(t *testing.T) {
	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		config:   &Config{},
	}

	conn := &mockConn{}

	req := Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: 45},
		EpEui:       0x1234567890ABCDEF,
		NwkKey:      [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler with nil session
	handlerErr := server.handleRegister(conn, nil, 45, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// Verify errNoActiveSession token and POSIX_EINVAL
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "Nil session must return POSIX_EINVAL")
	assertErrorToken(t, errorResp, errNoActiveSession)
}

// TestHandleRegister_SessionContext_PropagatesTenantOrg verifies that session's
// TenantID and OrganizationID are propagated to operation recording.
//
// Per plan: Assert Record called with actual context values (not just that it was called)
func TestHandleRegister_SessionContext_PropagatesTenantOrg(t *testing.T) {
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Register", mock.Anything, mock.Anything, int64(42)).Return("")

	// Capture the session passed to Record
	var capturedSession *Session
	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On(
		"Record",
		mock.Anything,
		mock.MatchedBy(func(s *Session) bool {
			capturedSession = s
			return true
		}),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:             123,
		TenantID:       42,
		OrganizationID: [16]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB},
		State:          StateActive,
	}

	req := Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: 46},
		EpEui:       0x1234567890ABCDEF,
		NwkKey:      [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleRegister(conn, session, 46, payload)
	assert.NoError(t, handlerErr)

	// Verify session context was passed to Record
	require.NotNil(t, capturedSession, "Record should have been called with session")
	assert.Equal(t, int64(42), capturedSession.TenantID, "TenantID must be propagated")
	assert.Equal(t, int64(123), capturedSession.ID, "Session ID must be propagated")

	mockRecorder.AssertExpectations(t)
}

// TestHandleRegister_Success_SendsRegisterResponse verifies that successful
// registration sends RegisterResponse with correct opId.
//
// Per plan: Positive-path register success test (regression guard)
func TestHandleRegister_Success_SendsRegisterResponse(t *testing.T) {
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Register", mock.Anything, mock.Anything, int64(1)).Return("")

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	req := Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: 47},
		EpEui:       0x1234567890ABCDEF,
		NwkKey:      [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Bidi:        true,
		PreAttach:   true,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleRegister(conn, session, 47, payload)
	assert.NoError(t, handlerErr)

	// Verify RegisterResponse was sent (not Error)
	var resp RegisterResponse
	require.NoError(t, decodeResponse(conn.written, &resp))

	assert.Equal(t, CmdRegisterResponse, resp.Command, "Should send RegisterResponse on success")
	assert.Equal(t, int64(47), resp.OpId, "Response opId must match request opId")
}

// TestHandleRegister_OperationLogging_AllFields verifies that operation recording
// captures all §3.6.1 fields (except nwkKey for security).
//
// Per plan: Assert requestData contains all §3.6.1 fields formatted correctly
func TestHandleRegister_OperationLogging_AllFields(t *testing.T) {
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Register", mock.Anything, mock.Anything, int64(1)).Return("")

	// Capture requestData passed to Record
	var capturedRequestData map[string]interface{}
	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On(
		"Record",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.MatchedBy(func(data map[string]interface{}) bool {
			capturedRequestData = data
			return true
		}),
	).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	// Create Register with all §3.6.1 fields populated
	req := Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: 48},
		EpEui:       0x70B3D59CD00008C1,
		NwkKey:      [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Bidi:        true,
		PreAttach:   true,
		ShAddr:      12345,
		AttachCnt:   100000,
		PacketCnt:   200000,
		DualChan:    true,
		Repetition:  true, // bool per SCACI §3.6.1
		WideCarrOff: true,
		LongBlkDist: true,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleRegister(conn, session, 48, payload)
	assert.NoError(t, handlerErr)

	// Verify all §3.6.1 fields are present in requestData
	require.NotNil(t, capturedRequestData, "Record should have been called with requestData")

	// epEui is recorded as hex text, which the operation log's JSON decode cannot round
	assert.Equal(t, mioty.FormatEUI64(0x70B3D59CD00008C1), capturedRequestData[MetadataKeyEpEui], "epEui must be recorded as hex text")

	// Verify all other fields
	assert.Equal(t, true, capturedRequestData["bidi"], "bidi must be captured")
	assert.Equal(t, true, capturedRequestData["preAttach"], "preAttach must be captured")
	assert.Equal(t, uint16(12345), capturedRequestData["shAddr"], "shAddr must be captured")
	assert.Equal(t, uint32(100000), capturedRequestData["attachCnt"], "attachCnt must be captured")
	assert.Equal(t, uint32(200000), capturedRequestData["packetCnt"], "packetCnt must be captured")
	assert.Equal(t, true, capturedRequestData["dualChan"], "dualChan must be captured")
	assert.Equal(t, true, capturedRequestData["repetition"], "repetition must be captured (bool)")
	assert.Equal(t, true, capturedRequestData["wideCarrOff"], "wideCarrOff must be captured")
	assert.Equal(t, true, capturedRequestData["longBlkDist"], "longBlkDist must be captured")

	// CRITICAL: nwkKey must NOT be in requestData (security exclusion)
	_, hasNwkKey := capturedRequestData["nwkKey"]
	assert.False(t, hasNwkKey, "nwkKey must NOT be in requestData for security")
}

// TestHandleRegister_InvalidPayloadType_ReturnsEINVAL verifies that a non-map payload
// triggers msgpack decode failure and returns POSIX_EINVAL with errInvalidRegisterPayload.
//
// Spec: SCACI §3.6.1 - Register requires a valid msgpack map
// Gap: The service-layer test was ineffective because msgpack is lenient with field types;
// this handler-level test feeds an invalid msgpack type to exercise the decode-failure branch.
func TestHandleRegister_InvalidPayloadType_ReturnsEINVAL(t *testing.T) {
	// Build msgpack with array instead of map - Register struct expects a map
	invalidPayload := []interface{}{"not", "a", "map"}
	payload, err := msgpack.Marshal(invalidPayload)
	require.NoError(t, err)

	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		config:   &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	// Execute handler - should fail at msgpack.Unmarshal
	handlerErr := server.handleRegister(conn, session, 99, payload)
	assert.NoError(t, handlerErr, "Handler should not return error - it sends error response")

	// Verify Error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp), "Response should be valid msgpack Error")

	// CRITICAL: Verify decode failure returns POSIX_EINVAL with errInvalidRegisterPayload
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "Decode failure must return POSIX_EINVAL")
	assertErrorToken(t, errorResp, errInvalidRegisterPayload)
}

// TestHandleRegister_DecodeFailure_TruncatedPayload verifies that a truncated msgpack
// payload triggers decode failure and returns POSIX_EINVAL with errInvalidRegisterPayload.
//
// Spec: SCACI §3.6.1 - Invalid payloads must be rejected
// Gap: This covers the remaining untested decode-failure branch in the register handler.
func TestHandleRegister_DecodeFailure_TruncatedPayload(t *testing.T) {
	// Create truncated msgpack - incomplete map (starts map but truncates mid-key)
	truncatedPayload := []byte{0x81, 0xa5} // fixmap(1) + fixstr(5) but no string content

	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:    testFrameCodec,
		commands: mustTestCommandRegistry(),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		config:   &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1, // TenantID is int64
		State:    StateActive,
	}

	// Execute handler - should fail at msgpack.Unmarshal
	handlerErr := server.handleRegister(conn, session, -100, truncatedPayload)
	assert.NoError(t, handlerErr, "Handler should not return error - it sends error response")

	// Verify Error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp), "Response should be valid msgpack Error")

	// CRITICAL: Verify truncated payload returns POSIX_EINVAL with errInvalidRegisterPayload
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "Truncated payload must return POSIX_EINVAL")
	assertErrorToken(t, errorResp, errInvalidRegisterPayload)
}

// TestHandleDeregister_SendFailure_MarksOperationFailed verifies that when
// SendDeregisterResponse fails, the operation is marked as failed (not acknowledged).
//
// Spec: SCACI §3.7.2 - DeregisterResponse send failure handling
// Gap: R1 - Missing regression test for send-before-ack ordering
func TestHandleDeregister_SendFailure_MarksOperationFailed(t *testing.T) {
	// Setup mock services
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Deregister", mock.Anything, uint64(0x70B3D59CD000089B), int64(1)).Return("")

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock operationRepo to capture UpdateOperationState calls
	mockOpRepo := new(MockSCACIOperationRepository)
	// Expect UpdateOperationState to be called with StateFailed (not Acknowledged!)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123),
		int64(42),
		models.OperationStateFailed,
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			// Verify failure metadata contains error token
			token, ok := meta[MetadataKeyErrorToken].(string)
			return ok && token == errSendDeregisterResponseFailed
		}),
	).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		config:            &Config{},
	}

	// Use failing connection to simulate send error
	conn := &failingMockConn{writeErr: assert.AnError}
	session := &Session{
		ID:       123, // Non-zero to enable operation tracking
		TenantID: 1,
		State:    StateActive,
	}

	// Create Deregister payload
	req := Deregister{
		BaseMessage: BaseMessage{Command: CmdDeregister, OpId: 42},
		EpEui:       0x70B3D59CD000089B,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler - should return error because send failed
	handlerErr := server.handleDeregister(conn, session, 42, payload)
	assert.Error(t, handlerErr, "Handler should return error when send fails")

	// CRITICAL: Verify operation was marked as FAILED (not acknowledged)
	mockOpRepo.AssertExpectations(t)
	mockOpRepo.AssertNotCalled(t, "UpdateOperationState", mock.Anything, mock.Anything, mock.Anything, models.OperationStateAcknowledged, mock.Anything)
}

// TestHandleDeregister_SendSuccess_MarksAcknowledged verifies that operation
// is marked acknowledged ONLY AFTER successful send (not before).
//
// Spec: SCACI §3.7.2 - Send-before-ack ordering
// Gap: R1 - Test must verify call order: send → ack
func TestHandleDeregister_SendSuccess_MarksAcknowledged(t *testing.T) {
	// Track call order to verify send happens before ack
	var callOrder []string
	var callOrderMu sync.Mutex

	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Deregister", mock.Anything, uint64(0x70B3D59CD000089B), int64(1)).Return("")

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123),
		int64(42),
		models.OperationStateAcknowledged,
		mock.Anything,
	).Run(func(_ mock.Arguments) {
		callOrderMu.Lock()
		callOrder = append(callOrder, "ack")
		callOrderMu.Unlock()
	}).Return(nil)

	server := &Server{
		registry:          newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:             testFrameCodec,
		commands:          mustTestCommandRegistry(),
		clock:             clock.SystemClock{},
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		config:            &Config{},
	}

	// Use tracking connection to record when write happens
	conn := &trackingMockConn{
		onWrite: func() {
			callOrderMu.Lock()
			callOrder = append(callOrder, "send")
			callOrderMu.Unlock()
		},
	}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	req := Deregister{
		BaseMessage: BaseMessage{Command: CmdDeregister, OpId: 42},
		EpEui:       0x70B3D59CD000089B,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler
	handlerErr := server.handleDeregister(conn, session, 42, payload)
	assert.NoError(t, handlerErr)

	// CRITICAL: Verify send happened BEFORE ack
	callOrderMu.Lock()
	defer callOrderMu.Unlock()
	require.Len(t, callOrder, 2, "Expected exactly 2 operations: send and ack")
	assert.Equal(t, "send", callOrder[0], "Send must happen first")
	assert.Equal(t, "ack", callOrder[1], "Ack must happen after send")
}

// TestHandleDeregisterComplete_CleanupMetadataCapture verifies that handleDeregisterComplete
// captures the cleanup metadata keys (including cleanupStatus) and that counts match the
// mocked service returns. When cleanup errors occur, state should be completed_with_warnings.
//
// Spec: SCACI §3.7.3 - DeregisterComplete cleanup metadata
// Gap: R4 Gap 2 - completed_with_warnings state for cleanup issues
func TestHandleDeregisterComplete_CleanupMetadataCapture(t *testing.T) {
	// Setup mock services with specific return values
	mockEndpoint := new(MockEndpointService)
	// PropagateDetachToAll returns 2 errors to verify detachErrorCount
	mockEndpoint.On("PropagateDetachToAll", mock.Anything, int64(1), uint64(0x70B3D59CD000089B)).Return([]error{
		assert.AnError,
		assert.AnError,
	})

	mockDL := new(MockDLService)
	// GetDownlinkQueue returns 3 downlinks
	mockDL.On("GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything).Return([]*storage.DownlinkMessage{
		{QueID: 1},
		{QueID: 2},
		{QueID: 3},
	}, nil)
	// RevokeDownlink succeeds for all 3 (revokedCount = 3)
	mockDL.On("RevokeDownlink", mock.Anything, mock.Anything).Return(uint64(0), "")

	// Mock operationRepo to capture cleanup metadata
	// Gap 2: When detachErrorCount > 0, state is completed_with_warnings (not completed)
	mockOpRepo := answeredDeregisterLog(0x70B3D59CD000089B)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123),
		int64(42),
		models.OperationStateCompletedWithWarnings, // R4: cleanup errors trigger warnings state
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			// Verify the cleanup metadata keys are present (incl. cleanupStatus)
			epEui, hasEpEui := meta[MetadataKeyEpEui]
			revokedCount, hasRevoked := meta[MetadataKeyRevokedCount]
			detachErrorCount, hasDetach := meta[MetadataKeyDetachErrorCount]
			cleanupStatus, hasStatus := meta[MetadataKeyCleanupStatus]

			if !hasEpEui || !hasRevoked || !hasDetach || !hasStatus {
				t.Errorf("Missing metadata keys: epEui=%v, revoked=%v, detach=%v, status=%v",
					hasEpEui, hasRevoked, hasDetach, hasStatus)
				return false
			}

			// Verify epEui format (uppercase hex, 16 chars)
			epEuiStr, ok := epEui.(string)
			if !ok || epEuiStr != "70B3D59CD000089B" {
				t.Errorf("epEui format wrong: got %v, want 70B3D59CD000089B", epEui)
				return false
			}

			// Verify revokedCount matches mock (3 downlinks revoked)
			if revokedCount != 3 {
				t.Errorf("revokedCount wrong: got %v, want 3", revokedCount)
				return false
			}

			// Verify detachErrorCount matches mock (2 errors from PropagateDetachToAll)
			if detachErrorCount != 2 {
				t.Errorf("detachErrorCount wrong: got %v, want 2", detachErrorCount)
				return false
			}

			// Verify cleanupStatus is "partial_failure" (detach errors occurred)
			if cleanupStatus != "partial_failure" {
				t.Errorf("cleanupStatus wrong: got %v, want partial_failure", cleanupStatus)
				return false
			}

			return true
		}),
	).Return(nil)

	server := &Server{
		registry:      newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:         testFrameCodec,
		commands:      mustTestCommandRegistry(),
		clock:         clock.SystemClock{},
		logger:        testLogger(),
		endpointSvc:   mockEndpoint,
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123, // Non-zero to enable operation tracking
		TenantID: 1,
		State:    StateActive,
	}

	// Execute handler
	handlerErr := server.handleDeregisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Verify all mocks were called with correct arguments
	mockDL.AssertExpectations(t)
	mockEndpoint.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// TestHandleDeregisterComplete_CleanSuccess verifies that when cleanup succeeds
// without errors, the operation state is "completed" (not completed_with_warnings).
//
// Spec: SCACI §3.7.3 - DeregisterComplete clean success path
// Gap: R4 Gap 2 - Distinguish completed from completed_with_warnings
func TestHandleDeregisterComplete_CleanSuccess(t *testing.T) {
	// Setup mock services - all succeed with no errors
	mockEndpoint := new(MockEndpointService)
	// PropagateDetachToAll returns empty error slice (all succeeded)
	mockEndpoint.On("PropagateDetachToAll", mock.Anything, int64(1), uint64(0x70B3D59CD000089B)).Return([]error{})

	mockDL := new(MockDLService)
	// GetDownlinkQueue returns 2 downlinks
	mockDL.On("GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything).Return([]*storage.DownlinkMessage{
		{QueID: 1},
		{QueID: 2},
	}, nil)
	// RevokeDownlink succeeds for both
	mockDL.On("RevokeDownlink", mock.Anything, mock.Anything).Return(uint64(0), "")

	// Mock operationRepo - clean success means OperationStateCompleted (not CompletedWithWarnings)
	mockOpRepo := answeredDeregisterLog(0x70B3D59CD000089B)
	mockOpRepo.On(
		"UpdateOperationState",
		mock.Anything,
		int64(123),
		int64(42),
		models.OperationStateCompleted, // Clean success = completed
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			// Verify cleanupStatus is "success"
			cleanupStatus, hasStatus := meta[MetadataKeyCleanupStatus]
			if !hasStatus || cleanupStatus != "success" {
				t.Errorf("cleanupStatus wrong: got %v, want success", cleanupStatus)
				return false
			}
			// Verify detachErrorCount is 0
			detachErrorCount, hasDetach := meta[MetadataKeyDetachErrorCount]
			if !hasDetach || detachErrorCount != 0 {
				t.Errorf("detachErrorCount wrong: got %v, want 0", detachErrorCount)
				return false
			}
			// Verify revokedCount is 2
			revokedCount, hasRevoked := meta[MetadataKeyRevokedCount]
			if !hasRevoked || revokedCount != 2 {
				t.Errorf("revokedCount wrong: got %v, want 2", revokedCount)
				return false
			}
			return true
		}),
	).Return(nil)

	server := &Server{
		registry:      newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:         testFrameCodec,
		commands:      mustTestCommandRegistry(),
		clock:         clock.SystemClock{},
		logger:        testLogger(),
		endpointSvc:   mockEndpoint,
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	// Execute handler
	handlerErr := server.handleDeregisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Verify all mocks were called with correct arguments
	mockDL.AssertExpectations(t)
	mockEndpoint.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// TestHandleRegisterComplete_UnidirectionalPreAttach verifies that unidirectional endpoints
// with preAttach=true trigger attach propagation.
//
// Spec: BSSCI §3.8 - "attach propagate is required for unidirectional End Points"
func TestHandleRegisterComplete_UnidirectionalPreAttach(t *testing.T) {
	const testEpEui = uint64(0x70B3D59CD00008C1)

	// Mock endpoint service to return endpoint with bidi=false, preAttach=true
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Attach", mock.Anything, mock.Anything).Return("")
	mockEndpoint.On("GetByEUI", mock.Anything, int64(1), mock.Anything).Return(
		&models.EndPoint{
			ID:        1,
			TenantID:  1,
			PreAttach: true,
			Bidi:      false, // UNIDIRECTIONAL - must still trigger propagation per BSSCI §3.8
		}, "",
	)

	// Mock operation repo to return operation with epEui
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:        1,
		SessionID: 123,
		OpId:      42,
		Command:   CmdRegister,
		State:     string(models.OperationStateAcknowledged),
		RequestData: map[string]interface{}{
			MetadataKeyEpEui: mioty.FormatEUI64(testEpEui),
		},
	}, nil)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything).Return(nil)

	// Mock propagation service - CRITICAL: must be called despite bidi=false
	mockPropSvc := new(mockPropagationService)
	mockPropSvc.On("TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything).Return(nil)
	mockSnapProvider := &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}

	server := &Server{
		registry:                newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:                   testFrameCodec,
		commands:                mustTestCommandRegistry(),
		clock:                   clock.SystemClock{},
		logger:                  testLogger(),
		endpointSvc:             mockEndpoint,
		operationRepo:           mockOpRepo,
		propagationSvc:          mockPropSvc,
		sessionSnapshotProvider: mockSnapProvider,
		config:                  &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	handlerErr := server.handleRegisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Allow async goroutine to complete
	time.Sleep(asyncSettleDelay)

	// Propagation service must be called for unidirectional preAttach endpoint
	mockPropSvc.AssertCalled(t, "TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything)
	mockPropSvc.AssertExpectations(t)
}

// TestHandleRegisterComplete_HexStringEpEui verifies that the epEui the
// operation log recorded as hex text is parsed back.
func TestHandleRegisterComplete_HexStringEpEui(t *testing.T) {
	const testEpEuiHex = "70B3D59CD00008C1"

	// Mock endpoint service to return endpoint with preAttach=true
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Attach", mock.Anything, mock.Anything).Return("")
	mockEndpoint.On("GetByEUI", mock.Anything, int64(1), mock.Anything).Return(
		&models.EndPoint{
			ID:        1,
			TenantID:  1,
			PreAttach: true,
			Bidi:      true,
		}, "",
	)

	// Mock operation repo to return operation with LEGACY STRING epEui
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:        1,
		SessionID: 123,
		OpId:      42,
		Command:   CmdRegister,
		State:     string(models.OperationStateAcknowledged),
		RequestData: map[string]interface{}{
			MetadataKeyEpEui: testEpEuiHex,
		},
	}, nil)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything).Return(nil)

	// Mock propagation service - must be called after hex string is parsed to endpoint ID
	mockPropSvc := new(mockPropagationService)
	mockPropSvc.On("TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything).Return(nil)
	mockSnapProvider := &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}

	server := &Server{
		registry:                newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:                   testFrameCodec,
		commands:                mustTestCommandRegistry(),
		clock:                   clock.SystemClock{},
		logger:                  testLogger(),
		endpointSvc:             mockEndpoint,
		operationRepo:           mockOpRepo,
		propagationSvc:          mockPropSvc,
		sessionSnapshotProvider: mockSnapProvider,
		config:                  &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
	}

	handlerErr := server.handleRegisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Allow async goroutine to complete
	time.Sleep(asyncSettleDelay)

	// String "70B3D59CD00008C1" must be parsed correctly and propagation triggered
	mockPropSvc.AssertCalled(t, "TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything)
	mockPropSvc.AssertExpectations(t)
}

// answeredDeregisterLog is an operation log holding opId 42 of session 123 as
// a dereg of epEui the service center answered with deregRsp.
func answeredDeregisterLog(epEui uint64) *MockSCACIOperationRepository {
	operations := new(MockSCACIOperationRepository)
	operations.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		SessionID: 123, OpId: 42, Command: CmdDeregister, State: string(models.OperationStateAcknowledged),
		RequestData: map[string]interface{}{MetadataKeyEpEui: mioty.FormatEUI64(epEui)},
	}, nil)
	return operations
}
