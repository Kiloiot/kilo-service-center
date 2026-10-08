package postgres

import (
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// ErrKeyMaterialNotEnvelope reports a stored key value that is not a valid
// authenticated envelope. Runtime reads never fall back to interpreting such
// bytes as cleartext: pre-envelope rows are converted once by the offline
// rekey command (KC-DB/cmd/rekey), which owns all format detection.
var ErrKeyMaterialNotEnvelope = errors.New("stored key material is not an authenticated envelope; run the rekey command")

// encryptKeyMaterial returns a binary envelope for non-empty key material, or
// nil for empty input so the column stays SQL NULL. Runtime writes always
// receive domain cleartext and always encrypt: a value that merely looks like
// an envelope is key material like any other and gets wrapped, so a caller
// cannot smuggle bytes past encryption.
func encryptKeyMaterial(cipher keycrypto.Cipher, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}
	envelope, err := cipher.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapEncryptKeyMaterial, err)
	}
	return envelope, nil
}

// decryptKeyMaterial decrypts a stored envelope. A non-empty value without a
// valid envelope is an error: the rekey command converts historical rows, so
// at runtime anything else indicates corruption or a missed migration.
func decryptKeyMaterial(cipher keycrypto.Cipher, stored []byte) ([]byte, error) {
	if len(stored) == 0 {
		return stored, nil
	}
	if !keycrypto.IsEnvelope(stored) {
		return nil, ErrKeyMaterialNotEnvelope
	}
	plaintext, err := cipher.Decrypt(stored)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapDecryptKeyMaterial, err)
	}
	return plaintext, nil
}

// decryptEndpointKeys decrypts the network and application session keys of an
// endpoint in place. It is called after every scan path that selects the
// nwk_key / app_key columns so callers always observe cleartext key material.
func decryptEndpointKeys(cipher keycrypto.Cipher, endpoint *models.EndPoint) error {
	if endpoint == nil {
		return nil
	}
	nwk, err := decryptKeyMaterial(cipher, endpoint.NwkSnKey)
	if err != nil {
		return err
	}
	endpoint.NwkSnKey = nwk

	app, err := decryptKeyMaterial(cipher, endpoint.AppKey)
	if err != nil {
		return err
	}
	endpoint.AppKey = app
	return nil
}

// decryptSessionKey decrypts the stored endpoint-session key in place.
func decryptSessionKey(cipher keycrypto.Cipher, session *models.EndPointSession) error {
	if session == nil {
		return nil
	}
	plaintext, err := decryptKeyMaterial(cipher, session.SessionKey)
	if err != nil {
		return err
	}
	session.SessionKey = plaintext
	return nil
}
