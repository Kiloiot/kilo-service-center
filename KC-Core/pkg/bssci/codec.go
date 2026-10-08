package bssci

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// validateFiniteFloat validates a float64 value is finite and within range.
// Returns empty string on success, error token on validation failure.
func validateFiniteFloat(value, minVal, maxVal float64) string {
	if math.IsNaN(value) {
		return errInvalidFloatNaN
	}
	if math.IsInf(value, 0) {
		return errInvalidFloatInf
	}
	if value < minVal || value > maxVal {
		return errFloatOutOfRange
	}
	return ""
}

// validateOutboundMessage validates an outbound message complies with BSSCI field catalog.
// The map is a validation projection of the payload (see outboundValidationProjection);
// it is inspected only and never replaces the encoded payload.
// Returns nil on success, CatalogError on validation failure.
func (s *Server) validateOutboundMessage(session *Session, msgMap map[string]interface{}) error {
	// Check command field exists
	cmdVal, hasCommand := msgMap["command"]
	if !hasCommand {
		return &CatalogError{Token: errOutboundMissingCommand, Posix: POSIX_EPROTO}
	}

	command, ok := cmdVal.(string)
	if !ok {
		return &CatalogError{Token: errOutboundMissingCommand, Posix: POSIX_EPROTO}
	}

	// Validate opId presence and type (MIOTY requires opId on all outbound frames per §2.5.1)
	opIdVal, hasOpId := msgMap["opId"]
	if !hasOpId {
		return &CatalogError{Token: errOutboundMissingField, Posix: POSIX_EPROTO}
	}

	// Type check - accept numeric types (int64, float64, int, etc.) from JSON/msgpack unmarshaling
	// Accept any value including 0 (valid for conCmp), negative (SC-initiated), or positive (BS-initiated)
	switch opIdVal.(type) {
	case int64, float64, int, int32, uint32, uint64, int8, uint8:
		// Valid numeric types - accept any value
	default:
		return &CatalogError{Token: errOutboundInvalidFieldType, Posix: POSIX_EINVAL}
	}

	// Get allowed fields for this command
	allowedFields, known := mioty.AllowedOutboundFields(command)
	if !known {
		return &CatalogError{Token: errOutboundUnknownCommand, Posix: POSIX_EPROTO}
	}

	// Get mandatory fields for this command
	mandatoryFields, _ := mioty.MandatoryOutboundFields(command)

	// Build allowed field set - INCLUDE BASE FIELDS (command, opId)
	allowedSet := make(map[string]bool)
	allowedSet["command"] = true // Base field - always allowed
	allowedSet["opId"] = true    // Base field - always allowed
	for _, field := range allowedFields {
		allowedSet[field] = true
	}

	// Validate all fields in message are allowed
	ctx := s.sessionContext(session)
	for field := range msgMap {
		if !allowedSet[field] {
			s.logger.WarnContext(ctx,
				LogBSSCIOutboundDisallowedField,
				logger.FieldCommand, command, logger.FieldField, field)
			return &CatalogError{Token: errOutboundExtraField, Posix: POSIX_EPROTO}
		}
	}

	// Validate all mandatory fields are present
	for _, field := range mandatoryFields {
		if _, present := msgMap[field]; !present {
			s.logger.WarnContext(ctx,
				LogBSSCIOutboundMissingMandatoryFieldText,
				logger.FieldCommand, command, logger.FieldField, field)
			return &CatalogError{Token: errOutboundMissingMandatoryField, Posix: POSIX_EPROTO}
		}
	}

	return nil
}

// trimJSONWhitespace strips leading ASCII whitespace and UTF-8 BOM from a byte slice.
// Per RFC 8259, JSON allows insignificant whitespace before or after any token.
// Returns the trimmed slice without modifying the original.
func trimJSONWhitespace(data []byte) []byte {
	// Strip UTF-8 BOM if present
	data = bytes.TrimPrefix(data, []byte(utf8BOM))

	// Strip leading ASCII whitespace (space, tab, newline, carriage return)
	start := 0
	for start < len(data) {
		b := data[start]
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			start++
		} else {
			break
		}
	}

	return data[start:]
}

// detectEncoding performs a non-mutating syntax check to detect message encoding
// Per BSSCI Section 1, messages can be either JSON or MessagePack
// Returns "json" or "msgpack" based on first byte inspection
// Falls back to configDefault for empty or ambiguous payloads
func detectEncoding(rawFrame []byte, configDefault string) string {
	if len(rawFrame) == 0 {
		return configDefault // Use configured default for empty frames
	}

	// Trim leading whitespace and BOM for JSON detection per RFC 8259
	trimmed := trimJSONWhitespace(rawFrame)
	if len(trimmed) == 0 {
		return configDefault
	}

	firstByte := trimmed[0]

	// JSON detection: objects start with '{', arrays with '['
	if firstByte == '{' || firstByte == '[' {
		return EncodingJSON
	}

	// MessagePack detection (map markers):
	// - fixmap: 0x80-0x8f (10000000 to 10001111)
	// - map 16: 0xde
	// - map 32: 0xdf
	// BSSCI messages are typically objects, so we check map markers
	if (firstByte >= msgpackFixmapMin && firstByte <= msgpackFixmapMax) || firstByte == msgpackMap16 || firstByte == msgpackMap32 {
		return EncodingMessagePack
	}

	// Use configured default for ambiguous payloads
	return configDefault
}

// encodeMessage encodes a message using the specified encoding
// Per BSSCI Section 1, supports both JSON and MessagePack encoding
func encodeMessage(msg interface{}, encoding string) ([]byte, error) {
	switch encoding {
	case EncodingJSON:
		return json.Marshal(msg)
	case EncodingMessagePack:
		return msgpack.Marshal(msg)
	default:
		// Default to msgpack per BSSCI spec
		return msgpack.Marshal(msg)
	}
}

// decodeMessage decodes a raw frame using the specified encoding
// Per BSSCI Section 1, supports both JSON and MessagePack encoding
func decodeMessage(rawFrame []byte, encoding string) (map[string]interface{}, error) {
	var data map[string]interface{}

	switch encoding {
	case EncodingJSON:
		return decodeJSONFrame(rawFrame)
	case EncodingMessagePack:
		if err := msgpack.Unmarshal(rawFrame, &data); err != nil {
			return nil, err
		}
		return data, nil
	default:
		// Default to msgpack per BSSCI spec
		if err := msgpack.Unmarshal(rawFrame, &data); err != nil {
			return nil, err
		}
		return data, nil
	}
}

// decodeJSONFrame decodes a JSON-encoded BSSCI frame strictly: numbers are
// preserved as json.Number (the full uint64 EUI range survives decoding), the
// frame must contain exactly one JSON object, and trailing content is rejected.
func decodeJSONFrame(rawFrame []byte) (map[string]interface{}, error) {
	// Trim BOM and leading whitespace per RFC 8259
	trimmed := trimJSONWhitespace(rawFrame)

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()

	var data map[string]interface{}
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf(errFmtTrailingContentAfterJSONFrame, ResolveErrorMessage(errInvalidMessageFormat))
	}
	return data, nil
}

// normalizeStrictDecodedMap converts a strict-decoded (UseNumber) map into the
// value shapes the legacy decoder produced: integral numbers within the exact
// float64 range become float64 (so existing consumers keep working), while
// integers beyond that range stay exact as int64/uint64 - this is what
// preserves full-range EUI-64 and counter values across resume.
func normalizeStrictDecodedMap(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		m[k] = normalizeStrictDecodedValue(v)
	}
	return m
}

func normalizeStrictDecodedValue(v interface{}) interface{} {
	switch t := v.(type) {
	case json.Number:
		if i, err := jsonNumberToInt64(t); err == nil {
			if i >= -int64(maxExactFloat64Integer) && i <= int64(maxExactFloat64Integer) {
				return float64(i)
			}
			return i
		}
		if u, err := jsonNumberToUint64(t); err == nil {
			return u
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t
	case map[string]interface{}:
		return normalizeStrictDecodedMap(t)
	case []interface{}:
		for i, e := range t {
			t[i] = normalizeStrictDecodedValue(e)
		}
		return t
	default:
		return v
	}
}
