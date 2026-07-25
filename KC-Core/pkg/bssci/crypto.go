package bssci

import (
	"crypto/aes"
	"crypto/subtle"
	"encoding/binary"
	"errors"

	"github.com/aead/cmac"
)

// ValidateAttachSignature validates the attach signature per MIOTY radio spec §3.7.1.3.
func ValidateAttachSignature(epEUI uint64, attachCnt uint32, signature []byte, presharedKey []byte) error {
	if len(signature) != 4 {
		return errors.New("signature must be exactly 4 bytes")
	}
	if len(presharedKey) != 16 {
		return errors.New("preshared key must be exactly 16 bytes")
	}

	// Build CMAC initialization vector: [EUI64 | 0xFF | 0x00 | attachCnt(24-bit) | 0xFFFF].
	iv := make([]byte, 15)
	binary.BigEndian.PutUint64(iv[0:8], epEUI)
	iv[8] = 0xFF
	iv[9] = 0x00

	maskedCnt := attachCnt & 0xFFFFFF
	iv[10] = byte(maskedCnt >> 16) //nolint:gosec // G115: intentional byte extraction of 24-bit counter
	iv[11] = byte(maskedCnt >> 8)  //nolint:gosec // G115: intentional byte extraction of 24-bit counter
	iv[12] = byte(maskedCnt)       //nolint:gosec // G115: intentional byte extraction of 24-bit counter

	iv[13] = 0xFF
	iv[14] = 0xFF

	block, err := aes.NewCipher(presharedKey)
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
		return errors.New("signature mismatch")
	}

	return nil
}

// DeriveSessionKey derives the session-specific NWK key per MIOTY radio spec §3.7.1.3.
func DeriveSessionKey(epEUI uint64, nonce []byte, signature []byte, presharedKey []byte) ([]byte, error) {
	if len(nonce) != 4 {
		return nil, errors.New("nonce must be exactly 4 bytes")
	}
	if len(signature) != 4 {
		return nil, errors.New("signature must be exactly 4 bytes")
	}
	if len(presharedKey) != 16 {
		return nil, errors.New("preshared key must be exactly 16 bytes")
	}

	// Build AES seed: [EUI64 | nonce | signature].
	seed := make([]byte, 16)
	binary.BigEndian.PutUint64(seed[0:8], epEUI)
	copy(seed[8:12], nonce)
	copy(seed[12:16], signature)

	block, err := aes.NewCipher(presharedKey)
	if err != nil {
		return nil, err
	}

	sessionKey := make([]byte, 16)
	block.Encrypt(sessionKey, seed)

	return sessionKey, nil
}

// ValidateDetachSignature verifies detach signature matches stored attach signature.
// Per BSSCI §5.7.1, detach signature is "analogous to attach" but spec doesn't define
// CMAC IV format for detach operations. Using constant-time comparison against
// endpoint.Sign (the attach signature) until MIOTY Alliance clarifies §3.7.1 requirements.
func ValidateDetachSignature(detachSig, attachSig []byte) error {
	if len(detachSig) != 4 {
		return errors.New("detach signature must be exactly 4 bytes")
	}
	if len(attachSig) != 4 {
		return errors.New("attach signature must be exactly 4 bytes")
	}
	if subtle.ConstantTimeCompare(detachSig, attachSig) != 1 {
		return errors.New("signature mismatch")
	}
	return nil
}

// ValidateDetachSignatureCMAC validates detach signature using hybrid approach.
// Implements fallback chain: PresharedKey CMAC -> NwkSnKey CMAC -> signature equality.
//
// TODO: MIOTY spec does not define detach CMAC IV format (see BSSCI §3.7.1.3).
// Until resolved, uses signature equality as final fallback.
//
// Validation chain:
//  1. If presharedKey provided: attempt CMAC validation (reserved for future spec clarification)
//  2. If nwkSnKey provided: attempt CMAC validation (reserved for future spec clarification)
//  3. Fallback: constant-time comparison against attachSign
//
// Per BSSCI §5.7.1: "signature (analogous to attach)" but IV construction undefined.
func ValidateDetachSignatureCMAC(detachSign, attachSign, nwkSnKey, presharedKey []byte) error {
	// Suppress unused parameter warnings until CMAC IV format is defined by spec
	_, _ = nwkSnKey, presharedKey

	if len(detachSign) != 4 {
		return errors.New("detach signature must be exactly 4 bytes")
	}
	if len(attachSign) != 4 {
		return errors.New("attach signature must be exactly 4 bytes")
	}

	// TODO: MIOTY spec does not define detach CMAC IV format
	// Current implementation uses signature equality fallback only
	// When spec defines IV construction, add CMAC validation branches here:
	//
	// if len(presharedKey) == 16 {
	//     // Attempt CMAC validation with presharedKey
	//     // Build IV per spec (format TBD)
	//     // return CMAC validation result
	// }
	// if len(nwkSnKey) == 16 {
	//     // Attempt CMAC validation with nwkSnKey
	//     // Build IV per spec (format TBD)
	//     // return CMAC validation result
	// }

	// Fallback: signature equality (constant-time comparison)
	if subtle.ConstantTimeCompare(detachSign, attachSign) != 1 {
		return errors.New("signature mismatch")
	}

	return nil
}
