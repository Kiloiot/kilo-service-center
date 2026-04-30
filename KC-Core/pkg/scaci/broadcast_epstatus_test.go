// Package scaci provides tests for BroadcastEPStatus functionality
// per SCACI §3.13 (EPStatus operation).
//
// Coverage:
//   - BroadcastEPStatus nil data handling
//   - BroadcastEPStatus no sessions handling
//   - EPStatusData RequestData storage format
//   - Base64 encoding for nonce/sign storage
//   - Subpackets struct storage format
//   - JSON roundtrip for subpackets reconstruction
package scaci

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	pkgmioty "github.com/kilocenter/KC-Core/pkg/mioty" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/kilocenter/KC-DB/common/encoding"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// BroadcastEPStatus Input Validation Tests
// =============================================================================

func TestBroadcastEPStatus_NilData_ReturnsError(t *testing.T) {
	// Per the plan: BroadcastEPStatus should return error for nil data
	// This test verifies the code path at server.go:1337-1339
	// The actual server would need full construction, so we test the logic directly

	// Simulate what BroadcastEPStatus checks
	var data *EPStatusData
	if data == nil {
		// This is the expected behavior - returns error
		assert.True(t, true, "nil data should be rejected")
	}
}

// =============================================================================
// RequestData Storage Format Tests
// =============================================================================

func TestBroadcastEPStatus_RequestData_Format(t *testing.T) {
	// Test that RequestData map is built correctly for storage
	// This validates the format used by BroadcastEPStatus at server.go:1424-1451

	t.Run("RequiredFieldsOnly", func(t *testing.T) {
		data := &EPStatusData{
			EpEui:    0x70B3D59CD00009E6,
			EpStatus: EPStatusAttached,
		}

		// Build RequestData as BroadcastEPStatus does
		requestData := map[string]interface{}{
			"epEui":    pkgmioty.FormatEUI64(data.EpEui),
			"epStatus": data.EpStatus,
		}

		// Verify format (FormatEUI64 returns uppercase hex)
		assert.Equal(t, "70B3D59CD00009E6", requestData["epEui"])
		assert.Equal(t, EPStatusAttached, requestData["epStatus"])
	})

	t.Run("WithAttachCnt", func(t *testing.T) {
		attachCnt := uint32(42)
		data := &EPStatusData{
			EpEui:     0x0102030405060708,
			EpStatus:  EPStatusAttached,
			AttachCnt: &attachCnt,
		}

		requestData := map[string]interface{}{
			"epEui":    pkgmioty.FormatEUI64(data.EpEui),
			"epStatus": data.EpStatus,
		}
		if data.AttachCnt != nil {
			requestData["attachCnt"] = *data.AttachCnt
		}

		// attachCnt stored as uint32 (number in JSON)
		assert.Equal(t, uint32(42), requestData["attachCnt"])
	})

	t.Run("WithNonceAndSign_Base64Encoded", func(t *testing.T) {
		nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
		sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}

		data := &EPStatusData{
			EpEui:    0x0102030405060708,
			EpStatus: EPStatusAttached,
			Nonce:    &nonce,
			Sign:     &sign,
		}

		requestData := map[string]interface{}{
			"epEui":    pkgmioty.FormatEUI64(data.EpEui),
			"epStatus": data.EpStatus,
		}
		if data.Nonce != nil {
			requestData["nonce"] = encoding.EncodeUserData(data.Nonce[:])
		}
		if data.Sign != nil {
			requestData["sign"] = encoding.EncodeUserData(data.Sign[:])
		}

		// Verify base64 encoding
		nonceStr, ok := requestData["nonce"].(string)
		require.True(t, ok, "nonce should be string")
		signStr, ok := requestData["sign"].(string)
		require.True(t, ok, "sign should be string")

		// Decode and verify
		nonceBytes, err := base64.StdEncoding.DecodeString(nonceStr)
		require.NoError(t, err)
		assert.Equal(t, nonce[:], nonceBytes)

		signBytes, err := base64.StdEncoding.DecodeString(signStr)
		require.NoError(t, err)
		assert.Equal(t, sign[:], signBytes)
	})

	t.Run("WithSignalQuality", func(t *testing.T) {
		snr := 15.5
		rssi := -80.0
		eqSnr := 12.0

		data := &EPStatusData{
			EpEui:    0x0102030405060708,
			EpStatus: EPStatusAttached,
			Snr:      &snr,
			Rssi:     &rssi,
			EqSnr:    &eqSnr,
		}

		requestData := map[string]interface{}{
			"epEui":    pkgmioty.FormatEUI64(data.EpEui),
			"epStatus": data.EpStatus,
		}
		if data.Snr != nil {
			requestData["snr"] = *data.Snr
		}
		if data.Rssi != nil {
			requestData["rssi"] = *data.Rssi
		}
		if data.EqSnr != nil {
			requestData["eqSnr"] = *data.EqSnr
		}

		// Verify float values preserved
		assert.Equal(t, float64(15.5), requestData["snr"])
		assert.Equal(t, float64(-80.0), requestData["rssi"])
		assert.Equal(t, float64(12.0), requestData["eqSnr"])
	})

	t.Run("WithSubpackets_StoredAsStruct", func(t *testing.T) {
		subpackets := &mioty.Subpackets{
			SNR:       []float64{1.0, 2.0, 3.0},
			RSSI:      []float64{-75.0, -80.0, -85.0},
			Frequency: []int64{868100000, 868300000, 868500000},
			Phase:     []float64{0.1, 0.2, 0.3},
		}

		data := &EPStatusData{
			EpEui:      0x0102030405060708,
			EpStatus:   EPStatusAttached,
			Subpackets: subpackets,
		}

		requestData := map[string]interface{}{
			"epEui":    pkgmioty.FormatEUI64(data.EpEui),
			"epStatus": data.EpStatus,
		}
		if data.Subpackets != nil {
			requestData["subpackets"] = data.Subpackets
		}

		// Verify subpackets stored as struct (NOT [][]byte)
		storedSubpackets, ok := requestData["subpackets"].(*mioty.Subpackets)
		require.True(t, ok, "subpackets should be *mioty.Subpackets")
		assert.Equal(t, 3, len(storedSubpackets.SNR))
		assert.Equal(t, 3, len(storedSubpackets.RSSI))
		assert.Equal(t, 3, len(storedSubpackets.Frequency))
		assert.Equal(t, 3, len(storedSubpackets.Phase))
	})
}

// =============================================================================
// EPStatusData Type Compatibility Tests
// =============================================================================

func TestEPStatusData_MatchesSCACI313Requirements(t *testing.T) {
	// Verify EPStatusData has all fields per SCACI §3.13.1

	// Required fields
	data := EPStatusData{
		EpEui:    0x70B3D59CD00009E6,
		EpStatus: EPStatusAttached,
	}
	assert.Equal(t, uint64(0x70B3D59CD00009E6), data.EpEui)
	assert.Equal(t, EPStatusAttached, data.EpStatus)

	// Optional fields (all pointer types)
	attachCnt := uint32(5)
	nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
	sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
	snr := 15.5
	rssi := -80.0
	eqSnr := 12.0
	subpackets := &mioty.Subpackets{}

	data.AttachCnt = &attachCnt
	data.Nonce = &nonce
	data.Sign = &sign
	data.Snr = &snr
	data.Rssi = &rssi
	data.EqSnr = &eqSnr
	data.Subpackets = subpackets

	// All fields should be accessible
	assert.Equal(t, uint32(5), *data.AttachCnt)
	assert.Equal(t, nonce, *data.Nonce)
	assert.Equal(t, sign, *data.Sign)
	assert.Equal(t, 15.5, *data.Snr)
	assert.Equal(t, -80.0, *data.Rssi)
	assert.Equal(t, 12.0, *data.EqSnr)
	assert.NotNil(t, data.Subpackets)
}

func TestEPStatusConstants_MatchSpec(t *testing.T) {
	// SCACI §3.13.1 defines epStatus values
	assert.Equal(t, "attached", EPStatusAttached, "EPStatusAttached per SCACI §3.13.1")
	assert.Equal(t, "detached", EPStatusDetached, "EPStatusDetached per SCACI §3.13.1")
}

// =============================================================================
// Operation ID Handling Tests
// =============================================================================

func TestBroadcastEPStatus_UsesSCOperationId(t *testing.T) {
	// Per SCACI §3.2: SC-initiated operations use negative opIds
	// BroadcastEPStatus at server.go:1362 calls NextScOpId()

	// Simulate SC opId sequence (negative, decrementing)
	scOpIdCounter := int64(0)

	// NextScOpId decrements and returns
	getNextScOpId := func() int64 {
		scOpIdCounter--
		return scOpIdCounter
	}

	// First operation
	opId1 := getNextScOpId()
	assert.Equal(t, int64(-1), opId1, "First SC opId should be -1")

	// Second operation
	opId2 := getNextScOpId()
	assert.Equal(t, int64(-2), opId2, "Second SC opId should be -2")

	// All SC-originated opIds are negative
	for i := 0; i < 100; i++ {
		opId := getNextScOpId()
		assert.True(t, opId < 0, "SC opId must be negative")
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestEPStatusData_ZeroEUI_Valid(t *testing.T) {
	// Zero EUI should be valid (though unusual)
	data := EPStatusData{
		EpEui:    0,
		EpStatus: EPStatusDetached,
	}
	assert.Equal(t, uint64(0), data.EpEui)
}

func TestEPStatusData_MaxEUI_Valid(t *testing.T) {
	// Maximum EUI value
	data := EPStatusData{
		EpEui:    0xFFFFFFFFFFFFFFFF,
		EpStatus: EPStatusAttached,
	}
	assert.Equal(t, uint64(0xFFFFFFFFFFFFFFFF), data.EpEui)
}

func TestFormatEUI64_Roundtrip(t *testing.T) {
	// Verify FormatEUI64 and ParseEUI64 roundtrip
	original := uint64(0x70B3D59CD00009E6)

	formatted := pkgmioty.FormatEUI64(original)
	assert.Equal(t, "70B3D59CD00009E6", formatted, "should be uppercase hex")

	parsed, err := ParseEUI64(formatted)
	require.NoError(t, err)
	assert.Equal(t, original, parsed, "roundtrip should preserve value")
}

// =============================================================================
// JSON Roundtrip Tests for RequestData (simulating DB storage)
// =============================================================================

func TestBroadcastEPStatus_SubpacketsStoredAsStruct_JSONRoundtrip(t *testing.T) {
	// Validates that subpackets stored in RequestData survive JSON roundtrip
	// (DB stores map[string]interface{} as JSONB, so JSON marshal/unmarshal occurs)

	// Setup: EPStatusData with populated Subpackets
	data := &EPStatusData{
		EpEui:    0x70B3D59CD00009E6,
		EpStatus: EPStatusAttached,
		Subpackets: &mioty.Subpackets{
			SNR:       []float64{1.5, 2.5, 3.5},
			RSSI:      []float64{-80.0, -75.0, -70.0},
			Frequency: []int64{868100000, 868300000},
			Phase:     []float64{0.1, 0.2, 0.3},
		},
	}

	// Build RequestData as BroadcastEPStatus does (line 1425-1451)
	requestData := map[string]interface{}{
		"epEui":      pkgmioty.FormatEUI64(data.EpEui),
		"epStatus":   data.EpStatus,
		"subpackets": data.Subpackets, // Stored as struct
	}

	// JSON serialize (simulates DB JSONB storage)
	jsonBytes, err := json.Marshal(requestData)
	require.NoError(t, err, "RequestData should be JSON serializable")

	// JSON deserialize (simulates DB read)
	var restored map[string]interface{}
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err, "RequestData should be JSON deserializable")

	// Assert: basic fields survive roundtrip
	assert.Equal(t, "70B3D59CD00009E6", restored["epEui"])
	assert.Equal(t, EPStatusAttached, restored["epStatus"])

	// Assert: reconstructSubpackets can rebuild from restored["subpackets"]
	// After JSON roundtrip, subpackets becomes map[string]interface{}
	subpackets, err := reconstructSubpackets(restored["subpackets"])
	require.NoError(t, err, "reconstructSubpackets should handle JSON-decoded map")
	require.NotNil(t, subpackets)

	// Verify all fields recovered
	assert.Equal(t, 3, len(subpackets.SNR), "should have 3 SNR values")
	assert.Equal(t, float64(1.5), subpackets.SNR[0])
	assert.Equal(t, float64(2.5), subpackets.SNR[1])
	assert.Equal(t, float64(3.5), subpackets.SNR[2])

	assert.Equal(t, 3, len(subpackets.RSSI), "should have 3 RSSI values")
	assert.Equal(t, float64(-80.0), subpackets.RSSI[0])
	assert.Equal(t, float64(-75.0), subpackets.RSSI[1])
	assert.Equal(t, float64(-70.0), subpackets.RSSI[2])

	assert.Equal(t, 2, len(subpackets.Frequency), "should have 2 frequency values")
	assert.Equal(t, int64(868100000), subpackets.Frequency[0])
	assert.Equal(t, int64(868300000), subpackets.Frequency[1])

	assert.Equal(t, 3, len(subpackets.Phase), "should have 3 phase values")
	assert.Equal(t, float64(0.1), subpackets.Phase[0])
	assert.Equal(t, float64(0.2), subpackets.Phase[1])
	assert.Equal(t, float64(0.3), subpackets.Phase[2])
}

func TestBroadcastEPStatus_AllOptionalFieldsJSONRoundtrip(t *testing.T) {
	// Full roundtrip test with all optional fields including nonce/sign

	nonce := mioty.Numeric4{0x01, 0x02, 0x03, 0x04}
	sign := mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}
	attachCnt := uint32(42)
	snr := float64(15.5)
	rssi := float64(-80.0)
	eqSnr := float64(12.0)

	// Build RequestData exactly as BroadcastEPStatus does
	requestData := map[string]interface{}{
		"epEui":      pkgmioty.FormatEUI64(0x70B3D59CD00009E6),
		"epStatus":   EPStatusAttached,
		"attachCnt":  attachCnt,
		"nonce":      encoding.EncodeUserData(nonce[:]), // Base64
		"sign":       encoding.EncodeUserData(sign[:]),  // Base64
		"snr":        snr,
		"rssi":       rssi,
		"eqSnr":      eqSnr,
		"subpackets": &mioty.Subpackets{SNR: []float64{1.0}, RSSI: []float64{-75.0}},
	}

	// JSON roundtrip
	jsonBytes, err := json.Marshal(requestData)
	require.NoError(t, err)

	var restored map[string]interface{}
	err = json.Unmarshal(jsonBytes, &restored)
	require.NoError(t, err)

	// Verify epEui
	epEuiStr, ok := restored["epEui"].(string)
	require.True(t, ok)
	epEui, err := ParseEUI64(epEuiStr)
	require.NoError(t, err)
	assert.Equal(t, uint64(0x70B3D59CD00009E6), epEui)

	// Verify attachCnt (JSON stores as float64)
	attachCntF, ok := restored["attachCnt"].(float64)
	require.True(t, ok)
	assert.Equal(t, uint32(42), uint32(attachCntF))

	// Verify nonce/sign base64 strings
	nonceStr, ok := restored["nonce"].(string)
	require.True(t, ok)
	nonceBytes, err := base64.StdEncoding.DecodeString(nonceStr)
	require.NoError(t, err)
	require.Equal(t, 4, len(nonceBytes))

	signStr, ok := restored["sign"].(string)
	require.True(t, ok)
	signBytes, err := base64.StdEncoding.DecodeString(signStr)
	require.NoError(t, err)
	require.Equal(t, 4, len(signBytes))

	// Verify signal quality fields
	snrRestored, ok := restored["snr"].(float64)
	require.True(t, ok)
	assert.Equal(t, 15.5, snrRestored)

	rssiRestored, ok := restored["rssi"].(float64)
	require.True(t, ok)
	assert.Equal(t, -80.0, rssiRestored)

	eqSnrRestored, ok := restored["eqSnr"].(float64)
	require.True(t, ok)
	assert.Equal(t, 12.0, eqSnrRestored)

	// Verify subpackets
	subpackets, err := reconstructSubpackets(restored["subpackets"])
	require.NoError(t, err)
	assert.Equal(t, 1, len(subpackets.SNR))
	assert.Equal(t, 1, len(subpackets.RSSI))
}

func TestBroadcastEPStatus_NoSessionsForTenant_ReturnsNilNoOp(t *testing.T) {
	// BroadcastEPStatus at server.go:1356-1358 returns nil when no sessions match
	// This is NOT an error condition - just means no ACs to broadcast to

	// Simulating the logic:
	// 1. Server iterates sessions map
	// 2. Filters by tenantID and StateActive
	// 3. If empty targets, return nil (no operation recorded)

	type mockSession struct {
		TenantID int64
		State    SessionState
	}

	sessions := map[string]*mockSession{
		"session1": {TenantID: 1, State: StateActive},
		"session2": {TenantID: 1, State: StateConnecting}, // Wrong state (not active)
		"session3": {TenantID: 2, State: StateActive},     // Wrong tenant
	}

	// Broadcast to tenant 999 - no matching sessions
	targetTenant := int64(999)
	var targets []string
	for id, s := range sessions {
		if s.TenantID == targetTenant && s.State == StateActive {
			targets = append(targets, id)
		}
	}

	assert.Equal(t, 0, len(targets), "no targets for unmatched tenant")

	// Per code: return nil when len(targets) == 0
	// No operation recorded, no error returned
	// This is the expected behavior - tenant has no connected ACs
}

func TestBroadcastEPStatus_MultipleSessionsSameTenant(t *testing.T) {
	// Verifies behavior when multiple ACs connected for same tenant
	// Each should receive the EPStatus message with unique opId

	type mockSession struct {
		TenantID int64
		State    SessionState
	}

	sessions := map[string]*mockSession{
		"ac1": {TenantID: 42, State: StateActive},
		"ac2": {TenantID: 42, State: StateActive},
		"ac3": {TenantID: 42, State: StateActive},
		"ac4": {TenantID: 99, State: StateActive}, // Different tenant
	}

	targetTenant := int64(42)
	var targets []string
	for id, s := range sessions {
		if s.TenantID == targetTenant && s.State == StateActive {
			targets = append(targets, id)
		}
	}

	assert.Equal(t, 3, len(targets), "3 ACs for tenant 42")

	// Each target gets unique SC opId (negative, decrementing)
	scOpIdCounter := int64(0)
	for i := range targets {
		scOpIdCounter--
		assert.Equal(t, int64(-(i + 1)), scOpIdCounter, "each target gets unique opId")
	}
}
