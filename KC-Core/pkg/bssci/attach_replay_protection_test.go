package bssci

import (
	"crypto/aes"
	"encoding/binary"
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/aead/cmac"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPresharedKey returns the test preshared key for attach signature generation.
func testPresharedKey() []byte {
	key := make([]byte, 16)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

// generateAttachSignature computes CMAC signature per MIOTY radio spec section 3.7.1.3.
func generateAttachSignature(epEUI uint64, attachCnt uint32, presharedKey []byte) []byte {
	// Build CMAC initialization vector: [EUI64 | 0xFF | 0x00 | attachCnt(24-bit) | 0xFFFF]
	iv := make([]byte, 15)
	binary.BigEndian.PutUint64(iv[0:8], epEUI)
	iv[8] = 0xFF
	iv[9] = 0x00
	maskedCnt := attachCnt & 0xFFFFFF
	iv[10] = byte(maskedCnt >> 16)
	iv[11] = byte(maskedCnt >> 8)
	iv[12] = byte(maskedCnt)
	iv[13] = 0xFF
	iv[14] = 0xFF

	block, _ := aes.NewCipher(presharedKey)
	mac, _ := cmac.New(block)
	mac.Write(iv)
	return mac.Sum(nil)[:4] // First 4 bytes
}

// TestAttachReplayProtection_RejectReplay validates BSSCI attach counter replay protection.
// When an attach message arrives with attachCnt <= stored value, it must be rejected.
// This test verifies server.go:2245-2267 enforcement of monotonic counter requirement.
func TestAttachReplayProtection_RejectReplay(t *testing.T) {
	t.Parallel()

	presharedKey := testPresharedKey()
	storedCnt := uint32(10)
	incomingCnt := uint32(10) // Equal to stored - should be rejected as replay

	// Generate valid signature for this attach counter
	validSign := generateAttachSignature(TestEpEui01, incomingCnt, presharedKey)

	// Build endpoint with proper EUI type
	var euiBytes models.EUI
	binary.BigEndian.PutUint64(euiBytes[:], TestEpEui01)
	endpoint := &models.EndPoint{
		ID:        1001,
		EUI:       euiBytes,
		TenantID:  1,
		AttachCnt: &storedCnt,
		Sign:      validSign,
		NwkSnKey:  presharedKey,
	}

	// Create test environment
	testLogger := logger.NewNop()
	eventStore := &recordingEventStore{}
	storage := newStubStorage()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(testLogger, eventStore)

	server := NewTestServer(
		testLogger,
		storage,
		eventStore,
		1, // tenantID
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver,
	)
	server.config = &Config{
		MessageEncoding:          EncodingJSON,
		DisableAttachPersistence: true,
	}
	server.endpointRepo = newFakeEndpointRepo(endpoint)
	server.SetStorageForTest(storage)
	server.orgResolver = &fakeOrgResolver{
		tenantToOrg: make(map[int64]uuid.UUID),
		orgToTenant: make(map[uuid.UUID]int64),
	}
	server.RegisterHandlers()

	// Create TestConn and session
	testConn := &bsscitest.TestConn{Encoding: "json"}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "test-session-replay",
			BaseStationEUI:    TestBsEui01,
			ResolvedTenantID:  1,
			DbSessionID:       1,
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: true,
		},
		Conn: testConn,
	}

	const opID = int64(1)

	// Pre-insert pending operation to verify cleanup
	ctx := testutil.TestContext()
	pendingOp := &PendingOperation{
		OperationType: mioty.CmdAttach,
		CreatedAt:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}
	err := statusSvc.RecordPendingOperation(ctx, session, opID, pendingOp, session.DbSessionID)
	require.NoError(t, err, "Failed to pre-insert pending operation")

	// Verify it was inserted
	_, err = statusSvc.GetPendingOperation(session, opID)
	require.NoError(t, err, "Pending operation should exist before handler call")

	// Convert signature bytes to float64 slice for validateByteArray
	signFloats := make([]interface{}, 4)
	for i, b := range validSign {
		signFloats[i] = float64(b)
	}

	// Build attach message data with all mandatory fields
	data := map[string]interface{}{
		"command":     mioty.CmdAttach,
		"opId":        opID,
		"epEui":       uint64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"attachCnt":   incomingCnt,
		"snr":         12.5,
		"rssi":        -75.5,
		"nonce":       []interface{}{float64(0), float64(0), float64(0), float64(0)},
		"sign":        signFloats,
		"dualChan":    false,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}

	msg := &Message{
		Command: mioty.CmdAttach,
		OpId:    opID,
		Data:    data,
	}

	// Call full handler (includes normalization)
	_ = server.CallHandleMessage(session, msg, data)

	// Assert results
	code, message := testConn.LastError()

	// Verify error response for replay rejection
	assert.Equal(t, POSIX_EPROTO, code, "Expected POSIX_EPROTO for replay rejection")
	assert.Equal(t, ResolveErrorMessage(errAttachCounterNotMonotonic), message,
		"Expected errAttachCounterNotMonotonic message")

	// Verify pending operation was removed
	_, err = statusSvc.GetPendingOperation(session, opID)
	assert.Error(t, err, "Pending operation should be removed after replay rejection")
}

// TestAttachReplayProtection_RejectLowerCounter validates that a lower counter is rejected.
// This verifies the strict monotonic requirement (incoming > stored).
func TestAttachReplayProtection_RejectLowerCounter(t *testing.T) {
	t.Parallel()

	presharedKey := testPresharedKey()
	storedCnt := uint32(100)
	incomingCnt := uint32(50) // Lower than stored - should be rejected

	validSign := generateAttachSignature(TestEpEui01, incomingCnt, presharedKey)

	var euiBytes models.EUI
	binary.BigEndian.PutUint64(euiBytes[:], TestEpEui01)
	endpoint := &models.EndPoint{
		ID:        1001,
		EUI:       euiBytes,
		TenantID:  1,
		AttachCnt: &storedCnt,
		Sign:      validSign,
		NwkSnKey:  presharedKey,
	}

	testLogger := logger.NewNop()
	eventStore := &recordingEventStore{}
	storage := newStubStorage()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(testLogger, eventStore)

	server := NewTestServer(
		testLogger,
		storage,
		eventStore,
		1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver,
	)
	server.config = &Config{MessageEncoding: EncodingJSON, DisableAttachPersistence: true}
	server.endpointRepo = newFakeEndpointRepo(endpoint)
	server.SetStorageForTest(storage)
	server.orgResolver = &fakeOrgResolver{
		tenantToOrg: make(map[int64]uuid.UUID),
		orgToTenant: make(map[uuid.UUID]int64),
	}
	server.RegisterHandlers()

	testConn := &bsscitest.TestConn{Encoding: "json"}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "test-session-lower",
			BaseStationEUI:    TestBsEui01,
			ResolvedTenantID:  1,
			DbSessionID:       1,
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: true,
		},
		Conn: testConn,
	}

	signFloats := make([]interface{}, 4)
	for i, b := range validSign {
		signFloats[i] = float64(b)
	}

	data := map[string]interface{}{
		"command":     mioty.CmdAttach,
		"opId":        int64(1),
		"epEui":       uint64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"attachCnt":   incomingCnt,
		"snr":         12.5,
		"rssi":        -75.5,
		"nonce":       []interface{}{float64(0), float64(0), float64(0), float64(0)},
		"sign":        signFloats,
		"dualChan":    false,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}

	msg := &Message{
		Command: mioty.CmdAttach,
		OpId:    1,
		Data:    data,
	}

	_ = server.CallHandleMessage(session, msg, data)

	code, message := testConn.LastError()
	assert.Equal(t, POSIX_EPROTO, code, "Expected POSIX_EPROTO for lower counter rejection")
	assert.Equal(t, ResolveErrorMessage(errAttachCounterNotMonotonic), message,
		"Expected errAttachCounterNotMonotonic message")
}

// TestAttachReplayProtection_RolloverEdgeCase validates 24-bit rollover exception.
// When stored counter is near max (>0xFFFF00) and incoming is low (<0x100),
// this is a valid rollover, not a replay attack.
// Note: This test verifies the rollover check passes - the handler will continue
// past replay protection but may fail later (at transaction) due to test stub limitations.
// The key assertion is that we DON'T get errAttachCounterNotMonotonic.
func TestAttachReplayProtection_RolloverEdgeCase(t *testing.T) {
	t.Parallel()

	presharedKey := testPresharedKey()
	storedCnt := uint32(0xFFFF10) // Near 24-bit max
	incomingCnt := uint32(5)      // Low value - valid rollover

	validSign := generateAttachSignature(TestEpEui01, incomingCnt, presharedKey)

	var euiBytes models.EUI
	binary.BigEndian.PutUint64(euiBytes[:], TestEpEui01)
	endpoint := &models.EndPoint{
		ID:        1001,
		EUI:       euiBytes,
		TenantID:  1,
		AttachCnt: &storedCnt,
		Sign:      validSign,
		NwkSnKey:  presharedKey,
	}

	testLogger := logger.NewNop()
	eventStore := &recordingEventStore{}
	storage := newStubStorage()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(testLogger, eventStore)

	server := NewTestServer(
		testLogger,
		storage,
		eventStore,
		1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver,
	)
	server.config = &Config{MessageEncoding: EncodingJSON, DisableAttachPersistence: true}
	server.endpointRepo = newFakeEndpointRepo(endpoint)
	server.SetStorageForTest(storage)
	server.orgResolver = &fakeOrgResolver{
		tenantToOrg: make(map[int64]uuid.UUID),
		orgToTenant: make(map[uuid.UUID]int64),
	}
	server.RegisterHandlers()

	testConn := &bsscitest.TestConn{Encoding: "json"}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "test-session-rollover",
			BaseStationEUI:    TestBsEui01,
			ResolvedTenantID:  1,
			DbSessionID:       1,
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: true,
		},
		Conn: testConn,
	}

	signFloats := make([]interface{}, 4)
	for i, b := range validSign {
		signFloats[i] = float64(b)
	}

	data := map[string]interface{}{
		"command":     mioty.CmdAttach,
		"opId":        int64(1),
		"epEui":       uint64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"attachCnt":   incomingCnt,
		"snr":         12.5,
		"rssi":        -75.5,
		"nonce":       []interface{}{float64(0), float64(0), float64(0), float64(0)},
		"sign":        signFloats,
		"dualChan":    false,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}

	msg := &Message{
		Command: mioty.CmdAttach,
		OpId:    1,
		Data:    data,
	}

	_ = server.CallHandleMessage(session, msg, data)

	_, message := testConn.LastError()
	assert.NotEqual(t, ResolveErrorMessage(errAttachCounterNotMonotonic), message,
		"Rollover should NOT trigger replay protection error")
}

// TestAttachReplayProtection_FirstAttachNilCounter validates first attach acceptance.
// When endpoint has no stored counter (nil), any incoming counter should pass replay check.
// Note: Similar to rollover test, may fail at transaction stage due to test stubs.
func TestAttachReplayProtection_FirstAttachNilCounter(t *testing.T) {
	t.Parallel()

	presharedKey := testPresharedKey()
	incomingCnt := uint32(5)

	validSign := generateAttachSignature(TestEpEui01, incomingCnt, presharedKey)

	var euiBytes models.EUI
	binary.BigEndian.PutUint64(euiBytes[:], TestEpEui01)
	endpoint := &models.EndPoint{
		ID:        1001,
		EUI:       euiBytes,
		TenantID:  1,
		AttachCnt: nil, // First attach - no stored counter
		Sign:      validSign,
		NwkSnKey:  presharedKey,
	}

	testLogger := logger.NewNop()
	eventStore := &recordingEventStore{}
	storage := newStubStorage()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(testLogger, eventStore)

	server := NewTestServer(
		testLogger,
		storage,
		eventStore,
		1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc,
		broadcaster, queueSerializer, auditLogger, tenantResolver,
	)
	server.config = &Config{MessageEncoding: EncodingJSON, DisableAttachPersistence: true}
	server.endpointRepo = newFakeEndpointRepo(endpoint)
	server.SetStorageForTest(storage)
	server.orgResolver = &fakeOrgResolver{
		tenantToOrg: make(map[int64]uuid.UUID),
		orgToTenant: make(map[uuid.UUID]int64),
	}
	server.RegisterHandlers()

	testConn := &bsscitest.TestConn{Encoding: "json"}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "test-session-first",
			BaseStationEUI:    TestBsEui01,
			ResolvedTenantID:  1,
			DbSessionID:       1,
			Encoding:          EncodingJSON,
			SessionUUID:       uuidBytes(),
			HandshakeComplete: true,
		},
		Conn: testConn,
	}

	signFloats := make([]interface{}, 4)
	for i, b := range validSign {
		signFloats[i] = float64(b)
	}

	data := map[string]interface{}{
		"command":     mioty.CmdAttach,
		"opId":        int64(1),
		"epEui":       uint64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"attachCnt":   incomingCnt,
		"snr":         12.5,
		"rssi":        -75.5,
		"nonce":       []interface{}{float64(0), float64(0), float64(0), float64(0)},
		"sign":        signFloats,
		"dualChan":    false,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}

	msg := &Message{
		Command: mioty.CmdAttach,
		OpId:    1,
		Data:    data,
	}

	_ = server.CallHandleMessage(session, msg, data)

	_, message := testConn.LastError()
	assert.NotEqual(t, ResolveErrorMessage(errAttachCounterNotMonotonic), message,
		"First attach should NOT trigger replay protection error")
}
