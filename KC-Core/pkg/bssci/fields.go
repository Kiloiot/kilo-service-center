package bssci

import (
	"math"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func getStringField(data map[string]interface{}, key string, defaultValue string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return defaultValue
}

// getNumericField extracts a signed 64-bit protocol field. Unsigned overflow,
// non-integral floats, and float magnitudes beyond the exact integer range are
// rejected via the canonical numeric coercion (coerceInt64).
func getNumericField(data map[string]interface{}, key string) (int64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}
	v, err := coerceInt64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// getUint64Field extracts an unsigned 64-bit protocol field (e.g. an EUI-64),
// preserving the full uint64 range including values above INT64_MAX. Negative
// values, non-integral floats, and float magnitudes beyond the exact integer
// range are rejected via the canonical numeric coercion (coerceUint64).
func getUint64Field(data map[string]interface{}, key string) (uint64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}
	v, err := coerceUint64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// parseOpID validates and extracts operation ID from interface{} value.
// Per BSSCI §5.2, operation IDs must be precise 64-bit integers; the canonical
// numeric coercion rejects non-integral floats and uint64 overflow.
// Returns (value, true) on success or (0, false) on validation failure.
func parseOpID(value interface{}) (int64, bool) {
	v, err := coerceInt64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

func getBoolField(data map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := data[key].(bool); ok {
		return v
	}
	return defaultValue
}

func getNumericFieldInt(data map[string]interface{}, key string, defaultValue int) int {
	if val, ok := getNumericField(data, key); ok {
		return int(val)
	}
	return defaultValue
}

// getFloatFieldValidated extracts a float field and validates it's actually
// numeric via the canonical numeric coercion (coerceFloat64).
// Returns the value and true if field exists and is numeric, or 0 and false otherwise
func getFloatFieldValidated(data map[string]interface{}, key string) (float64, bool) {
	value, exists := data[key]
	if !exists {
		return 0, false
	}
	v, err := coerceFloat64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// validateByteArray validates a MessagePack byte array (e.g., 4-byte nonce/sign).
// Returns (validBytes, "") on success or (nil, errToken) on validation failure.
// Accepts []byte directly or []interface{} (MessagePack numeric array format).
// All array elements must be integers in range 0-255.
func validateByteArray(data interface{}, fieldName string, expectedLen int) ([]byte, string) {
	switch v := data.(type) {
	case []byte:
		if len(v) != expectedLen {
			// Return appropriate error token based on field name
			if fieldName == fieldNameNonce {
				return nil, errInvalidNonceElement
			}
			return nil, errInvalidSignElement
		}
		return v, ""
	case []interface{}:
		if len(v) != expectedLen {
			if fieldName == fieldNameNonce {
				return nil, errInvalidNonceElement
			}
			return nil, errInvalidSignElement
		}
		result := make([]byte, expectedLen)
		for i, elem := range v {
			// Canonical numeric coercion enforces integer 0-255 range
			b, err := numericToByte(elem)
			if err != nil {
				if fieldName == fieldNameNonce {
					return nil, errInvalidNonceElement
				}
				return nil, errInvalidSignElement
			}
			result[i] = b
		}
		return result, ""
	default:
		// Invalid type
		if fieldName == fieldNameNonce {
			return nil, errInvalidNonceElement
		}
		return nil, errInvalidSignElement
	}
}

func parseMetadataEUI(value interface{}) (uint64, bool) {
	if s, ok := value.(string); ok {
		parsed, err := validation.ParseEUI(s)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	v, err := coerceUint64(value)
	if err != nil {
		return 0, false
	}
	return v, true
}

// extractFloatSlice validates and extracts float64 slice from a wire array
// via the canonical numeric coercion (coerceFloat64).
// Rejects non-numeric values. Returns (slice, true) on success or (nil, false) on failure.
func extractFloatSlice(values []interface{}) ([]float64, bool) {
	result := make([]float64, len(values))
	for i, v := range values {
		f, err := coerceFloat64(v)
		if err != nil {
			// Non-numeric value - reject entire array
			return nil, false
		}
		result[i] = f
	}
	return result, true
}

// extractHertzSlice reads subpacket frequencies at the canonical whole-Hz
// resolution; base stations report them with sub-Hz fractions.
func extractHertzSlice(values []interface{}) ([]int64, bool) {
	frequencies, ok := extractFloatSlice(values)
	if !ok {
		return nil, false
	}
	hertz := make([]int64, len(frequencies))
	for i, f := range frequencies {
		if math.Abs(f) > float64(maxExactFloat64Integer) {
			return nil, false
		}
		hertz[i] = int64(math.Round(f))
	}
	return hertz, true
}

// extractEndpointEUIFromMetadata reads the endpoint EUI persisted in pending
// operation metadata, falling back from "epEui" (attach) to "endpointEUI"
// (detach). Returns 0 when neither key carries a parseable EUI.
func extractEndpointEUIFromMetadata(metadata map[string]interface{}) uint64 {
	if v, ok := metadata["epEui"]; ok {
		if eui, parsed := parseMetadataEUI(v); parsed && eui != 0 {
			return eui
		}
	}
	if v, ok := metadata["endpointEUI"]; ok {
		if eui, parsed := parseMetadataEUI(v); parsed {
			return eui
		}
	}
	return 0
}

// geoFix is a base station position report.
type geoFix struct {
	latitude  float64
	longitude float64
	altitude  float64
}

// geoReport classifies a geoLocation report.
type geoReport int

const (
	// geoAbsent is no report, or one malformed or out of range.
	geoAbsent geoReport = iota
	// geoNoFix is a well-formed report at 0°/0°, how a base station without
	// a position fix fills the field.
	geoNoFix
	// geoFixed is a position.
	geoFixed
)

// parseGeoReport reads an optional geoLocation Numeric[3] [latitude,
// longitude, altitude] (rev1 §5.3.1, §5.5.2) and classifies it.
func parseGeoReport(value interface{}) (geoFix, geoReport) {
	values, ok := value.([]interface{})
	if !ok || len(values) != geoLocationComponents {
		return geoFix{}, geoAbsent
	}
	coords, ok := extractFloatSlice(values)
	if !ok {
		return geoFix{}, geoAbsent
	}
	fix := geoFix{latitude: coords[0], longitude: coords[1], altitude: coords[2]}
	if fix.latitude == 0 && fix.longitude == 0 {
		return geoFix{}, geoNoFix
	}
	inRange := fix.latitude >= models.LatitudeMin && fix.latitude <= models.LatitudeMax &&
		fix.longitude >= models.LongitudeMin && fix.longitude <= models.LongitudeMax
	if !inRange {
		return geoFix{}, geoAbsent
	}
	return fix, geoFixed
}

// parseGeoFix reads a geoLocation report as a position; a report without a
// fix, like an absent, malformed or out-of-range value, yields none.
func parseGeoFix(value interface{}) (geoFix, bool) {
	fix, report := parseGeoReport(value)
	return fix, report == geoFixed
}

// extractSessionUUID extracts UUID from various formats (BSSCI-3.3)
func extractSessionUUID(data interface{}) ([]byte, *CatalogError) {
	if data == nil {
		return nil, &CatalogError{Token: errUUIDDataNil, Posix: POSIX_EPROTO}
	}

	switch v := data.(type) {
	case []interface{}:
		if len(v) != 16 {
			return nil, &CatalogError{Token: errInvalidUUIDLength, Posix: POSIX_EPROTO}
		}
		uuid := make([]byte, 16)
		for i, val := range v {
			if b, ok := val.(int8); ok {
				// MessagePack encodes bytes above 0x7F as negative int8;
				// the two's-complement bit pattern is the intended byte
				uuid[i] = byte(b) //nolint:gosec // G115: intentional two's-complement byte extraction
				continue
			}
			b, err := numericToByte(val)
			if err != nil {
				return nil, &CatalogError{Token: errInvalidUUIDByteType, Posix: POSIX_EPROTO}
			}
			uuid[i] = b
		}
		return uuid, nil
	case []byte:
		if len(v) != 16 {
			return nil, &CatalogError{Token: errInvalidUUIDLength, Posix: POSIX_EPROTO}
		}
		return v, nil
	default:
		return nil, &CatalogError{Token: errUnsupportedUUIDType, Posix: POSIX_EPROTO}
	}
}
