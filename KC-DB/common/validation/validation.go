// Package validation provides centralized validation functions for KiloCenter
// All input validation should use these functions for consistency
package validation

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/kilocenter/KC-DB/common/config"
	"github.com/kilocenter/KC-DB/common/errors"
)

// Regular expressions for validation
var (
	// EUI validation - 16 hex characters (8 bytes)
	euiRegex = regexp.MustCompile(`^[0-9a-fA-F]{16}$`)
)

// ValidateEUI validates an EUI64 string
func ValidateEUI(eui string) error {
	if eui == "" {
		return errors.Wrap(errors.ErrMissingField, "EUI is required")
	}

	// Remove any hyphens or colons
	cleaned := strings.ReplaceAll(strings.ReplaceAll(eui, "-", ""), ":", "")

	// Check if it's 16 hex characters
	if !euiRegex.MatchString(cleaned) {
		return errors.Wrapf(errors.ErrInvalidEUI, "EUI must be 16 hex characters, got %s", eui)
	}

	return nil
}

// ParseEUI parses an EUI string to uint64
func ParseEUI(eui string) (uint64, error) {
	if err := ValidateEUI(eui); err != nil {
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
		return 0, errors.Wrapf(errors.ErrInvalidEUI, "EUI must be %d bytes", config.EUISize)
	}

	// Convert bytes to uint64 (big endian)
	var result uint64
	for i := 0; i < 8; i++ {
		result = (result << 8) | uint64(bytes[i])
	}

	return result, nil
}

// FormatEUI formats a uint64 EUI to hex string
func FormatEUI(eui uint64) string {
	return fmt.Sprintf("%016x", eui)
}
