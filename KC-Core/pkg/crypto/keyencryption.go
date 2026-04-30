// Package crypto provides encryption utilities for sensitive data
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// KeyEncryptor provides methods for encrypting and decrypting sensitive keys
type KeyEncryptor struct {
	key []byte
}

// NewKeyEncryptor creates a new key encryptor using a master key from environment
// or a default key (for development only)
func NewKeyEncryptor() (*KeyEncryptor, error) {
	// Get encryption key from environment or use a development default
	masterKey := os.Getenv("KC_ENCRYPTION_KEY")
	if masterKey == "" {
		// Development-only default key - MUST be overridden in production
		// This provides basic protection even in development environments
		masterKey = "development-key-do-not-use-in-production-2024"
	}

	// Derive a 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(masterKey))

	return &KeyEncryptor{
		key: hash[:],
	}, nil
}

// EncryptKey encrypts a network session key or other sensitive key material
// Returns the encrypted data as base64-encoded string for storage
func (ke *KeyEncryptor) EncryptKey(plaintext []byte) (string, error) {
	if len(plaintext) == 0 {
		return "", nil // Don't encrypt empty keys
	}

	// Create AES cipher
	block, err := aes.NewCipher(ke.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Return base64-encoded for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptKey decrypts a previously encrypted key
// Input should be base64-encoded ciphertext as returned by EncryptKey
func (ke *KeyEncryptor) DecryptKey(encryptedBase64 string) ([]byte, error) {
	if encryptedBase64 == "" {
		return nil, nil // Handle empty keys
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(ke.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptIfNotEmpty encrypts a key only if it's not empty
// This is a convenience method for optional encryption
func (ke *KeyEncryptor) EncryptIfNotEmpty(key []byte) (string, error) {
	if len(key) == 0 {
		return "", nil
	}
	return ke.EncryptKey(key)
}

// DecryptIfNotEmpty decrypts a key only if it's not empty
// This is a convenience method for optional decryption
func (ke *KeyEncryptor) DecryptIfNotEmpty(encrypted string) ([]byte, error) {
	if encrypted == "" {
		return nil, nil
	}
	return ke.DecryptKey(encrypted)
}

// EncryptKeyRaw encrypts a session key and returns raw GCM blob (nonce + ciphertext + tag)
// Returns ~44 bytes for 16-byte plaintext: 12-byte nonce + 16-byte ciphertext + 16-byte tag
// This maintains authenticated encryption (AEAD) while removing base64 overhead.
// Use this for BSSCI session keys to satisfy database constraint while keeping authentication.
func (ke *KeyEncryptor) EncryptKeyRaw(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil // Don't encrypt empty keys
	}

	// Create AES cipher
	block, err := aes.NewCipher(ke.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	// GCM.Seal prepends nonce to ciphertext: [nonce | encrypted_data | auth_tag]
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Return raw bytes (no base64 encoding)
	return ciphertext, nil
}

// DecryptKeyWithMigration attempts raw GCM decryption first (new format), falls back to base64 GCM (legacy).
// Returns: (plaintext, wasLegacy, error)
func (ke *KeyEncryptor) DecryptKeyWithMigration(encryptedKey []byte) ([]byte, bool, error) {
	if len(encryptedKey) == 0 {
		return nil, false, nil
	}
	// Try raw GCM first (new format: 44 bytes typical)
	if len(encryptedKey) >= 40 && len(encryptedKey) <= 50 {
		plaintext, err := ke.DecryptKeyRaw(encryptedKey)
		if err == nil {
			return plaintext, false, nil
		}
	}
	// Fall back to legacy base64 format
	plaintext, err := ke.DecryptKey(string(encryptedKey))
	if err != nil {
		return nil, false, fmt.Errorf("failed to decrypt with both raw (%d bytes) and base64: %w", len(encryptedKey), err)
	}
	return plaintext, true, nil
}

// DecryptKeyRaw decrypts a raw GCM blob (nonce + ciphertext + tag)
// Expects ~44 bytes input for 16-byte plaintext, returns decrypted plaintext
// Validates GCM authentication tag - will error if tampered
func (ke *KeyEncryptor) DecryptKeyRaw(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil // Handle empty keys
	}

	// Create AES cipher
	block, err := aes.NewCipher(ke.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short: got %d bytes, need at least %d", len(ciphertext), nonceSize)
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and validate authentication tag
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt or authenticate: %w", err)
	}

	return plaintext, nil
}
