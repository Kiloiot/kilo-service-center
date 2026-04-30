package scaci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrVersionMismatchOnResume_HasCorrectPOSIXCode validates that version mismatch
// errors map to POSIX_ENOTSUP (95) per SCACI §§2.1-2.3 wire protocol requirements.
func TestErrVersionMismatchOnResume_HasCorrectPOSIXCode(t *testing.T) {
	def := GetErrorDefinition(ErrVersionMismatchOnResume)
	require.NotEmpty(t, def.Token, "ErrVersionMismatchOnResume must have a definition")
	assert.Equal(t, POSIX_ENOTSUP, def.POSIXCode, "must map to POSIX_ENOTSUP=95 per §2.1-2.3")
}

// TestErrMajorVersionUnsupported_HasCorrectPOSIXCode validates major version errors.
func TestErrMajorVersionUnsupported_HasCorrectPOSIXCode(t *testing.T) {
	def := GetErrorDefinition(ErrMajorVersionUnsupported)
	require.NotEmpty(t, def.Token, "ErrMajorVersionUnsupported must have a definition")
	assert.Equal(t, POSIX_ENOTSUP, def.POSIXCode, "must map to POSIX_ENOTSUP=95 per §2.1")
}

// TestErrMinorVersionUnsupported_HasCorrectPOSIXCode validates minor version errors.
func TestErrMinorVersionUnsupported_HasCorrectPOSIXCode(t *testing.T) {
	def := GetErrorDefinition(ErrMinorVersionUnsupported)
	require.NotEmpty(t, def.Token, "ErrMinorVersionUnsupported must have a definition")
	assert.Equal(t, POSIX_ENOTSUP, def.POSIXCode, "must map to POSIX_ENOTSUP=95 per §2.2")
}

// TestGetErrorDefinition_UnknownToken returns default definition with echoed token.
func TestGetErrorDefinition_UnknownToken(t *testing.T) {
	unknownToken := "unknown.token.that.does.not.exist"
	def := GetErrorDefinition(unknownToken)
	// Unknown tokens get echoed back with default values
	assert.Equal(t, unknownToken, def.Token, "unknown token should be echoed back")
	assert.Equal(t, 0, def.POSIXCode, "unknown token should have zero POSIXCode")
	assert.Equal(t, "unknown", def.SpecSection, "unknown token should have 'unknown' spec section")
}

// TestVersionErrorDefinitions_HaveSpecSections validates compliance traceability.
func TestVersionErrorDefinitions_HaveSpecSections(t *testing.T) {
	tests := []struct {
		token       string
		specSection string
	}{
		{ErrMajorVersionUnsupported, "§2.1"},
		{ErrMinorVersionUnsupported, "§2.2"},
		{ErrVersionMismatchOnResume, "§2.1-2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			def := GetErrorDefinition(tt.token)
			require.NotEmpty(t, def.Token, "token %s must have a definition", tt.token)
			assert.Contains(t, def.SpecSection, tt.specSection,
				"token %s should reference spec section %s", tt.token, tt.specSection)
		})
	}
}
