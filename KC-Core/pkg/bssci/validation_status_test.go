package bssci

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetachValidationStatusKnownEndpoint verifies that detach operations
// for known endpoints are marked with "validated" status (Issue #4).
func TestDetachValidationStatusKnownEndpoint(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(201)
		epEui    = TestEpEuiValidation
		tenantID = int64(100)
	)

	endpoint := buildTestEndpoint(epEui, tenantID)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: "detach", OpId: opID, Data: payload}

	// Execute detach handler
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// Use StatusService to verify pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "detach must store pending operation")
	require.NotNil(t, pending, "pending operation should exist")

	// Verify ValidationStatus in metadata
	validationStatus, exists := pending.Metadata["validationStatus"]
	require.True(t, exists, "validationStatus must be present in metadata")
	assert.Equal(t, ValidationStatusValidated, validationStatus,
		"known endpoint must have 'validated' status")
}

// TestDetachValidationStatusUnknownEndpoint verifies that detach operations
// for unknown endpoints are marked with "unknown_endpoint" status (Issue #4).
func TestDetachValidationStatusUnknownEndpoint(t *testing.T) {
	t.Parallel()

	const (
		opID  = int64(202)
		epEui = TestEpEuiValidation
	)

	// Create test environment with NO endpoint in repository
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, nil) // nil endpoint = unknown endpoint

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: "detach", OpId: opID, Data: payload}

	// Execute detach handler
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// Use StatusService to verify pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "detach must store pending operation even for unknown endpoints")
	require.NotNil(t, pending, "pending operation should exist")

	// Verify ValidationStatus in metadata
	validationStatus, exists := pending.Metadata["validationStatus"]
	require.True(t, exists, "validationStatus must be present in metadata")
	assert.Equal(t, ValidationStatusUnknownEndpoint, validationStatus,
		"unknown endpoint must have 'unknown_endpoint' status")
}

// TestDetachMetadataValidationStatusRoundTrip verifies that ValidationStatus
// survives the JSON round-trip through detachMetadataToMap/mapToDetachMetadata (Issue #4).
func TestDetachMetadataValidationStatusRoundTrip(t *testing.T) {
	t.Parallel()

	// Test all three validation statuses
	testCases := []struct {
		name             string
		validationStatus string
	}{
		{"Validated", ValidationStatusValidated},
		{"UnknownEndpoint", ValidationStatusUnknownEndpoint},
		{"Disabled", ValidationStatusDisabled},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create typed metadata with validation status
			original := &detachMetadata{
				EpEui:            TestEpEui01, // Issue #3-4 Fix A3: Endpoint EUI, not Service Center EUI
				EndpointID:       1001,
				PacketCnt:        42,
				Signature:        []byte{0xAA, 0xBB, 0xCC, 0xDD},
				RxTime:           1234567890000000000,
				SNR:              12.5,
				RSSI:             -80.0,
				TenantID:         100,
				OrgUUID:          uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				ValidationStatus: tc.validationStatus,
			}

			// Convert to map
			metadataMap := detachMetadataToMap(original)

			// Simulate JSON marshal/unmarshal which converts numbers to float64
			simulatedJSONMap := map[string]interface{}{
				"epEui":            metadataMap["epEui"],
				"endpointID":       float64(metadataMap["endpointID"].(int64)),
				"packetCnt":        float64(metadataMap["packetCnt"].(uint32)),
				"signature":        metadataMap["signature"],
				"rxTime":           float64(metadataMap["rxTime"].(int64)),
				"snr":              metadataMap["snr"],
				"rssi":             metadataMap["rssi"],
				"tenantId":         float64(metadataMap["tenantId"].(int64)),
				"validationStatus": metadataMap["validationStatus"],
			}
			if orgUUID, exists := metadataMap["orgUuid"]; exists {
				simulatedJSONMap["orgUuid"] = orgUUID
			}

			// Verify validationStatus is in map
			validationStatus, exists := simulatedJSONMap["validationStatus"]
			require.True(t, exists, "validationStatus must be in map")
			assert.Equal(t, tc.validationStatus, validationStatus)

			// Reconstruct from JSON-decoded map
			reconstructed := mapToDetachMetadata(simulatedJSONMap)
			require.NotNil(t, reconstructed, "mapToDetachMetadata must succeed")

			// Verify all fields preserved, especially ValidationStatus
			assert.Equal(t, original.EpEui, reconstructed.EpEui)
			assert.Equal(t, original.EndpointID, reconstructed.EndpointID)
			assert.Equal(t, original.PacketCnt, reconstructed.PacketCnt)
			assert.Equal(t, original.Signature, reconstructed.Signature)
			assert.Equal(t, original.RxTime, reconstructed.RxTime)
			assert.Equal(t, original.SNR, reconstructed.SNR)
			assert.Equal(t, original.RSSI, reconstructed.RSSI)
			assert.Equal(t, original.TenantID, reconstructed.TenantID)
			assert.Equal(t, original.OrgUUID, reconstructed.OrgUUID)
			assert.Equal(t, original.ValidationStatus, reconstructed.ValidationStatus,
				"ValidationStatus must survive round-trip")
		})
	}
}

// TestMapToDetachMetadataBackwardCompatibility verifies that old pending operations
// without validationStatus field default to "validated" for backward compatibility (Issue #4).
func TestMapToDetachMetadataBackwardCompatibility(t *testing.T) {
	t.Parallel()

	// Create metadata map WITHOUT validationStatus (simulates old persisted operation)
	oldMetadataMap := map[string]interface{}{
		"epEui":      float64(TestEpEui01), // Issue #3-4 Fix A3: Endpoint EUI, not Service Center EUI
		"endpointID": float64(1001),
		"packetCnt":  float64(42),
		"signature":  []byte{0xAA, 0xBB, 0xCC, 0xDD},
		"rxTime":     float64(1234567890000000000),
		"snr":        12.5,
		"rssi":       -80.0,
		"tenantId":   float64(100),
		// validationStatus intentionally omitted
	}

	// Reconstruct from map
	reconstructed := mapToDetachMetadata(oldMetadataMap)
	require.NotNil(t, reconstructed, "mapToDetachMetadata must succeed for old metadata")

	// Verify ValidationStatus defaults to "validated" for backward compatibility
	assert.Equal(t, ValidationStatusValidated, reconstructed.ValidationStatus,
		"old operations without validationStatus must default to 'validated'")
}

// TestDetachValidationStatusConstants verifies that validation status constants
// are properly defined and have expected string values (Issue #4).
func TestDetachValidationStatusConstants(t *testing.T) {
	t.Parallel()

	// Verify constant values match spec
	assert.Equal(t, "validated", ValidationStatusValidated,
		"ValidationStatusValidated must be 'validated'")
	assert.Equal(t, "unknown_endpoint", ValidationStatusUnknownEndpoint,
		"ValidationStatusUnknownEndpoint must be 'unknown_endpoint'")
	assert.Equal(t, "disabled", ValidationStatusDisabled,
		"ValidationStatusDisabled must be 'disabled'")
}

// TestDetachMetadataValidationStatusPersistence verifies that ValidationStatus
// is correctly persisted in the detachMetadata struct and survives operations (Issue #4).
func TestDetachMetadataValidationStatusPersistence(t *testing.T) {
	t.Parallel()

	const (
		opID     = int64(203)
		epEui    = TestEpEuiValidation
		tenantID = int64(100)
	)

	endpoint := buildTestEndpoint(epEui, tenantID)
	env := newDetachTestEnv(t, &Config{
		DetachSignatureValidationEnabled: false,
		MessageEncoding:                  EncodingJSON,
	}, endpoint)

	payload := buildDetachPayload(epEui)
	msg := &Message{Command: "detach", OpId: opID, Data: payload}

	// Execute detach handler
	require.NoError(t, env.server.handleDetach(env.server, env.session, msg, payload))

	// Use StatusService to retrieve pending operation
	pending, err := env.server.statusSvc.GetPendingOperation(env.session, opID)
	require.NoError(t, err, "detach must store pending operation")
	require.NotNil(t, pending, "pending operation should exist")

	// Verify ValidationStatus in metadata map (native types, not JSON-decoded)
	validationStatus, exists := pending.Metadata["validationStatus"]
	require.True(t, exists, "validationStatus must be in metadata")
	assert.Equal(t, ValidationStatusValidated, validationStatus,
		"ValidationStatus must be 'validated' for known endpoint")

	// Verify other key fields are set
	endpointID, exists := pending.Metadata["endpointID"]
	require.True(t, exists, "endpointID must be in metadata")
	assert.Equal(t, endpoint.ID, endpointID.(int64),
		"EndpointID must be set for known endpoint")

	tenantIDValue, exists := pending.Metadata["tenantId"]
	require.True(t, exists, "tenantId must be in metadata")
	assert.Equal(t, tenantID, tenantIDValue.(int64),
		"TenantID must match endpoint owner")
}

// TestDetachValidationStatusWithOptionalFields verifies that ValidationStatus
// works correctly alongside optional detachMetadata fields (Issue #4).
func TestDetachValidationStatusWithOptionalFields(t *testing.T) {
	t.Parallel()

	// Create metadata with all optional fields populated
	original := &detachMetadata{
		EpEui:            TestEpEui01, // Issue #3-4 Fix A3: Endpoint EUI, not Service Center EUI
		EndpointID:       1001,
		PacketCnt:        42,
		Signature:        []byte{0xAA, 0xBB, 0xCC, 0xDD},
		RxTime:           1234567890000000000,
		SNR:              12.5,
		RSSI:             -80.0,
		TenantID:         100,
		OrgUUID:          uuid.MustParse("12345678-1234-1234-1234-123456789012"),
		ValidationStatus: ValidationStatusValidated,
	}

	// Add optional fields
	eqSnr := 13.0
	original.EqSnr = &eqSnr
	profile := "eu1"
	original.Profile = &profile
	rxDuration := int64(500000)
	original.RxDuration = &rxDuration

	// Convert to map
	metadataMap := detachMetadataToMap(original)

	// Simulate JSON marshal/unmarshal (epEui is stored as hex string by detachMetadataToMap)
	simulatedJSONMap := map[string]interface{}{
		"epEui":            metadataMap["epEui"], // string (hex-formatted EUI)
		"endpointID":       float64(metadataMap["endpointID"].(int64)),
		"packetCnt":        float64(metadataMap["packetCnt"].(uint32)),
		"signature":        metadataMap["signature"],
		"rxTime":           float64(metadataMap["rxTime"].(int64)),
		"snr":              metadataMap["snr"],
		"rssi":             metadataMap["rssi"],
		"tenantId":         float64(metadataMap["tenantId"].(int64)),
		"validationStatus": metadataMap["validationStatus"],
		"eqSnr":            metadataMap["eqSnr"],
		"profile":          metadataMap["profile"],
		"rxDuration":       float64(metadataMap["rxDuration"].(int64)),
	}
	if orgUUID, exists := metadataMap["orgUuid"]; exists {
		simulatedJSONMap["orgUuid"] = orgUUID
	}

	// Reconstruct from JSON-decoded map
	reconstructed := mapToDetachMetadata(simulatedJSONMap)
	require.NotNil(t, reconstructed, "mapToDetachMetadata must succeed")

	// Verify all fields including ValidationStatus
	assert.Equal(t, original.ValidationStatus, reconstructed.ValidationStatus)
	require.NotNil(t, reconstructed.EqSnr)
	assert.Equal(t, *original.EqSnr, *reconstructed.EqSnr)
	require.NotNil(t, reconstructed.Profile)
	assert.Equal(t, *original.Profile, *reconstructed.Profile)
	require.NotNil(t, reconstructed.RxDuration)
	assert.Equal(t, *original.RxDuration, *reconstructed.RxDuration)
}
