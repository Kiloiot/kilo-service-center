package testsupport

import "github.com/Kiloiot/kilo-service-center/pkg/keycrypto"

// testMasterKey is a fixed 32-byte AES-256 key used only by tests so that
// key-material round-trips are deterministic and reproducible.
var testMasterKey = []byte("kilocenter-test-master-key-32byt")

// TestCipher returns a deterministic keycrypto.Cipher for tests that open the
// concrete store. It panics on failure because a broken test key is a
// programming error, never a runtime condition.
func TestCipher() keycrypto.Cipher {
	cipher, err := keycrypto.NewCipher(testMasterKey)
	if err != nil {
		panic(err)
	}
	return cipher
}
