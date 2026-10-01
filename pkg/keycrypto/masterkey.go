package keycrypto

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// EnvMasterKey names the environment variable holding the 32-byte master key
// (64 hex chars or standard base64) used to encrypt key material at rest.
const EnvMasterKey = "KILOCENTER_MASTER_KEY"

// ParseMasterKey decodes a master key from its configured string form: either
// 64 hexadecimal characters or standard base64, each yielding exactly 32
// bytes. Anything else is ErrKeyLength.
func ParseMasterKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, ErrKeyLength
	}
	if len(s) == hex.EncodedLen(keyLen) {
		if raw, err := hex.DecodeString(s); err == nil && len(raw) == keyLen {
			return raw, nil
		}
	}
	if raw, err := base64.StdEncoding.DecodeString(s); err == nil && len(raw) == keyLen {
		return raw, nil
	}
	return nil, ErrKeyLength
}

// NewCipherFromMasterKey parses the configured key string and builds a Cipher.
func NewCipherFromMasterKey(s string) (*AESGCMCipher, error) {
	key, err := ParseMasterKey(s)
	if err != nil {
		return nil, err
	}
	return NewCipher(key)
}
