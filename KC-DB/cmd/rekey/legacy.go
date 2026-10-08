package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// legacyCipher decrypts the retired KC-Core/pkg/crypto format: AES-256-GCM
// under SHA-256 of a passphrase, stored as nonce||ciphertext, either raw or
// base64-encoded. It never encrypts.
type legacyCipher struct {
	gcm cipher.AEAD
}

var errLegacyCiphertextTooShort = errors.New("legacy ciphertext shorter than nonce")

func newLegacyCipher(passphrase string) *legacyCipher {
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		// aes.NewCipher cannot fail for a 32-byte key.
		panic(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	return &legacyCipher{gcm: gcm}
}

// decryptRaw reads nonce||ciphertext.
func (l *legacyCipher) decryptRaw(data []byte) ([]byte, error) {
	if len(data) < l.gcm.NonceSize() {
		return nil, errLegacyCiphertextTooShort
	}
	nonce, ciphertext := data[:l.gcm.NonceSize()], data[l.gcm.NonceSize():]
	return l.gcm.Open(nil, nonce, ciphertext, nil)
}

// decryptBase64 reads base64(nonce||ciphertext), the form EncryptKey stored.
func (l *legacyCipher) decryptBase64(s string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return l.decryptRaw(data)
}
