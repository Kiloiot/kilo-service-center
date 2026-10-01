package bssci

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Known-answer vectors computed outside this code base with OpenSSL
// (`openssl mac -cipher AES-128-CBC CMAC`, `openssl enc -aes-128-ecb -nopad`)
// and Python `cryptography`, over the radio spec Fig. 3-15 IV
// 70B3D56770111505 FF 00 00123456 FFFF and the Fig. 3-16 seed.
const (
	katPresharedKeyHex = "2b7e151628aed2a6abf7158809cf4f3c"
	katEpEUI           = uint64(0x70B3D56770111505)
	katAttachCnt       = uint32(0x123456)
	katSignHex         = "d7173412"
	katNonceHex        = "a1b2c3d4"
	katSessionKeyHex   = "e77429f27a4d62080a8cef1447a42777"
	// katLegacySignHex is the CMAC over the 15-byte IV with a 3-byte counter.
	katLegacySignHex = "c31aeb23"
)

func katBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestValidateAttachSignature_KnownAnswer(t *testing.T) {
	key := katBytes(t, katPresharedKeyHex)

	require.NoError(t, ValidateAttachSignature(katEpEUI, katAttachCnt, katBytes(t, katSignHex), key),
		"a signature over the 16-byte Fig. 3-15 IV must verify")
	require.ErrorIs(t, ValidateAttachSignature(katEpEUI, katAttachCnt, katBytes(t, katLegacySignHex), key),
		errCryptoSignatureMismatch, "a signature over a 15-byte IV must not verify")
}

func TestDeriveSessionKey_KnownAnswer(t *testing.T) {
	got, err := DeriveSessionKey(katEpEUI, katBytes(t, katNonceHex), katBytes(t, katSignHex), katBytes(t, katPresharedKeyHex))
	require.NoError(t, err)
	require.Equal(t, katSessionKeyHex, hex.EncodeToString(got))
}

func TestCurrentNetworkSessionKey(t *testing.T) {
	presharedKey := katBytes(t, katPresharedKeyHex)
	sessionKey := katBytes(t, katSessionKeyHex)
	rotatedKey := katBytes(t, "000102030405060708090a0b0c0d0e0f")

	endpoint := func(key []byte, attachedOverTheAir bool) *models.EndPoint {
		ep := &models.EndPoint{NwkSnKey: key}
		binary.BigEndian.PutUint64(ep.EUI[:], katEpEUI)
		if attachedOverTheAir {
			ep.Nonce = katBytes(t, katNonceHex)
			ep.Sign = katBytes(t, katSignHex)
		}
		return ep
	}

	tests := []struct {
		name          string
		endpoint      *models.EndPoint
		activeSession []byte
		want          []byte
	}{
		{"never attached", endpoint(presharedKey, false), nil, presharedKey},
		{"pre-attached session", endpoint(presharedKey, false), presharedKey, presharedKey},
		{"over-the-air session", endpoint(presharedKey, true), sessionKey, sessionKey},
		{"pre-attached after an earlier over-the-air attach", endpoint(presharedKey, true), presharedKey, presharedKey},
		{"pre-shared key re-provisioned after the session began", endpoint(rotatedKey, true), sessionKey, rotatedKey},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, CurrentNetworkSessionKey(tc.endpoint, tc.activeSession))
		})
	}
}
