// Package encoding provides shared encoding utilities for MIOTY protocol data
package encoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEncodeUserData_NilReturnsEmptyString verifies nil input returns empty string
// This behavior is relied upon by SCACI §3.9.1 operation logging
func TestEncodeUserData_NilReturnsEmptyString(t *testing.T) {
	result := EncodeUserData(nil)
	assert.Equal(t, "", result, "nil should encode to empty string")
}

// TestEncodeUserData_EmptySliceReturnsEmptyString verifies empty slice returns empty string
// This behavior ensures empty userData (allowed per SCACI §3.9.1) is recorded consistently
func TestEncodeUserData_EmptySliceReturnsEmptyString(t *testing.T) {
	result := EncodeUserData([]byte{})
	assert.Equal(t, "", result, "empty slice should encode to empty string")
}

// TestEncodeUserData_NonEmptyReturnsBase64 verifies populated data returns base64 encoding
func TestEncodeUserData_NonEmptyReturnsBase64(t *testing.T) {
	result := EncodeUserData([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	assert.Equal(t, "3q2+7w==", result, "should encode as base64")
}

// TestDecodeUserData_EmptyReturnsNil verifies empty string decodes to nil
func TestDecodeUserData_EmptyReturnsNil(t *testing.T) {
	result, err := DecodeUserData("")
	assert.NoError(t, err)
	assert.Nil(t, result, "empty string should decode to nil")
}

// TestDecodeUserData_ValidBase64 verifies valid base64 decodes correctly
func TestDecodeUserData_ValidBase64(t *testing.T) {
	result, err := DecodeUserData("3q2+7w==")
	assert.NoError(t, err)
	assert.Equal(t, []byte{0xDE, 0xAD, 0xBE, 0xEF}, result)
}

// TestDecodeUserData_InvalidBase64 verifies invalid base64 returns error
func TestDecodeUserData_InvalidBase64(t *testing.T) {
	_, err := DecodeUserData("not-valid-base64!!!")
	assert.Error(t, err)
}

// TestEncodeDecodeRoundTrip verifies encode/decode round trip preserves data
func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	encoded := EncodeUserData(original)
	decoded, err := DecodeUserData(encoded)
	assert.NoError(t, err)
	assert.Equal(t, original, decoded)
}
