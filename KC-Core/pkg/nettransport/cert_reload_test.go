package nettransport

import (
	"crypto/tls"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRenewalDelay = time.Minute

// renew writes next's server key pair over current's, stamped later, as a
// renewal of the server certificate does.
func renew(t *testing.T, current, next *testPKI) []byte {
	t.Helper()
	certPEM, err := os.ReadFile(next.files.Cert)
	require.NoError(t, err)
	keyPEM, err := os.ReadFile(next.files.Key)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(current.files.Cert, certPEM, testFileMode))
	require.NoError(t, os.WriteFile(current.files.Key, keyPEM, testFileMode))
	later := time.Now().Add(testRenewalDelay)
	require.NoError(t, os.Chtimes(current.files.Cert, later, later))
	require.NoError(t, os.Chtimes(current.files.Key, later, later))
	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	return block.Bytes
}

// A renewed server certificate serves the next handshake without a restart.
func TestBuildServerTLSConfig_ServesARenewedCertificate(t *testing.T) {
	current := newTestPKI(t)
	cfg, err := BuildServerTLSConfig(current.files, TLSOptions{MinVersion: "1.2"})
	require.NoError(t, err)
	require.NotNil(t, cfg.GetCertificate, "the listener reads its key pair per handshake")

	renewedDER := renew(t, current, newTestPKI(t))

	served, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	assert.Equal(t, renewedDER, served.Certificate[0])
}

// A key pair that does not load (a renewal half written) keeps the last good one serving.
func TestBuildServerTLSConfig_KeepsServingWhileARenewalIsIncomplete(t *testing.T) {
	current := newTestPKI(t)
	cfg, err := BuildServerTLSConfig(current.files, TLSOptions{MinVersion: "1.2"})
	require.NoError(t, err)
	served, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)

	later := time.Now().Add(testRenewalDelay)
	require.NoError(t, os.WriteFile(current.files.Key, []byte("half written"), testFileMode))
	require.NoError(t, os.Chtimes(current.files.Key, later, later))

	again, err := cfg.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	assert.Equal(t, served.Certificate[0], again.Certificate[0])
}
