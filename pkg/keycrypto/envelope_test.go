package keycrypto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func testKey() []byte {
	k := make([]byte, keyLen)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}

func TestRoundTripBinary(t *testing.T) {
	c, err := NewCipher(testKey())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	plain := []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2, 3, 4, 5, 6}
	env, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !IsEnvelope(env) {
		t.Fatal("output is not recognized as an envelope")
	}
	if bytes.Contains(env, plain) {
		t.Fatal("plaintext key material appears in the envelope")
	}
	got, err := c.Decrypt(env)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round-trip mismatch: %x != %x", got, plain)
	}
}

func TestRoundTripString(t *testing.T) {
	c, _ := NewCipher(testKey())
	plain := []byte("tls-private-key-bytes")
	s, err := c.EncryptString(plain)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if !strings.HasPrefix(s, textPrefix) {
		t.Fatalf("missing text prefix: %q", s)
	}
	if strings.Contains(s, string(plain)) {
		t.Fatal("plaintext appears in the text envelope")
	}
	got, err := c.DecryptString(s)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatal("string round-trip mismatch")
	}
}

func TestMarkersMatchThePredicates(t *testing.T) {
	c, _ := NewCipher(testKey())
	env, err := c.Encrypt([]byte("key-material"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !bytes.HasPrefix(env, EnvelopeMagic()) {
		t.Fatal("a binary envelope does not begin with EnvelopeMagic")
	}
	if !IsEnvelope(EnvelopeMagic()) {
		t.Fatal("IsEnvelope rejects the bare EnvelopeMagic marker")
	}
	s, err := c.EncryptString([]byte("key-material"))
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if !strings.HasPrefix(s, TextEnvelopePrefix()) || !IsTextEnvelope(TextEnvelopePrefix()) {
		t.Fatal("TextEnvelopePrefix does not match the text envelope form")
	}

	marker := EnvelopeMagic()
	marker[0] ^= 0xFF
	if !IsEnvelope(env) || !bytes.HasPrefix(env, EnvelopeMagic()) {
		t.Fatal("mutating a returned marker changed envelope detection")
	}
}

func TestTamperedHeaderFailsAuth(t *testing.T) {
	c, _ := NewCipher(testKey())
	env, _ := c.Encrypt([]byte("secret"))
	env[magicLen] = 0x02 // flip the version byte (authenticated header)
	if _, err := c.Decrypt(env); !errors.Is(err, errUnsupportedVersion) && !errors.Is(err, errDecrypt) {
		t.Fatalf("tampered header must fail, got %v", err)
	}
}

func TestTamperedBodyFailsAuth(t *testing.T) {
	c, _ := NewCipher(testKey())
	env, _ := c.Encrypt([]byte("secret"))
	env[len(env)-1] ^= 0xFF // corrupt the tag
	if _, err := c.Decrypt(env); !errors.Is(err, errDecrypt) {
		t.Fatalf("tampered body must fail with errDecrypt, got %v", err)
	}
}

func TestRejectsUnversionedCiphertext(t *testing.T) {
	c, _ := NewCipher(testKey())
	// A legacy raw-GCM blob starts with a random nonce, not the magic.
	legacy := make([]byte, 44)
	for i := range legacy {
		legacy[i] = byte(i)
	}
	if _, err := c.Decrypt(legacy); !errors.Is(err, errBadEnvelope) {
		t.Fatalf("unversioned ciphertext must be rejected, got %v", err)
	}
}

func TestKeyLengthEnforced(t *testing.T) {
	if _, err := NewCipher(make([]byte, 16)); !errors.Is(err, ErrKeyLength) {
		t.Fatalf("short key must fail, got %v", err)
	}
}

func TestKeyIsCopied(t *testing.T) {
	key := testKey()
	c, _ := NewCipher(key)
	env, _ := c.Encrypt([]byte("x"))
	for i := range key {
		key[i] = 0 // mutate the caller's slice after construction
	}
	if _, err := c.Decrypt(env); err != nil {
		t.Fatalf("caller key mutation altered the active key: %v", err)
	}
}

func TestEntropyFailure(t *testing.T) {
	c, _ := newCipherWithRand(testKey(), failReader{})
	if _, err := c.Encrypt([]byte("x")); !errors.Is(err, errEntropy) {
		t.Fatalf("entropy failure must surface, got %v", err)
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("no entropy") }

func TestParseMasterKey(t *testing.T) {
	raw := testKey()
	hexKey := hex.EncodeToString(raw)
	if got, err := ParseMasterKey(hexKey); err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("hex parse: %v", err)
	}
	if _, err := ParseMasterKey("tooshort"); !errors.Is(err, ErrKeyLength) {
		t.Fatalf("short key must fail, got %v", err)
	}
	if _, err := ParseMasterKey(""); !errors.Is(err, ErrKeyLength) {
		t.Fatal("empty key must fail")
	}
}
