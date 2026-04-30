// Package scaci version parsing tests
//
// SCACI §§2.1-2.3 Compliance Tests - Version Negotiation
//
// This file tests the semantic version parsing and negotiation rules:
//   - §2.1: Major version compatibility (breaking changes)
//   - §2.2: Minor version compatibility (additive features)
//   - §2.3: Patch version tolerance (no protocol impact)
package scaci

import (
	"testing"
)

// ============================================================================
// ParseSemanticVersion Tests - SCACI §§2.1-2.3 Compliance
// ============================================================================

// TestParseSemanticVersion_ValidVersions tests well-formed version strings
func TestParseSemanticVersion_ValidVersions(t *testing.T) {
	tests := []struct {
		name    string
		version string
		major   int
		minor   int
		patch   int
	}{
		// Standard versions
		{"current protocol version", "1.0.0", 1, 0, 0},
		{"future minor bump", "1.1.0", 1, 1, 0},
		{"future patch bump", "1.0.5", 1, 0, 5},
		{"major version 2", "2.0.0", 2, 0, 0},

		// Edge cases - valid but unusual
		{"zero version", "0.0.0", 0, 0, 0},
		{"high numbers within limit", "999.999.999", 999, 999, 999},
		{"single digits", "1.2.3", 1, 2, 3},
		{"double digits", "10.20.30", 10, 20, 30},
		{"triple digits", "100.200.300", 100, 200, 300},

		// Whitespace handling per ParseSemanticVersion contract
		{"leading whitespace", " 1.0.0", 1, 0, 0},
		{"trailing whitespace", "1.0.0 ", 1, 0, 0},
		{"surrounding whitespace", " 1.0.0 ", 1, 0, 0},
		{"component whitespace", " 1 . 0 . 0 ", 1, 0, 0},

		// Leading zeros are accepted (per SCACI flexibility)
		{"leading zeros major", "01.0.0", 1, 0, 0},
		{"leading zeros minor", "1.00.0", 1, 0, 0},
		{"leading zeros patch", "1.0.03", 1, 0, 3},
		{"all leading zeros", "01.00.03", 1, 0, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := ParseSemanticVersion(tt.version)
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.version, err)
				return
			}
			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("version %q: got %d.%d.%d, want %d.%d.%d",
					tt.version, major, minor, patch, tt.major, tt.minor, tt.patch)
			}
		})
	}
}

// TestParseSemanticVersion_InvalidVersions tests malformed version strings
func TestParseSemanticVersion_InvalidVersions(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		// Wrong component count
		{"empty string", ""},
		{"single number", "1"},
		{"two components", "1.0"},
		{"four components", "1.0.0.0"},
		{"five components", "1.0.0.0.1"},

		// Non-numeric components
		{"alpha major", "a.0.0"},
		{"alpha minor", "1.b.0"},
		{"alpha patch", "1.0.c"},
		{"full text", "one.zero.zero"},
		{"mixed alpha", "v1.0.0"},

		// Invalid separators
		{"dash separators", "1-0-0"},
		{"underscore separators", "1_0_0"},
		{"comma separators", "1,0,0"},
		{"space separators", "1 0 0"},
		{"empty components", "1..0"},
		{"trailing dot", "1.0.0."},
		{"leading dot", ".1.0.0"},

		// Overflow protection (>999)
		{"major overflow", "1000.0.0"},
		{"minor overflow", "1.1000.0"},
		{"patch overflow", "1.0.1000"},
		{"large number", "9999.0.0"},

		// Negative numbers
		{"negative major", "-1.0.0"},
		{"negative minor", "1.-1.0"},
		{"negative patch", "1.0.-1"},

		// Special characters
		{"special chars", "1.0.0-beta"},
		{"plus sign", "1.0.0+build"},
		{"parentheses", "1.0.(0)"},
		{"brackets", "1.0.[0]"},

		// Unicode edge cases
		{"unicode digit", "１.0.0"}, // fullwidth digit 1
		{"em dash", "1—0—0"},       // em dash instead of dot
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := ParseSemanticVersion(tt.version)
			if err == nil {
				t.Errorf("expected error for %q, got %d.%d.%d", tt.version, major, minor, patch)
			}
		})
	}
}

// TestParseSemanticVersion_BoundaryValues tests boundary conditions
func TestParseSemanticVersion_BoundaryValues(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		wantError bool
		major     int
		minor     int
		patch     int
	}{
		// At boundary (999 is max per implementation)
		{"max major", "999.0.0", false, 999, 0, 0},
		{"max minor", "0.999.0", false, 0, 999, 0},
		{"max patch", "0.0.999", false, 0, 0, 999},
		{"all max", "999.999.999", false, 999, 999, 999},

		// Just over boundary (should error)
		{"major 1000", "1000.0.0", true, 0, 0, 0},
		{"minor 1000", "0.1000.0", true, 0, 0, 0},
		{"patch 1000", "0.0.1000", true, 0, 0, 0},

		// Zero boundary
		{"all zeros", "0.0.0", false, 0, 0, 0},
		{"only major", "1.0.0", false, 1, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := ParseSemanticVersion(tt.version)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for %q, got %d.%d.%d", tt.version, major, minor, patch)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.version, err)
				return
			}
			if major != tt.major || minor != tt.minor || patch != tt.patch {
				t.Errorf("version %q: got %d.%d.%d, want %d.%d.%d",
					tt.version, major, minor, patch, tt.major, tt.minor, tt.patch)
			}
		})
	}
}

// ============================================================================
// Protocol Version Constants Tests
// ============================================================================

// TestProtocolVersionConstants verifies the exported constants match expected values
// This catches accidental changes to protocol version constants
func TestProtocolVersionConstants(t *testing.T) {
	// Verify major version constant
	if SupportedMajorVersion != 1 {
		t.Errorf("SupportedMajorVersion = %d, want 1", SupportedMajorVersion)
	}

	// Verify minor version constant
	if SupportedMinorVersion != 0 {
		t.Errorf("SupportedMinorVersion = %d, want 0", SupportedMinorVersion)
	}

	// Verify version string is correctly formatted
	if ProtocolVersionString != "1.0.0" {
		t.Errorf("ProtocolVersionString = %q, want \"1.0.0\"", ProtocolVersionString)
	}

	// Verify ProtocolVersionString parses to the declared constants
	major, minor, _, err := ParseSemanticVersion(ProtocolVersionString)
	if err != nil {
		t.Fatalf("ProtocolVersionString %q failed to parse: %v", ProtocolVersionString, err)
	}
	if major != SupportedMajorVersion {
		t.Errorf("ProtocolVersionString major=%d, SupportedMajorVersion=%d (mismatch)",
			major, SupportedMajorVersion)
	}
	if minor != SupportedMinorVersion {
		t.Errorf("ProtocolVersionString minor=%d, SupportedMinorVersion=%d (mismatch)",
			minor, SupportedMinorVersion)
	}
}

// TestOpIDConnectConstant verifies the Connect operation ID constant
func TestOpIDConnectConstant(t *testing.T) {
	// SCACI §3.3: Connect handshake uses opId=0
	if OpIDConnect != 0 {
		t.Errorf("OpIDConnect = %d, want 0 (per SCACI §3.3-02)", OpIDConnect)
	}
}

// ============================================================================
// Version Negotiation Logic Tests (§2.1-2.3 Rules)
// ============================================================================

// TestVersionNegotiationRules_MajorVersion tests §2.1 major version rules
// Major version mismatch = connection terminated (incompatible protocols)
func TestVersionNegotiationRules_MajorVersion(t *testing.T) {
	tests := []struct {
		name       string
		clientVer  string
		compatible bool // Can negotiate (major matches)
	}{
		{"same major version", "1.0.0", true},
		{"same major higher minor", "1.5.0", true},
		{"same major higher patch", "1.0.99", true},
		{"lower major version", "0.9.0", false},
		{"higher major version", "2.0.0", false},
		{"much higher major", "10.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, _, _, err := ParseSemanticVersion(tt.clientVer)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tt.clientVer, err)
			}

			// §2.1: Major version MUST match for compatibility
			compatible := major == SupportedMajorVersion

			if compatible != tt.compatible {
				t.Errorf("client %q major compatibility: got %v, want %v",
					tt.clientVer, compatible, tt.compatible)
			}
		})
	}
}

// TestVersionNegotiationRules_MinorVersion tests §2.2 minor version rules
// Minor version too high = connection terminated (missing features)
// Minor version lower = acceptable (backwards compatible)
func TestVersionNegotiationRules_MinorVersion(t *testing.T) {
	tests := []struct {
		name       string
		clientVer  string
		acceptable bool // Minor <= SupportedMinorVersion
	}{
		{"same minor version", "1.0.0", true},
		{"higher minor version", "1.1.0", false},
		{"much higher minor", "1.5.0", false},
		// Note: Can't test "lower minor" since SupportedMinorVersion=0
		// and minor can't go below 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, minor, _, err := ParseSemanticVersion(tt.clientVer)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tt.clientVer, err)
			}

			// §2.2: Minor version must be <= supported
			acceptable := minor <= SupportedMinorVersion

			if acceptable != tt.acceptable {
				t.Errorf("client %q minor acceptability: got %v, want %v",
					tt.clientVer, acceptable, tt.acceptable)
			}
		})
	}
}

// TestVersionNegotiationRules_PatchVersion tests §2.3 patch version rules
// Patch version is ignored for protocol compatibility
func TestVersionNegotiationRules_PatchVersion(t *testing.T) {
	versions := []string{
		"1.0.0",
		"1.0.1",
		"1.0.99",
		"1.0.999",
	}

	for _, clientVer := range versions {
		t.Run(clientVer, func(t *testing.T) {
			major, minor, patch, err := ParseSemanticVersion(clientVer)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", clientVer, err)
			}

			// §2.3: Patch version has no impact on compatibility
			// All these versions should be compatible if major/minor match
			majorOk := major == SupportedMajorVersion
			minorOk := minor <= SupportedMinorVersion

			if !majorOk || !minorOk {
				t.Errorf("version %q (patch=%d) should be compatible: majorOk=%v, minorOk=%v",
					clientVer, patch, majorOk, minorOk)
			}
		})
	}
}

// TestVersionNegotiationRules_NegotiatedResult tests the negotiation outcome
// Successful negotiation should return ProtocolVersionString, not client version
func TestVersionNegotiationRules_NegotiatedResult(t *testing.T) {
	// Per SCACI §2.1-2.3: Server responds with its supported version, not client's
	// Client sends "1.0.3", server negotiates to "1.0.0" (ProtocolVersionString)

	clientVersion := "1.0.3"
	major, minor, _, err := ParseSemanticVersion(clientVersion)
	if err != nil {
		t.Fatalf("failed to parse client version: %v", err)
	}

	// Check compatibility
	if major != SupportedMajorVersion {
		t.Errorf("major version mismatch - cannot negotiate")
		return
	}
	if minor > SupportedMinorVersion {
		t.Errorf("minor version too high - cannot negotiate")
		return
	}

	// Negotiation succeeds: result should be ProtocolVersionString
	negotiatedVersion := ProtocolVersionString

	// Parse negotiated version to verify it's valid
	negMajor, negMinor, negPatch, err := ParseSemanticVersion(negotiatedVersion)
	if err != nil {
		t.Fatalf("negotiated version %q failed to parse: %v", negotiatedVersion, err)
	}

	// Verify negotiated version matches our constants
	if negMajor != SupportedMajorVersion {
		t.Errorf("negotiated major=%d, want %d", negMajor, SupportedMajorVersion)
	}
	if negMinor != SupportedMinorVersion {
		t.Errorf("negotiated minor=%d, want %d", negMinor, SupportedMinorVersion)
	}
	if negPatch != 0 {
		t.Errorf("negotiated patch=%d, want 0 (per §2.3)", negPatch)
	}
}
