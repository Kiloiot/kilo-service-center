package keymaterial

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAdapterForTest(t *testing.T) *CipherKeyEncryptor {
	t.Helper()
	cipher, err := keycrypto.NewCipher([]byte("keymaterial-test-master-key-32by"))
	require.NoError(t, err)
	return NewCipherKeyEncryptor(cipher)
}

// TestCipherKeyEncryptor_RoundTrip verifies key material round-trips through
// the text-envelope form and the stored string is a valid envelope, never the
// cleartext.
func TestCipherKeyEncryptor_RoundTrip(t *testing.T) {
	adapter := newAdapterForTest(t)
	material := []byte("-----BEGIN EC PRIVATE KEY-----\ntest-material\n-----END EC PRIVATE KEY-----\n")

	stored, err := adapter.EncryptKey(material)
	require.NoError(t, err)
	assert.True(t, keycrypto.IsTextEnvelope(stored), "stored value must be a text envelope")
	assert.NotContains(t, stored, "test-material")

	roundTrip, err := adapter.DecryptKey(stored)
	require.NoError(t, err)
	assert.Equal(t, material, roundTrip)
}

// TestCipherKeyEncryptor_EmptyValuesPassThrough keeps the optional-column
// semantics: empty input stays empty in both directions.
func TestCipherKeyEncryptor_EmptyValuesPassThrough(t *testing.T) {
	adapter := newAdapterForTest(t)

	stored, err := adapter.EncryptKey(nil)
	require.NoError(t, err)
	assert.Empty(t, stored)

	decrypted, err := adapter.DecryptKey("")
	require.NoError(t, err)
	assert.Nil(t, decrypted)
}

// TestCipherKeyEncryptor_RejectsNonEnvelope verifies reads never interpret
// legacy or cleartext values: only valid text envelopes decrypt.
func TestCipherKeyEncryptor_RejectsNonEnvelope(t *testing.T) {
	adapter := newAdapterForTest(t)

	_, err := adapter.DecryptKey("bm90LWFuLWVudmVsb3Bl")
	require.Error(t, err, "a legacy base64 value must not decrypt")
}
