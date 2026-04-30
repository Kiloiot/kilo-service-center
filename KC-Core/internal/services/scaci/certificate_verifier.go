// Package scaciservices implements the SCACI (Service Center Application Center Interface) protocol server.
//
// certificate_verifier.go implements CertificateVerifier interface.
//
// Extracted Logic:
//   - Certificate expiry validation (NotBefore/NotAfter checks)
//   - Extended key usage validation (ClientAuth requirement for mutual TLS)
//   - Subject validation (basic presence checks)
//
// Dependencies (injected):
//   - logger.Logger: Structured logging
//
// Error Handling:
//   - Returns error tokens from errors_catalog.go (package-private constants)
//   - NO Go errors returned from public methods
//   - Transport layer resolves tokens → POSIX codes
//
// Infrastructure Reuse:
//   - errors_catalog.go: All error tokens (err*)
//   - log_messages.go: All log constants (Log*)
package scaciservices

import (
	"crypto/x509"
	"time"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/scaci"
)

// certificateVerifier implements CertificateVerifier interface
//
// This service provides defense-in-depth certificate validation beyond TLS handshake.
// It ensures client certificates meet KiloCenter security requirements before tenant
// resolution and session establishment.
//
// Validation Flow:
//  1. Check certificate expiry (NotBefore/NotAfter)
//  2. Verify extended key usage (must include ClientAuth)
//  3. Validate subject presence
//
// Used by HandshakeService.ValidateConnect() as first security gate.
type certificateVerifier struct {
	logger logger.Logger
}

// NewCertificateVerifier creates a new certificate verifier service
//
// Parameters:
//   - logger: Structured logger
//
// Returns:
//   - CertificateVerifier: Service instance implementing interface
func NewCertificateVerifier(
	log logger.Logger) scaci.CertificateVerifier {
	return &certificateVerifier{
		logger: log,
	}
}

// VerifyCertificate implements CertificateVerifier.VerifyCertificate
//
// Validates client certificate security properties:
//  1. Certificate expiry: NotBefore <= now <= NotAfter
//  2. Key usage: ExtKeyUsageClientAuth present (required for mutual TLS)
//  3. Subject: Has valid subject (CommonName or Organization)
//
// Parameters:
//   - cert: Client certificate from TLS handshake
//
// Returns:
//   - string: Error token if validation fails, "" on success
func (cv *certificateVerifier) VerifyCertificate(cert *x509.Certificate) string {
	if cert == nil {
		cv.logger.Error(scaci.LogSCACINoClientCertificate)
		return scaci.ErrNilCertificate
	}

	now := time.Now()

	// Check 1: Certificate expiry validation
	if now.Before(cert.NotBefore) {
		cv.logger.Error(scaci.LogSCACICertNotYetValid,
			"notBefore", cert.NotBefore,
			"now", now,
			"subject", cert.Subject.String())
		return scaci.ErrCertNotYetValid
	}

	if now.After(cert.NotAfter) {
		cv.logger.Error(scaci.LogSCACICertExpired,
			"notAfter", cert.NotAfter,
			"now", now,
			"subject", cert.Subject.String())
		return scaci.ErrCertExpired
	}

	// Check 2: Extended key usage validation (ClientAuth required for mutual TLS)
	hasClientAuth := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
			break
		}
	}

	if !hasClientAuth {
		cv.logger.Error(scaci.LogSCACICertMissingClientAuth,
			"subject", cert.Subject.String(),
			"extKeyUsage", cert.ExtKeyUsage)
		return scaci.ErrCertMissingClientAuth
	}

	// Check 3: Subject validation (basic presence check)
	if cert.Subject.CommonName == "" && len(cert.Subject.Organization) == 0 {
		cv.logger.Error(scaci.LogSCACICertInvalidSubject,
			"subject", cert.Subject.String())
		return scaci.ErrCertInvalidSubject
	}

	cv.logger.Debug(scaci.LogSCACICertValidationPassed,
		"subject", cert.Subject.String(),
		"cn", cert.Subject.CommonName,
		"notBefore", cert.NotBefore,
		"notAfter", cert.NotAfter)

	return ""
}
