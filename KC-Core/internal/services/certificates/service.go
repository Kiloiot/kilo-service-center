// Package certificates provides certificate service implementation for gRPC layer.
package certificates

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Certificate generator flags and canonical certificate file names.
const (
	certGenFlagDir        = "-dir"
	certGenFlagDays       = "-days"
	certGenFlagClient     = "-client"
	certGenFlagClientOnly = "-client-only"
	certGenFlagServer     = "-server"
	certGenFlagServerOnly = "-server-only"
	certGenFlagAltNames   = "-san"
	// altNameSeparator joins the names of the -san flag.
	altNameSeparator = ","

	caCertFileName     = "ca.crt"
	caKeyFileName      = "ca.key"
	serverCertFileName = "server.crt"
	clientCertFileName = "client.crt"
	clientKeyFileName  = "client.key"
	certInfoFileName   = "info.json"

	// Download filenames handed to the browser.
	downloadNameCACert        = "kilocenter-ca-certificate.crt"
	downloadNameFmtClientCert = "basestation-%s-client-certificate.crt"
	downloadNameFmtPrivateKey = "basestation-%s-private-key.key"
	downloadNameFmtBSCACert   = "basestation-%s-ca-certificate.crt"

	// Directory and secret-file permissions for generated material.
	certDirPerm  = 0o750
	certFilePerm = 0o600

	// maxValidityDays caps requested certificate validity (three years).
	maxValidityDays = 1095
)

// KeyEncryptor is the narrow key-encryption contract the certificate service
// consumes (implemented by the keymaterial cipher adapter over pkg/keycrypto).
type KeyEncryptor interface {
	EncryptKey(key []byte) (string, error)
	DecryptKey(encrypted string) ([]byte, error)
}

// CertGenRunner executes the certificate generator with the given arguments.
// Injected so tests can produce real PEM material without the external
// certgen binary; ExecCertGen shells out to the configured path.
type CertGenRunner func(ctx context.Context, certGenPath string, args ...string) (stdout, stderr string, err error)

// ExecCertGen is the production CertGenRunner: it runs the certgen binary,
// reporting a typed generator-not-found error when the binary is absent.
func ExecCertGen(ctx context.Context, certGenPath string, args ...string) (string, string, error) {
	if _, err := os.Stat(certGenPath); err != nil {
		return "", "", fmt.Errorf("%w: %w", ErrGeneratorNotFound, errors.New(certGenPath))
	}
	cmd := exec.CommandContext(ctx, certGenPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// BaseStationStore reads a base station by EUI, persists certificate fields
// and hands out a stored private key once, reporting a station without one as
// storage.ErrNotFound.
type BaseStationStore interface {
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)
	Update(ctx context.Context, tenantID, id int64, updates map[string]interface{}) error
	TakeTLSKey(ctx context.Context, tenantID int64, eui []byte, open func(sealed string) error) error
}

// DisclosureRecorder records the audit event a private key cannot leave the
// service center without, returning the write failure so the key is kept.
type DisclosureRecorder interface {
	RecordRequired(ctx context.Context, ev audit.Event) error
}

// settings are the validated certificate settings the service works with.
type settings struct {
	certGenPath        string
	certsDir           string
	tempDir            string
	serverValidityDays int
	serverNames        []string
	protocol           *config.ProtocolConfig
}

// Service implements grpcservices.CertificateService.
type Service struct {
	clock        clock.Clock
	logger       logger.Logger
	settings     settings
	bsRepo       BaseStationStore
	keyEncryptor KeyEncryptor
	certGen      CertGenRunner
	disclosures  DisclosureRecorder
}

// New creates a certificate service over the validated certificate settings
// of cfg. Every collaborator is required: ownership verification, durable
// persistence and the record of every private key download are part of the
// service, so an incompletely wired one must never mint or hand out a
// certificate. ctx scopes the startup logging.
func New(ctx context.Context, cfg *config.Config, log logger.Logger, bsRepo BaseStationStore, keyEncryptor KeyEncryptor, certGen CertGenRunner, clk clock.Clock, disclosures DisclosureRecorder) (*Service, error) {
	if cfg == nil || log == nil || bsRepo == nil || keyEncryptor == nil || certGen == nil || clk == nil || disclosures == nil {
		return nil, ErrServiceNotConfigured
	}
	s := &Service{
		clock:  clk,
		logger: log,
		settings: settings{
			certGenPath:        cfg.Certificates.CertGenPath,
			certsDir:           cfg.Certificates.CertsDir,
			tempDir:            cfg.Certificates.TempDir,
			serverValidityDays: cfg.Certificates.ServerValidityDays,
			serverNames:        cfg.Certificates.ServerNames,
			protocol:           &cfg.Protocol,
		},
		bsRepo:       bsRepo,
		keyEncryptor: keyEncryptor,
		certGen:      certGen,
		disclosures:  disclosures,
	}
	s.reportStartupPaths(ctx)
	return s, nil
}

// reportStartupPaths logs a missing certgen binary or certificate directory
// at start-up, which helps debug relative paths and working directory
// mismatches, and creates the bundle directory.
func (s *Service) reportStartupPaths(ctx context.Context) {
	if _, err := os.Stat(s.settings.certGenPath); err != nil {
		s.logger.WarnContext(ctx, LogCertConfigMissingPath,
			logger.FieldConfiguredPath, s.settings.certGenPath,
			logger.FieldWorkingDirectory, workingDirectory(),
			logger.FieldHint, LogCertConfigMissingPathHint)
	} else {
		s.logger.InfoContext(ctx, LogCertGeneratorPathInfo, logger.FieldPath, s.settings.certGenPath)
	}

	if _, err := os.Stat(s.settings.certsDir); err != nil {
		s.logger.WarnContext(ctx, LogCertDirectoryNotFound,
			logger.FieldPath, s.settings.certsDir,
			logger.FieldHint, LogCertDirectoryNotFoundHint)
	}

	if err := os.MkdirAll(s.settings.tempDir, certDirPerm); err != nil {
		s.logger.ErrorContext(ctx, LogCertTempDirCreateFailed, logger.FieldError, err)
	}
}

// caCertPath locates the service center CA certificate, the one station
// certificates are issued with.
func (s *Service) caCertPath() string {
	return filepath.Join(s.settings.certsDir, caCertFileName)
}

// Ensure Service implements grpcservices.CertificateService
var _ grpcservices.CertificateService = (*Service)(nil)

// workingDirectory names the directory relative paths resolve against, or
// the reason it cannot be determined, for a start-up diagnostic.
func workingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	return wd
}
