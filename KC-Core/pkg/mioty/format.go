// Package mioty provides shared MIOTY protocol helpers used by both BSSCI and SCACI.
// IMPORTANT: This package must NOT import SCACI or BSSCI to avoid import cycles.
// Both protocol packages import from here, not vice versa.
package mioty

import (
	"encoding/hex"
	"fmt"
	"time"
)

// Timestamp formatting constants.
// These MUST NOT be defined inline in adapter files.
const (
	// TimestampFormatRFC3339 is the standard timestamp format for exports,
	// with the nanoseconds a MIOTY reception time carries
	TimestampFormatRFC3339 = time.RFC3339Nano
)

// Numeric format strings.
// These MUST NOT be defined inline in adapter files.
const (
	// FormatFloat2DecPattern for SNR/RSSI values with 2 decimal places (used by FormatFloat2Dec)
	FormatFloat2DecPattern = "%.2f"
)

// FormatTimestampRFC3339 formats a Unix nanosecond timestamp as RFC 3339 in
// UTC, keeping the nanoseconds. Centralized formatter for message exports.
func FormatTimestampRFC3339(unixNano int64) string {
	return time.Unix(0, unixNano).UTC().Format(TimestampFormatRFC3339)
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
