package certificates

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// GenerateServerCertificates generates new server certificates.
func (s *Service) GenerateServerCertificates(ctx context.Context) error {
	s.logger.InfoContext(ctx, LogServerCertGenRequested)
	s.logger.InfoContext(ctx, LogCertCertsPathInfo, logger.FieldPath, s.settings.certsDir)

	if err := os.MkdirAll(s.settings.certsDir, certDirPerm); err != nil {
		s.logger.ErrorContext(ctx, LogCertsDirCreateFailed, logger.FieldError, err)
		return fmt.Errorf("%w: %w", ErrDirectoryCreate, err)
	}
	return s.runServerCertGen(ctx)
}

// RenewServerCertificates renews server certificates using -server-only to preserve the existing CA.
func (s *Service) RenewServerCertificates(ctx context.Context) error {
	s.logger.InfoContext(ctx, LogServerCertRenewalRequested)

	certsDir := s.settings.certsDir
	if _, err := os.Stat(filepath.Join(certsDir, serverCertFileName)); os.IsNotExist(err) {
		return ErrNoCertificatesToRenew
	}
	// -server-only signs with the existing CA, so both CA files are required.
	if _, err := os.Stat(s.caCertPath()); os.IsNotExist(err) {
		return ErrCACertRead
	}
	if _, err := os.Stat(filepath.Join(certsDir, caKeyFileName)); os.IsNotExist(err) {
		return ErrCAKeyRead
	}
	return s.runServerCertGen(ctx, certGenFlagServerOnly)
}

// runServerCertGen runs certgen for the server certificate of the planned
// names into the certificates directory, with any extra flags.
func (s *Service) runServerCertGen(ctx context.Context, extraFlags ...string) error {
	names := s.plannedServerNames(ctx)
	genArgs := append([]string{
		certGenFlagDir, s.settings.certsDir,
		certGenFlagDays, fmt.Sprintf("%d", s.settings.serverValidityDays),
		certGenFlagServer, names.subject,
	}, extraFlags...)
	if len(names.altNames) > 0 {
		genArgs = append(genArgs, certGenFlagAltNames, strings.Join(names.altNames, altNameSeparator))
	}
	s.logger.InfoContext(ctx, LogServerCertGenExecuting, logger.FieldCommand, s.settings.certGenPath, logger.FieldArgs, genArgs, logger.FieldHostname, names.subject)

	stdout, stderr, err := s.certGen(ctx, s.settings.certGenPath, genArgs...)
	if err != nil {
		s.logger.ErrorContext(ctx, LogServerCertGenFailed, logger.FieldError, err)
		s.logger.DebugContext(ctx, LogCertGenerationStdout, logger.FieldOutput, stdout)
		s.logger.DebugContext(ctx, LogCertGenerationStderr, logger.FieldOutput, stderr)
		if errors.Is(err, ErrGeneratorNotFound) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrServerGenerationFailed, fmt.Errorf("%s: %w", stderr, err))
	}

	s.logger.InfoContext(ctx, LogServerCertGenSuccess)
	return nil
}

// GetServerCertificateStatus returns the status of server certificates.
func (s *Service) GetServerCertificateStatus(ctx context.Context) (*grpcservices.CertificateStatus, error) {
	now := s.clock.Now()
	return &grpcservices.CertificateStatus{
		Server:       s.getCertificateInfo(ctx, filepath.Join(s.settings.certsDir, serverCertFileName), now),
		CA:           s.getCertificateInfo(ctx, s.caCertPath(), now),
		RenewalNames: s.plannedServerNames(ctx).all(),
	}, nil
}

// getCertificateInfo reads the certificate at certPath and judges it as of now; nil when absent or unreadable.
func (s *Service) getCertificateInfo(ctx context.Context, certPath string, now time.Time) *grpcservices.CertificateInfo {
	cert := s.readCertificate(ctx, certPath)
	if cert == nil {
		return nil
	}

	issuer := cert.Issuer.CommonName
	if issuer == "" && len(cert.Issuer.Organization) > 0 {
		issuer = cert.Issuer.Organization[0]
	}

	subject := cert.Subject.CommonName
	if subject == "" && len(cert.Subject.Organization) > 0 {
		subject = cert.Subject.Organization[0]
	}

	return &grpcservices.CertificateInfo{
		Subject:         subject,
		Issuer:          issuer,
		NotBefore:       cert.NotBefore,
		NotAfter:        cert.NotAfter,
		DaysUntilExpiry: int32(cert.NotAfter.Sub(now).Hours() / hoursPerDay),
		Valid:           !now.Before(cert.NotBefore) && !now.After(cert.NotAfter),
	}
}

// readCertificate parses the PEM certificate at certPath; nil when absent or unreadable.
func (s *Service) readCertificate(ctx context.Context, certPath string) *x509.Certificate {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil
	}

	certPEM, err := os.ReadFile(certPath) // #nosec G304 - path from trusted KC-Core certs dir
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertFileReadFailed,
			logger.FieldCertPath, certPath, logger.FieldError, err)
		return nil
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		s.logger.ErrorContext(ctx, LogCertPEMBlockParseFailed, logger.FieldCertPath, certPath)
		return nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertParseFailed,
			logger.FieldCertPath, certPath, logger.FieldError, err)
		return nil
	}
	return cert
}
