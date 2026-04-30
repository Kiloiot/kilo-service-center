// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseSemanticVersion parses "major.minor.patch" format with validation
//
// Validates semantic version strings per SCACI §2.1-2.3:
//   - Major version: Required for compatibility determination
//   - Minor version: Required for capability negotiation
//   - Patch version: Ignored for protocol compatibility (§2.3)
//
// Input validation:
//   - Trims whitespace from input and components
//   - Requires exactly 3 components separated by dots
//   - Each component must be 0-999 (prevents overflow)
//   - Leading zeros are accepted (e.g., "01.00.03" → 1.0.3)
//
// Parameters:
//   - v: Version string in "major.minor.patch" format
//
// Returns:
//   - major, minor, patch: Parsed version components
//   - error: Parse error if format is invalid
//
// Examples:
//
//	ParseSemanticVersion("1.0.0")   → 1, 0, 0, nil
//	ParseSemanticVersion(" 1.0.3 ") → 1, 0, 3, nil
//	ParseSemanticVersion("2.5")     → 0, 0, 0, error
//	ParseSemanticVersion("1000.0.0") → 0, 0, 0, error (overflow)
func ParseSemanticVersion(v string) (major, minor, patch int, err error) {
	v = strings.TrimSpace(v)
	parts := strings.Split(v, ".")

	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("version must be major.minor.patch, got: %s", v)
	}

	// Parse major version with overflow protection
	major, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || major < 0 || major > 999 {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	// Parse minor version with overflow protection
	minor, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || minor < 0 || minor > 999 {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	// Parse patch version with overflow protection
	patch, err = strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil || patch < 0 || patch > 999 {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return major, minor, patch, nil
}
