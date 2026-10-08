package auth

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/passwordpolicy"
)

// PHC format: $pbkdf2-sha512$v=1$i=<rounds>$<salt>$<hash>
// This is the Modular Crypt Format (MCF) for password hashing.

const (
	phcPrefix = "$pbkdf2-sha512$"
)

const (
	phcVersionPrefix    = "v="
	phcIterationsPrefix = "i="
	phcHashFmt          = "$pbkdf2-sha512$v=1$i=%d$%s$%s"
	phcMinParts         = 3
)

// VerifyPassword verifies a password against a PHC-format hash.
// Returns nil if the password matches, or an error if it doesn't.
// PHC string components: version and iteration field prefixes, the rendered
// hash format, and the minimum field count (iterations, salt, hash).
func VerifyPassword(password, phcHash string) error {
	// Parse PHC format
	salt, storedHash, iterations, err := parsePHCHash(phcHash)
	if err != nil {
		return fmt.Errorf("%s: %w", errPrefixVerifyPassword, err)
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
	remainder := strings.TrimPrefix(phcHash, phcPrefix)
	parts := strings.Split(remainder, "$")

	if len(parts) < phcMinParts {
		return nil, nil, 0, ErrInvalidPHCFormat
	}

	// Parse version (optional, skip if present)
	idx := 0
	if strings.HasPrefix(parts[idx], phcVersionPrefix) {
		idx++
	}

	// Check remaining parts
	if len(parts)-idx < phcMinParts {
		return nil, nil, 0, ErrInvalidPHCFormat
	}

	// Parse iterations (i=<rounds>)
	if !strings.HasPrefix(parts[idx], phcIterationsPrefix) {
		return nil, nil, 0, ErrInvalidPHCFormat
	}
	iterations, err = strconv.Atoi(strings.TrimPrefix(parts[idx], phcIterationsPrefix))
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

	return fmt.Sprintf(phcHashFmt, iterations, saltB64, hashB64)
}

// ValidatePassword checks the password against passwordpolicy.Rules;
// ErrUserPasswordWeak when it falls short.
func ValidatePassword(password string) error {
	if !passwordpolicy.Rules.Accepts(password) {
		return ErrUserPasswordWeak
	}
	return nil
}
