// Package certificates provides certificate service implementation for gRPC layer.
package certificates

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	pkggrpc "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/google/uuid"
)

// Service implements grpcservices.CertificateService.
type Service struct {
	config             *config.Config
	logger             logger.Logger
	certGenPath        string
	certsDir           string
	tempDir            string
	serverValidityDays int
	protocolConfig     *config.ProtocolConfig
	bsRepo             interfaces.BaseStationRepository
	keyEncryptor       *crypto.KeyEncryptor
}

// New creates a new certificate service.
func New(cfg *config.Config, log logger.Logger) *Service {
	ctx := context.Background()

	// Use certificate config paths if available, fall back to constants
	certGenPath := cfg.Certificates.CertGenPath
	if certGenPath == "" {
		certGenPath = config.DefaultCertificatesCertGenPath
	}
	certsDir := cfg.Certificates.CertsDir
	if certsDir == "" {
		certsDir = config.DefaultCertificatesCertsDir
	}
	tempDir := cfg.Certificates.TempDir
	if tempDir == "" {
		tempDir = config.DefaultCertificatesTempDir
	}

	// Fail-fast validation: check if certgen binary exists at startup
	// This helps debug issues with relative paths and working directory mismatches
	if _, err := os.Stat(certGenPath); err != nil {
		wd, _ := os.Getwd()
		log.WarnContext(ctx, pkggrpc.LogCertConfigMissingPath,
			"configured_path", certGenPath,
			"working_directory", wd,
			"hint", pkggrpc.LogCertConfigMissingPathHint)
	} else {
		log.InfoContext(ctx, pkggrpc.LogCertGeneratorPathInfo, "path", certGenPath)
	}

	if _, err := os.Stat(certsDir); err != nil {
		log.WarnContext(ctx, "Certificate directory not found",
			"path", certsDir,
			"hint", "Run certgen to generate certificates before starting")
	}

	// Resolve server validity days with bounds check
	serverValidityDays := cfg.Certificates.ServerValidityDays
	if serverValidityDays <= 0 {
		serverValidityDays = config.DefaultCertificatesServerValidityDays
		log.WarnContext(ctx, "Invalid server_validity_days, using default", "default", serverValidityDays)
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0750); err != nil {
		log.ErrorContext(ctx, pkggrpc.LogCertTempDirCreateFailed, "error", err)
	}

	return &Service{
		config:             cfg,
		logger:             log,
		certGenPath:        certGenPath,
		certsDir:           certsDir,
		tempDir:            tempDir,
		serverValidityDays: serverValidityDays,
		protocolConfig:     &cfg.Protocol,
	}
}

// WithBaseStationRepo adds TLS persistence capability.
func (s *Service) WithBaseStationRepo(repo interfaces.BaseStationRepository) *Service {
	s.bsRepo = repo
	return s
}

// WithKeyEncryptor adds private key encryption capability.
func (s *Service) WithKeyEncryptor(enc *crypto.KeyEncryptor) *Service {
	s.keyEncryptor = enc
	return s
}

// GenerateCertificate generates a new certificate for a base station.
func (s *Service) GenerateCertificate(ctx context.Context, req *grpcservices.CertificateRequest) (*grpcservices.CertificateResponse, error) {
	s.logger.InfoContext(ctx, pkggrpc.LogCertGenerationRequested,
		"bs_eui", req.BsEUI,
		"name", req.BaseStationName,
		"validity_days", req.ValidityDays)

	// Validate and normalize the Base Station EUI (accepts dashed, colon-separated, or plain 16-hex)
	euiValue, euiErr := validation.ParseEUI(req.BsEUI)
	if euiErr != nil {
		s.logger.WarnContext(ctx, pkggrpc.LogCertInvalidEUI, "bs_eui", req.BsEUI, "error", euiErr)
		return nil, fmt.Errorf("%s: %s", pkggrpc.ErrTokenInvalidBasestationEUIFormat, pkggrpc.ResolveErrorMessage(pkggrpc.ErrTokenInvalidBasestationEUIFormat))
	}
	// Canonical uppercase-dashed form used as the certificate CN and in stored metadata
	bsEUIDashed := mioty.FormatEUI64Dashed(euiValue)

	// Validate validity days (max 3 years = 1095 days)
	if req.ValidityDays < 1 || req.ValidityDays > 1095 {
		s.logger.WarnContext(ctx, pkggrpc.LogCertInvalidValidityDays, "validity_days", req.ValidityDays)
		return nil, fmt.Errorf("%s: %s", pkggrpc.ErrTokenInvalidValidityPeriod, pkggrpc.ResolveErrorMessage(pkggrpc.ErrTokenInvalidValidityPeriod))
	}

	// Generate unique ID for this certificate set
	certID := uuid.New().String()
	certDir := filepath.Join(s.tempDir, certID)
	s.logger.InfoContext(ctx, pkggrpc.LogCertDirectoryInfo, "path", certDir)

	// Create temporary directory for certificates
	if err := os.MkdirAll(certDir, 0750); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertDirCreateFailed, "error", err)
		return nil, fmt.Errorf("%s: %w", pkggrpc.ErrTokenCertDirCreateFailed, err)
	}

	// Log the certgen path and check if it exists
	s.logger.InfoContext(ctx, pkggrpc.LogCertGeneratorPathInfo, "path", s.certGenPath)
	if _, err := os.Stat(s.certGenPath); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertGeneratorNotFound, "path", s.certGenPath, "error", err)
		if rmErr := os.RemoveAll(certDir); rmErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogCertDirRemoveFailed, "error", rmErr)
		}
		return nil, fmt.Errorf("%s: %s", pkggrpc.ErrTokenCertGeneratorNotFound, s.certGenPath)
	}

	// Copy the existing CA certificate from KC-Core instead of generating a new one
	kcCoreCertsPath := s.certsDir
	caCertSrc := filepath.Clean(filepath.Join(kcCoreCertsPath, "ca.crt"))
	caCertDst := filepath.Join(certDir, "ca.crt")

	// Copy CA certificate
	caCertData, err := os.ReadFile(caCertSrc) // #nosec G304 - path validated via filepath.Clean
	if err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertCACertReadFailed, "error", err)
		if rmErr := os.RemoveAll(certDir); rmErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogCertDirRemoveFailed, "error", rmErr)
		}
		return nil, fmt.Errorf("%s: %w", pkggrpc.ErrTokenCACertReadFailed, err)
	}

	if err := os.WriteFile(caCertDst, caCertData, 0600); err != nil { //nolint:gosec // G703: path built from configured cert dir and canonically validated EUI
		s.logger.ErrorContext(ctx, pkggrpc.LogCertCACertCopyFailed, "error", err)
		if rmErr := os.RemoveAll(certDir); rmErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogCertDirRemoveFailed, "error", rmErr)
		}
		return nil, fmt.Errorf("%s: %w", pkggrpc.ErrTokenCACertCopyFailed, err)
	}

	s.logger.InfoContext(ctx, pkggrpc.LogCertCACertCopied, "from", caCertSrc, "to", caCertDst)

	// Also copy the CA key so we can sign client certificates
	caKeySrc := filepath.Clean(filepath.Join(kcCoreCertsPath, "ca.key"))
	caKeyDst := filepath.Join(certDir, "ca.key")

	caKeyData, err := os.ReadFile(caKeySrc) // #nosec G304 - path validated via filepath.Clean
	if err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertCAKeyReadFailed, "error", err)
		if rmErr := os.RemoveAll(certDir); rmErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogCertDirRemoveFailed, "error", rmErr)
		}
		return nil, fmt.Errorf("%s: %w", pkggrpc.ErrTokenCAKeyReadFailed, err)
	}

	if err := os.WriteFile(caKeyDst, caKeyData, 0600); err != nil { //nolint:gosec // G703: path built from configured cert dir and canonically validated EUI
		s.logger.ErrorContext(ctx, pkggrpc.LogCertCAKeyCopyFailed, "error", err)
		if rmErr := os.RemoveAll(certDir); rmErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogCertDirRemoveFailed, "error", rmErr)
		}
		return nil, fmt.Errorf("%s: %w", pkggrpc.ErrTokenCAKeyCopyFailed, err)
	}

	// Execute certificate generation command with -client-only flag
	cmd := exec.Command(s.certGenPath,
		"-dir", certDir,
		"-days", fmt.Sprintf("%d", req.ValidityDays),
		"-client", bsEUIDashed,
		"-client-only",
	)

	s.logger.InfoContext(ctx, pkggrpc.LogCertGenerationExecuting, "command", s.certGenPath, "args", cmd.Args[1:])

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertGenerationFailed, "error", err)
		s.logger.DebugContext(ctx, pkggrpc.LogCertGenerationStdout, "output", stdout.String())
		s.logger.DebugContext(ctx, pkggrpc.LogCertGenerationStderr, "output", stderr.String())
		if rmErr := os.RemoveAll(certDir); rmErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogCertDirRemoveFailed, "error", rmErr)
		}
		return nil, fmt.Errorf("%s: %s", pkggrpc.ErrTokenCertGenerationFailed, stderr.String())
	}

	s.logger.InfoContext(ctx, pkggrpc.LogCertGenerationSuccess)
	s.logger.DebugContext(ctx, pkggrpc.LogCertGenerationStdout, "output", stdout.String())

	// Read the generated certificate to extract expiry date
	certPath := filepath.Join(certDir, "client.crt")
	certPEM, err := os.ReadFile(certPath) // #nosec G304 - path constructed from UUID certDir
	var certExpiryStr string
	var certExpiryTime time.Time
	if err == nil {
		block, _ := pem.Decode(certPEM)
		if block != nil {
			cert, parseErr := x509.ParseCertificate(block.Bytes)
			if parseErr == nil {
				certExpiryStr = cert.NotAfter.Format(time.RFC3339)
				certExpiryTime = cert.NotAfter
			}
		}
	}

	// Save certificate info for later retrieval
	certInfoData := map[string]interface{}{
		"bsEui":        bsEUIDashed,
		"createdAt":    time.Now().Format(time.RFC3339),
		"expiresAt":    certExpiryStr,
		"validityDays": req.ValidityDays,
	}
	infoJSON, _ := json.Marshal(certInfoData)
	if err := os.WriteFile(filepath.Join(certDir, "info.json"), infoJSON, 0600); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertInfoWriteFailed, "error", err)
	}

	// Persist certs to base station when dependencies are configured
	if s.bsRepo != nil && s.keyEncryptor != nil && req.TenantID > 0 {
		euiBytes := binary.BigEndian.AppendUint64(nil, euiValue)
		if persistErr := s.persistCertsToBaseStation(ctx, certDir, euiBytes, req.TenantID, certExpiryTime); persistErr != nil {
			s.logger.WarnContext(ctx, pkggrpc.LogCertPersistenceSkipped, "error", persistErr)
		}
	} else if s.bsRepo != nil && s.keyEncryptor == nil && req.TenantID > 0 {
		s.logger.WarnContext(ctx, pkggrpc.LogCertPersistenceSkipped, "error", pkggrpc.ErrTokenServiceNotConfigured)
	}

	// Build canonical Service Center URL from config
	serviceCenterURL := config.GetServiceCenterURL(s.protocolConfig)

	// Create download references with certID for gRPC download (valid for 15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute)
	downloadUrls := map[string]string{
		"ca_cert":     certID,
		"client_cert": certID,
		"private_key": certID,
	}

	return &grpcservices.CertificateResponse{
		BsEUI:            bsEUIDashed,
		ServiceCenterURL: serviceCenterURL,
		DownloadURLs:     downloadUrls,
		ExpiresAt:        &expiresAt,
	}, nil
}

// DownloadCertificateByID downloads a generated certificate by ID and type.
func (s *Service) DownloadCertificateByID(ctx context.Context, certType, certID string) ([]byte, string, error) {
	// Read the certificate info to get the EUI
	infoPath := filepath.Join(s.tempDir, certID, "info.json")
	infoData, err := os.ReadFile(infoPath) // #nosec G304 - certID is UUID-based
	var certInfo struct {
		BsEui string `json:"bsEui"`
	}
	if err == nil {
		if unmarshalErr := json.Unmarshal(infoData, &certInfo); unmarshalErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogCertUnmarshalFailed, "error", unmarshalErr)
		}
	}

	// Use EUI in filename if available, otherwise use certID
	euiPart := certInfo.BsEui
	if euiPart == "" {
		if len(certID) >= 8 {
			euiPart = certID[:8]
		} else {
			euiPart = certID // Use full certID when shorter than 8
		}
	}

	// Validate certificate type and set descriptive filename
	var filename, downloadName string
	switch certType {
	case pkggrpc.CertTypeCA:
		filename = "ca.crt"
		downloadName = "kilocenter-ca-certificate.crt"
	case "server":
		filename = "server.crt"
		downloadName = fmt.Sprintf("basestation-%s-server-certificate.crt", euiPart)
	case pkggrpc.CertTypeClient:
		filename = "client.crt"
		downloadName = fmt.Sprintf("basestation-%s-client-certificate.crt", euiPart)
	case pkggrpc.CertTypeKey:
		filename = "client.key"
		downloadName = fmt.Sprintf("basestation-%s-private-key.key", euiPart)
	default:
		s.logger.ErrorContext(ctx, pkggrpc.LogDownloadCertFailed, "cert_type", certType, "error", "invalid cert type")
		return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenCertTypeRequired)
	}

	// Build file path
	filePath := filepath.Join(s.tempDir, certID, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, "", fmt.Errorf("%s: %s", pkggrpc.ErrTokenCertNotFound, certType)
	}

	// Read file
	data, err := os.ReadFile(filePath) // #nosec G304 - validated certType and UUID certID
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", pkggrpc.ErrTokenCertNotFound, err)
	}

	// If this was the private key, schedule cleanup
	if certType == pkggrpc.CertTypeKey {
		go func() {
			time.Sleep(1 * time.Second)
			certPath := filepath.Join(s.tempDir, certID)
			if rmErr := os.RemoveAll(certPath); rmErr != nil {
				s.logger.WarnContext(ctx, pkggrpc.LogCertTempDirRemoveFailed,
					"certID", certID, "path", certPath, "error", rmErr)
			}
		}()
	}

	return data, downloadName, nil
}

// deriveServerHostname extracts the hostname for server certificate generation.
// Priority: BSCIExternalURL host > BSCIHost (if not wildcard) > DefaultCertificatesHostname
func (s *Service) deriveServerHostname() string {
	if s.protocolConfig.BSCIExternalURL != "" {
		raw := s.protocolConfig.BSCIExternalURL
		raw = strings.TrimPrefix(raw, "tls://")
		raw = strings.TrimPrefix(raw, "tcp://")
		host := raw
		if h, _, err := net.SplitHostPort(raw); err == nil && h != "" {
			host = h
		}
		if host != "" && host != config.DefaultProtocolBSCIHost {
			return host
		}
	}
	if s.protocolConfig.BSCIHost != "" && s.protocolConfig.BSCIHost != config.DefaultProtocolBSCIHost {
		return s.protocolConfig.BSCIHost
	}
	return config.DefaultCertificatesHostname
}

// GenerateServerCertificates generates new server certificates.
func (s *Service) GenerateServerCertificates(ctx context.Context) error {
	s.logger.InfoContext(ctx, pkggrpc.LogServerCertGenRequested)

	// Get the KC-Core certificates directory
	kcCorePath := s.certsDir
	s.logger.InfoContext(ctx, pkggrpc.LogCertCertsPathInfo, "path", kcCorePath)

	// Ensure directory exists
	if err := os.MkdirAll(kcCorePath, 0750); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertsDirCreateFailed, "error", err)
		return fmt.Errorf("%s: %w", pkggrpc.ErrTokenCertDirCreateFailed, err)
	}

	// Check if certgen exists
	if _, err := os.Stat(s.certGenPath); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertGeneratorNotFound, "path", s.certGenPath, "error", err)
		return fmt.Errorf("%s: %s", pkggrpc.ErrTokenCertGeneratorNotFound, s.certGenPath)
	}

	// Execute certificate generation command for server certificates
	serverHostname := s.deriveServerHostname()
	cmd := exec.Command(s.certGenPath,
		"-dir", kcCorePath,
		"-days", fmt.Sprintf("%d", s.serverValidityDays),
		"-server", serverHostname,
	)

	s.logger.InfoContext(ctx, pkggrpc.LogServerCertGenExecuting, "command", s.certGenPath, "args", cmd.Args[1:], "hostname", serverHostname)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogServerCertGenFailed, "error", err)
		s.logger.DebugContext(ctx, pkggrpc.LogCertGenerationStdout, "output", stdout.String())
		s.logger.DebugContext(ctx, pkggrpc.LogCertGenerationStderr, "output", stderr.String())
		return fmt.Errorf("%s: %s", pkggrpc.ErrTokenCertServerGenerationFailed, stderr.String())
	}

	s.logger.InfoContext(ctx, pkggrpc.LogServerCertGenSuccess)
	return nil
}

// RenewServerCertificates renews server certificates using -server-only to preserve the existing CA.
func (s *Service) RenewServerCertificates(ctx context.Context) error {
	s.logger.InfoContext(ctx, pkggrpc.LogServerCertRenewalRequested)

	kcCorePath := s.certsDir

	// Require existing server certificate (nothing to renew otherwise)
	if _, err := os.Stat(filepath.Join(kcCorePath, "server.crt")); os.IsNotExist(err) {
		return fmt.Errorf("%s", pkggrpc.ErrTokenNoCertsToRenew)
	}

	// Require CA files (-server-only needs them to sign)
	if _, err := os.Stat(filepath.Join(kcCorePath, "ca.crt")); os.IsNotExist(err) {
		return fmt.Errorf("%s: %s", pkggrpc.ErrTokenCACertReadFailed, pkggrpc.ResolveErrorMessage(pkggrpc.ErrTokenCACertReadFailed))
	}
	if _, err := os.Stat(filepath.Join(kcCorePath, "ca.key")); os.IsNotExist(err) {
		return fmt.Errorf("%s: %s", pkggrpc.ErrTokenCAKeyReadFailed, pkggrpc.ResolveErrorMessage(pkggrpc.ErrTokenCAKeyReadFailed))
	}

	// Check if certgen exists
	if _, err := os.Stat(s.certGenPath); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertGeneratorNotFound, "path", s.certGenPath, "error", err)
		return fmt.Errorf("%s: %s", pkggrpc.ErrTokenCertGeneratorNotFound, s.certGenPath)
	}

	// Execute certgen with -server-only to regenerate server cert without touching the CA
	serverHostname := s.deriveServerHostname()
	cmd := exec.Command(s.certGenPath,
		"-dir", kcCorePath,
		"-days", fmt.Sprintf("%d", s.serverValidityDays),
		"-server", serverHostname,
		"-server-only",
	)

	s.logger.InfoContext(ctx, pkggrpc.LogServerCertGenExecuting, "command", s.certGenPath, "args", cmd.Args[1:], "hostname", serverHostname)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogServerCertGenFailed, "error", err)
		s.logger.DebugContext(ctx, pkggrpc.LogCertGenerationStdout, "output", stdout.String())
		s.logger.DebugContext(ctx, pkggrpc.LogCertGenerationStderr, "output", stderr.String())
		return fmt.Errorf("%s: %s", pkggrpc.ErrTokenCertServerGenerationFailed, stderr.String())
	}

	s.logger.InfoContext(ctx, pkggrpc.LogServerCertGenSuccess)
	return nil
}

// GetServerCertificateStatus returns the status of server certificates.
func (s *Service) GetServerCertificateStatus(_ context.Context) (*grpcservices.CertificateStatus, error) {
	status := &grpcservices.CertificateStatus{}

	// Get the KC-Core certificates directory
	kcCorePath := s.certsDir

	// Check server certificate
	serverCertPath := filepath.Join(kcCorePath, "server.crt")
	if serverInfo := s.getCertificateInfo(serverCertPath); serverInfo != nil {
		status.HasServerCert = true
		status.ServerCertExpiry = &serverInfo.ExpiryDate
		status.Subject = serverInfo.Subject
		status.Issuer = serverInfo.Issuer
		status.NeedsRenewal = serverInfo.ExpiryDate.Before(time.Now().AddDate(0, 1, 0))
	}

	// Check CA certificate
	caCertPath := filepath.Join(kcCorePath, "ca.crt")
	if caInfo := s.getCertificateInfo(caCertPath); caInfo != nil {
		status.HasCACert = true
		status.CACertExpiry = &caInfo.ExpiryDate
		if caInfo.ExpiryDate.Before(time.Now().AddDate(0, 1, 0)) {
			status.NeedsRenewal = true
		}
	}

	return status, nil
}

// CleanupExpiredCertificates removes certificate directories older than 15 minutes.
func (s *Service) CleanupExpiredCertificates(ctx context.Context) {
	entries, err := os.ReadDir(s.tempDir)
	if err != nil {
		return
	}

	cutoffTime := time.Now().Add(-15 * time.Minute)

	for _, entry := range entries {
		if entry.IsDir() {
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}

			if info.ModTime().Before(cutoffTime) {
				if rmErr := os.RemoveAll(filepath.Join(s.tempDir, entry.Name())); rmErr != nil {
					s.logger.WarnContext(ctx, pkggrpc.LogCertExpiredDirRemoveFailed,
						"dir", entry.Name(), "error", rmErr)
				}
			}
		}
	}
}

// certInfo holds parsed certificate information
type certInfo struct {
	ExpiryDate time.Time
	Subject    string
	Issuer     string
}

// getCertificateInfo reads and parses certificate information
func (s *Service) getCertificateInfo(certPath string) *certInfo {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil
	}

	certPEM, err := os.ReadFile(certPath) // #nosec G304 - path from trusted KC-Core certs dir
	if err != nil {
		s.logger.ErrorContext(context.Background(), pkggrpc.LogCertFileReadFailed,
			"certPath", certPath, "error", err)
		return nil
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		s.logger.ErrorContext(context.Background(), pkggrpc.LogCertPEMBlockParseFailed, "certPath", certPath)
		return nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		s.logger.ErrorContext(context.Background(), pkggrpc.LogCertParseFailed,
			"certPath", certPath, "error", err)
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

	return &certInfo{
		ExpiryDate: cert.NotAfter,
		Subject:    subject,
		Issuer:     issuer,
	}
}

// GetStoredCertificate retrieves TLS certificates stored in base station record.
func (s *Service) GetStoredCertificate(ctx context.Context, tenantID int64, bsEui []byte, certType string) ([]byte, string, error) {
	if s.bsRepo == nil {
		return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenServiceNotConfigured)
	}

	// Validate cert type using canonical constants
	if certType != pkggrpc.CertTypeCA && certType != pkggrpc.CertTypeClient && certType != pkggrpc.CertTypeKey {
		s.logger.ErrorContext(ctx, pkggrpc.LogDownloadCertFailed, "cert_type", certType, "error", pkggrpc.ErrTokenCertTypeRequired)
		return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenCertTypeRequired)
	}

	bs, err := s.bsRepo.GetByEUI(ctx, tenantID, bsEui)
	if err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertBSNotFound, "bs_eui", hex.EncodeToString(bsEui), "error", err)
		return nil, "", fmt.Errorf("%s: %w", pkggrpc.ErrTokenBaseStationNotFound, err)
	}

	euiHex := hex.EncodeToString(bsEui)
	var data []byte
	var filename string

	switch certType {
	case pkggrpc.CertTypeCA:
		if bs.TLSCACertificate == nil || *bs.TLSCACertificate == "" {
			return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenCertNotFound)
		}
		data = []byte(*bs.TLSCACertificate)
		filename = fmt.Sprintf("basestation-%s-ca-certificate.crt", euiHex)
	case pkggrpc.CertTypeClient:
		if bs.TLSCertificate == nil || *bs.TLSCertificate == "" {
			return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenCertNotFound)
		}
		data = []byte(*bs.TLSCertificate)
		filename = fmt.Sprintf("basestation-%s-client-certificate.crt", euiHex)
	case pkggrpc.CertTypeKey:
		if bs.TLSKey == nil || *bs.TLSKey == "" {
			return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenCertNotFound)
		}
		if s.keyEncryptor == nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogDownloadCertFailed, "bs_eui", euiHex, "error", pkggrpc.ErrTokenServiceNotConfigured)
			return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenServiceNotConfigured)
		}
		decrypted, decryptErr := s.keyEncryptor.DecryptKey(*bs.TLSKey)
		if decryptErr != nil {
			s.logger.ErrorContext(ctx, pkggrpc.LogDownloadCertFailed, "bs_eui", euiHex, "error", decryptErr)
			return nil, "", fmt.Errorf("%s", pkggrpc.ErrTokenCertNotFound)
		}
		data = decrypted
		filename = fmt.Sprintf("basestation-%s-private-key.key", euiHex)
	}

	return data, filename, nil
}

// persistCertsToBaseStation stores generated certs in base station TLS fields.
func (s *Service) persistCertsToBaseStation(ctx context.Context, certDir string, bsEui []byte, tenantID int64, expiryTime time.Time) error {
	// First lookup the base station to get its ID (Update requires int64 ID, not EUI)
	bs, err := s.bsRepo.GetByEUI(ctx, tenantID, bsEui)
	if err != nil {
		return fmt.Errorf("%s", pkggrpc.ErrTokenBaseStationNotFound)
	}

	// Read certificate files with proper error handling
	caCertData, err := os.ReadFile(filepath.Join(certDir, "ca.crt")) // #nosec G304 - path from UUID-based certDir
	if err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertFileReadFailed, "file", "ca.crt", "error", err)
		return fmt.Errorf("%s", pkggrpc.ErrTokenCACertReadFailed)
	}
	clientCertData, err := os.ReadFile(filepath.Join(certDir, "client.crt")) // #nosec G304 - path from UUID-based certDir
	if err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertFileReadFailed, "file", "client.crt", "error", err)
		return fmt.Errorf("%s", pkggrpc.ErrTokenCertGenerationFailed)
	}
	clientKeyData, err := os.ReadFile(filepath.Join(certDir, "client.key")) // #nosec G304 - path from UUID-based certDir
	if err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertFileReadFailed, "file", "client.key", "error", err)
		return fmt.Errorf("%s", pkggrpc.ErrTokenCertGenerationFailed)
	}

	// Parse certificate and compute fingerprint for audit/identity
	block, _ := pem.Decode(clientCertData)
	if block == nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertPEMBlockParseFailed, "file", "client.crt")
		return fmt.Errorf("%s", pkggrpc.ErrTokenCertGenerationFailed)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertParseFailed, "file", "client.crt", "error", err)
		return fmt.Errorf("%s", pkggrpc.ErrTokenCertGenerationFailed)
	}
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(cert.Raw))

	// Encrypt private key (required for storage)
	encrypted, encErr := s.keyEncryptor.EncryptKey(clientKeyData)
	if encErr != nil {
		s.logger.ErrorContext(ctx, pkggrpc.LogCertGenerationFailed, "operation", "encrypt", "error", encErr)
		return fmt.Errorf("%s", pkggrpc.ErrTokenCertGenerationFailed)
	}

	updates := map[string]interface{}{
		"tls_ca_certificate":   string(caCertData),
		"tls_certificate":      string(clientCertData),
		"tls_key":              encrypted,
		"tls_cert_fingerprint": fingerprint,
		"tls_cert_expires_at":  expiryTime,
	}

	// Update uses (ctx, tenantID, id int64, updates) - use bs.ID from lookup
	return s.bsRepo.Update(ctx, tenantID, bs.ID, updates)
}

// Ensure Service implements grpcservices.CertificateService
var _ grpcservices.CertificateService = (*Service)(nil)
