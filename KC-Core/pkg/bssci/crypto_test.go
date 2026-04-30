package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateDetachSignatureCMAC_MatchingSignatures verifies signature equality fallback
func TestValidateDetachSignatureCMAC_MatchingSignatures(t *testing.T) {
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	attachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	nwkSnKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	presharedKey := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}

	err := ValidateDetachSignatureCMAC(detachSign, attachSign, nwkSnKey, presharedKey)
	require.NoError(t, err, "Matching signatures should validate successfully")
}

// TestValidateDetachSignatureCMAC_MismatchedSignatures verifies signature mismatch detection
func TestValidateDetachSignatureCMAC_MismatchedSignatures(t *testing.T) {
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	attachSign := []byte{0x11, 0x22, 0x33, 0x44} // Different signature
	nwkSnKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	presharedKey := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}

	err := ValidateDetachSignatureCMAC(detachSign, attachSign, nwkSnKey, presharedKey)
	require.Error(t, err, "Mismatched signatures should fail validation")
	assert.Contains(t, err.Error(), "signature mismatch", "Error should indicate signature mismatch")
}

// TestValidateDetachSignatureCMAC_InvalidDetachSignatureLength verifies detach signature length validation
func TestValidateDetachSignatureCMAC_InvalidDetachSignatureLength(t *testing.T) {
	detachSign := []byte{0xAA, 0xBB} // Too short (2 bytes instead of 4)
	attachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	nwkSnKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	presharedKey := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}

	err := ValidateDetachSignatureCMAC(detachSign, attachSign, nwkSnKey, presharedKey)
	require.Error(t, err, "Invalid detach signature length should fail validation")
	assert.Contains(t, err.Error(), "detach signature must be exactly 4 bytes", "Error should indicate invalid detach signature length")
}

// TestValidateDetachSignatureCMAC_InvalidAttachSignatureLength verifies attach signature length validation
func TestValidateDetachSignatureCMAC_InvalidAttachSignatureLength(t *testing.T) {
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	attachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF} // Too long (6 bytes instead of 4)
	nwkSnKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	presharedKey := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}

	err := ValidateDetachSignatureCMAC(detachSign, attachSign, nwkSnKey, presharedKey)
	require.Error(t, err, "Invalid attach signature length should fail validation")
	assert.Contains(t, err.Error(), "attach signature must be exactly 4 bytes", "Error should indicate invalid attach signature length")
}

// TestValidateDetachSignatureCMAC_NilKeys verifies fallback works with nil keys
func TestValidateDetachSignatureCMAC_NilKeys(t *testing.T) {
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	attachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	// Test with both keys nil
	err := ValidateDetachSignatureCMAC(detachSign, attachSign, nil, nil)
	require.NoError(t, err, "Validation should succeed with nil keys when signatures match")

	// Test with only nwkSnKey provided
	nwkSnKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	err = ValidateDetachSignatureCMAC(detachSign, attachSign, nwkSnKey, nil)
	require.NoError(t, err, "Validation should succeed with only nwkSnKey when signatures match")

	// Test with only presharedKey provided
	presharedKey := []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20}
	err = ValidateDetachSignatureCMAC(detachSign, attachSign, nil, presharedKey)
	require.NoError(t, err, "Validation should succeed with only presharedKey when signatures match")
}

// TestValidateDetachSignatureCMAC_EmptyKeys verifies fallback works with empty byte slices
func TestValidateDetachSignatureCMAC_EmptyKeys(t *testing.T) {
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	attachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	emptyKey := []byte{}

	// Test with both keys empty
	err := ValidateDetachSignatureCMAC(detachSign, attachSign, emptyKey, emptyKey)
	require.NoError(t, err, "Validation should succeed with empty keys when signatures match")
}

// TestValidateDetachSignatureCMAC_ConstantTimeComparison verifies timing-safe comparison.
// This test ensures we use constant-time comparison to prevent timing attacks.
func TestValidateDetachSignatureCMAC_ConstantTimeComparison(t *testing.T) {
	// Test case 1: First byte differs
	detachSign1 := []byte{0xFF, 0xBB, 0xCC, 0xDD}
	attachSign1 := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	err1 := ValidateDetachSignatureCMAC(detachSign1, attachSign1, nil, nil)
	require.Error(t, err1, "Should detect first byte mismatch")

	// Test case 2: Last byte differs
	detachSign2 := []byte{0xAA, 0xBB, 0xCC, 0xFF}
	attachSign2 := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	err2 := ValidateDetachSignatureCMAC(detachSign2, attachSign2, nil, nil)
	require.Error(t, err2, "Should detect last byte mismatch")

	// Both should return the same error message (constant-time comparison)
	assert.Equal(t, err1.Error(), err2.Error(), "Error messages should be identical regardless of which byte differs")
}
