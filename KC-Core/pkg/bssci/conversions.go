// Package bssci type conversion helpers with overflow protection
package bssci

import (
	"strconv"
	"strings"
)

// safeUint8 converts int64 to uint8 with bounds checking for BSSCI field assignments
// Returns (converted value, error token if overflow)
//
// Usage pattern - caller MUST propagate error through sendError:
//
//	val, errToken := safeUint8(int64Var, "macType")
//	if errToken != "" {
//	    return s.sendError(session, msg.OpId, ResolveErrorMessage(errToken), mioty.ErrorInvalid)
//	}
//
// POSIX mapping: Always use mioty.ErrorInvalid (EINVAL = 22) for field range violations
// per BSSCI §4 error handling.
func safeUint8(v int64, _ string) (uint8, string) {
	if v < 0 || v > 255 {
		return 0, errInvalidFieldValue
	}
	return uint8(v), ""
}

// safeUint32 converts int64 to uint32 with bounds checking
// See safeUint8 for usage pattern - caller must propagate via sendError
func safeUint32(v int64, _ string) (uint32, string) {
	if v < 0 || v > 4294967295 {
		return 0, errInvalidFieldValue
	}
	return uint32(v), ""
}

// safeUint64 converts int64 to uint64 with bounds checking (negative check only)
// See safeUint8 for usage pattern - caller must propagate via sendError
func safeUint64(v int64, _ string) (uint64, string) {
	if v < 0 {
		return 0, errInvalidFieldValue
	}
	return uint64(v), ""
}

// ParseVersion parses a BSSCI protocol version string "major.minor.patch"
// (BSSCI rev1 §4). Exactly three unsigned ASCII-decimal components are
// required; whitespace, signs, empty or extra components, and values
// overflowing the int range are rejected. Returns a specific CatalogError per
// component to preserve diagnostic precision.
func ParseVersion(version string) (major, minor, patch int, cerr *CatalogError) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, 0, NewCatalogError(errInvalidVersionFormat, POSIX_EPROTO)
	}

	componentTokens := [3]string{errInvalidMajorVersion, errInvalidMinorVersion, errInvalidPatchVersion}
	var values [3]int
	for i, part := range parts {
		v, ok := parseVersionComponent(part)
		if !ok {
			return 0, 0, 0, NewCatalogError(componentTokens[i], POSIX_EPROTO)
		}
		values[i] = v
	}

	return values[0], values[1], values[2], nil
}

// parseVersionComponent parses one unsigned ASCII-decimal version component,
// rejecting signs, whitespace, non-digits, empty strings, and overflow.
func parseVersionComponent(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	// Digits already validated above; Atoi rejects overflow of the native int.
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}
