package certificates

import (
	"context"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// issuanceTarget is the base station a client certificate is issued for.
type issuanceTarget struct {
	id     int64  // the base station's row id
	eui    []byte // big-endian EUI, the storage key
	dashed string // canonical uppercase-dashed EUI, the certificate CN
}

// GenerateCertificate issues a client certificate for a base station the
// requesting tenant owns, records it on the base station and returns the
// download references of its bundle.
func (s *Service) GenerateCertificate(ctx context.Context, req *grpcservices.CertificateRequest) (*grpcservices.CertificateResponse, error) {
	s.logger.InfoContext(ctx, LogCertGenerationRequested,
		logger.FieldBsEuiSnake, req.BsEUI,
		logger.FieldName, req.BaseStationName,
		logger.FieldValidityDays, req.ValidityDays)

	target, err := s.authorizeIssuance(ctx, req)
	if err != nil {
		return nil, err
	}

	certID := uuid.New().String()
	certDir := filepath.Join(s.settings.tempDir, certID)
	expiry, err := s.issueBundle(ctx, certDir, target, req.ValidityDays)
	if err != nil {
		s.discardBundle(ctx, certDir)
		return nil, err
	}

	// Persistence is part of issuance: a station never receives a
	// certificate the service center did not record.
	if err := s.persistCertsToBaseStation(ctx, certDir, target.id, req.TenantID, expiry); err != nil {
		s.logger.ErrorContext(ctx, LogCertPersistenceSkipped, logger.FieldError, err)
		s.discardBundle(ctx, certDir)
		return nil, fmt.Errorf("%w: %w", ErrPersistenceFailed, err)
	}

	expiresAt := s.clock.Now().Add(certDownloadWindow)
	return &grpcservices.CertificateResponse{
		BsEUI:            target.dashed,
		BaseStationID:    target.id,
		ServiceCenterURL: config.GetServiceCenterURL(s.settings.protocol),
		DownloadURLs: map[string]string{
			"ca_cert":     certID,
			"client_cert": certID,
			"private_key": certID,
		},
		ExpiresAt: &expiresAt,
	}, nil
}

// authorizeIssuance validates the request and verifies, before any file or
// certgen work, that the requesting tenant owns a base station with the EUI;
// otherwise a certificate could be minted for another tenant's station.
func (s *Service) authorizeIssuance(ctx context.Context, req *grpcservices.CertificateRequest) (issuanceTarget, error) {
	euiValue, err := validation.ParseEUI(req.BsEUI)
	if err != nil {
		s.logger.WarnContext(ctx, LogCertInvalidEUI, logger.FieldBsEuiSnake, req.BsEUI, logger.FieldError, err)
		return issuanceTarget{}, ErrInvalidBaseStationEUI
	}
	if req.ValidityDays < 1 || req.ValidityDays > maxValidityDays {
		s.logger.WarnContext(ctx, LogCertInvalidValidityDays, logger.FieldValidityDays, req.ValidityDays)
		return issuanceTarget{}, ErrInvalidValidityPeriod
	}
	if req.TenantID <= 0 {
		s.logger.WarnContext(ctx, LogCertPersistenceSkipped,
			logger.FieldBsEuiSnake, req.BsEUI, logger.FieldTenantIDSnake, req.TenantID)
		return issuanceTarget{}, ErrTenantRequired
	}
	euiBytes := binary.BigEndian.AppendUint64(nil, euiValue)
	station, err := s.bsRepo.GetByEUI(ctx, req.TenantID, euiBytes)
	if err != nil {
		s.logger.WarnContext(ctx, LogCertPersistenceSkipped,
			logger.FieldBsEuiSnake, req.BsEUI, logger.FieldTenantIDSnake, req.TenantID, logger.FieldError, err)
		return issuanceTarget{}, fmt.Errorf("%w: %w", ErrBaseStationNotFound, err)
	}
	return issuanceTarget{id: station.ID, eui: euiBytes, dashed: mioty.FormatEUI64Dashed(euiValue)}, nil
}

// issueBundle generates the client certificate of target into certDir, signed
// by the service CA, and returns its expiry; the CA key never stays in the
// bundle. The caller discards certDir on failure.
func (s *Service) issueBundle(ctx context.Context, certDir string, target issuanceTarget, validityDays int32) (time.Time, error) {
	s.logger.InfoContext(ctx, LogCertDirectoryInfo, logger.FieldPath, certDir)
	if err := os.MkdirAll(certDir, certDirPerm); err != nil {
		s.logger.ErrorContext(ctx, LogCertDirCreateFailed, logger.FieldError, err)
		return time.Time{}, fmt.Errorf("%w: %w", ErrDirectoryCreate, err)
	}
	s.logger.InfoContext(ctx, LogCertGeneratorPathInfo, logger.FieldPath, s.settings.certGenPath)

	if err := s.stageSigningCA(ctx, certDir); err != nil {
		return time.Time{}, err
	}
	if err := s.runClientCertGen(ctx, certDir, target.dashed, validityDays); err != nil {
		return time.Time{}, err
	}
	if err := os.Remove(filepath.Join(certDir, caKeyFileName)); err != nil {
		s.logger.ErrorContext(ctx, LogCertCAKeyRemoveFailed, logger.FieldError, err)
		return time.Time{}, fmt.Errorf("%w: %w", ErrGenerationFailed, err)
	}

	expiry := clientCertificateExpiry(certDir)
	s.writeBundleInfo(ctx, certDir, target.dashed, validityDays, expiry)
	return expiry, nil
}

// stageSigningCA copies the service CA certificate and key into certDir so
// certgen can sign the client certificate with the existing CA.
func (s *Service) stageSigningCA(ctx context.Context, certDir string) error {
	if err := s.copyFromCertsDir(ctx, certDir, caCertFileName,
		LogCertCACertReadFailed, ErrCACertRead, LogCertCACertCopyFailed, ErrCACertCopy); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, LogCertCACertCopied,
		logger.FieldFrom, s.caCertPath(),
		logger.FieldTo, filepath.Join(certDir, caCertFileName))
	return s.copyFromCertsDir(ctx, certDir, caKeyFileName,
		LogCertCAKeyReadFailed, ErrCAKeyRead, LogCertCAKeyCopyFailed, ErrCAKeyCopy)
}

// copyFromCertsDir copies one file of the service certificate directory into
// certDir, reporting a read or write failure with its own message and sentinel.
func (s *Service) copyFromCertsDir(ctx context.Context, certDir, name, readLog string, readErr error, copyLog string, copyErr error) error {
	data, err := os.ReadFile(filepath.Clean(filepath.Join(s.settings.certsDir, name))) // #nosec G304 - path validated via filepath.Clean
	if err != nil {
		s.logger.ErrorContext(ctx, readLog, logger.FieldError, err)
		return fmt.Errorf("%w: %w", readErr, err)
	}
	if err := os.WriteFile(filepath.Join(certDir, name), data, certFilePerm); err != nil { //nolint:gosec // G703: path built from configured cert dir and canonically validated EUI
		s.logger.ErrorContext(ctx, copyLog, logger.FieldError, err)
		return fmt.Errorf("%w: %w", copyErr, err)
	}
	return nil
}

// runClientCertGen runs certgen with -client-only for the base station CN.
func (s *Service) runClientCertGen(ctx context.Context, certDir, commonName string, validityDays int32) error {
	genArgs := []string{
		certGenFlagDir, certDir,
		certGenFlagDays, fmt.Sprintf("%d", validityDays),
		certGenFlagClient, commonName,
		certGenFlagClientOnly,
	}
	s.logger.InfoContext(ctx, LogCertGenerationExecuting, logger.FieldCommand, s.settings.certGenPath, logger.FieldArgs, genArgs)

	stdout, stderr, err := s.certGen(ctx, s.settings.certGenPath, genArgs...)
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertGenerationFailed, logger.FieldError, err)
		s.logger.DebugContext(ctx, LogCertGenerationStdout, logger.FieldOutput, stdout)
		s.logger.DebugContext(ctx, LogCertGenerationStderr, logger.FieldOutput, stderr)
		if errors.Is(err, ErrGeneratorNotFound) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrGenerationFailed, fmt.Errorf("%s: %w", stderr, err))
	}
	s.logger.InfoContext(ctx, LogCertGenerationSuccess)
	s.logger.DebugContext(ctx, LogCertGenerationStdout, logger.FieldOutput, stdout)
	return nil
}

// clientCertificateExpiry reads the generated client certificate's NotAfter;
// the zero time when it cannot be read.
func clientCertificateExpiry(certDir string) time.Time {
	certPEM, err := os.ReadFile(filepath.Join(certDir, clientCertFileName)) // #nosec G304 - path constructed from UUID certDir
	if err != nil {
		return time.Time{}
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return time.Time{}
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}
	}
	return cert.NotAfter
}

// writeBundleInfo records the bundle's base station and validity for the
// download and cleanup paths; a failure is logged, the bundle stays usable.
func (s *Service) writeBundleInfo(ctx context.Context, certDir, bsEUIDashed string, validityDays int32, expiry time.Time) {
	var expiresAt string
	if !expiry.IsZero() {
		expiresAt = expiry.Format(time.RFC3339)
	}
	infoJSON, err := json.Marshal(map[string]interface{}{
		"bsEui":        bsEUIDashed,
		"createdAt":    s.clock.Now().Format(time.RFC3339),
		"expiresAt":    expiresAt,
		"validityDays": validityDays,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertInfoWriteFailed, logger.FieldError, err)
		return
	}
	if err := os.WriteFile(filepath.Join(certDir, certInfoFileName), infoJSON, certFilePerm); err != nil {
		s.logger.ErrorContext(ctx, LogCertInfoWriteFailed, logger.FieldError, err)
	}
}

// discardBundle removes a bundle directory whose issuance failed.
func (s *Service) discardBundle(ctx context.Context, certDir string) {
	if err := os.RemoveAll(certDir); err != nil {
		s.logger.ErrorContext(ctx, LogCertDirRemoveFailed, logger.FieldError, err)
	}
}

// persistCertsToBaseStation stores the generated certificate material and the
// encrypted private key in the TLS fields of the base station with stationID.
func (s *Service) persistCertsToBaseStation(ctx context.Context, certDir string, stationID, tenantID int64, expiryTime time.Time) error {
	caCertData, err := os.ReadFile(filepath.Join(certDir, caCertFileName)) // #nosec G304 - path from UUID-based certDir
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertFileReadFailed, logger.FieldFile, caCertFileName, logger.FieldError, err)
		return ErrCACertRead
	}
	clientCertData, err := os.ReadFile(filepath.Join(certDir, clientCertFileName)) // #nosec G304 - path from UUID-based certDir
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertFileReadFailed, logger.FieldFile, clientCertFileName, logger.FieldError, err)
		return ErrGenerationFailed
	}
	clientKeyData, err := os.ReadFile(filepath.Join(certDir, clientKeyFileName)) // #nosec G304 - path from UUID-based certDir
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertFileReadFailed, logger.FieldFile, clientKeyFileName, logger.FieldError, err)
		return ErrGenerationFailed
	}

	block, _ := pem.Decode(clientCertData)
	if block == nil {
		s.logger.ErrorContext(ctx, LogCertPEMBlockParseFailed, logger.FieldFile, clientCertFileName)
		return ErrGenerationFailed
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertParseFailed, logger.FieldFile, clientCertFileName, logger.FieldError, err)
		return ErrGenerationFailed
	}

	encrypted, err := s.keyEncryptor.EncryptKey(clientKeyData)
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertGenerationFailed, logger.FieldOperation, operationEncryptKey, logger.FieldError, err)
		return ErrGenerationFailed
	}

	return s.bsRepo.Update(ctx, tenantID, stationID, map[string]interface{}{
		"tls_ca_certificate":   string(caCertData),
		"tls_certificate":      string(clientCertData),
		"tls_key":              encrypted,
		"tls_cert_fingerprint": crypto.CertFingerprintSHA256(cert.Raw),
		"tls_cert_expires_at":  expiryTime,
	})
}
