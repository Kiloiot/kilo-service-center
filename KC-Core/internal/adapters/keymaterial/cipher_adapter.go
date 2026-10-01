// Package keymaterial adapts the deployment's mandatory keycrypto cipher to
// the narrow key-encryption ports consumers declare.
package keymaterial

import (
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// CipherKeyEncryptor implements the certificate service's KeyEncryptor port
// over the same mandatory keycrypto cipher every other key surface uses, in
// its text-envelope form for TEXT columns.
type CipherKeyEncryptor struct {
	cipher keycrypto.Cipher
}

// NewCipherKeyEncryptor wraps the cipher. The cipher is mandatory at startup,
// so a nil cipher is a wiring defect surfaced by the consumer's constructor
// guard rather than here.
func NewCipherKeyEncryptor(cipher keycrypto.Cipher) *CipherKeyEncryptor {
	return &CipherKeyEncryptor{cipher: cipher}
}

// EncryptKey returns the text-envelope form of the key material; empty input
// stays empty so optional columns remain unset.
func (a *CipherKeyEncryptor) EncryptKey(key []byte) (string, error) {
	if len(key) == 0 {
		return "", nil
	}
	return a.cipher.EncryptString(key)
}

// DecryptKey reads the text-envelope form; empty input stays empty.
func (a *CipherKeyEncryptor) DecryptKey(encrypted string) ([]byte, error) {
	if encrypted == "" {
		return nil, nil
	}
	return a.cipher.DecryptString(encrypted)
}
