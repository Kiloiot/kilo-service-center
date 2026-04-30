package auth

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kilocenter/KC-Core/pkg/config"
	"golang.org/x/crypto/pbkdf2"
)

// PHC format: $pbkdf2-sha512$v=1$i=<rounds>$<salt>$<hash>
// This is the Modular Crypt Format (MCF) for password hashing.

const (
	phcPrefix = "$pbkdf2-sha512$"
)

// VerifyPassword verifies a password against a PHC-format hash.
// Returns nil if the password matches, or an error if it doesn't.
func VerifyPassword(password, phcHash string) error {
	// Parse PHC format
	salt, storedHash, iterations, err := parsePHCHash(phcHash)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}

	// Compute PBKDF2-SHA512 hash
	computedHash := pbkdf2.Key([]byte(password), salt, iterations, len(storedHash), sha512.New)

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(computedHash, storedHash) != 1 {
		return ErrInvalidCredentials
	}

	return nil
}

// parsePHCHash parses a PHC format hash string.
// Format: $pbkdf2-sha512$v=1$i=<rounds>$<salt>$<hash>
func parsePHCHash(phcHash string) (salt, hash []byte, iterations int, err error) {
	if !strings.HasPrefix(phcHash, phcPrefix) {
		return nil, nil, 0, ErrInvalidPHCFormat
	}

	// Remove prefix and split by $
	remainder := strings.TrimPrefix(phcHash, "$pbkdf2-sha512$")
	parts := strings.Split(remainder, "$")

	if len(parts) < 3 {
		return nil, nil, 0, ErrInvalidPHCFormat
	}

	// Parse version (optional, skip if present)
	idx := 0
	if strings.HasPrefix(parts[idx], "v=") {
		idx++
	}

	// Check remaining parts
	if len(parts)-idx < 3 {
		return nil, nil, 0, ErrInvalidPHCFormat
	}

	// Parse iterations (i=<rounds>)
	if !strings.HasPrefix(parts[idx], "i=") {
		return nil, nil, 0, ErrInvalidPHCFormat
	}
	iterations, err = strconv.Atoi(strings.TrimPrefix(parts[idx], "i="))
	if err != nil {
		return nil, nil, 0, ErrInvalidPHCFormat
	}
	idx++

	// Parse salt (base64 encoded)
	salt, err = base64.RawStdEncoding.DecodeString(parts[idx])
	if err != nil {
		// Try with padding
		salt, err = base64.StdEncoding.DecodeString(parts[idx])
		if err != nil {
			return nil, nil, 0, ErrInvalidPHCFormat
		}
	}
	idx++

	// Parse hash (base64 encoded)
	hash, err = base64.RawStdEncoding.DecodeString(parts[idx])
	if err != nil {
		// Try with padding
		hash, err = base64.StdEncoding.DecodeString(parts[idx])
		if err != nil {
			return nil, nil, 0, ErrInvalidPHCFormat
		}
	}

	return salt, hash, iterations, nil
}

// HashPassword creates a PHC-format hash for a password.
func HashPassword(password string, salt []byte, iterations int) string {
	hash := pbkdf2.Key([]byte(password), salt, iterations, config.AuthPBKDF2KeyLength, sha512.New)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$pbkdf2-sha512$v=1$i=%d$%s$%s", iterations, saltB64, hashB64)
}

// ValidatePassword checks if password meets length and complexity requirements.
// Requires at least one letter and one digit. Uses rune counting for Unicode support.
// Returns nil if valid, ErrUserPasswordWeak if invalid.
func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < config.AuthPasswordMinLength || length > config.AuthPasswordMaxLength {
		return ErrUserPasswordWeak
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
		if hasLetter && hasDigit {
			break
		}
	}
	if !hasLetter || !hasDigit {
		return ErrUserPasswordWeak
	}

	return nil
}
