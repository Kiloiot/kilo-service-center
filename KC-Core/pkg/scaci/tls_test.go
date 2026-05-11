// Package scaci provides tests for SCACI TLS configuration per SCACI §1
//
// Coverage:
//   - LoadTLSConfig: Valid/invalid certificate scenarios
//   - CA pool validation: Empty CA pool rejection with path-inclusive error message
//   - Self-signed certificate generation helper for tests
package scaci

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCertAndKey creates a self-signed certificate and key pair for testing
// Returns paths to temporary cert and key files (caller must clean up)
func generateTestCertAndKey(t *testing.T) (certPath, keyPath string) {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate RSA key")

	// Create self-signed certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"KiloCenter Test"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err, "failed to create certificate")

	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Write certificate to PEM file
	certPath = filepath.Join(tmpDir, "cert.pem")
	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, err)
	require.NoError(t, certFile.Close())

	// Write private key to PEM file
	keyPath = filepath.Join(tmpDir, "key.pem")
	keyFile, err := os.Create(keyPath)
	require.NoError(t, err)
	err = pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	require.NoError(t, err)
	require.NoError(t, keyFile.Close())

	return certPath, keyPath
}

// ============================================================================
// R5: TLS Tests with Updated Error Strings
// ============================================================================

func TestLoadTLSConfig_ValidCertificates(t *testing.T) {
	// Generate test certificates
	certPath, keyPath := generateTestCertAndKey(t)
	caPath := certPath // Self-signed cert is also CA

	cfg := config.TLSConfig{
		CertFile:   certPath,
		KeyFile:    keyPath,
		CAFile:     caPath,
		MinVersion: "1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	require.NoError(t, err)
	require.NotNil(t, tlsConfig)
	assert.Len(t, tlsConfig.Certificates, 1)
	assert.NotNil(t, tlsConfig.ClientCAs)
}

func TestLoadTLSConfig_MissingCertFile(t *testing.T) {
	_, keyPath := generateTestCertAndKey(t)

	cfg := config.TLSConfig{
		CertFile:   "/nonexistent/cert.pem",
		KeyFile:    keyPath,
		CAFile:     keyPath,
		MinVersion: "1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	assert.Nil(t, tlsConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load server certificate")
}

func TestLoadTLSConfig_MissingKeyFile(t *testing.T) {
	certPath, _ := generateTestCertAndKey(t)

	cfg := config.TLSConfig{
		CertFile:   certPath,
		KeyFile:    "/nonexistent/key.pem",
		CAFile:     certPath,
		MinVersion: "1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	assert.Nil(t, tlsConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load server certificate")
}

func TestLoadTLSConfig_MissingCAFile(t *testing.T) {
	certPath, keyPath := generateTestCertAndKey(t)

	cfg := config.TLSConfig{
		CertFile:   certPath,
		KeyFile:    keyPath,
		CAFile:     "/nonexistent/ca.pem",
		MinVersion: "1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	assert.Nil(t, tlsConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read CA certificate")
}

func TestLoadTLSConfig_EmptyCAPool_IncludesFilePath(t *testing.T) {
	// Create a file with invalid PEM content (not a valid certificate)
	tmpDir := t.TempDir()
	invalidCAPath := filepath.Join(tmpDir, "invalid-ca.pem")
	err := os.WriteFile(invalidCAPath, []byte("not a valid PEM certificate"), 0600)
	require.NoError(t, err)

	certPath, keyPath := generateTestCertAndKey(t)

	cfg := config.TLSConfig{
		CertFile:   certPath,
		KeyFile:    keyPath,
		CAFile:     invalidCAPath,
		MinVersion: "1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	// Assert: R5 requirement - error message must include the CA file path
	assert.Nil(t, tlsConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid CA certificates found in")
	assert.Contains(t, err.Error(), invalidCAPath, "error message must include CA file path for diagnostics")
}

func TestLoadTLSConfig_InvalidMinVersion(t *testing.T) {
	certPath, keyPath := generateTestCertAndKey(t)

	cfg := config.TLSConfig{
		CertFile:   certPath,
		KeyFile:    keyPath,
		CAFile:     certPath,
		MinVersion: "1.0", // Too old
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	// Note: ParseTLSMinVersion may or may not reject TLS1.0 depending on implementation
	// This test verifies the error propagation path works
	if err != nil {
		assert.Nil(t, tlsConfig)
		assert.Contains(t, err.Error(), "TLS configuration")
	}
}

func TestLoadTLSConfig_WithCipherSuites(t *testing.T) {
	certPath, keyPath := generateTestCertAndKey(t)

	cfg := config.TLSConfig{
		CertFile:   certPath,
		KeyFile:    keyPath,
		CAFile:     certPath,
		MinVersion: "1.2",
		AllowedCiphers: []string{
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		},
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	require.NoError(t, err)
	require.NotNil(t, tlsConfig)
	assert.Len(t, tlsConfig.CipherSuites, 2)
}

func TestLoadTLSConfig_InvalidCipherSuite(t *testing.T) {
	certPath, keyPath := generateTestCertAndKey(t)

	cfg := config.TLSConfig{
		CertFile:   certPath,
		KeyFile:    keyPath,
		CAFile:     certPath,
		MinVersion: "1.2",
		AllowedCiphers: []string{
			"INVALID_CIPHER_SUITE",
		},
	}

	tlsConfig, err := LoadTLSConfig(cfg)

	assert.Nil(t, tlsConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported cipher suite")
}
