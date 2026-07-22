package crypto

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
)

// CertFingerprintSHA256 returns the lowercase-hex SHA-256 fingerprint of a
// DER-encoded certificate. This is the canonical format stored in the
// tls_cert_fingerprint column and compared during BSSCI connect enforcement.
func CertFingerprintSHA256(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

// CertFingerprintFromPEM parses a PEM-encoded certificate and returns its
// canonical SHA-256 fingerprint.
func CertFingerprintFromPEM(pemData []byte) (string, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return "", errors.New("no PEM block in certificate data")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse certificate: %w", err)
	}
	return CertFingerprintSHA256(cert.Raw), nil
}
