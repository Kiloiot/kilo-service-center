// Package mioty provides shared MIOTY protocol helpers used by both BSSCI and SCACI.
// IMPORTANT: This package must NOT import SCACI or BSSCI to avoid import cycles.
// Both protocol packages import from here, not vice versa.
package mioty

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// FormatEUI64 returns uppercase hex representation of an EUI64 (16 chars, zero-padded).
// Used for logging and operation metadata across both protocol packages.
func FormatEUI64(eui uint64) string {
	return fmt.Sprintf("%016X", eui)
}

// EUIDashSeparator joins EUI64 byte pairs in the dashed display format.
const EUIDashSeparator = "-"

// FormatEUI64Dashed returns the uppercase dashed representation of an EUI64
// (XX-XX-XX-XX-XX-XX-XX-XX), used for certificate common names.
func FormatEUI64Dashed(eui uint64) string {
	plain := FormatEUI64(eui)
	pairs := make([]string, 0, len(plain)/2)
	for i := 0; i < len(plain); i += 2 {
		pairs = append(pairs, plain[i:i+2])
	}
	return strings.Join(pairs, EUIDashSeparator)
}

// Timestamp formatting constants.
// These MUST NOT be defined inline in adapter files.
const (
	// TimestampFormatRFC3339 is the standard timestamp format for exports
	TimestampFormatRFC3339 = time.RFC3339
)

// Numeric format strings.
// These MUST NOT be defined inline in adapter files.
const (
	// FormatInt64Pattern for integer values (used by FormatInt)
	FormatInt64Pattern = "%d"
	// FormatFloat2DecPattern for SNR/RSSI values with 2 decimal places (used by FormatFloat2Dec)
	FormatFloat2DecPattern = "%.2f"
)

// FormatTimestampRFC3339 formats a Unix nanosecond timestamp to RFC3339.
// Centralized formatter for message exports.
func FormatTimestampRFC3339(unixNano int64) string {
	return time.Unix(0, unixNano).Format(TimestampFormatRFC3339)
}

// FormatInt formats an int64 to string.
// Centralized formatter for message exports.
func FormatInt(v int64) string {
	return fmt.Sprintf(FormatInt64Pattern, v)
}

// FormatFloat2Dec formats a float64 with 2 decimal places.
// Centralized formatter for SNR/RSSI in message exports.
func FormatFloat2Dec(v float64) string {
	return fmt.Sprintf(FormatFloat2DecPattern, v)
}

// FormatUserDataHex formats user data bytes as lowercase hex string.
// Centralized formatter for message exports.
func FormatUserDataHex(data []byte) string {
	return hex.EncodeToString(data)
}

// DateFormat is the standard date format for analytics/reports.
// Centralized constant for date parsing.
const DateFormat = "2006-01-02"

// EPStatus constants per SCACI §3.13.1 - shared between BSSCI and SCACI
const (
	EPStatusAttached = "attached" // Endpoint attached (OTA or commissioned)
	EPStatusDetached = "detached" // Endpoint detached (OTA or decommissioned)
)
