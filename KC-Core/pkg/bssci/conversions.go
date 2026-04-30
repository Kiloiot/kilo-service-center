// Package bssci type conversion helpers with overflow protection
package bssci

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
