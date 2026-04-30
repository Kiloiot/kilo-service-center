package crypto

import (
	"bytes"
	"testing"
)

// TestEncryptDecryptKeyRaw tests raw GCM encryption/decryption roundtrip
func TestEncryptDecryptKeyRaw(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	// Test with 16-byte session key (BSSCI §5.6.2 requirement)
	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	// Encrypt
	encrypted, err := ke.EncryptKeyRaw(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKeyRaw failed: %v", err)
	}

	// Verify encrypted length is ~44 bytes (12-byte nonce + 16-byte ciphertext + 16-byte tag)
	if len(encrypted) < 40 || len(encrypted) > 50 {
		t.Errorf("Encrypted key length %d out of expected range 40-50 bytes", len(encrypted))
	}

	// Verify encrypted data is different from plaintext
	if bytes.Equal(encrypted, sessionKey) {
		t.Error("Encrypted key should differ from plaintext")
	}

	// Decrypt
	decrypted, err := ke.DecryptKeyRaw(encrypted)
	if err != nil {
		t.Fatalf("DecryptKeyRaw failed: %v", err)
	}

	// Verify roundtrip produces original key
	if !bytes.Equal(decrypted, sessionKey) {
		t.Errorf("Decrypted key %x does not match original %x", decrypted, sessionKey)
	}
}

// TestDecryptKeyRaw_Authentication tests that GCM authentication tag validation works
func TestDecryptKeyRaw_Authentication(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	encrypted, err := ke.EncryptKeyRaw(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKeyRaw failed: %v", err)
	}

	// Tamper with the ciphertext (flip a bit in the middle)
	if len(encrypted) > 20 {
		encrypted[20] ^= 0x01
	}

	// Decryption should fail due to authentication tag mismatch
	_, err = ke.DecryptKeyRaw(encrypted)
	if err == nil {
		t.Error("DecryptKeyRaw should fail with tampered ciphertext")
	}
}

// TestEncryptDecryptKey_Base64Legacy tests the base64-encoded legacy format
func TestEncryptDecryptKey_Base64Legacy(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	// Encrypt with legacy base64 method
	encrypted, err := ke.EncryptKey(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKey failed: %v", err)
	}

	// Verify base64 string length is ~60 chars
	if len(encrypted) < 55 || len(encrypted) > 65 {
		t.Errorf("Base64 encrypted key length %d out of expected range 55-65 chars", len(encrypted))
	}

	// Decrypt with legacy method
	decrypted, err := ke.DecryptKey(encrypted)
	if err != nil {
		t.Fatalf("DecryptKey failed: %v", err)
	}

	// Verify roundtrip
	if !bytes.Equal(decrypted, sessionKey) {
		t.Errorf("Decrypted key %x does not match original %x", decrypted, sessionKey)
	}
}

// TestEncryptDecryptKeyRaw_Roundtrip tests raw GCM encrypt/decrypt roundtrip
func TestEncryptDecryptKeyRaw_Roundtrip(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	// Encrypt with raw format
	encrypted, err := ke.EncryptKeyRaw(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKeyRaw failed: %v", err)
	}

	// Decrypt with raw format
	decrypted, err := ke.DecryptKeyRaw(encrypted)
	if err != nil {
		t.Fatalf("DecryptKeyRaw failed: %v", err)
	}

	// Verify correct decryption
	if !bytes.Equal(decrypted, sessionKey) {
		t.Errorf("Decrypted key %x does not match original %x", decrypted, sessionKey)
	}
}

// TestEncryptDecryptKey_Base64Roundtrip tests base64 encrypt/decrypt roundtrip
func TestEncryptDecryptKey_Base64Roundtrip(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	// Encrypt with base64 format
	encryptedBase64, err := ke.EncryptKey(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKey failed: %v", err)
	}

	// Decrypt with base64 format
	decrypted, err := ke.DecryptKey(encryptedBase64)
	if err != nil {
		t.Fatalf("DecryptKey failed: %v", err)
	}

	// Verify correct decryption
	if !bytes.Equal(decrypted, sessionKey) {
		t.Errorf("Decrypted key %x does not match original %x", decrypted, sessionKey)
	}
}

// TestEncryptKeyRaw_EmptyKey tests handling of empty keys
func TestEncryptKeyRaw_EmptyKey(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	encrypted, err := ke.EncryptKeyRaw(nil)
	if err != nil {
		t.Errorf("EncryptKeyRaw should handle nil key: %v", err)
	}
	if encrypted != nil {
		t.Error("EncryptKeyRaw should return nil for nil input")
	}

	encrypted, err = ke.EncryptKeyRaw([]byte{})
	if err != nil {
		t.Errorf("EncryptKeyRaw should handle empty key: %v", err)
	}
	if encrypted != nil {
		t.Error("EncryptKeyRaw should return nil for empty input")
	}
}

// TestDecryptKeyRaw_EmptyKey tests handling of empty encrypted keys
func TestDecryptKeyRaw_EmptyKey(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	decrypted, err := ke.DecryptKeyRaw(nil)
	if err != nil {
		t.Errorf("DecryptKeyRaw should handle nil key: %v", err)
	}
	if decrypted != nil {
		t.Error("DecryptKeyRaw should return nil for nil input")
	}

	decrypted, err = ke.DecryptKeyRaw([]byte{})
	if err != nil {
		t.Errorf("DecryptKeyRaw should handle empty key: %v", err)
	}
	if decrypted != nil {
		t.Error("DecryptKeyRaw should return nil for empty input")
	}
}

// TestDecryptKeyRaw_InvalidCiphertext tests error handling for malformed ciphertext
func TestDecryptKeyRaw_InvalidCiphertext(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	// Test with ciphertext too short (< nonce size)
	tooShort := []byte{0x01, 0x02, 0x03}
	_, err = ke.DecryptKeyRaw(tooShort)
	if err == nil {
		t.Error("DecryptKeyRaw should fail with ciphertext shorter than nonce")
	}
}

// TestDecryptKeyWithMigration_RawFormat verifies that raw GCM encrypted keys
// are recognized as the current format (wasLegacy=false)
func TestDecryptKeyWithMigration_RawFormat(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	encrypted, err := ke.EncryptKeyRaw(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKeyRaw failed: %v", err)
	}

	decrypted, wasLegacy, err := ke.DecryptKeyWithMigration(encrypted)
	if err != nil {
		t.Fatalf("DecryptKeyWithMigration failed: %v", err)
	}
	if wasLegacy {
		t.Error("Raw GCM format should not be flagged as legacy")
	}
	if !bytes.Equal(decrypted, sessionKey) {
		t.Errorf("Decrypted key %x does not match original %x", decrypted, sessionKey)
	}
}

// TestDecryptKeyWithMigration_LegacyFormat verifies that base64-encoded GCM keys
// are recognized as legacy (wasLegacy=true) and decrypted correctly
func TestDecryptKeyWithMigration_LegacyFormat(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	// Encrypt with legacy base64 method
	encryptedBase64, err := ke.EncryptKey(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKey failed: %v", err)
	}

	// Convert to bytes as would be stored in DB
	encryptedBytes := []byte(encryptedBase64)

	decrypted, wasLegacy, err := ke.DecryptKeyWithMigration(encryptedBytes)
	if err != nil {
		t.Fatalf("DecryptKeyWithMigration failed: %v", err)
	}
	if !wasLegacy {
		t.Error("Base64 GCM format should be flagged as legacy")
	}
	if !bytes.Equal(decrypted, sessionKey) {
		t.Errorf("Decrypted key %x does not match original %x", decrypted, sessionKey)
	}
}

// TestEncryptKeyRaw_Deterministic tests that encryption is non-deterministic (due to nonce)
func TestEncryptKeyRaw_Deterministic(t *testing.T) {
	ke, err := NewKeyEncryptor()
	if err != nil {
		t.Fatalf("Failed to create key encryptor: %v", err)
	}

	sessionKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	// Encrypt same key twice
	encrypted1, err := ke.EncryptKeyRaw(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKeyRaw failed: %v", err)
	}

	encrypted2, err := ke.EncryptKeyRaw(sessionKey)
	if err != nil {
		t.Fatalf("EncryptKeyRaw failed: %v", err)
	}

	// Results should differ (due to random nonce)
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("EncryptKeyRaw should produce different ciphertexts for same plaintext (nonce randomness)")
	}

	// But both should decrypt to same key
	decrypted1, err := ke.DecryptKeyRaw(encrypted1)
	if err != nil {
		t.Fatalf("DecryptKeyRaw failed: %v", err)
	}

	decrypted2, err := ke.DecryptKeyRaw(encrypted2)
	if err != nil {
		t.Fatalf("DecryptKeyRaw failed: %v", err)
	}

	if !bytes.Equal(decrypted1, sessionKey) || !bytes.Equal(decrypted2, sessionKey) {
		t.Error("Both encrypted versions should decrypt to original key")
	}
}
