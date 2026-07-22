package scaciservices

import (
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
)

// sessionValidator implements scaci.SessionValidator interface
//
// This service validates SCACI Connect message fields per SCACI §3.3.
// All validations return error tokens from pkg/scaci/errors_catalog.go
// for consistent error reporting across the SCACI protocol layer.
//
// Spec References:
//   - §3.3-02: Connect must use opId == 0 (validated in handler, not here)
//   - §3.3.1-01: Mandatory fields (version, acEui, snAcUuid)
//   - §3.3: Resume fields must be paired (both nil or both non-nil)
type sessionValidator struct{}

// NewSessionValidator creates a new session validator service
//
// This is a stateless service - no dependencies required.
func NewSessionValidator() scaci.SessionValidator {
	return &sessionValidator{}
}

// ValidateConnectFields validates Connect message fields per SCACI §3.3
//
// Validation Rules:
//  1. Version field must not be empty (§3.3.1-01)
//  2. AcEui must not be zero (§3.3.1-01)
//  3. SnAcUUID must not be zero-valued (§3.3.1-01)
//  4. Resume fields must be paired:
//     - Fresh session: both SnAcOpID and SnScOpID must be nil
//     - Resume session: both SnAcOpID and SnScOpID must be non-nil
//     - Partial resume (one nil, one non-nil) is INVALID
//
// Parameters:
//   - req: Connect message to validate
//   - opId: Operation ID (validated by handler, not used here)
//
// Returns:
//   - Empty string if validation passes
//   - Error token string if validation fails (from errors_catalog.go)
func (v *sessionValidator) ValidateConnectFields(req *scaci.Connect) string {
	// Validation 1: Version must not be empty (§3.3.1-01)
	if req.Version == "" {
		return scaci.ErrMissingVersion
	}

	// Validation 2: AcEui must not be zero (§3.3.1-01)
	if req.AcEui == 0 {
		return scaci.ErrMissingAcEui
	}

	// Validation 3: SnAcUUID must not be zero-valued (§3.3.1-01)
	if isUUIDZero(req.SnAcUUID) {
		return scaci.ErrSnAcUUIDZero
	}

	// Validation 4: Resume fields must be paired (§3.3)
	hasAcOpID := req.SnAcOpId != nil
	hasScOpID := req.SnScOpId != nil

	// Case 1: Fresh session (both nil) - OK
	// Case 2: Resume session (both non-nil) - OK
	// Case 3: Partial resume (one nil, one non-nil) - ERROR
	if hasAcOpID && !hasScOpID {
		return scaci.ErrSnScOpIdRequired
	}
	if !hasAcOpID && hasScOpID {
		return scaci.ErrSnAcOpIdRequired
	}

	// All validations passed
	return ""
}

// isUUIDZero checks if a UUID16 is all zeros
//
// This is a local helper to avoid exporting uuidIsZero from pkg/scaci/session.go.
// Duplicating this simple logic keeps the validator self-contained.
func isUUIDZero(uuid scaci.UUID16) bool {
	for _, b := range uuid {
		if b != 0 {
			return false
		}
	}
	return true
}
