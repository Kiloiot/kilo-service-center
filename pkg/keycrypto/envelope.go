// Package keycrypto encrypts key material at rest under a single authenticated
// envelope format. It lives in the shared pkg module so both the persistence
// layer (KC-DB) and the composition roots (KC-Core) depend on it without
// KC-DB reaching up into KC-Core.
//
// Envelope layout (binary, used for BYTEA columns):
//
//	magic   4 bytes  "KCE1"   fixed, so an envelope is never confused with a
//	                          legacy raw-GCM blob whose first byte is a random
//	                          nonce.
//	version 1 byte   0x01
//	algo    1 byte   0x01 = AES-256-GCM
//	keyID   1 byte   rotation slot (0x00 = the single current key)
//	nonce   12 bytes crypto/rand
//	body    N bytes  AES-256-GCM Seal output (ciphertext + 16-byte tag)
//
// The 7-byte header (magic|version|algo|keyID) is authenticated as the GCM
// additional data, so a tampered header fails decryption. Text columns store
// textPrefix + base64(binary envelope).
package keycrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const (
	magicLen  = 4
	headerLen = 7 // magic(4) + version(1) + algo(1) + keyID(1)
	nonceLen  = 12
	tagLen    = 16
	keyLen    = 32 // AES-256

	versionV1  = 0x01
	algoAESGCM = 0x01

	// textPrefix marks a text-column envelope: textPrefix + base64(binary).
	textPrefix = "kcenc:v1:"
)

// currentKeyID is the single active rotation slot.
const currentKeyID byte = 0x00

var magic = [magicLen]byte{'K', 'C', 'E', '1'}

// Exported sentinels so callers can assert failure modes.
var (
	// ErrKeyLength reports a master key that did not decode to exactly 32 bytes.
	ErrKeyLength = errors.New("keycrypto: master key must be exactly 32 bytes")
	// errBadEnvelope reports ciphertext that is not a well-formed envelope.
	errBadEnvelope = errors.New("keycrypto: not a valid ciphertext envelope")
	// errUnsupportedVersion reports an envelope version or algorithm this build cannot read.
	errUnsupportedVersion = errors.New("keycrypto: unsupported envelope version or algorithm")
	// errDecrypt reports an authentication or decryption failure.
	errDecrypt = errors.New("keycrypto: decryption failed")
	// errEntropy reports a failure reading randomness for the nonce.
	errEntropy = errors.New("keycrypto: entropy source failed")
)

// Error format strings for wrapping construction failures.
const (
	errFmtCipherInit          = "keycrypto: %w"
	errFmtUnexpectedNonceSize = "keycrypto: unexpected GCM nonce size %d"
)

// Cipher encrypts and decrypts key material under the envelope format.
type Cipher interface {
	// Encrypt returns a binary envelope for a BYTEA column.
	Encrypt(plaintext []byte) ([]byte, error)
	// Decrypt reads a binary envelope. It rejects anything without the magic
	// prefix - runtime never accepts unversioned ciphertext.
	Decrypt(envelope []byte) ([]byte, error)
	// EncryptString returns textPrefix + base64(envelope) for a TEXT column.
	EncryptString(plaintext []byte) (string, error)
	// DecryptString reads the textPrefix + base64 form.
	DecryptString(s string) ([]byte, error)
}

// AESGCMCipher implements Cipher with AES-256-GCM.
type AESGCMCipher struct {
	gcm   cipher.AEAD
	keyID byte
	rand  io.Reader
}

// NewCipher builds an AES-256-GCM cipher from a 32-byte key. The key is copied
// so later caller mutation cannot alter the active key.
func NewCipher(key []byte) (*AESGCMCipher, error) {
	return newCipherWithRand(key, rand.Reader)
}

// newCipherWithRand is NewCipher with an explicit entropy source, so a
// nonce-generation failure is testable.
func newCipherWithRand(key []byte, r io.Reader) (*AESGCMCipher, error) {
	if len(key) != keyLen {
		return nil, ErrKeyLength
	}
	keyCopy := make([]byte, keyLen)
	copy(keyCopy, key)
	block, err := aes.NewCipher(keyCopy)
	if err != nil {
		return nil, fmt.Errorf(errFmtCipherInit, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf(errFmtCipherInit, err)
	}
	if gcm.NonceSize() != nonceLen {
		return nil, fmt.Errorf(errFmtUnexpectedNonceSize, gcm.NonceSize())
	}
	return &AESGCMCipher{gcm: gcm, keyID: currentKeyID, rand: r}, nil
}

func (c *AESGCMCipher) header() []byte {
	h := make([]byte, 0, headerLen)
	h = append(h, magic[:]...)
	h = append(h, versionV1, algoAESGCM, c.keyID)
	return h
}

// Encrypt returns a binary envelope.
func (c *AESGCMCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(c.rand, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", errEntropy, err)
	}
	header := c.header()
	body := c.gcm.Seal(nil, nonce, plaintext, header)
	out := make([]byte, 0, headerLen+nonceLen+len(body))
	out = append(out, header...)
	out = append(out, nonce...)
	out = append(out, body...)
	return out, nil
}

// Decrypt reads a binary envelope, rejecting anything not in the versioned
// format.
func (c *AESGCMCipher) Decrypt(envelope []byte) ([]byte, error) {
	if len(envelope) < headerLen+nonceLen+tagLen {
		return nil, errBadEnvelope
	}
	if [magicLen]byte(envelope[:magicLen]) != magic {
		return nil, errBadEnvelope
	}
	version := envelope[magicLen]
	algo := envelope[magicLen+1]
	if version != versionV1 || algo != algoAESGCM {
		return nil, errUnsupportedVersion
	}
	header := envelope[:headerLen]
	nonce := envelope[headerLen : headerLen+nonceLen]
	body := envelope[headerLen+nonceLen:]
	plaintext, err := c.gcm.Open(nil, nonce, body, header)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errDecrypt, err)
	}
	return plaintext, nil
}

// EncryptString returns the text-column form.
func (c *AESGCMCipher) EncryptString(plaintext []byte) (string, error) {
	env, err := c.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	return textPrefix + base64.StdEncoding.EncodeToString(env), nil
}

// DecryptString reads the text-column form.
func (c *AESGCMCipher) DecryptString(s string) ([]byte, error) {
	if len(s) < len(textPrefix) || s[:len(textPrefix)] != textPrefix {
		return nil, errBadEnvelope
	}
	env, err := base64.StdEncoding.DecodeString(s[len(textPrefix):])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errBadEnvelope, err)
	}
	return c.Decrypt(env)
}

// IsEnvelope reports whether b begins with the envelope magic - useful for
// migration code distinguishing new from legacy formats.
func IsEnvelope(b []byte) bool {
	return len(b) >= magicLen && [magicLen]byte(b[:magicLen]) == magic
}

// IsTextEnvelope reports whether s carries the text-column prefix.
func IsTextEnvelope(s string) bool {
	return len(s) >= len(textPrefix) && s[:len(textPrefix)] == textPrefix
}

// EnvelopeMagic returns a copy of the marker every binary envelope begins
// with, for detection where IsEnvelope cannot run, such as a SQL filter.
func EnvelopeMagic() []byte {
	m := magic
	return m[:]
}

// TextEnvelopePrefix returns the marker every text-column envelope begins
// with, for detection where IsTextEnvelope cannot run.
func TextEnvelopePrefix() string {
	return textPrefix
}
