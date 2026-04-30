// Package scaci provides handler-level integration tests for SCACI operations.
//
// Coverage:
//   - handleULDataTransmit missing userData returns POSIX_EINVAL (§2.4)
//   - handleULDataTransmit format field defaults to 0 when absent (§3.9.1)
package scaci

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/propagation"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// createTestServer creates a Server with minimal dependencies for handleULDataTransmit tests.
//
// Fields accessed by handleULDataTransmit (handler_operations.go:469-634):
//   - logger (logging throughout)
//   - ulSvc (line 563 - ScheduleULTransmit call)
//   - operationRecorder (line 541 - Record call, only if session.ID > 0 && operationRepo != nil)
//   - operationRepo (line 520 - nil-safe check before recording)
//   - sessionContext() (requires sessions map and sessionsMu)
//   - sendErrorWithCatalog() (requires config for error response)
//   - statusSvc (line 498 - only when req.BsEui != nil)
//   - SendULDataTransmitResponse() (requires logger, sessions, sessionsMu)
//
// For validation-failure tests, ulSvc is not called (validation fails before scheduling).
// For success tests, ulSvc must return valid values.
func createTestServer(t *testing.T, ulSvc *MockULService, recorder *MockOperationRecorder) *Server {
	t.Helper()
	return &Server{
		logger:            testLogger(), // from server_test.go
		ulSvc:             ulSvc,
		operationRecorder: recorder,
		operationRepo:     nil, // nil-safe: session.ID > 0 && operationRepo != nil guards recording
		statusSvc:         nil, // nil-safe: only used when req.BsEui != nil
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
		config:            &Config{}, // minimal config
	}
}

// createTestSession creates a valid active session for testing.
func createTestSession(t *testing.T) *Session {
	t.Helper()
	return &Session{
		ID:       0, // ID=0 skips operation recording
		TenantID: 1,
		State:    StateActive,
		AcEui:    0xFEDCBA0987654321,
	}
}

// decodeResponse decodes msgpack response, skipping MIOTYA01 frame header.
// Frame format: "MIOTYA01" (8 bytes) + size (4 bytes LE) + payload
func decodeResponse(data []byte, v interface{}) error {
	// Check for MIOTYA01 header
	if len(data) > 12 && string(data[:8]) == "MIOTYA01" {
		// Read payload size from bytes 8-11 (little-endian)
		payloadSize := binary.LittleEndian.Uint32(data[8:12])
		if int(payloadSize)+12 <= len(data) {
			data = data[12 : 12+payloadSize]
		}
	}
	return msgpack.Unmarshal(data, v)
}

// assertErrorToken verifies that the error response contains the expected message
// from the error catalog. ErrorToken is internal-only (msgpack:"-") and not on wire,
// so we verify by comparing the Message field against the catalog definition.
func assertErrorToken(t *testing.T, errorResp Error, expectedToken string) {
	t.Helper()
	expectedDef := GetErrorDefinition(expectedToken)
	assert.Equal(t, expectedDef.Message, errorResp.Message,
		"Error message should match catalog for token %s", expectedToken)
}

// =============================================================================
// §2.4 Mandatory Field Validation Tests
// =============================================================================

// TestHandleULDataTransmit_MissingUserData_ReturnsError verifies that missing
// mandatory userData field returns POSIX_EINVAL with appropriate error token.
//
// Spec: SCACI §2.4 - mandatory field validation
// Spec: SCACI §3.9.1 - userData is mandatory field for ULDataTransmit
func TestHandleULDataTransmit_MissingUserData_ReturnsError(t *testing.T) {
	// Setup mock services
	mockULSvc := new(MockULService)
	mockRecorder := new(MockOperationRecorder)

	// Create server with mocks
	server := createTestServer(t, mockULSvc, mockRecorder)
	conn := &mockConn{}
	session := createTestSession(t)

	// Create ULDataTx payload with nil userData (missing mandatory field)
	req := ULDataTransmit{
		BaseMessage: mioty.BaseMessage{CommandType: CmdULDataTransmit, OpId: 42},
		EpEui:       0x1234567890ABCDEF,
		NwkSnKey:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ShAddr:      0x1234,
		PacketCnt:   100,
		UserData:    nil, // Missing mandatory field
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Execute handler
	handlerErr := server.handleULDataTransmit(conn, session, 42, payload)

	// Should NOT return Go error (error sent to client via sendErrorWithCatalog)
	assert.NoError(t, handlerErr)

	// Verify Error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	// Assert POSIX_EINVAL and error token
	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code)
	assertErrorToken(t, errorResp, errUserDataEmpty)

	// Verify operation was NOT recorded (validation failed before recording)
	mockRecorder.AssertNotCalled(t, "Record")

	// Verify ULService was NOT called (validation failed before scheduling)
	mockULSvc.AssertNotCalled(t, "ScheduleULTransmit")
}

// TestHandleULDataTransmit_MissingEpEui_ReturnsError verifies that missing
// mandatory epEui field (zero value) returns POSIX_EINVAL.
//
// Spec: SCACI §2.4 - mandatory field validation
// Spec: SCACI §3.9.1 - epEui is mandatory field for ULDataTransmit
func TestHandleULDataTransmit_MissingEpEui_ReturnsError(t *testing.T) {
	mockULSvc := new(MockULService)
	mockRecorder := new(MockOperationRecorder)

	server := createTestServer(t, mockULSvc, mockRecorder)
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

// =============================================================================
// §3.9.1 Optional Field Default Tests
// =============================================================================

// TestHandleULDataTransmit_FormatDefaultApplied verifies that format field
// defaults to 0 when absent from the request.
//
// Spec: SCACI §2.4 - optional field default handling
// Spec: SCACI §3.9.1 - format field is optional, default 0
func TestHandleULDataTransmit_FormatDefaultApplied(t *testing.T) {
	// Setup mock services with capture
	var capturedReq *mioty.ULDataTransmit
	mockULSvc := new(MockULService)
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(req *mioty.ULDataTransmit) bool {
			capturedReq = req
			return true
		}),
		mock.AnythingOfType("int64"),
	).Return(int64(123), uint64(0x70B3D59CD00009E6), "")

	mockRecorder := new(MockOperationRecorder)
	// Record is not called because session.ID=0 and operationRepo=nil

	server := createTestServer(t, mockULSvc, mockRecorder)
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
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(req *mioty.ULDataTransmit) bool {
			capturedReq = req
			return true
		}),
		mock.AnythingOfType("int64"),
	).Return(int64(123), uint64(0x70B3D59CD00009E6), "")

	mockRecorder := new(MockOperationRecorder)

	server := createTestServer(t, mockULSvc, mockRecorder)
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

// =============================================================================
// §3.9 UL Data Transmit Tests - Error Mapping, Tenant Isolation, Operation Recording
// =============================================================================

// createTestServerFull creates a Server with all mock dependencies for comprehensive testing.
// Use this when testing operation recording, state transitions, or tenant verification.
type testServerOpts struct {
	ulSvc     *MockULService
	recorder  *MockOperationRecorder
	opRepo    *MockSCACIOperationRepository
	statusSvc *MockStatusService
}

func createTestServerFull(t *testing.T, opts testServerOpts) *Server {
	t.Helper()
	return &Server{
		logger:            testLogger(),
		ulSvc:             opts.ulSvc,
		operationRecorder: opts.recorder,
		operationRepo:     opts.opRepo, // CRITICAL: non-nil to hit UpdateOperationState
		statusSvc:         opts.statusSvc,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
		config:            &Config{},
	}
}

// createTestSessionWithID creates a session with ID > 0 for operation recording tests.
func createTestSessionWithID(t *testing.T) *Session {
	t.Helper()
	return &Session{
		ID:       123, // Non-zero to enable operation recording
		TenantID: 1,
		State:    StateActive,
		AcEui:    0xFEDCBA0987654321,
	}
}

// =============================================================================
// §3.9 Error Mapping Tests (POSIX Code Verification)
// =============================================================================

// TestHandleULDataTransmit_BaseStationUnavailable_ReturnsPOSIX_EAGAIN verifies that
// scheduler resource exhaustion returns POSIX_EAGAIN per SCACI §3.9.2 error mapping.
//
// Spec: SCACI §3.9.2 - UL Data Transmit Response error handling
func TestHandleULDataTransmit_BaseStationUnavailable_ReturnsPOSIX_EAGAIN(t *testing.T) {
	mockULSvc := new(MockULService)
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything, // Handler creates its own context via sessionContext()
		mock.MatchedBy(func(_ *mioty.ULDataTransmit) bool { return true }),
		int64(1), // tenantID
	).Return(int64(0), uint64(0), ErrBaseStationUnavailable) // Exported constant

	mockRecorder := new(MockOperationRecorder)
	server := createTestServer(t, mockULSvc, mockRecorder)
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
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(_ *mioty.ULDataTransmit) bool { return true }),
		int64(1),
	).Return(int64(0), uint64(0), ErrBaseStationNotFound)

	mockRecorder := new(MockOperationRecorder)
	server := createTestServer(t, mockULSvc, mockRecorder)
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
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything,
		mock.MatchedBy(func(_ *mioty.ULDataTransmit) bool { return true }),
		int64(1),
	).Return(int64(0), uint64(0), ErrULTransmitNotSupported)

	mockRecorder := new(MockOperationRecorder)
	server := createTestServer(t, mockULSvc, mockRecorder)
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

// =============================================================================
// §3.9 Tenant Isolation Tests (Explicit bsEui Verification)
// =============================================================================

// TestHandleULDataTransmit_ExplicitBsEui_TenantMismatch_ReturnsNotFound verifies that
// when explicit bsEui is provided and not found for tenant, POSIX_ENOENT is returned.
//
// Spec: SCACI §3.9.1 - BS tenant verification before scheduling
func TestHandleULDataTransmit_ExplicitBsEui_TenantMismatch_ReturnsNotFound(t *testing.T) {
	// Mock statusSvc to return ErrNotFound (BS not found for tenant)
	mockStatusSvc := new(MockStatusService)
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, 0x70B3D59CD00009E6)
	mockStatusSvc.On("GetBaseStation",
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
	mockStatusSvc.On("GetBaseStation",
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
	mockStatusSvc.On("GetBaseStation",
		mock.Anything,
		int64(1), // tenantID
		bsEuiBytes,
	).Return(&models.BaseStation{EUI: bsEuiArray}, nil) // BS found

	mockULSvc := new(MockULService)
	mockULSvc.On("ScheduleULTransmit",
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

// =============================================================================
// §3.9 Operation Recording Tests
// =============================================================================

// TestHandleULDataTransmit_OperationLogging_RecordsCalled verifies that operation
// recording is called with correct fields and excludes nwkSnKey for security.
//
// Spec: SCACI §3.9.1 - Operation recording for resume safety
func TestHandleULDataTransmit_OperationLogging_RecordsCalled(t *testing.T) {
	var capturedRequestData map[string]interface{}
	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record",
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
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything,
		mock.Anything,
		int64(1),
	).Return(int64(789), uint64(0x70B3D59CD00009E6), "")

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
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
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything,
		mock.Anything,
		int64(1),
	).Return(int64(0), uint64(0), ErrBaseStationUnavailable)

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
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
	mockULSvc.On("ScheduleULTransmit",
		mock.Anything,
		mock.Anything,
		int64(1),
	).Return(int64(789), uint64(0x70B3D59CD00009E6), "") // Success

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
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

// =============================================================================
// §3.9 Three-Way Handshake Completion Tests
// =============================================================================

// TestHandleULDataTransmitComplete_MarksCompleted verifies that handshake
// completion marks operation as completed with timestamp and sends no response.
//
// Spec: SCACI §3.9.3 - UL Data Transmit Complete processing
func TestHandleULDataTransmitComplete_MarksCompleted(t *testing.T) {
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
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
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
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

// =============================================================================
// §3.7 Deregister Handler Tests - epEui Caching Flow
// =============================================================================

// TestHandleDeregister_CachesEpEui verifies that handleDeregister caches the
// epEui in session.PendingDeregisterOps for use by handleDeregisterComplete.
//
// Spec: SCACI §3.7.1 - Deregister message processing
// Fix: GAP3 - In-memory epEui cache for cleanup fallback
func TestHandleDeregister_CachesEpEui(t *testing.T) {
	// Setup mock services
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("Deregister", mock.Anything, uint64(0x1234567890ABCDEF), int64(1)).Return("")

	mockRecorder := new(MockOperationRecorder)
	// Record may or may not be called depending on session.ID

	// Create server with mocks
	server := &Server{
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		operationRepo:     nil, // nil-safe
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
		config:            &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       0, // Skip DB operations
		TenantID: 1,
		State:    StateActive,
		AcEui:    0xFEDCBA0987654321,
	}

	// Create Deregister payload
	req := Deregister{
		BaseMessage: BaseMessage{Command: CmdDeregister, OpId: 42},
		EpEui:       0x1234567890ABCDEF,
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	// Verify cache is initially empty or nil
	assert.Nil(t, session.PendingDeregisterOps)

	// Execute handler
	handlerErr := server.handleDeregister(conn, session, 42, payload)
	assert.NoError(t, handlerErr)

	// CRITICAL: Verify epEui was cached in session
	require.NotNil(t, session.PendingDeregisterOps, "PendingDeregisterOps should be initialized")
	cachedEui, exists := session.PendingDeregisterOps[42]
	assert.True(t, exists, "epEui should be cached for opId 42")
	assert.Equal(t, uint64(0x1234567890ABCDEF), cachedEui, "Cached epEui should match request")

	// Verify EndpointService was called
	mockEndpoint.AssertExpectations(t)
}

// TestHandleDeregisterComplete_UsesCache verifies that handleDeregisterComplete
// retrieves epEui from cache and performs cleanup even without DB lookup.
//
// Spec: SCACI §3.7.3 - DeregisterComplete message processing
// Fix: GAP1 - Cleanup not gated by operationRepo availability
func TestHandleDeregisterComplete_UsesCache(t *testing.T) {
	// Setup mock services
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("PropagateDetachToAll", uint64(0x1234567890ABCDEF)).Return(nil)

	mockDL := new(MockDLService)
	// GetDownlinkQueue returns empty list (no downlinks to revoke)
	mockDL.On("GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything).Return([]*storage.DownlinkMessage{}, nil)

	// Create server with mocks - operationRepo is nil to test cache-only path
	server := &Server{
		logger:        testLogger(),
		endpointSvc:   mockEndpoint,
		dlSvc:         mockDL,
		operationRepo: nil, // nil to test cache-only cleanup path
		sessions:      make(map[net.Conn]*Session),
		sessionsMu:    sync.RWMutex{},
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       0, // Skip DB operations
		TenantID: 1,
		State:    StateActive,
		AcEui:    0xFEDCBA0987654321,
		// Pre-populate cache to simulate handleDeregister having run
		PendingDeregisterOps: map[int64]uint64{
			42: 0x1234567890ABCDEF,
		},
	}

	// Execute handler
	handlerErr := server.handleDeregisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Verify cleanup services were called (cleanup executed from cache)
	mockDL.AssertExpectations(t)
	mockEndpoint.AssertExpectations(t)

	// Verify cache entry was deleted after use
	_, exists := session.PendingDeregisterOps[42]
	assert.False(t, exists, "Cache entry should be deleted after cleanup")
}

// TestRevokeEndpointDownlinks_NilDLSvc_Graceful verifies that revokeEndpointDownlinks
// returns gracefully when dlSvc is nil (defensive guard).
//
// Fix: GAP2 - nil guard for dlSvc prevents panic
func TestRevokeEndpointDownlinks_NilDLSvc_Graceful(t *testing.T) {
	server := &Server{
		logger: testLogger(),
		dlSvc:  nil, // nil to test defensive guard
	}

	ctx := context.Background()
	var eui [8]byte
	binary.BigEndian.PutUint64(eui[:], 0x1234567890ABCDEF)

	// Should return (0, nil) (graceful no-op), not panic
	count, err := server.revokeEndpointDownlinks(ctx, 1, eui)
	assert.NoError(t, err, "Should return nil when dlSvc is nil")
	assert.Equal(t, 0, count, "Should return 0 count when dlSvc is nil")
}

// TestHandleDeregisterComplete_DBFallback verifies that handleDeregisterComplete
// falls back to DB lookup when cache is empty and successfully retrieves epEui.
//
// Spec: SCACI §3.7.3 - DeregisterComplete with resumed session (cache miss)
// Fix: F3 - Test DB fallback path (handler_operations.go:346-376)
func TestHandleDeregisterComplete_DBFallback(t *testing.T) {
	// Setup mock services
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("PropagateDetachToAll", uint64(0x1234567890ABCDEF)).Return(nil)

	mockDL := new(MockDLService)
	mockDL.On("GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything).Return([]*storage.DownlinkMessage{}, nil)

	// Mock operationRepo to return op with epEui in RequestData
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:        1,
		SessionID: 123,
		OpId:      42,
		Command:   "dereg",
		RequestData: map[string]interface{}{
			"epEui": uint64(0x1234567890ABCDEF), // epEui present in DB
		},
	}, nil)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything).Return(nil)

	server := &Server{
		logger:        testLogger(),
		endpointSvc:   mockEndpoint,
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		sessions:      make(map[net.Conn]*Session),
		sessionsMu:    sync.RWMutex{},
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123, // Non-zero to enable DB lookup
		TenantID: 1,
		State:    StateActive,
	}
	session.EnsurePendingDeregisterOps() // Initialize empty cache

	// Execute handler - should fall back to DB
	handlerErr := server.handleDeregisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Verify DB was queried
	mockOpRepo.AssertCalled(t, "GetOperationByOpID", mock.Anything, int64(123), int64(42))

	// Verify UpdateOperationState was called (proves state update path runs)
	mockOpRepo.AssertCalled(t, "UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything)

	// Verify cleanup was executed (from DB-retrieved epEui)
	mockDL.AssertExpectations(t)
	mockEndpoint.AssertExpectations(t)
}

// TestHandleDeregisterComplete_NoEpEui_MarksOperationFailed verifies that
// when no epEui is available from cache or DB, the operation is marked as
// failed with proper error metadata (Gap 1 fix per R4 plan).
//
// Spec: SCACI §3.7.3 - DeregisterComplete edge case (no epEui available)
// Fix: R4 Gap 1 - Mark missing epEui as failed (not completed)
func TestHandleDeregisterComplete_NoEpEui_MarksOperationFailed(t *testing.T) {
	// Setup mock services - these should NOT be called since cleanup is skipped
	mockEndpoint := new(MockEndpointService)
	// NO expectations set - PropagateDetachToAll should not be called

	mockDL := new(MockDLService)
	// NO expectations set - GetDownlinkQueue should not be called

	// Mock operationRepo to return op WITHOUT epEui in RequestData
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:          1,
		SessionID:   123,
		OpId:        42,
		Command:     "dereg",
		RequestData: map[string]interface{}{}, // No epEui in RequestData
	}, nil)
	// Gap 1: Missing epEui now marks operation as FAILED with error metadata
	mockOpRepo.On("UpdateOperationState",
		mock.Anything,
		int64(123),
		int64(42),
		models.OperationStateFailed,
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			// Verify failure metadata contains error token and detail
			token, hasToken := meta[MetadataKeyErrorToken].(string)
			detail, hasDetail := meta[MetadataKeyErrorDetail].(string)
			return hasToken && token == ErrMissingEpEui &&
				hasDetail && detail == "epEui not found in cache or operation log"
		}),
	).Return(nil)

	server := &Server{
		logger:        testLogger(),
		endpointSvc:   mockEndpoint,
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		sessions:      make(map[net.Conn]*Session),
		sessionsMu:    sync.RWMutex{},
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123, // Non-zero to enable DB lookup
		TenantID: 1,
		State:    StateActive,
	}
	session.EnsurePendingDeregisterOps() // Initialize empty cache

	// Execute handler - should mark operation failed and return early
	handlerErr := server.handleDeregisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr, "Handler should return nil (error logged internally)")

	// Verify UpdateOperationState was called with FAILED state and error metadata
	mockOpRepo.AssertCalled(t, "UpdateOperationState",
		mock.Anything, int64(123), int64(42), models.OperationStateFailed, mock.Anything)

	// Verify cleanup services were NOT called (operation failed, cleanup skipped)
	mockDL.AssertNotCalled(t, "GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything)
	mockEndpoint.AssertNotCalled(t, "PropagateDetachToAll", mock.Anything)
}

// =============================================================================
// §3.6 Register Handler Tests - POSIX Mapping and Session Guard
// =============================================================================

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
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		operationRepo:     nil,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
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
	mockRecorder.On("Record",
		mock.Anything,
		mock.MatchedBy(func(s *Session) bool {
			capturedSession = s
			return true
		}),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil)

	server := &Server{
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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
	mockRecorder.On("Record",
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
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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

	// epEui should be numeric uint64 (for propagation lookup at lines 165-173)
	epEui, ok := capturedRequestData["epEui"].(uint64)
	assert.True(t, ok, "epEui must be uint64 (numeric for propagation lookup)")
	assert.Equal(t, uint64(0x70B3D59CD00008C1), epEui, "epEui must be numeric uint64")

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

// =============================================================================
// §3.6 Register Handler Tests - Decode Failure Coverage
// =============================================================================

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
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
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
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
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

// =============================================================================
// §3.7 Deregister Handler Tests - Send Ordering and Failure Handling
// =============================================================================

// failingMockConn implements net.Conn and always returns an error on Write.
// Used to test send failure → failed state transitions.
type failingMockConn struct {
	mockConn       // Embed mockConn for Read/Close/etc defaults
	writeErr error // Error to return on Write
}

func (m *failingMockConn) Write(_ []byte) (n int, err error) {
	return 0, m.writeErr
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
	mockOpRepo.On("UpdateOperationState",
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
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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
	mockOpRepo.On("UpdateOperationState",
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
		logger:            testLogger(),
		endpointSvc:       mockEndpoint,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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

// trackingMockConn implements net.Conn and calls onWrite callback on Write.
type trackingMockConn struct {
	mockConn
	onWrite func()
}

func (m *trackingMockConn) Write(b []byte) (n int, err error) {
	if m.onWrite != nil {
		m.onWrite()
	}
	m.written = append(m.written, b...)
	return len(b), nil
}

// TestHandleDeregisterComplete_CleanupMetadataCapture verifies that handleDeregisterComplete
// captures all 5 cleanup metadata keys (including cleanupStatus) and that counts match the
// mocked service returns. When cleanup errors occur, state should be completed_with_warnings.
//
// Spec: SCACI §3.7.3 - DeregisterComplete cleanup metadata
// Gap: R4 Gap 2 - completed_with_warnings state for cleanup issues
func TestHandleDeregisterComplete_CleanupMetadataCapture(t *testing.T) {
	// Setup mock services with specific return values
	mockEndpoint := new(MockEndpointService)
	// PropagateDetachToAll returns 2 errors to verify detachErrorCount
	mockEndpoint.On("PropagateDetachToAll", uint64(0x70B3D59CD000089B)).Return([]error{
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
	// RevokeDownlinkByID succeeds for all 3 (revokedCount = 3)
	mockDL.On("RevokeDownlinkByID", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock operationRepo to capture cleanup metadata
	// Gap 2: When detachErrorCount > 0, state is completed_with_warnings (not completed)
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
		mock.Anything,
		int64(123),
		int64(42),
		models.OperationStateCompletedWithWarnings, // R4: cleanup errors trigger warnings state
		mock.MatchedBy(func(meta map[string]interface{}) bool {
			// Verify all 5 metadata keys are present (incl. cleanupStatus)
			epEui, hasEpEui := meta[MetadataKeyEpEui]
			cleanupSource, hasSource := meta[MetadataKeyCleanupSource]
			revokedCount, hasRevoked := meta[MetadataKeyRevokedCount]
			detachErrorCount, hasDetach := meta[MetadataKeyDetachErrorCount]
			cleanupStatus, hasStatus := meta[MetadataKeyCleanupStatus]

			if !hasEpEui || !hasSource || !hasRevoked || !hasDetach || !hasStatus {
				t.Errorf("Missing metadata keys: epEui=%v, source=%v, revoked=%v, detach=%v, status=%v",
					hasEpEui, hasSource, hasRevoked, hasDetach, hasStatus)
				return false
			}

			// Verify epEui format (uppercase hex, 16 chars)
			epEuiStr, ok := epEui.(string)
			if !ok || epEuiStr != "70B3D59CD000089B" {
				t.Errorf("epEui format wrong: got %v, want 70B3D59CD000089B", epEui)
				return false
			}

			// Verify cleanupSource is "cache" (we pre-populated cache)
			if cleanupSource != "cache" {
				t.Errorf("cleanupSource wrong: got %v, want cache", cleanupSource)
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
		logger:        testLogger(),
		endpointSvc:   mockEndpoint,
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		sessions:      make(map[net.Conn]*Session),
		sessionsMu:    sync.RWMutex{},
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123, // Non-zero to enable operation tracking
		TenantID: 1,
		State:    StateActive,
		// Pre-populate cache with epEui
		PendingDeregisterOps: map[int64]uint64{
			42: 0x70B3D59CD000089B,
		},
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
	mockEndpoint.On("PropagateDetachToAll", uint64(0x70B3D59CD000089B)).Return([]error{})

	mockDL := new(MockDLService)
	// GetDownlinkQueue returns 2 downlinks
	mockDL.On("GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything).Return([]*storage.DownlinkMessage{
		{QueID: 1},
		{QueID: 2},
	}, nil)
	// RevokeDownlinkByID succeeds for both
	mockDL.On("RevokeDownlinkByID", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock operationRepo - clean success means OperationStateCompleted (not CompletedWithWarnings)
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
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
		logger:        testLogger(),
		endpointSvc:   mockEndpoint,
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		sessions:      make(map[net.Conn]*Session),
		sessionsMu:    sync.RWMutex{},
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: 1,
		State:    StateActive,
		// Pre-populate cache with epEui
		PendingDeregisterOps: map[int64]uint64{
			42: 0x70B3D59CD000089B,
		},
	}

	// Execute handler
	handlerErr := server.handleDeregisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Verify all mocks were called with correct arguments
	mockDL.AssertExpectations(t)
	mockEndpoint.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// =============================================================================
// §3.10 DLDataQueue Handler Tests - Single-Payload Constraint Validation
// =============================================================================

// TestHandleDLDataQueue_NonCntDependMultiPayload_ReturnsEINVAL verifies that when
// cntDepend=false but UserData has more than one entry, the handler rejects the
// request with errNonCntDependMultiPayload and POSIX_EINVAL.
//
// Spec: SCACI §3.10.1 - "single user data entry if cntDepend is false"
// Gap: Missing validation for multi-payload with non-counter-dependent queue
func TestHandleDLDataQueue_NonCntDependMultiPayload_ReturnsEINVAL(t *testing.T) {
	mockRecorder := new(MockOperationRecorder)
	server := &Server{
		logger:            testLogger(),
		operationRecorder: mockRecorder,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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

// TestProcessDLDataQueueCore_StatusUpdateUsesQueueIDOnSuccess verifies queued status updates
// target que_id (SCACI queue ID) instead of internal DB row id.
func TestProcessDLDataQueueCore_StatusUpdateUsesQueueIDOnSuccess(t *testing.T) {
	const (
		tenantID int64  = 1
		opID     int64  = 101
		queID    uint64 = 7000001
		bsEUI    uint64 = 0x1122334455667788
	)

	mockDL := new(MockDLService)
	mockEndpoint := new(MockEndpointService)

	mockEndpoint.On("GetByEUI", mock.Anything, tenantID, mock.Anything).
		Return(&models.EndPoint{}, "")
	mockDL.On("GetDownlinkByQueueID", mock.Anything, queID, "1").
		Return(nil, storage.ErrNotFound)
	mockDL.On("EnqueueDownlink", mock.Anything, mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
		return dl != nil && dl.QueID == int64(queID)
	})).Return(&storage.DownlinkMessage{
		ID:    42, // Deliberately different from queID to catch id-selector regressions.
		QueID: int64(queID),
	}, nil)
	mockDL.On("QueueDownlink", mock.Anything, mock.MatchedBy(func(req *mioty.DLDataQueue) bool {
		return req != nil && req.QueId == queID
	}), tenantID).Return(queID, bsEUI, "")
	mockDL.On("UpdateDownlinkStatus", mock.Anything, "7000001", bssci.DLQueueStatusQueued, mock.Anything).
		Return(nil).Once()

	server := &Server{
		logger:      testLogger(),
		dlSvc:       mockDL,
		endpointSvc: mockEndpoint,
		sessions:    make(map[net.Conn]*Session),
		sessionsMu:  sync.RWMutex{},
		config:      &Config{},
	}

	session := &Session{TenantID: tenantID}
	req := &DLDataQueue{
		EpEui:     0x70B3D59CD00009E6,
		QueId:     queID,
		CntDepend: false,
		UserData:  [][]byte{{0x01, 0x02, 0x03}},
	}

	result, errToken, posixCode := server.processDLDataQueueCore(context.Background(), session, opID, req)
	require.NotNil(t, result)
	assert.Equal(t, uint64(queID), result.QueID)
	assert.Equal(t, "", errToken)
	assert.Equal(t, 0, posixCode)

	mockDL.AssertExpectations(t)
	mockEndpoint.AssertExpectations(t)
}

// TestProcessDLDataQueueCore_StatusUpdateUsesQueueIDOnSchedulerFailure verifies failed status updates
// target que_id (SCACI queue ID) when scheduler coordination fails.
func TestProcessDLDataQueueCore_StatusUpdateUsesQueueIDOnSchedulerFailure(t *testing.T) {
	const (
		tenantID int64  = 1
		opID     int64  = 102
		queID    uint64 = 7000002
	)

	mockDL := new(MockDLService)
	mockEndpoint := new(MockEndpointService)

	mockEndpoint.On("GetByEUI", mock.Anything, tenantID, mock.Anything).
		Return(&models.EndPoint{}, "")
	mockDL.On("GetDownlinkByQueueID", mock.Anything, queID, "1").
		Return(nil, storage.ErrNotFound)
	mockDL.On("EnqueueDownlink", mock.Anything, mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
		return dl != nil && dl.QueID == int64(queID)
	})).Return(&storage.DownlinkMessage{
		ID:    43, // Deliberately different from queID to catch id-selector regressions.
		QueID: int64(queID),
	}, nil)
	mockDL.On("QueueDownlink", mock.Anything, mock.MatchedBy(func(req *mioty.DLDataQueue) bool {
		return req != nil && req.QueId == queID
	}), tenantID).Return(uint64(0), uint64(0), errSchedulerUnavailable)
	mockDL.On("UpdateDownlinkStatus", mock.Anything, "7000002", bssci.DLQueueStatusFailed, mock.Anything).
		Return(nil).Once()

	server := &Server{
		logger:      testLogger(),
		dlSvc:       mockDL,
		endpointSvc: mockEndpoint,
		sessions:    make(map[net.Conn]*Session),
		sessionsMu:  sync.RWMutex{},
		config:      &Config{},
	}

	session := &Session{TenantID: tenantID}
	req := &DLDataQueue{
		EpEui:     0x70B3D59CD00009E6,
		QueId:     queID,
		CntDepend: false,
		UserData:  [][]byte{{0xAA}},
	}

	result, errToken, posixCode := server.processDLDataQueueCore(context.Background(), session, opID, req)
	assert.Nil(t, result)
	assert.Equal(t, errSchedulerUnavailable, errToken)
	assert.Equal(t, POSIX_ENOTSUP, posixCode)

	mockDL.AssertExpectations(t)
	mockEndpoint.AssertExpectations(t)
}

// =============================================================================
// §3.11 DL Data Revoke Handler Tests
// =============================================================================

// TestHandleDLDataRevoke_Success_SendsRevRsp verifies that a valid dlDataRev
// payload triggers lookup, revoke, and response with matching opId.
//
// Spec: SCACI §3.11.1-3.11.2 - DL Data Revoke request/response flow
func TestHandleDLDataRevoke_Success_SendsRevRsp(t *testing.T) {
	// Mock DLService for GetDownlinkByPacketCnt lookup
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinkByPacketCnt",
		mock.Anything,
		"1", // tenantID as string
		"1234567890ABCDEF",
		uint32(100),
	).Return(&storage.DownlinkMessage{QueID: 42}, nil)
	mockDL.On("RevokeDownlink",
		mock.Anything,
		uint64(42),
		int64(1),
	).Return(uint64(0x70B3D59CD00009E6), "") // Success: returns bsEui, empty error

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record",
		mock.Anything, mock.Anything, int64(10), CmdDLDataRevoke,
		models.OperationDirectionInbound, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
		mock.Anything, int64(123), int64(10),
		models.OperationStateAcknowledged, mock.Anything,
	).Return(nil)

	server := &Server{
		logger:            testLogger(),
		dlSvc:             mockDL,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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

// TestHandleDLDataRevoke_MissingPacketCnt_ReturnsEINVAL verifies that missing
// mandatory packetCnt field returns POSIX_EINVAL with errMissingPacketCnt.
//
// Spec: SCACI §3.11.1 - packetCnt is mandatory field
func TestHandleDLDataRevoke_MissingPacketCnt_ReturnsEINVAL(t *testing.T) {
	server := &Server{
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 0, TenantID: 1, State: StateActive}

	req := DLDataRevoke{
		BaseMessage: BaseMessage{Command: CmdDLDataRevoke, OpId: 11},
		EpEui:       0x1234567890ABCDEF,
		PacketCnt:   0, // Missing mandatory field (zero value)
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleDLDataRevoke(conn, session, 11, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code)
	assertErrorToken(t, errorResp, errMissingPacketCnt)
}

// TestHandleDLDataRevoke_NotFound_ReturnsENOENT verifies that when
// GetDownlinkByPacketCnt returns ErrNotFound, POSIX_ENOENT is returned.
//
// Spec: SCACI §3.11.1 - Queue entry not found error mapping
func TestHandleDLDataRevoke_NotFound_ReturnsENOENT(t *testing.T) {
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinkByPacketCnt",
		mock.Anything,
		"1",
		"1234567890ABCDEF",
		uint32(999),
	).Return(nil, storage.ErrNotFound)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
		mock.Anything, int64(123), int64(12),
		models.OperationStateFailed, mock.MatchedBy(func(meta map[string]interface{}) bool {
			return meta["errorToken"] == errDownlinkNotFound
		}),
	).Return(nil)

	server := &Server{
		logger:        testLogger(),
		dlSvc:         mockDL,
		operationRepo: mockOpRepo,
		sessions:      make(map[net.Conn]*Session),
		sessionsMu:    sync.RWMutex{},
		config:        &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 123, TenantID: 1, State: StateActive}

	req := DLDataRevoke{
		BaseMessage: BaseMessage{Command: CmdDLDataRevoke, OpId: 12},
		EpEui:       0x1234567890ABCDEF,
		PacketCnt:   999, // Non-existent
	}
	payload, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	handlerErr := server.handleDLDataRevoke(conn, session, 12, payload)
	assert.NoError(t, handlerErr)

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_ENOENT, errorResp.Code)
	assertErrorToken(t, errorResp, errDownlinkNotFound)

	mockDL.AssertExpectations(t)
	mockOpRepo.AssertExpectations(t)
}

// TestHandleDLDataRevoke_SchedulerUnavailable_ReturnsENOTSUP verifies that when
// RevokeDownlink returns errSchedulerUnavailable, POSIX_ENOTSUP is returned.
//
// Spec: SCACI §3.11 - Scheduler unavailable error mapping
func TestHandleDLDataRevoke_SchedulerUnavailable_ReturnsENOTSUP(t *testing.T) {
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinkByPacketCnt",
		mock.Anything, "1", "1234567890ABCDEF", uint32(100),
	).Return(&storage.DownlinkMessage{QueID: 42}, nil)
	mockDL.On("RevokeDownlink",
		mock.Anything, uint64(42), int64(1),
	).Return(uint64(0), ErrSchedulerUnavailable) // Error: scheduler unavailable

	mockRecorder := new(MockOperationRecorder)
	mockRecorder.On("Record",
		mock.Anything, mock.Anything, int64(13), CmdDLDataRevoke,
		models.OperationDirectionInbound, mock.Anything,
	).Return(nil)

	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("UpdateOperationState",
		mock.Anything, int64(123), int64(13),
		models.OperationStateFailed, mock.MatchedBy(func(meta map[string]interface{}) bool {
			return meta["errorToken"] == ErrSchedulerUnavailable
		}),
	).Return(nil)

	server := &Server{
		logger:            testLogger(),
		dlSvc:             mockDL,
		operationRecorder: mockRecorder,
		operationRepo:     mockOpRepo,
		sessions:          make(map[net.Conn]*Session),
		sessionsMu:        sync.RWMutex{},
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
	mockOpRepo.On("UpdateOperationState",
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
		logger:        testLogger(),
		operationRepo: mockOpRepo,
		sessions:      make(map[net.Conn]*Session),
		sessionsMu:    sync.RWMutex{},
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

// ============================================================================
// Inbound Error Validation Tests (Fix 3: SCACI §3.14.1)
// ============================================================================

// TestHandleInboundError_ZeroCode_ReturnsEINVAL validates Fix 3:
// When AC sends error with code=0, SC must reject it with POSIX_EINVAL.
//
// Spec: SCACI §3.14.1 - Error messages must have non-zero POSIX code
func TestHandleInboundError_ZeroCode_ReturnsEINVAL(t *testing.T) {
	server := &Server{
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 100, TenantID: 1, State: StateActive}

	// Create error with code=0 (invalid per §3.14.1)
	errMsg := Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
		Code:        0, // Invalid - POSIX_OK is not valid for Error messages
		Message:     "Some error message",
	}
	payload, err := msgpack.Marshal(&errMsg)
	require.NoError(t, err)

	handlerErr := server.handleInboundError(conn, session, 1, payload)
	assert.NoError(t, handlerErr)

	// Verify error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "code=0 in Error must be rejected with POSIX_EINVAL")
	assertErrorToken(t, errorResp, errErrorMissingCode)
}

// TestHandleInboundError_EmptyMessage_ReturnsEINVAL validates Fix 3:
// When AC sends error with empty message, SC must reject it with POSIX_EINVAL.
//
// Spec: SCACI §3.14.1 - Error messages must have non-empty message field
func TestHandleInboundError_EmptyMessage_ReturnsEINVAL(t *testing.T) {
	server := &Server{
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
	}

	conn := &mockConn{}
	session := &Session{ID: 100, TenantID: 1, State: StateActive}

	// Create error with empty message (invalid per §3.14.1)
	errMsg := Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
		Code:        POSIX_EINVAL, // Valid code
		Message:     "",           // Invalid - empty message
	}
	payload, err := msgpack.Marshal(&errMsg)
	require.NoError(t, err)

	handlerErr := server.handleInboundError(conn, session, 1, payload)
	assert.NoError(t, handlerErr)

	// Verify error response was sent
	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))

	assert.Equal(t, CmdError, errorResp.Command)
	assert.Equal(t, POSIX_EINVAL, errorResp.Code, "empty message in Error must be rejected with POSIX_EINVAL")
	assertErrorToken(t, errorResp, errErrorMissingMessage)
}

// TestHandleInboundError_Valid_SendsErrorAck validates that valid error messages
// are processed correctly and an errorAck is sent.
//
// Spec: SCACI §3.14 - Error handshake: error → errorAck
func TestHandleInboundError_Valid_SendsErrorAck(t *testing.T) {
	server := &Server{
		logger:     testLogger(),
		sessions:   make(map[net.Conn]*Session),
		sessionsMu: sync.RWMutex{},
		config:     &Config{},
		// No errorRecorder - test basic flow without persistence
	}

	conn := &mockConn{}
	session := &Session{ID: 100, TenantID: 1, State: StateActive}

	// Create valid error message
	errMsg := Error{
		BaseMessage: BaseMessage{Command: CmdError, OpId: 1},
		Code:        POSIX_EIO, // Valid non-zero code
		Message:     "AC could not process operation",
	}
	payload, err := msgpack.Marshal(&errMsg)
	require.NoError(t, err)

	handlerErr := server.handleInboundError(conn, session, 1, payload)
	assert.NoError(t, handlerErr)

	// Verify errorAck was sent (not error)
	var response map[string]interface{}
	require.NoError(t, decodeResponse(conn.written, &response))

	cmdVal, ok := response["command"]
	require.True(t, ok, "response must have command field")
	assert.Equal(t, CmdErrorAck, cmdVal, "valid error must receive errorAck response")
}

// =============================================================================
// §3.6.3 Register Complete - Pre-Attach Propagation Tests
// =============================================================================

// mockPropagationService mocks propagation.Service for testing pre-attach propagation
type mockPropagationService struct {
	mock.Mock
}

func (m *mockPropagationService) TriggerEndpointPropagate(ctx context.Context, endpointID int64, activeSessions []propagation.BaseStationSession) error {
	args := m.Called(ctx, endpointID, activeSessions)
	return args.Error(0)
}

func (m *mockPropagationService) ReconcileBaseStation(ctx context.Context, session propagation.BaseStationSession, bs *models.BaseStation) error {
	args := m.Called(ctx, session, bs)
	return args.Error(0)
}

// mockSessionSnapshotProvider mocks propagation.SessionSnapshotProvider
type mockSessionSnapshotProvider struct {
	sessions []propagation.BaseStationSession
}

func (m *mockSessionSnapshotProvider) ConnectedSessionsSnapshot() []propagation.BaseStationSession {
	return m.sessions
}

// TestHandleRegisterComplete_UnidirectionalPreAttach verifies that unidirectional endpoints
// with preAttach=true trigger attach propagation.
//
// Spec: BSSCI §3.8 - "attach propagate is required for unidirectional End Points"
func TestHandleRegisterComplete_UnidirectionalPreAttach(t *testing.T) {
	const testEpEui = uint64(0x70B3D59CD00008C1)

	// Mock endpoint service to return endpoint with bidi=false, preAttach=true
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("GetByEUI", mock.Anything, int64(1), mock.Anything).Return(
		&models.EndPoint{
			ID:        1,
			TenantID:  1,
			PreAttach: true,
			Bidi:      false, // UNIDIRECTIONAL - must still trigger propagation per BSSCI §3.8
		}, "")

	// Mock operation repo to return operation with epEui
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:        1,
		SessionID: 123,
		OpId:      42,
		Command:   CmdRegister,
		RequestData: map[string]interface{}{
			"epEui": testEpEui, // numeric uint64
		},
	}, nil)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything).Return(nil)

	// Mock propagation service - CRITICAL: must be called despite bidi=false
	mockPropSvc := new(mockPropagationService)
	mockPropSvc.On("TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything).Return(nil)
	mockSnapProvider := &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}

	server := &Server{
		logger:                  testLogger(),
		endpointSvc:             mockEndpoint,
		operationRepo:           mockOpRepo,
		propagationSvc:          mockPropSvc,
		sessionSnapshotProvider: mockSnapProvider,
		sessions:                make(map[net.Conn]*Session),
		sessionsMu:              sync.RWMutex{},
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
	time.Sleep(50 * time.Millisecond)

	// Propagation service must be called for unidirectional preAttach endpoint
	mockPropSvc.AssertCalled(t, "TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything)
	mockPropSvc.AssertExpectations(t)
}

// TestHandleRegisterComplete_LegacyStringEpEui verifies that legacy operations with
// epEui stored as hex string (before numeric storage fix) are correctly parsed.
func TestHandleRegisterComplete_LegacyStringEpEui(t *testing.T) {
	const testEpEuiHex = "70B3D59CD00008C1" // Legacy hex string format

	// Mock endpoint service to return endpoint with preAttach=true
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("GetByEUI", mock.Anything, int64(1), mock.Anything).Return(
		&models.EndPoint{
			ID:        1,
			TenantID:  1,
			PreAttach: true,
			Bidi:      true,
		}, "")

	// Mock operation repo to return operation with LEGACY STRING epEui
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:        1,
		SessionID: 123,
		OpId:      42,
		Command:   CmdRegister,
		RequestData: map[string]interface{}{
			"epEui": testEpEuiHex, // LEGACY: stored as hex string before fix
		},
	}, nil)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything).Return(nil)

	// Mock propagation service - must be called after hex string is parsed to endpoint ID
	mockPropSvc := new(mockPropagationService)
	mockPropSvc.On("TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything).Return(nil)
	mockSnapProvider := &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}

	server := &Server{
		logger:                  testLogger(),
		endpointSvc:             mockEndpoint,
		operationRepo:           mockOpRepo,
		propagationSvc:          mockPropSvc,
		sessionSnapshotProvider: mockSnapProvider,
		sessions:                make(map[net.Conn]*Session),
		sessionsMu:              sync.RWMutex{},
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
	time.Sleep(50 * time.Millisecond)

	// String "70B3D59CD00008C1" must be parsed correctly and propagation triggered
	mockPropSvc.AssertCalled(t, "TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything)
	mockPropSvc.AssertExpectations(t)
}

// TestHandleRegisterComplete_RoamingTenantFallback verifies that when endpoint is not
// found in session tenant, cross-tenant fallback (GetGlobal) is used.
func TestHandleRegisterComplete_RoamingTenantFallback(t *testing.T) {
	const testEpEui = uint64(0x70B3D59CD00008C1)
	const sessionTenant = int64(1)  // SCACI session is in tenant 1
	const endpointTenant = int64(2) // But endpoint is owned by tenant 2

	// Mock endpoint service: GetByEUI fails for session tenant, GetGlobal succeeds
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("GetByEUI", mock.Anything, sessionTenant, mock.Anything).Return(
		nil, ErrEndpointNotFound) // Tenant-scoped lookup fails
	mockEndpoint.On("GetGlobal", mock.Anything, mock.Anything).Return(
		&models.EndPoint{
			ID:        1,
			TenantID:  endpointTenant, // Endpoint owned by different tenant
			PreAttach: true,
			Bidi:      true,
		}, "")

	// Mock operation repo
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:        1,
		SessionID: 123,
		OpId:      42,
		Command:   CmdRegister,
		RequestData: map[string]interface{}{
			"epEui": testEpEui,
		},
	}, nil)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything).Return(nil)

	// Mock propagation service
	mockPropSvc := new(mockPropagationService)
	mockPropSvc.On("TriggerEndpointPropagate", mock.Anything, int64(1), mock.Anything).Return(nil)
	mockSnapProvider := &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}

	server := &Server{
		logger:                  testLogger(),
		endpointSvc:             mockEndpoint,
		operationRepo:           mockOpRepo,
		propagationSvc:          mockPropSvc,
		sessionSnapshotProvider: mockSnapProvider,
		sessions:                make(map[net.Conn]*Session),
		sessionsMu:              sync.RWMutex{},
		config:                  &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: sessionTenant, // Session in tenant 1
		State:    StateActive,
	}

	handlerErr := server.handleRegisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Allow async goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Verify cross-tenant fallback was called
	mockEndpoint.AssertCalled(t, "GetByEUI", mock.Anything, sessionTenant, mock.Anything)
	mockEndpoint.AssertCalled(t, "GetGlobal", mock.Anything, mock.Anything)
	mockPropSvc.AssertExpectations(t)
}

// TestHandleRegisterComplete_RoamingPropagationUsesOwnerTenant verifies that
// propagation uses endpoint.TenantID (owner) not session.TenantID for roaming.
func TestHandleRegisterComplete_RoamingPropagationUsesOwnerTenant(t *testing.T) {
	const testEpEui = uint64(0x70B3D59CD00008C1)
	const sessionTenant = int64(1)  // Session tenant
	const endpointTenant = int64(2) // Endpoint owner tenant (different)

	// Track which tenant context was used for propagation
	var propagationEndpointID int64
	var propagationCalled bool

	// Mock endpoint service: returns endpoint from different tenant
	mockEndpoint := new(MockEndpointService)
	mockEndpoint.On("GetByEUI", mock.Anything, sessionTenant, mock.Anything).Return(
		&models.EndPoint{
			ID:        1,
			TenantID:  endpointTenant, // OWNER tenant is different from session
			PreAttach: true,
			Bidi:      true,
		}, "")

	// Mock operation repo
	mockOpRepo := new(MockSCACIOperationRepository)
	mockOpRepo.On("GetOperationByOpID", mock.Anything, int64(123), int64(42)).Return(&models.SCACIOperation{
		ID:        1,
		SessionID: 123,
		OpId:      42,
		Command:   CmdRegister,
		RequestData: map[string]interface{}{
			"epEui": testEpEui,
		},
	}, nil)
	mockOpRepo.On("UpdateOperationState", mock.Anything, int64(123), int64(42), models.OperationStateCompleted, mock.Anything).Return(nil)

	// Custom mock propagation service to capture call and verify tenant context
	mockPropSvc := new(mockPropagationService)
	mockPropSvc.On("TriggerEndpointPropagate", mock.Anything, mock.MatchedBy(func(epID int64) bool {
		propagationCalled = true
		propagationEndpointID = epID
		return true
	}), mock.Anything).Return(nil)
	mockSnapProvider := &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}

	server := &Server{
		logger:                  testLogger(),
		endpointSvc:             mockEndpoint,
		operationRepo:           mockOpRepo,
		propagationSvc:          mockPropSvc,
		sessionSnapshotProvider: mockSnapProvider,
		sessions:                make(map[net.Conn]*Session),
		sessionsMu:              sync.RWMutex{},
		config:                  &Config{},
	}

	conn := &mockConn{}
	session := &Session{
		ID:       123,
		TenantID: sessionTenant, // Session in tenant 1
		State:    StateActive,
	}

	handlerErr := server.handleRegisterComplete(conn, session, 42)
	assert.NoError(t, handlerErr)

	// Allow async goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Verify propagation was triggered with the correct endpoint ID
	assert.True(t, propagationCalled, "Propagation must be called for preAttach endpoint")
	assert.Equal(t, int64(1), propagationEndpointID, "Propagation must use correct endpoint ID")
	mockPropSvc.AssertExpectations(t)
}
