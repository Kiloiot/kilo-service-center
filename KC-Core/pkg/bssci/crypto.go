package bssci

import (
	"crypto/aes"
	"crypto/subtle"
	"encoding/binary"

	"github.com/aead/cmac"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// ValidateAttachSignature validates the attach signature per MIOTY radio spec §3.7.1.3:
// the CMAC over the Fig. 3-15 initialization vector under the 16-byte pre-shared network key.
func ValidateAttachSignature(epEUI uint64, attachCnt uint32, signature []byte, key []byte) error {
	if len(signature) != 4 {
		return errSignatureMustBeExactly4Bytes
	}
	if len(key) != 16 {
		return errKeyMustBeExactly16
	}

	iv := make([]byte, attachIVSize)
	binary.BigEndian.PutUint64(iv, epEUI)
	iv[attachIVPadFFOffset] = attachIVPadFF
	iv[attachIVPad00Offset] = attachIVPad00
	binary.BigEndian.PutUint32(iv[attachIVCounterOffset:], attachCnt)
	iv[attachIVTrailerOffset] = attachIVPadFF
	iv[attachIVTrailerOffset+1] = attachIVPadFF

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	mac, err := cmac.New(block)
	if err != nil {
		return err
	}

	if _, err := mac.Write(iv); err != nil {
		return err
	}
	cmacResult := mac.Sum(nil)

	if subtle.ConstantTimeCompare(cmacResult[:4], signature) != 1 {
		return errCryptoSignatureMismatch
	}

	return nil
}

// DeriveSessionKey derives the session-specific network key per MIOTY radio spec §3.7.1.3:
// one AES-128 ECB block over the Fig. 3-16 seed under the 16-byte pre-shared network key.
func DeriveSessionKey(epEUI uint64, nonce []byte, signature []byte, key []byte) ([]byte, error) {
	if len(nonce) != 4 {
		return nil, errNonceMustBeExactly4Bytes
	}
	if len(signature) != 4 {
		return nil, errSignatureMustBeExactly4Bytes
	}
	if len(key) != 16 {
		return nil, errKeyMustBeExactly16
	}

	// Build AES seed: [EUI64 | nonce | signature].
	seed := make([]byte, 16)
	binary.BigEndian.PutUint64(seed[0:8], epEUI)
	copy(seed[8:12], nonce)
	copy(seed[12:16], signature)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	sessionKey := make([]byte, 16)
	block.Encrypt(sessionKey, seed)

	return sessionKey, nil
}

// CurrentNetworkSessionKey returns the network session key the endpoint
// operates on (radio spec §3.7.1.3): its over-the-air session key while the
// active session holds the key its recorded nonce and signature derive from
// its pre-shared key, otherwise the pre-shared key, which is the session key of
// a pre-attached endpoint.
func CurrentNetworkSessionKey(endpoint *models.EndPoint, activeSessionKey []byte) []byte {
	if len(activeSessionKey) == 0 {
		return endpoint.NwkSnKey
	}
	overTheAirKey, err := DeriveSessionKey(endpoint.EUI.ToUint64(), endpoint.Nonce, endpoint.Sign, endpoint.NwkSnKey)
	if err != nil || subtle.ConstantTimeCompare(overTheAirKey, activeSessionKey) != 1 {
		return endpoint.NwkSnKey
	}
	return overTheAirKey
}
