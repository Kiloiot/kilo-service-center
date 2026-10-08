package bssci

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"

	// Shared MIOTY helpers (FormatEUI64, EPStatus)

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Message represents a BSSCI protocol message
type Message struct {
	Command    string      `json:"command" msgpack:"command"`
	OpId       int64       `json:"opId" msgpack:"opId"` //nolint:revive // BSSCI §2.5 requires lowercase 'd' (opId not opId)
	Data       interface{} `json:"-" msgpack:"-"`
	RawPayload []byte      `json:"-" msgpack:"-"` // Original wire bytes (MessagePack/JSON) for forensic analysis
}

// tryExtractFromMap attempts to extract command and opId from a map.
// Returns (command, opId, ok). If ok=false, caller should fall back to JSON round-trip.
func tryExtractFromMap(v map[string]interface{}) (string, int64, bool) {
	// Try "command" first, then "commandType"
	var command string
	if cmdVal, exists := v["command"]; exists {
		if cmd, ok := cmdVal.(string); ok {
			command = cmd
		} else {
			return "", 0, false // Wrong type, needs JSON fallback
		}
	} else if typeVal, exists := v["commandType"]; exists {
		if cmd, ok := typeVal.(string); ok {
			command = cmd
		} else {
			return "", 0, false
		}
	} else {
		return "", 0, false // Neither field present
	}

	// Extract opId via the canonical operation ID parsing (BSSCI §5.2)
	opIdVal, hasOpId := v["opId"]
	if !hasOpId {
		return "", 0, false
	}

	opId, ok := parseOpID(opIdVal)
	if !ok {
		return "", 0, false // Wrong type, needs JSON fallback
	}

	return command, opId, true
}

// outboundEnvelope exposes the wire envelope (command, opId) of typed BSSCI
// messages embedding mioty.BaseMessage without serialization round-trips.
type outboundEnvelope interface {
	EnvelopeCommand() string
	EnvelopeOpID() int64
}

// wrapOutboundMessage converts interface{} to *Message for consistent RawPayload capture.
// Map payloads keep their original values; typed structs (ConnectResponse, etc.)
// are retained as the original typed payload so uint64 fields such as scEui are
// encoded exactly (no JSON float64 projection).
func (s *Server) wrapOutboundMessage(msg interface{}) (*Message, error) {
	switch v := msg.(type) {
	case *Message:
		return v, nil
	case map[string]interface{}:
		command, opId, ok := tryExtractFromMap(v)
		if !ok {
			return nil, errMapMissingCommandOpIDEnvelope
		}
		// Ensure canonical envelope keys on the original map
		v["command"] = command
		v["opId"] = opId

		return &Message{
			Command: command,
			OpId:    opId,
			Data:    v,
		}, nil
	default:
		env, ok := msg.(outboundEnvelope)
		if !ok {
			return nil, fmt.Errorf(errFmtMessageTypeNoOpID, msg)
		}
		command := env.EnvelopeCommand()
		if command == "" {
			return nil, fmt.Errorf(errFmtMessageMissingCommandField, msg)
		}

		return &Message{
			Command: command,
			OpId:    env.EnvelopeOpID(),
			Data:    msg,
		}, nil
	}
}

// outboundValidationProjection builds the map used for outbound field-catalog
// validation. Typed payloads are projected through MessagePack (uint64-exact);
// map payloads are validated directly. The projection is only inspected -
// encoding always uses the original payload.
func outboundValidationProjection(payload interface{}) (map[string]interface{}, error) {
	if m, ok := payload.(map[string]interface{}); ok {
		return m, nil
	}
	raw, err := msgpack.Marshal(payload)
	if err != nil {
		return nil, &CatalogError{Token: errOutboundMarshalFailed, Posix: POSIX_EPROTO}
	}
	var projection map[string]interface{}
	if err := msgpack.Unmarshal(raw, &projection); err != nil {
		return nil, &CatalogError{Token: errOutboundMarshalFailed, Posix: POSIX_EPROTO}
	}
	return projection, nil
}

// HandlerFunc handles a specific command
type HandlerFunc func(s *Server, session *Session, msg *Message, data map[string]interface{}) error

// PendingOperation represents an operation that needs to be tracked for MIOTY session resume
type PendingOperation struct {
	OperationID   int64                  // MIOTY operation ID
	OperationType string                 // Operation type (attPrp, detPrp, dlDataQueue, etc.)
	Message       map[string]interface{} // Complete operation message for reissue
	Endpoint      []byte                 // Endpoint EUI (if applicable)
	MACType       int                    // MAC type for VM operations
	Data          []byte                 // Data payload for operations
	Timestamp     time.Time              // Timestamp for operations
	Metadata      map[string]interface{} // Additional metadata
	CreatedAt     time.Time              // When operation was created
}

// SessionOpKey is a composite key for pending operations to prevent operation ID collision
// across concurrent base station sessions (BSSCI §§5.7-5.8.3).
// Example: BS_A and BS_B can both send detach opId=100 in parallel without conflict.
// Exported for use by StatusService to construct keys directly.
type SessionOpKey struct {
	SessionID   string // Session.ID for in-memory map lookups (ephemeral, not persisted)
	OperationID int64  // MIOTY operation ID (BS-issued positive or SC-issued negative)
}

// detachMetadata holds typed detach operation metadata for crash-safe persistence.
// Per BSSCI §5.7, prevents JSON round-trip type drift (float64/base64 issues).
// Use detachMetadataToMap() before persistPendingOperation, mapToDetachMetadata() on resume.
type detachMetadata struct {
	EpEui            uint64    // End Point EUI as uint64 (not float64 after JSON round-trip)
	EndpointID       int64     // Database endpoint.ID to avoid refetch in handleDetachComplete
	PacketCnt        uint32    // Packet counter as uint32
	Signature        []byte    // 4-byte signature (not base64-encoded string)
	RxTime           int64     // Reception timestamp (nanoseconds)
	SNR              float64   // Signal-to-noise ratio (dB)
	RSSI             float64   // Signal strength (dBm)
	EqSnr            *float64  // Optional AWGN equivalent SNR (dB)
	Profile          *string   // Optional MIOTY profile (e.g., "eu1")
	RxDuration       *int64    // Optional reception duration (ns)
	TenantID         int64     // Endpoint owner tenant ID for crash-safe tenant propagation
	OrgUUID          uuid.UUID // Endpoint owner organization UUID for roaming support
	ValidationStatus string    // One of: ValidationStatusValidated | ValidationStatusUnverified | ValidationStatusInvalidSignature
}

// detachMetadataToMap converts typed detachMetadata to map for JSON persistence.
// Prevents type drift by explicitly converting uint64/uint32/[]byte before marshal.
func detachMetadataToMap(m *detachMetadata) map[string]interface{} {
	result := map[string]interface{}{
		"epEui":               mioty.FormatEUI64(m.EpEui),
		metadataKeyEndpointID: m.EndpointID,
		"packetCnt":           m.PacketCnt,
		"signature":           m.Signature,
		"rxTime":              m.RxTime,
		"snr":                 m.SNR,
		"rssi":                m.RSSI,
		"tenantId":            m.TenantID,
		"validationStatus":    m.ValidationStatus, // Issue #4: signature validation tracking for crash-safe resume
	}
	// Store orgUuid only when valid (not nil UUID)
	if m.OrgUUID != uuid.Nil {
		result["orgUuid"] = m.OrgUUID.String()
	}
	if m.EqSnr != nil {
		result["eqSnr"] = *m.EqSnr
	}
	if m.Profile != nil {
		result["profile"] = *m.Profile
	}
	if m.RxDuration != nil {
		result["rxDuration"] = *m.RxDuration
	}
	return result
}

// recordBytes reads a byte field of a recovery record: the bytes it was
// recorded with, or the base64 text its JSON storage turned them into.
func recordBytes(value interface{}) ([]byte, bool) {
	switch v := value.(type) {
	case []byte:
		return v, true
	case string:
		decoded, err := base64.StdEncoding.DecodeString(v)
		return decoded, err == nil
	default:
		return nil, false
	}
}

// mapToDetachMetadata reconstructs typed detachMetadata from JSON-decoded map.
// Normalizes float64 → int64 → uint32/uint64 and validates 4-byte signature.
// Returns nil if required fields missing or signature invalid.
func mapToDetachMetadata(m map[string]interface{}) *detachMetadata {
	if m == nil {
		return nil
	}

	// Extract required fields with type safety
	epEui, ok := parseMetadataEUI(m["epEui"])
	if !ok {
		return nil
	}

	// Extract endpointID (optional for backward compatibility with old pending operations)
	// Canonical numeric coercion accepts both legacy float64-decoded values and
	// the strict json.Number decode used on resume (exact uint64/int64 range).
	var endpointID int64
	if id, err := coerceInt64(m[metadataKeyEndpointID]); err == nil {
		endpointID = id
	}

	packetCntInt, err := coerceInt64(m["packetCnt"])
	if err != nil {
		return nil
	}
	rxTime, err := coerceInt64(m["rxTime"])
	if err != nil {
		return nil
	}
	snr, err := coerceFloat64(m["snr"])
	if err != nil {
		return nil
	}
	rssi, err := coerceFloat64(m["rssi"])
	if err != nil {
		return nil
	}

	signature, ok := recordBytes(m["signature"])
	if !ok || len(signature) != 4 {
		return nil
	}

	// Build typed metadata with safe conversions (using conversions.go helpers)
	packetCnt, errToken := safeUint32(packetCntInt)
	if errToken != "" {
		return nil // Failed conversion
	}

	result := &detachMetadata{
		EpEui:      epEui,
		EndpointID: endpointID,
		PacketCnt:  packetCnt,
		Signature:  signature,
		RxTime:     rxTime,
		SNR:        snr,
		RSSI:       rssi,
	}

	// Extract tenantId (optional for backward compatibility with old pending operations)
	if tenantID, err := coerceInt64(m["tenantId"]); err == nil {
		result.TenantID = tenantID
	}

	// Extract orgUuid (optional for backward compatibility)
	if orgUUIDStr, ok := m["orgUuid"].(string); ok {
		if parsedUUID, err := uuid.Parse(orgUUIDStr); err == nil {
			result.OrgUUID = parsedUUID
		}
	}

	// Optional fields
	if eqSnr, err := coerceFloat64(m["eqSnr"]); err == nil && m["eqSnr"] != nil {
		result.EqSnr = &eqSnr
	}
	if profileStr, ok := m["profile"].(string); ok {
		result.Profile = &profileStr
	}
	if rxDur, err := coerceInt64(m["rxDuration"]); err == nil && m["rxDuration"] != nil {
		result.RxDuration = &rxDur
	}

	// validationStatus is optional for operations persisted before it existed;
	// a record that never carried one defaults to unverified - a missing field
	// must never grant a validation the operation did not earn.
	if validationStatus, ok := m["validationStatus"].(string); ok {
		result.ValidationStatus = validationStatus
	} else {
		result.ValidationStatus = ValidationStatusUnverified
	}

	return result
}
