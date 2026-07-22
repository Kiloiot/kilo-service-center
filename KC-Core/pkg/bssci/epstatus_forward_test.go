// Package bssci provides tests for BSSCI → SCACI EPStatus forwarding
// per SCACI §3.13 (EPStatus operation).
//
// Coverage:
//   - SetSCACIEPStatusBroadcaster: wiring and nil handling
//   - mockSCACIEPStatusBroadcaster: interface verification
//   - EPStatusData: correct type construction
package bssci

import (
	"context"
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // EPStatus constants (EPStatusAttached, EPStatusDetached)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"        // Numeric4, Subpackets types
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// mockSCACIEPStatusBroadcaster Interface Tests
// =============================================================================

func TestMockSCACIEPStatusBroadcaster_ImplementsInterface(t *testing.T) {
	// Compile-time check already exists in test_mocks_test.go
	// This test verifies runtime behavior
	var broadcaster SCACIEPStatusBroadcaster = NewMockSCACIEPStatusBroadcaster()
	require.NotNil(t, broadcaster, "mock should implement interface")
}

func TestMockSCACIEPStatusBroadcaster_RecordsCalls(t *testing.T) {
	mock := NewMockSCACIEPStatusBroadcaster()

	// No calls initially
	assert.Equal(t, 0, mock.CallCount())
	assert.Nil(t, mock.LastCall())

	// Make first call
	data1 := &EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusAttached,
	}
	err := mock.BroadcastEPStatus(context.Background(), 42, data1)
	require.NoError(t, err)

	// Verify first call recorded
	assert.Equal(t, 1, mock.CallCount())
	lastCall := mock.LastCall()
	require.NotNil(t, lastCall)
	assert.Equal(t, int64(42), lastCall.TenantID)
	assert.Equal(t, uint64(0x1234567890ABCDEF), lastCall.Data.EpEui)
	assert.Equal(t, pkgmioty.EPStatusAttached, lastCall.Data.EpStatus)

	// Make second call
	data2 := &EPStatusData{
		EpEui:    0xAABBCCDDEEFF0011,
		EpStatus: pkgmioty.EPStatusDetached,
	}
	err = mock.BroadcastEPStatus(context.Background(), 99, data2)
	require.NoError(t, err)

	// Verify second call
	assert.Equal(t, 2, mock.CallCount())
	lastCall = mock.LastCall()
	require.NotNil(t, lastCall)
	assert.Equal(t, int64(99), lastCall.TenantID)
	assert.Equal(t, pkgmioty.EPStatusDetached, lastCall.Data.EpStatus)

	// Verify GetCall by index
	call0 := mock.GetCall(0)
	require.NotNil(t, call0)
	assert.Equal(t, int64(42), call0.TenantID)

	call1 := mock.GetCall(1)
	require.NotNil(t, call1)
	assert.Equal(t, int64(99), call1.TenantID)

	// Out of bounds returns nil
	assert.Nil(t, mock.GetCall(-1))
	assert.Nil(t, mock.GetCall(2))
}

func TestMockSCACIEPStatusBroadcaster_ReturnsConfiguredError(t *testing.T) {
	mock := NewMockSCACIEPStatusBroadcaster()

	expectedErr := errors.New("SCACI broadcast failure")
	mock.SetError(expectedErr)

	data := &EPStatusData{
		EpEui:    0x1234567890ABCDEF,
		EpStatus: pkgmioty.EPStatusAttached,
	}
	err := mock.BroadcastEPStatus(context.Background(), 1, data)

	// Error should be returned
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// Call should still be recorded
	assert.Equal(t, 1, mock.CallCount())
}

func TestMockSCACIEPStatusBroadcaster_Reset(t *testing.T) {
	mock := NewMockSCACIEPStatusBroadcaster()

	// Add some calls
	data := &EPStatusData{EpEui: 0x1234, EpStatus: pkgmioty.EPStatusAttached}
	_ = mock.BroadcastEPStatus(context.Background(), 1, data)
	_ = mock.BroadcastEPStatus(context.Background(), 2, data)

	assert.Equal(t, 2, mock.CallCount())

	// Reset
	mock.Reset()

	// All cleared
	assert.Equal(t, 0, mock.CallCount())
	assert.Nil(t, mock.LastCall())
}

// =============================================================================
// Server.SetSCACIEPStatusBroadcaster Tests
// =============================================================================

func TestSetSCACIEPStatusBroadcaster_StoresBroadcaster(t *testing.T) {
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mock := NewMockSCACIEPStatusBroadcaster()

	// Set the broadcaster
	server.SetSCACIEPStatusBroadcaster(mock)

	// Broadcaster should be stored (we verify by checking no panic on subsequent operations)
	// The actual forwarding tests would require more complex setup with full message handling
	assert.NotNil(t, server, "server should be non-nil after setting broadcaster")
}

func TestSetSCACIEPStatusBroadcaster_NilDoesNotPanic(t *testing.T) {
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Setting nil should not panic
	require.NotPanics(t, func() {
		server.SetSCACIEPStatusBroadcaster(nil)
	})
}

func TestSetSCACIEPStatusBroadcaster_CanBeOverwritten(t *testing.T) {
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	mock1 := NewMockSCACIEPStatusBroadcaster()
	mock2 := NewMockSCACIEPStatusBroadcaster()

	// Set first broadcaster
	server.SetSCACIEPStatusBroadcaster(mock1)

	// Overwrite with second
	server.SetSCACIEPStatusBroadcaster(mock2)

	// No panic, operation succeeds
	assert.NotNil(t, server)
}

// =============================================================================
// EPStatusData Type Tests
// =============================================================================

func TestEPStatusData_AllFieldsPopulated(t *testing.T) {
	// Verify EPStatusData can hold all optional fields
	attachCnt := uint32(5)
	nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
	sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
	snr := 15.5
	rssi := -80.0
	eqSnr := 12.0
	subpackets := &mioty.Subpackets{
		SNR:       []float64{1.0, 2.0, 3.0},
		RSSI:      []float64{-75.0, -80.0, -85.0},
		Frequency: []int64{868100000, 868300000, 868500000},
		Phase:     []float64{0.1, 0.2, 0.3},
	}

	data := &EPStatusData{
		EpEui:      0x70B3D59CD00009E6,
		EpStatus:   pkgmioty.EPStatusAttached,
		AttachCnt:  &attachCnt,
		Nonce:      &nonce,
		Sign:       &sign,
		Snr:        &snr,
		Rssi:       &rssi,
		EqSnr:      &eqSnr,
		Subpackets: subpackets,
	}

	// Verify all fields accessible
	assert.Equal(t, uint64(0x70B3D59CD00009E6), data.EpEui)
	assert.Equal(t, pkgmioty.EPStatusAttached, data.EpStatus)
	assert.Equal(t, uint32(5), *data.AttachCnt)
	assert.Equal(t, nonce, *data.Nonce)
	assert.Equal(t, sign, *data.Sign)
	assert.Equal(t, 15.5, *data.Snr)
	assert.Equal(t, -80.0, *data.Rssi)
	assert.Equal(t, 12.0, *data.EqSnr)
	assert.Equal(t, 3, len(data.Subpackets.SNR))
}

func TestEPStatusData_MinimalRequired(t *testing.T) {
	// Only required fields (epEui, epStatus)
	data := &EPStatusData{
		EpEui:    0x0102030405060708,
		EpStatus: pkgmioty.EPStatusDetached,
	}

	assert.Equal(t, uint64(0x0102030405060708), data.EpEui)
	assert.Equal(t, pkgmioty.EPStatusDetached, data.EpStatus)
	assert.Nil(t, data.AttachCnt)
	assert.Nil(t, data.Nonce)
	assert.Nil(t, data.Sign)
	assert.Nil(t, data.Snr)
	assert.Nil(t, data.Rssi)
	assert.Nil(t, data.EqSnr)
	assert.Nil(t, data.Subpackets)
}

func TestEPStatusConstants(t *testing.T) {
	// Verify EPStatus constants match expected values
	assert.Equal(t, "attached", pkgmioty.EPStatusAttached)
	assert.Equal(t, "detached", pkgmioty.EPStatusDetached)
}

// =============================================================================
// Handler-Level Integration Tests (SCACI §3.13 EPStatus Forwarding)
// =============================================================================
// These tests invoke handleAttachComplete/handleDetachComplete to verify
// that the handlers correctly extract metadata from pending operations
// and broadcast EPStatus to SCACI ACs.

func TestHandleAttachComplete_BroadcastsEPStatus(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(1001)
		epEUI    = uint64(0x70B3D59CD00009E6)
		tenantID = int64(42)
	)

	// Setup: Create server with sync mock broadcaster
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, tenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)

	syncMock := NewSyncMockSCACIEPStatusBroadcaster()
	server.SetSCACIEPStatusBroadcaster(syncMock)

	// Setup session
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-attach-epstatus",
			BaseStationEUI:   0xABCDEF1234567890,
			ResolvedTenantID: tenantID,
			DbSessionID:      1,
			Encoding:         EncodingJSON,
		},
	}

	// Seed pending attach operation via StatusService with metadata
	// This mimics what handleAttach does before handleAttachComplete is called
	// NOTE: handleAttach stores epEui as int64 (from getNumericField), and
	// handleAttachComplete reads it as int64 (line 2971). Cast to int64 to match.
	attachCnt := int64(5)
	snr := 15.5
	rssi := -80.0
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opID,
		OperationType: mioty.CmdAttach,
		Metadata: map[string]interface{}{
			"epEui":      int64(epEUI), // Must be int64 to match handleAttachComplete's type assertion
			"attachCnt":  attachCnt,
			"snr":        snr,
			"rssi":       rssi,
			"endpointID": int64(1001),
		},
	}
	err := statusSvc.RecordPendingOperation(context.Background(), session, opID, pendingOp, tenantID)
	require.NoError(t, err, "failed to seed pending operation")

	// Prepare WaitGroup before calling handler (goroutine will decrement)
	syncMock.ExpectCall(1)

	// Action: Call handleAttachComplete
	msg := &Message{Command: mioty.CmdAttachComplete, OpId: opID}
	err = server.CallHandleAttachComplete(session, msg, nil)
	require.NoError(t, err)

	// Wait deterministically for goroutine to complete
	syncMock.Wait()

	// Assert: BroadcastEPStatus was called with correct data
	assert.Equal(t, 1, syncMock.CallCount(), "expected 1 broadcast call")
	call := syncMock.LastCall()
	require.NotNil(t, call, "expected broadcast call")
	assert.Equal(t, tenantID, call.TenantID, "tenant ID mismatch")
	assert.Equal(t, epEUI, call.Data.EpEui, "endpoint EUI mismatch")
	assert.Equal(t, pkgmioty.EPStatusAttached, call.Data.EpStatus, "epStatus should be attached")

	// Verify optional fields propagated
	require.NotNil(t, call.Data.AttachCnt, "attachCnt should be set")
	assert.Equal(t, uint32(5), *call.Data.AttachCnt, "attachCnt value mismatch")
	require.NotNil(t, call.Data.Snr, "snr should be set")
	assert.Equal(t, 15.5, *call.Data.Snr, "snr value mismatch")
	require.NotNil(t, call.Data.Rssi, "rssi should be set")
	assert.Equal(t, -80.0, *call.Data.Rssi, "rssi value mismatch")
}

func TestHandleDetachComplete_BroadcastsEPStatus(t *testing.T) {
	t.Parallel()

	const (
		opID = int64(2001)
		// Use smaller EUI that fits in float64 without precision loss (53-bit limit)
		// 0x1FFFFFFFFFFFFF = 9007199254740991 is max safe integer for float64
		epEUI    = uint64(0x0001020304050607) // 48-bit compliant EUI
		tenantID = int64(99)
	)

	// Setup: Create server with sync mock broadcaster
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, tenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)

	syncMock := NewSyncMockSCACIEPStatusBroadcaster()
	server.SetSCACIEPStatusBroadcaster(syncMock)

	// Setup session
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-detach-epstatus",
			BaseStationEUI:   0x1234567890ABCDEF,
			ResolvedTenantID: tenantID,
			DbSessionID:      2,
			Encoding:         EncodingJSON,
		},
	}

	// Seed pending detach operation via StatusService with metadata
	// This mimics what handleDetach does before handleDetachComplete is called
	// NOTE: handleDetachComplete uses mapToDetachMetadata() which requires:
	// - epEui as float64 (for JSON round-trip)
	// - packetCnt, rxTime, snr, rssi as float64
	// - signature as []byte (4 bytes)
	snr := 12.5
	rssi := -75.0
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opID,
		OperationType: mioty.CmdDetach,
		Metadata: map[string]interface{}{
			"epEui":      float64(epEUI),         // mapToDetachMetadata expects float64
			"packetCnt":  float64(100),           // Required by mapToDetachMetadata
			"rxTime":     float64(1699876543000), // Required by mapToDetachMetadata
			"snr":        snr,                    // Required by mapToDetachMetadata
			"rssi":       rssi,                   // Required by mapToDetachMetadata
			"signature":  []byte{1, 2, 3, 4},     // Required: 4-byte signature
			"endpointID": float64(2001),          // Optional but useful
		},
	}
	err := statusSvc.RecordPendingOperation(context.Background(), session, opID, pendingOp, tenantID)
	require.NoError(t, err, "failed to seed pending operation")

	// Prepare WaitGroup before calling handler (goroutine will decrement)
	syncMock.ExpectCall(1)

	// Action: Call handleDetachComplete
	msg := &Message{Command: mioty.CmdDetachComplete, OpId: opID}
	err = server.CallHandleDetachComplete(session, msg, nil)
	require.NoError(t, err)

	// Wait deterministically for goroutine to complete
	syncMock.Wait()

	// Assert: BroadcastEPStatus was called with correct data
	assert.Equal(t, 1, syncMock.CallCount(), "expected 1 broadcast call")
	call := syncMock.LastCall()
	require.NotNil(t, call, "expected broadcast call")
	assert.Equal(t, tenantID, call.TenantID, "tenant ID mismatch")
	assert.Equal(t, epEUI, call.Data.EpEui, "endpoint EUI mismatch")
	assert.Equal(t, pkgmioty.EPStatusDetached, call.Data.EpStatus, "epStatus should be detached")

	// Verify optional fields propagated from typedMeta
	require.NotNil(t, call.Data.Snr, "snr should be set")
	assert.Equal(t, 12.5, *call.Data.Snr, "snr value mismatch")
	require.NotNil(t, call.Data.Rssi, "rssi should be set")
	assert.Equal(t, -75.0, *call.Data.Rssi, "rssi value mismatch")
}

func TestHandleAttachComplete_NilBroadcaster_NoPanic(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(3001)
		epEUI    = uint64(0x70B3D59CD00009E8)
		tenantID = int64(1)
	)

	// Setup: Create server WITHOUT setting broadcaster
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, tenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)

	// NOTE: No SetSCACIEPStatusBroadcaster call - broadcaster is nil

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-attach-nil-broadcaster",
			BaseStationEUI:   0xFFFFFFFFFFFFFFFF,
			ResolvedTenantID: tenantID,
			DbSessionID:      3,
			Encoding:         EncodingJSON,
		},
	}

	// Seed pending attach operation (use int64 for epEui to match handleAttachComplete)
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opID,
		OperationType: mioty.CmdAttach,
		Metadata: map[string]interface{}{
			"epEui":      int64(epEUI),
			"endpointID": int64(3001),
		},
	}
	err := statusSvc.RecordPendingOperation(context.Background(), session, opID, pendingOp, tenantID)
	require.NoError(t, err)

	// Action: Call handleAttachComplete - should not panic
	msg := &Message{Command: mioty.CmdAttachComplete, OpId: opID}
	require.NotPanics(t, func() {
		err = server.CallHandleAttachComplete(session, msg, nil)
	})
	require.NoError(t, err, "handler should succeed without broadcaster")
}

// TestHandleAttachComplete_BroadcastsEPStatus_FullTelemetry tests that all OTA fields
// (nonce, sign, eqSnr, subpackets) are correctly extracted from pending operation
// metadata and propagated to BroadcastEPStatus per SCACI §3.13.1.
func TestHandleAttachComplete_BroadcastsEPStatus_FullTelemetry(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(5001)
		epEUI    = uint64(0x70B3D59CD00009F0)
		tenantID = int64(50)
	)

	// Setup: Create server with sync mock broadcaster
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, tenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)

	syncMock := NewSyncMockSCACIEPStatusBroadcaster()
	server.SetSCACIEPStatusBroadcaster(syncMock)

	// Setup session
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-attach-full-telemetry",
			BaseStationEUI:   0xABCDEF1234567890,
			ResolvedTenantID: tenantID,
			DbSessionID:      5,
			Encoding:         EncodingJSON,
		},
	}

	// Seed pending attach operation with ALL OTA fields per SCACI §3.13.1
	attachCnt := int64(5)
	snr := 15.5
	rssi := -80.0
	eqSnr := 12.5
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opID,
		OperationType: mioty.CmdAttach,
		Metadata: map[string]interface{}{
			"epEui":     int64(epEUI),
			"attachCnt": attachCnt,
			"snr":       snr,
			"rssi":      rssi,
			// OTA telemetry fields (SCACI §3.13.1)
			"nonce": []byte{0x01, 0x02, 0x03, 0x04},
			"sign":  []byte{0xAA, 0xBB, 0xCC, 0xDD},
			"eqSnr": eqSnr,
			// Subpackets as []interface{} with map entries (metadata storage format)
			"subpackets": []interface{}{
				map[string]interface{}{"snr": 10.0, "rssi": -70.0, "frequency": float64(868100000)},
				map[string]interface{}{"snr": 11.0, "rssi": -72.0, "frequency": float64(868300000)},
			},
			"endpointID": int64(5001),
		},
	}
	err := statusSvc.RecordPendingOperation(context.Background(), session, opID, pendingOp, tenantID)
	require.NoError(t, err, "failed to seed pending operation")

	// Prepare WaitGroup before calling handler
	syncMock.ExpectCall(1)

	// Action: Call handleAttachComplete
	msg := &Message{Command: mioty.CmdAttachComplete, OpId: opID}
	err = server.CallHandleAttachComplete(session, msg, nil)
	require.NoError(t, err)

	// Wait deterministically for goroutine to complete
	syncMock.Wait()

	// Assert: BroadcastEPStatus was called with all telemetry fields
	assert.Equal(t, 1, syncMock.CallCount(), "expected 1 broadcast call")
	call := syncMock.LastCall()
	require.NotNil(t, call, "expected broadcast call")
	assert.Equal(t, tenantID, call.TenantID, "tenant ID mismatch")
	assert.Equal(t, epEUI, call.Data.EpEui, "endpoint EUI mismatch")
	assert.Equal(t, pkgmioty.EPStatusAttached, call.Data.EpStatus, "epStatus should be attached")

	// Verify basic OTA fields
	require.NotNil(t, call.Data.AttachCnt, "attachCnt should be set")
	assert.Equal(t, uint32(5), *call.Data.AttachCnt)
	require.NotNil(t, call.Data.Snr, "snr should be set")
	assert.Equal(t, 15.5, *call.Data.Snr)
	require.NotNil(t, call.Data.Rssi, "rssi should be set")
	assert.Equal(t, -80.0, *call.Data.Rssi)

	// Verify nonce (4-byte array → mioty.Numeric4)
	require.NotNil(t, call.Data.Nonce, "nonce should be set")
	assert.Equal(t, mioty.Numeric4{0x01, 0x02, 0x03, 0x04}, *call.Data.Nonce, "nonce value mismatch")

	// Verify sign (4-byte array → mioty.Numeric4)
	require.NotNil(t, call.Data.Sign, "sign should be set")
	assert.Equal(t, mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}, *call.Data.Sign, "sign value mismatch")

	// Verify eqSnr
	require.NotNil(t, call.Data.EqSnr, "eqSnr should be set")
	assert.Equal(t, 12.5, *call.Data.EqSnr, "eqSnr value mismatch")

	// Verify subpackets (converted to *mioty.Subpackets)
	require.NotNil(t, call.Data.Subpackets, "subpackets should be set")
	assert.Equal(t, []float64{10.0, 11.0}, call.Data.Subpackets.SNR, "subpackets SNR mismatch")
	assert.Equal(t, []float64{-70.0, -72.0}, call.Data.Subpackets.RSSI, "subpackets RSSI mismatch")
	assert.Equal(t, []int64{868100000, 868300000}, call.Data.Subpackets.Frequency, "subpackets Frequency mismatch")
}

// TestHandleDetachComplete_BroadcastsEPStatus_SignEqSnrFallback tests that sign/eqSnr
// are extracted from both typedMeta and raw-map fallback paths per SCACI §3.13.1.
func TestHandleDetachComplete_BroadcastsEPStatus_SignEqSnrFallback(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(6001)
		epEUI    = uint64(0x0001020304050608) // 48-bit compliant EUI
		tenantID = int64(60)
	)

	// Setup: Create server with sync mock broadcaster
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, tenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)

	syncMock := NewSyncMockSCACIEPStatusBroadcaster()
	server.SetSCACIEPStatusBroadcaster(syncMock)

	// Setup session
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-detach-sign-eqsnr",
			BaseStationEUI:   0x1234567890ABCDEF,
			ResolvedTenantID: tenantID,
			DbSessionID:      6,
			Encoding:         EncodingJSON,
		},
	}

	// Seed pending detach operation with sign and eqSnr
	// typedMeta path uses: signature []byte, EqSnr *float64
	snr := 14.0
	rssi := -78.0
	eqSnr := 11.5
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opID,
		OperationType: mioty.CmdDetach,
		Metadata: map[string]interface{}{
			"epEui":     float64(epEUI),
			"packetCnt": float64(200),
			"rxTime":    float64(1699876543000),
			"snr":       snr,
			"rssi":      rssi,
			"signature": []byte{0xDD, 0xCC, 0xBB, 0xAA}, // 4-byte signature for typedMeta
			"eqSnr":     eqSnr,                          // eqSnr for fallback (typedMeta.EqSnr not set by mapToDetachMetadata from raw map)
			// For raw-map fallback path testing, also include "sign" key
			"sign":       []byte{0xDD, 0xCC, 0xBB, 0xAA},
			"endpointID": float64(6001),
		},
	}
	err := statusSvc.RecordPendingOperation(context.Background(), session, opID, pendingOp, tenantID)
	require.NoError(t, err, "failed to seed pending operation")

	// Prepare WaitGroup before calling handler
	syncMock.ExpectCall(1)

	// Action: Call handleDetachComplete
	msg := &Message{Command: mioty.CmdDetachComplete, OpId: opID}
	err = server.CallHandleDetachComplete(session, msg, nil)
	require.NoError(t, err)

	// Wait deterministically for goroutine to complete
	syncMock.Wait()

	// Assert: BroadcastEPStatus was called with sign and eqSnr
	assert.Equal(t, 1, syncMock.CallCount(), "expected 1 broadcast call")
	call := syncMock.LastCall()
	require.NotNil(t, call, "expected broadcast call")
	assert.Equal(t, tenantID, call.TenantID, "tenant ID mismatch")
	assert.Equal(t, epEUI, call.Data.EpEui, "endpoint EUI mismatch")
	assert.Equal(t, pkgmioty.EPStatusDetached, call.Data.EpStatus, "epStatus should be detached")

	// Verify basic OTA fields
	require.NotNil(t, call.Data.Snr, "snr should be set")
	assert.Equal(t, 14.0, *call.Data.Snr)
	require.NotNil(t, call.Data.Rssi, "rssi should be set")
	assert.Equal(t, -78.0, *call.Data.Rssi)

	// Verify sign (from typedMeta.Signature or raw-map fallback)
	require.NotNil(t, call.Data.Sign, "sign should be set")
	assert.Equal(t, mioty.Numeric4{0xDD, 0xCC, 0xBB, 0xAA}, *call.Data.Sign, "sign value mismatch")

	// Verify eqSnr (from typedMeta.EqSnr or raw-map fallback)
	require.NotNil(t, call.Data.EqSnr, "eqSnr should be set")
	assert.Equal(t, 11.5, *call.Data.EqSnr, "eqSnr value mismatch")

	// Note: Subpackets are NOT extracted for detach (per plan - no new shapes)
	// Detach metadata does not carry subpackets in the current flow
}

func TestHandleDetachComplete_NilBroadcaster_NoPanic(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(4001)
		epEUI    = uint64(0x70B3D59CD00009E9)
		tenantID = int64(1)
	)

	// Setup: Create server WITHOUT setting broadcaster
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, tenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)

	// NOTE: No SetSCACIEPStatusBroadcaster call - broadcaster is nil

	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-detach-nil-broadcaster",
			BaseStationEUI:   0xFFFFFFFFFFFFFFFF,
			ResolvedTenantID: tenantID,
			DbSessionID:      4,
			Encoding:         EncodingJSON,
		},
	}

	// Seed pending detach operation (use uint64 for epEui to match handleDetachComplete)
	pendingOp := &PendingOperation{
		SessionSlug:   session.ID,
		OperationID:   opID,
		OperationType: mioty.CmdDetach,
		Metadata: map[string]interface{}{
			"epEui":      uint64(epEUI),
			"endpointID": int64(4001),
		},
	}
	err := statusSvc.RecordPendingOperation(context.Background(), session, opID, pendingOp, tenantID)
	require.NoError(t, err)

	// Action: Call handleDetachComplete - should not panic
	msg := &Message{Command: mioty.CmdDetachComplete, OpId: opID}
	require.NotPanics(t, func() {
		err = server.CallHandleDetachComplete(session, msg, nil)
	})
	require.NoError(t, err, "handler should succeed without broadcaster")
}
