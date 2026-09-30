// Package validation provides centralized validation functions for KiloCenter
// All input validation should use these functions for consistency
package validation

import (
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
)

// Regular expressions for validation
var (
	// EUI validation - 16 hex characters (8 bytes)
	euiRegex = regexp.MustCompile(`^[0-9a-fA-F]{16}$`)
)

const (
	msgEUIRequired      = "EUI is required"
	msgFmtEUIHexLength  = "EUI must be 16 hex characters, got %s"
	msgFmtEUIByteLength = "EUI must be %d bytes"
)

// validateEUI validates an EUI64 string
// EUI validation messages wrapped onto the sentinel errors.
func validateEUI(eui string) error {
	if eui == "" {
		return errors.Wrap(errors.ErrMissingField, msgEUIRequired)
	}

	// Remove any hyphens or colons
	cleaned := strings.ReplaceAll(strings.ReplaceAll(eui, "-", ""), ":", "")

	// Check if it's 16 hex characters
	if !euiRegex.MatchString(cleaned) {
		return errors.Wrapf(errors.ErrInvalidEUI, msgFmtEUIHexLength, eui)
	}

	return nil
}

// ParseEUI parses an EUI string to uint64
func ParseEUI(eui string) (uint64, error) {
	if err := validateEUI(eui); err != nil {
		return 0, err
	}

	// Remove any hyphens or colons
	cleaned := strings.ReplaceAll(strings.ReplaceAll(eui, "-", ""), ":", "")

	// Decode hex string to bytes
	bytes, err := hex.DecodeString(cleaned)
	if err != nil {
		return 0, errors.Wrap(errors.ErrInvalidEUI, err.Error())
	}

	if len(bytes) != config.EUISize {
		return 0, errors.Wrapf(errors.ErrInvalidEUI, msgFmtEUIByteLength, config.EUISize)
	}

	// Convert bytes to uint64 (big endian)
	var result uint64
	for _, b := range bytes {
		result = (result << 8) | uint64(b)
	}

	return result, nil
}

// ParseEUIBytes parses an EUI string into its 8 big-endian bytes.
func ParseEUIBytes(eui string) ([]byte, error) {
	value, err := ParseEUI(eui)
	if err != nil {
		return nil, err
	}
	bytes := make([]byte, config.EUISize)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes, nil
}
