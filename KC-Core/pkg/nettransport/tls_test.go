package nettransport

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testServerName   = "kilocenter.test"
	testClientCN     = "70B3D59CD0000001"
	testCertLifetime = time.Hour
	testSerialCA     = 1
	testSerialServer = 2
	testSerialClient = 3
	testFileMode     = 0o600
)

// Paths that never exist, used to exercise TLS file-loading failures.
const (
	missingCertPath = "/nonexistent/cert.pem"
	missingKeyPath  = "/nonexistent/key.pem"
	missingCAPath   = "/nonexistent/ca.pem"
)

// testPKI is a throwaway CA with one server and one client leaf, written to
// disk in the layout the listener loads.
type testPKI struct {
	files      TLSFiles
	caPool     *x509.CertPool
	clientCert tls.Certificate
}

func newTestPKI(t *testing.T) *testPKI {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(testSerialCA),
		Subject:               pkix.Name{CommonName: "nettransport-test-ca"},
		NotBefore:             time.Now().Add(-testCertLifetime),
		NotAfter:              time.Now().Add(testCertLifetime),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	issue := func(serial int64, cn string, usage x509.ExtKeyUsage, dnsName string) (tls.Certificate, []byte, []byte) {
		key, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, keyErr)
		template := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: cn},
			NotBefore:    time.Now().Add(-testCertLifetime),
			NotAfter:     time.Now().Add(testCertLifetime),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		}
		if dnsName != "" {
			template.DNSNames = []string{dnsName}
		}
		der, certErr := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
		require.NoError(t, certErr)
		keyDER, marshalErr := x509.MarshalECPrivateKey(key)
		require.NoError(t, marshalErr)
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
		pair, pairErr := tls.X509KeyPair(certPEM, keyPEM)
		require.NoError(t, pairErr)
		return pair, certPEM, keyPEM
	}

	_, serverPEM, serverKeyPEM := issue(testSerialServer, testServerName, x509.ExtKeyUsageServerAuth, testServerName)
	clientPair, _, _ := issue(testSerialClient, testClientCN, x509.ExtKeyUsageClientAuth, "")

	files := TLSFiles{
		Cert: filepath.Join(dir, "server.crt"),
		Key:  filepath.Join(dir, "server.key"),
		CA:   filepath.Join(dir, "ca.crt"),
	}
	require.NoError(t, os.WriteFile(files.CA, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), testFileMode))
	require.NoError(t, os.WriteFile(files.Cert, serverPEM, testFileMode))
	require.NoError(t, os.WriteFile(files.Key, serverKeyPEM, testFileMode))

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	return &testPKI{files: files, caPool: pool, clientCert: clientPair}
}

func TestTLSFilesExist(t *testing.T) {
	pki := newTestPKI(t)
	assert.True(t, pki.files.Exist())
	assert.False(t, TLSFiles{Cert: pki.files.Cert, Key: pki.files.Key, CA: missingCAPath}.Exist())
	assert.False(t, TLSFiles{}.Exist())
}

func TestBuildServerTLSConfig(t *testing.T) {
	pki := newTestPKI(t)

	t.Run("valid files yield mutual TLS with the default suites", func(t *testing.T) {
		cfg, err := BuildServerTLSConfig(pki.files, TLSOptions{MinVersion: "1.2"})
		require.NoError(t, err)
		assert.NotNil(t, cfg.GetCertificate)
		assert.NotNil(t, cfg.ClientCAs)
		assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Equal(t, defaultCipherSuites, cfg.CipherSuites)
	})
	t.Run("missing certificate", func(t *testing.T) {
		cfg, err := BuildServerTLSConfig(TLSFiles{Cert: missingCertPath, Key: pki.files.Key, CA: pki.files.CA}, TLSOptions{MinVersion: "1.2"})
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load server certificate")
	})
	t.Run("missing key", func(t *testing.T) {
		cfg, err := BuildServerTLSConfig(TLSFiles{Cert: pki.files.Cert, Key: missingKeyPath, CA: pki.files.CA}, TLSOptions{MinVersion: "1.2"})
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load server certificate")
	})
	t.Run("missing CA", func(t *testing.T) {
		cfg, err := BuildServerTLSConfig(TLSFiles{Cert: pki.files.Cert, Key: pki.files.Key, CA: missingCAPath}, TLSOptions{MinVersion: "1.2"})
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read CA certificate")
	})
	t.Run("CA file without certificates names the path", func(t *testing.T) {
		invalidCA := filepath.Join(t.TempDir(), "invalid-ca.pem")
		require.NoError(t, os.WriteFile(invalidCA, []byte("not a certificate"), testFileMode))
		cfg, err := BuildServerTLSConfig(TLSFiles{Cert: pki.files.Cert, Key: pki.files.Key, CA: invalidCA}, TLSOptions{MinVersion: "1.2"})
		assert.Nil(t, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid CA certificates found in")
		assert.Contains(t, err.Error(), invalidCA)
	})
	t.Run("unsupported minimum version", func(t *testing.T) {
		for _, version := range []string{"1.0", "1.1", "2.0", "invalid"} {
			cfg, err := BuildServerTLSConfig(pki.files, TLSOptions{MinVersion: version})
			assert.Nil(t, cfg, version)
			require.Error(t, err, version)
			assert.Contains(t, err.Error(), "TLS configuration", version)
		}
	})
	t.Run("named cipher suites", func(t *testing.T) {
		cfg, err := BuildServerTLSConfig(pki.files, TLSOptions{
			MinVersion:   "1.2",
			CipherSuites: []string{"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", "TLS_AES_256_GCM_SHA384"},
		})
		require.NoError(t, err)
		assert.Equal(t, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, tls.TLS_AES_256_GCM_SHA384}, cfg.CipherSuites)
	})
	t.Run("unknown cipher suite", func(t *testing.T) {
		cfg, err := BuildServerTLSConfig(pki.files, TLSOptions{MinVersion: "1.2", CipherSuites: []string{"TLS_INVALID_CIPHER"}})
		assert.Nil(t, cfg)
		require.ErrorIs(t, err, ErrUnsupportedCipherSuite)
		assert.Contains(t, err.Error(), "TLS_INVALID_CIPHER")
	})
}
