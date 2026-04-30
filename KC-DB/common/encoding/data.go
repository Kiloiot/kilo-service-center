// Package encoding provides shared encoding utilities for MIOTY protocol data
package encoding

import (
	"encoding/base64"
	"fmt"
)

// EncodeUserData encodes user data bytes to base64 string for storage/logging
// Returns empty string for empty input
func EncodeUserData(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeUserData decodes base64 string back to bytes
// Returns error if decode fails; returns nil bytes for empty input
func DecodeUserData(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return data, nil
}
