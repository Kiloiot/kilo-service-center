//go:build integration

package probe

import (
	"crypto/aes"
	"encoding/binary"
	"testing"

	"github.com/aead/cmac"
	"github.com/stretchr/testify/require"
)

// specAttachSignature is the SIM-BS's own implementation of the endpoint's
// attach signature, written from RADIO §3.7.1.3 and not from the Service
// Center: CMAC over the 16-byte IV of Fig. 3-15 (EUI64 | 0xFF | 0x00 | attach
// counter, 4 bytes big-endian | 0xFFFF) with the pre-shared key, truncated to
// the 4 most significant bytes.
func specAttachSignature(t *testing.T, eui uint64, attachCnt uint32, key []byte) []byte {
	t.Helper()
	iv := make([]byte, attachIVLen)
	binary.BigEndian.PutUint64(iv[0:8], eui)
	iv[8] = attachIVMarkerHi
	iv[9] = attachIVMarkerLo
	binary.BigEndian.PutUint32(iv[10:14], attachCnt)
	binary.BigEndian.PutUint16(iv[14:16], attachIVTrailer)
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	mac, err := cmac.New(block)
	require.NoError(t, err)
	_, err = mac.Write(iv)
	require.NoError(t, err)
	return mac.Sum(nil)[:nonceLen]
}

// specSessionKey is the network session key of RADIO §3.7.1.3 Fig. 3-16: the
// seed EUI64 | nonce | signature encrypted as one AES-128-ECB block with the
// pre-shared key.
func specSessionKey(t *testing.T, eui uint64, nonce, sign, key []byte) []byte {
	t.Helper()
	seed := make([]byte, sessionKeyLen)
	binary.BigEndian.PutUint64(seed[0:8], eui)
	copy(seed[8:12], nonce)
	copy(seed[12:16], sign)
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	out := make([]byte, sessionKeyLen)
	block.Encrypt(out, seed)
	return out
}
