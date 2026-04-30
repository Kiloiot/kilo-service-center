package scaciservices

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/scaci"
)

// Helper to create test certificate with custom properties
func createTestCert(notBefore, notAfter time.Time, extKeyUsage []x509.ExtKeyUsage, subject pkix.Name) *x509.Certificate {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      subject,
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		ExtKeyUsage:  extKeyUsage,
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	cert, _ := x509.ParseCertificate(certDER)
	return cert
}

func TestVerifyCertificate_Valid(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create valid cert: not expired, has ClientAuth, valid subject
	now := time.Now()
	cert := createTestCert(
		now.Add(-24*time.Hour),                                                  // NotBefore: yesterday
		now.Add(365*24*time.Hour),                                               // NotAfter: 1 year from now
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},                          // Has ClientAuth
		pkix.Name{CommonName: "test-client", Organization: []string{"TestOrg"}}, // Valid subject
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != "" {
		t.Errorf("expected valid certificate to pass, got error token: %s", errToken)
	}
}

func TestVerifyCertificate_Nil(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	errToken := verifier.VerifyCertificate(nil)
	if errToken != scaci.ErrNilCertificate {
		t.Errorf("expected ErrNilCertificate, got: %s", errToken)
	}
}

func TestVerifyCertificate_NotYetValid(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create cert that's not yet valid (NotBefore is in the future)
	now := time.Now()
	cert := createTestCert(
		now.Add(24*time.Hour),                          // NotBefore: tomorrow
		now.Add(365*24*time.Hour),                      // NotAfter: 1 year from now
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, // Has ClientAuth
		pkix.Name{CommonName: "test-client"},
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != scaci.ErrCertNotYetValid {
		t.Errorf("expected ErrCertNotYetValid, got: %s", errToken)
	}
}

func TestVerifyCertificate_Expired(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create expired cert (NotAfter is in the past)
	now := time.Now()
	cert := createTestCert(
		now.Add(-365*24*time.Hour),                     // NotBefore: 1 year ago
		now.Add(-24*time.Hour),                         // NotAfter: yesterday
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, // Has ClientAuth
		pkix.Name{CommonName: "test-client"},
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != scaci.ErrCertExpired {
		t.Errorf("expected ErrCertExpired, got: %s", errToken)
	}
}

func TestVerifyCertificate_MissingClientAuth(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create cert without ClientAuth (has ServerAuth instead)
	now := time.Now()
	cert := createTestCert(
		now.Add(-24*time.Hour),
		now.Add(365*24*time.Hour),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, // Wrong key usage
		pkix.Name{CommonName: "test-client"},
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != scaci.ErrCertMissingClientAuth {
		t.Errorf("expected ErrCertMissingClientAuth, got: %s", errToken)
	}
}

func TestVerifyCertificate_NoExtKeyUsage(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create cert with no ExtKeyUsage at all
	now := time.Now()
	cert := createTestCert(
		now.Add(-24*time.Hour),
		now.Add(365*24*time.Hour),
		[]x509.ExtKeyUsage{}, // Empty ExtKeyUsage
		pkix.Name{CommonName: "test-client"},
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != scaci.ErrCertMissingClientAuth {
		t.Errorf("expected ErrCertMissingClientAuth, got: %s", errToken)
	}
}

func TestVerifyCertificate_InvalidSubject_Empty(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create cert with empty subject (no CN, no Organization)
	now := time.Now()
	cert := createTestCert(
		now.Add(-24*time.Hour),
		now.Add(365*24*time.Hour),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		pkix.Name{}, // Empty subject
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != scaci.ErrCertInvalidSubject {
		t.Errorf("expected ErrCertInvalidSubject, got: %s", errToken)
	}
}

func TestVerifyCertificate_ValidSubject_OnlyOrganization(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create cert with Organization but no CN (should be valid)
	now := time.Now()
	cert := createTestCert(
		now.Add(-24*time.Hour),
		now.Add(365*24*time.Hour),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		pkix.Name{Organization: []string{"TestOrg"}}, // Only org, no CN
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != "" {
		t.Errorf("expected valid certificate (org without CN), got error token: %s", errToken)
	}
}

func TestVerifyCertificate_MultipleExtKeyUsages(t *testing.T) {
	logger := logger.NewNop()
	verifier := NewCertificateVerifier(logger)

	// Create cert with multiple ExtKeyUsages including ClientAuth
	now := time.Now()
	cert := createTestCert(
		now.Add(-24*time.Hour),
		now.Add(365*24*time.Hour),
		[]x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth, // Has ClientAuth among others
			x509.ExtKeyUsageEmailProtection,
		},
		pkix.Name{CommonName: "test-client"},
	)

	errToken := verifier.VerifyCertificate(cert)
	if errToken != "" {
		t.Errorf("expected valid certificate (multiple usages), got error token: %s", errToken)
	}
}
