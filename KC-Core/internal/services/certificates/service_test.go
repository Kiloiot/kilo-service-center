package certificates

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	pkggrpc "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...interface{})                           {}
func (m *mockLogger) Info(_ string, _ ...interface{})                            {}
func (m *mockLogger) Warn(_ string, _ ...interface{})                            {}
func (m *mockLogger) Error(_ string, _ ...interface{})                           {}
func (m *mockLogger) Fatal(_ string, _ ...interface{})                           {}
func (m *mockLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) WithField(_ string, _ interface{}) logger.Logger            { return m }
func (m *mockLogger) WithFields(_ map[string]interface{}) logger.Logger          { return m }

type mockKeyEncryptor struct {
	encryptErr error
}

func (m *mockKeyEncryptor) EncryptKey(key []byte) (string, error) {
	if m.encryptErr != nil {
		return "", m.encryptErr
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func (m *mockKeyEncryptor) DecryptKey(encrypted string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encrypted)
}

func TestDownloadCertificateByID_ShortCertID(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		Certificates: config.CertificateConfig{
			TempDir: tempDir,
		},
	}

	svc, err := New(cfg, &mockLogger{}, &mockBaseStationRepo{}, &mockKeyEncryptor{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	shortCertID := "abc123"
	certDir := filepath.Join(tempDir, shortCertID)

	if err := os.MkdirAll(certDir, 0750); err != nil {
		t.Fatalf("failed to create cert dir: %v", err)
	}

	infoData, _ := json.Marshal(struct {
		BsEui string `json:"bsEui"`
	}{BsEui: ""})
	if err := os.WriteFile(filepath.Join(certDir, "info.json"), infoData, 0600); err != nil {
		t.Fatalf("failed to write info.json: %v", err)
	}

	if err := os.WriteFile(filepath.Join(certDir, "client.crt"), []byte("dummy cert"), 0600); err != nil {
		t.Fatalf("failed to write client.crt: %v", err)
	}

	ctx := testutil.TestContext()
	_, filename, err := svc.DownloadCertificateByID(ctx, "client", shortCertID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFilename := "basestation-abc123-client-certificate.crt"
	if filename != expectedFilename {
		t.Fatalf("filename = %q, want %q", filename, expectedFilename)
	}
}

// newGenerateTestService builds a Service whose certgen binary is absent so that
// GenerateCertificate proceeds past EUI validation but fails at the generator check.
func newGenerateTestService(t *testing.T) *Service {
	t.Helper()
	tmpDir := t.TempDir()
	for _, name := range []string{"ca.crt", "ca.key"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("staged"), 0600); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	return &Service{
		config:             &config.Config{},
		logger:             &mockLogger{},
		certGenPath:        filepath.Join(tmpDir, "certgen-missing"),
		certsDir:           tmpDir,
		tempDir:            filepath.Join(tmpDir, "temp"),
		serverValidityDays: config.DefaultCertificatesServerValidityDays,
		protocolConfig:     &config.ProtocolConfig{},
		bsRepo:             &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: 42}},
		keyEncryptor:       &mockKeyEncryptor{},
		certGen:            execCertGen,
	}
}

func TestGenerateCertificate_RejectsMalformedEUI(t *testing.T) {
	svc := newGenerateTestService(t)
	ctx := testutil.TestContext()

	_, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "not-a-valid-eui",
		ValidityDays: 365,
	})
	if err == nil {
		t.Fatal("expected error for malformed EUI, got nil")
	}
	if !strings.Contains(err.Error(), pkggrpc.ErrTokenInvalidBasestationEUIFormat) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), pkggrpc.ErrTokenInvalidBasestationEUIFormat)
	}
}

func TestGenerateCertificate_AcceptsDashedHighBitEUI(t *testing.T) {
	svc := newGenerateTestService(t)
	ctx := testutil.TestContext()

	_, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "CA-FE-CA-FE-CA-FE-CA-FE",
		ValidityDays: 365,
		TenantID:     42,
	})
	if err == nil {
		t.Fatal("expected generator-not-found error, got nil")
	}
	// EUI validation must succeed; the failure comes from the absent certgen binary.
	if strings.Contains(err.Error(), pkggrpc.ErrTokenInvalidBasestationEUIFormat) {
		t.Errorf("dashed high-bit EUI was rejected as invalid: %v", err)
	}
	if !strings.Contains(err.Error(), pkggrpc.ErrTokenCertGeneratorNotFound) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), pkggrpc.ErrTokenCertGeneratorNotFound)
	}
}

func TestGenerateCertificate_AcceptsPlainHexEUI(t *testing.T) {
	svc := newGenerateTestService(t)
	ctx := testutil.TestContext()

	_, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "cafecafecafecafe",
		ValidityDays: 365,
		TenantID:     42,
	})
	if err == nil {
		t.Fatal("expected generator-not-found error, got nil")
	}
	if strings.Contains(err.Error(), pkggrpc.ErrTokenInvalidBasestationEUIFormat) {
		t.Errorf("plain 16-hex EUI was rejected as invalid: %v", err)
	}
}

func TestGetStoredCertificate_InvalidCertType(t *testing.T) {
	svc := &Service{
		logger: &mockLogger{},
		bsRepo: &mockBaseStationRepo{
			bs: &models.BaseStation{
				ID: 1,
			},
		},
	}

	ctx := testutil.TestContext()
	_, _, err := svc.GetStoredCertificate(ctx, 1, []byte{0x01, 0x02}, "bad")
	if err == nil {
		t.Fatal("expected error")
	}
	if tok, ok := pkggrpc.TokenOf(err); !ok || tok != pkggrpc.ErrTokenCertTypeRequired {
		t.Fatalf("error = %q, want typed token %q", err.Error(), pkggrpc.ErrTokenCertTypeRequired)
	}
}

func TestGetStoredCertificate_KeyEncryptorRequired(t *testing.T) {
	encrypted := "encrypted-key"
	svc := &Service{
		logger: &mockLogger{},
		bsRepo: &mockBaseStationRepo{
			bs: &models.BaseStation{
				ID:     1,
				TLSKey: &encrypted,
			},
		},
	}

	ctx := testutil.TestContext()
	_, _, err := svc.GetStoredCertificate(ctx, 1, []byte{0x01, 0x02}, pkggrpc.CertTypeKey)
	if err == nil {
		t.Fatal("expected error")
	}
	if tok, ok := pkggrpc.TokenOf(err); !ok || tok != pkggrpc.ErrTokenServiceNotConfigured {
		t.Fatalf("error = %q, want typed token %q", err.Error(), pkggrpc.ErrTokenServiceNotConfigured)
	}
}

func TestPersistCertsToBaseStation_MissingCACert(t *testing.T) {
	tempDir := t.TempDir()
	svc := &Service{
		logger: &mockLogger{},
		bsRepo: &mockBaseStationRepo{
			bs: &models.BaseStation{
				ID: 1,
			},
		},
	}

	ctx := testutil.TestContext()
	err := svc.persistCertsToBaseStation(ctx, tempDir, []byte{0x01, 0x02}, 1, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if tok, ok := pkggrpc.TokenOf(err); !ok || tok != pkggrpc.ErrTokenCACertReadFailed {
		t.Fatalf("error = %q, want typed token %q", err.Error(), pkggrpc.ErrTokenCACertReadFailed)
	}
}

type mockBaseStationRepo struct {
	bs         *models.BaseStation
	getErr     error
	updateErr  error
	updateArgs map[string]interface{}
}

func (m *mockBaseStationRepo) Create(_ context.Context, _ *models.BaseStation) error {
	return errors.New("not implemented")
}

func (m *mockBaseStationRepo) GetByID(_ context.Context, _ int64, _ int64) (*models.BaseStation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBaseStationRepo) GetByEUI(_ context.Context, _ int64, _ []byte) (*models.BaseStation, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.bs, nil
}

func (m *mockBaseStationRepo) Update(_ context.Context, _ int64, _ int64, updates map[string]interface{}) error {
	m.updateArgs = updates
	return m.updateErr
}

func (m *mockBaseStationRepo) Delete(_ context.Context, _ int64, _ int64) error {
	return errors.New("not implemented")
}

func (m *mockBaseStationRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (m *mockBaseStationRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	return errors.New("not implemented")
}

func (m *mockBaseStationRepo) UpdateSessionInfo(_ context.Context, _ int64, _ []byte, _ string) error {
	return errors.New("not implemented")
}

func (m *mockBaseStationRepo) GetStatistics(_ context.Context, _ int64) (*interfaces.BaseStationStatistics, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBaseStationRepo) UpdateEUI(_ context.Context, _ int64, _ []byte, _ []byte) (*models.BaseStation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBaseStationRepo) GetPropagationState(_ context.Context, _ int64) (*models.BaseStationPropagationState, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBaseStationRepo) UpsertPropagationState(_ context.Context, _ *models.BaseStationPropagationState) error {
	return errors.New("not implemented")
}

func (m *mockBaseStationRepo) UpdatePropagationStatus(_ context.Context, _ int64, _ string, _ *string) error {
	return errors.New("not implemented")
}

func (m *mockBaseStationRepo) IncrementRetryCount(_ context.Context, _ int64, _ time.Time) error {
	return errors.New("not implemented")
}

func (m *mockBaseStationRepo) GetByEUIGlobal(_ context.Context, _ []byte) (*models.BaseStation, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.bs, nil
}

func (m *mockBaseStationRepo) ListAllLocations(_ context.Context) ([]*models.BaseStation, error) {
	return nil, nil
}

func TestRenewServerCertificates_FailsWhenCAFilesMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Place a dummy server.crt so the "no certs to renew" check passes
	if err := os.WriteFile(filepath.Join(tmpDir, "server.crt"), []byte("dummy"), 0600); err != nil {
		t.Fatalf("failed to write dummy server.crt: %v", err)
	}

	svc := &Service{
		logger:             &mockLogger{},
		certsDir:           tmpDir,
		certGenPath:        "/nonexistent/certgen",
		serverValidityDays: 365,
		protocolConfig:     &config.ProtocolConfig{},
	}

	ctx := testutil.TestContext()
	err := svc.RenewServerCertificates(ctx)
	if err == nil {
		t.Fatal("expected error when CA files are missing, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, pkggrpc.ErrTokenCACertReadFailed) && !strings.Contains(errStr, pkggrpc.ErrTokenCAKeyReadFailed) {
		t.Errorf("expected error to contain %s or %s, got: %s",
			pkggrpc.ErrTokenCACertReadFailed, pkggrpc.ErrTokenCAKeyReadFailed, errStr)
	}
	// Error should contain the catalog-resolved message
	caCertMsg := pkggrpc.ResolveErrorMessage(pkggrpc.ErrTokenCACertReadFailed)
	caKeyMsg := pkggrpc.ResolveErrorMessage(pkggrpc.ErrTokenCAKeyReadFailed)
	if !strings.Contains(errStr, caCertMsg) && !strings.Contains(errStr, caKeyMsg) {
		t.Errorf("expected error to contain catalog message %q or %q, got: %s", caCertMsg, caKeyMsg, errStr)
	}
}

func TestRenewServerCertificates_FailsWhenNoServerCert(t *testing.T) {
	tmpDir := t.TempDir()

	svc := &Service{
		logger:             &mockLogger{},
		certsDir:           tmpDir,
		certGenPath:        "/nonexistent/certgen",
		serverValidityDays: 365,
		protocolConfig:     &config.ProtocolConfig{},
	}

	ctx := testutil.TestContext()
	err := svc.RenewServerCertificates(ctx)
	if err == nil {
		t.Fatal("expected error when server.crt is missing, got nil")
	}

	if !strings.Contains(err.Error(), pkggrpc.ErrTokenNoCertsToRenew) {
		t.Errorf("expected error to contain %s, got: %s", pkggrpc.ErrTokenNoCertsToRenew, err.Error())
	}
}

func TestNew_ServerValidityDaysFromConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Certificates: config.CertificateConfig{
			CertGenPath:        filepath.Join(tmpDir, "certgen"),
			CertsDir:           tmpDir,
			TempDir:            filepath.Join(tmpDir, "temp"),
			ServerValidityDays: 730,
		},
	}

	svc, err := New(cfg, &mockLogger{}, &mockBaseStationRepo{}, &mockKeyEncryptor{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if svc.serverValidityDays != 730 {
		t.Errorf("expected serverValidityDays=730, got %d", svc.serverValidityDays)
	}
}

func TestDeriveServerHostname_WildcardExternalURL(t *testing.T) {
	svc := &Service{
		protocolConfig: &config.ProtocolConfig{
			BSCIExternalURL: "tls://0.0.0.0:5000",
			BSCIHost:        "",
		},
	}

	hostname := svc.deriveServerHostname()
	if hostname == "0.0.0.0" {
		t.Errorf("deriveServerHostname should not return wildcard address, got %q", hostname)
	}
	if hostname != config.DefaultCertificatesHostname {
		t.Errorf("expected %q, got %q", config.DefaultCertificatesHostname, hostname)
	}
}

func TestDeriveServerHostname_ValidExternalURL(t *testing.T) {
	svc := &Service{
		protocolConfig: &config.ProtocolConfig{
			BSCIExternalURL: "tls://bssci.example.com:5000",
		},
	}

	hostname := svc.deriveServerHostname()
	if hostname != "bssci.example.com" {
		t.Errorf("expected 'bssci.example.com', got %q", hostname)
	}
}

func TestDeriveServerHostname_FallbackToBSCIHost(t *testing.T) {
	svc := &Service{
		protocolConfig: &config.ProtocolConfig{
			BSCIHost: "192.168.1.10",
		},
	}

	hostname := svc.deriveServerHostname()
	if hostname != "192.168.1.10" {
		t.Errorf("expected '192.168.1.10', got %q", hostname)
	}
}

func TestDeriveServerHostname_WildcardBSCIHost(t *testing.T) {
	svc := &Service{
		protocolConfig: &config.ProtocolConfig{
			BSCIHost: "0.0.0.0",
		},
	}

	hostname := svc.deriveServerHostname()
	if hostname != config.DefaultCertificatesHostname {
		t.Errorf("expected %q, got %q", config.DefaultCertificatesHostname, hostname)
	}
}

func TestNew_ServerValidityDaysFallsBackToDefault(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Certificates: config.CertificateConfig{
			CertGenPath:        filepath.Join(tmpDir, "certgen"),
			CertsDir:           tmpDir,
			TempDir:            filepath.Join(tmpDir, "temp"),
			ServerValidityDays: 0,
		},
	}

	svc, err := New(cfg, &mockLogger{}, &mockBaseStationRepo{}, &mockKeyEncryptor{}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if svc.serverValidityDays != config.DefaultCertificatesServerValidityDays {
		t.Errorf("expected serverValidityDays=%d (default), got %d",
			config.DefaultCertificatesServerValidityDays, svc.serverValidityDays)
	}
}

// newGenerateTestServiceWithRepo mirrors newGenerateTestService but injects a
// base-station repository so the tenant-ownership check runs.
func newGenerateTestServiceWithRepo(t *testing.T, repo *mockBaseStationRepo) *Service {
	t.Helper()
	tmpDir := t.TempDir()
	for _, name := range []string{"ca.crt", "ca.key"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("staged"), 0600); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	return &Service{
		config:             &config.Config{},
		logger:             &mockLogger{},
		certGenPath:        filepath.Join(tmpDir, "certgen-missing"),
		certsDir:           tmpDir,
		tempDir:            filepath.Join(tmpDir, "temp"),
		serverValidityDays: config.DefaultCertificatesServerValidityDays,
		protocolConfig:     &config.ProtocolConfig{},
		bsRepo:             repo,
		keyEncryptor:       &mockKeyEncryptor{},
		certGen:            execCertGen,
	}
}

// TestGenerateCertificate_CrossTenantDenied verifies a certificate cannot be
// minted for an EUI the requesting tenant does not own: the ownership lookup
// fails, so issuance is rejected before any certgen work.
func TestGenerateCertificate_CrossTenantDenied(t *testing.T) {
	svc := newGenerateTestServiceWithRepo(t, &mockBaseStationRepo{getErr: errors.New("not found for tenant")})
	ctx := testutil.TestContext()

	_, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "cafecafecafecafe",
		ValidityDays: 365,
		TenantID:     42,
	})
	if err == nil {
		t.Fatal("expected cross-tenant issuance to be denied")
	}
	if !strings.Contains(err.Error(), pkggrpc.ErrTokenBaseStationNotFound) {
		t.Errorf("error = %q, want base-station-not-found ownership rejection", err.Error())
	}
	// Must fail BEFORE reaching the (missing) certgen binary.
	if strings.Contains(err.Error(), pkggrpc.ErrTokenCertGeneratorNotFound) {
		t.Errorf("ownership check must run before generation, got %q", err.Error())
	}
}

// TestGenerateCertificate_OwnedProceedsToGeneration verifies that when the
// requesting tenant owns the EUI, issuance proceeds past the ownership check
// (and here fails only at the intentionally-absent certgen binary).
func TestGenerateCertificate_OwnedProceedsToGeneration(t *testing.T) {
	svc := newGenerateTestServiceWithRepo(t, &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: 42}})
	ctx := testutil.TestContext()

	_, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "cafecafecafecafe",
		ValidityDays: 365,
		TenantID:     42,
	})
	if err == nil {
		t.Fatal("expected certgen-missing failure after a passing ownership check")
	}
	if strings.Contains(err.Error(), pkggrpc.ErrTokenBaseStationNotFound) {
		t.Errorf("ownership check must pass for an owned EUI, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), pkggrpc.ErrTokenCertGeneratorNotFound) {
		t.Errorf("error = %q, want it to reach the certgen step", err.Error())
	}
}

func (m *mockBaseStationRepo) UpdateTLSFingerprintIfBlank(_ context.Context, _, _ int64, _ string) (bool, error) {
	return true, nil
}

// fakeCertGen is a CertGenRunner producing real PEM material without the
// external certgen binary.
func fakeCertGen(t *testing.T) CertGenRunner {
	t.Helper()
	return func(_ context.Context, _ string, args ...string) (string, string, error) {
		var dir string
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-dir" {
				dir = args[i+1]
			}
		}
		if dir == "" {
			return "", "no -dir argument", errors.New("missing -dir")
		}

		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return "", err.Error(), err
		}
		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: "CA-FE-CA-FE-CA-FE-CA-FE"},
			NotAfter:     time.Now().Add(24 * time.Hour),
		}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			return "", err.Error(), err
		}
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyDER, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return "", err.Error(), err
		}
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

		for name, data := range map[string][]byte{
			"ca.crt":     certPEM,
			"client.crt": certPEM,
			"client.key": keyPEM,
		} {
			if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
				return "", err.Error(), err
			}
		}
		return "generated", "", nil
	}
}

// newIssuanceTestService builds a fully wired service whose generator emits
// real PEM, with CA material staged in certsDir for the copy step.
func newIssuanceTestService(t *testing.T, repo *mockBaseStationRepo) *Service {
	t.Helper()
	tmpDir := t.TempDir()
	// Stage CA files the issuance path copies into the working dir
	for _, name := range []string{"ca.crt", "ca.key"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("staged"), 0600); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	return &Service{
		config:             &config.Config{},
		logger:             &mockLogger{},
		certGenPath:        filepath.Join(tmpDir, "certgen-unused"),
		certsDir:           tmpDir,
		tempDir:            filepath.Join(tmpDir, "temp"),
		serverValidityDays: config.DefaultCertificatesServerValidityDays,
		protocolConfig:     &config.ProtocolConfig{},
		bsRepo:             repo,
		keyEncryptor:       &mockKeyEncryptor{},
		certGen:            fakeCertGen(t),
	}
}

// TestGenerateCertificate_PersistsMatchingFingerprint: real issuance persists
// the certificate with a fingerprint equal to the canonical SHA-256 of the
// generated certificate.
func TestGenerateCertificate_PersistsMatchingFingerprint(t *testing.T) {
	repo := &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: 42}}
	svc := newIssuanceTestService(t, repo)
	ctx := testutil.TestContext()

	resp, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "cafecafecafecafe",
		ValidityDays: 365,
		TenantID:     42,
	})
	if err != nil {
		t.Fatalf("issuance failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected a certificate response")
	}

	stored, ok := repo.updateArgs["tls_cert_fingerprint"].(string)
	if !ok || stored == "" {
		t.Fatalf("fingerprint not persisted: %v", repo.updateArgs)
	}
	certPEM, ok := repo.updateArgs["tls_certificate"].(string)
	if !ok {
		t.Fatal("certificate not persisted")
	}
	derived, err := crypto.CertFingerprintFromPEM([]byte(certPEM))
	if err != nil {
		t.Fatalf("derive fingerprint: %v", err)
	}
	if stored != derived {
		t.Fatalf("stored fingerprint %q != derived %q", stored, derived)
	}
	if stored != strings.ToLower(stored) {
		t.Fatalf("fingerprint must be lowercase hex, got %q", stored)
	}
}

// TestGenerateCertificate_PersistenceFailureReturnsNoCert: a persistence
// failure surfaces as the typed persistence token and no certificate is
// returned (the station must never hold a certificate the service center did
// not record).
func TestGenerateCertificate_PersistenceFailureReturnsNoCert(t *testing.T) {
	repo := &mockBaseStationRepo{
		bs:        &models.BaseStation{ID: 1, TenantID: 42},
		updateErr: errors.New("db down"),
	}
	svc := newIssuanceTestService(t, repo)
	ctx := testutil.TestContext()

	resp, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "cafecafecafecafe",
		ValidityDays: 365,
		TenantID:     42,
	})
	if err == nil {
		t.Fatal("expected persistence failure")
	}
	if resp != nil {
		t.Fatal("no certificate may be returned when persistence failed")
	}
	if tok, ok := pkggrpc.TokenOf(err); !ok || tok != pkggrpc.ErrTokenCertPersistenceFailed {
		t.Fatalf("error = %q, want typed token %q", err.Error(), pkggrpc.ErrTokenCertPersistenceFailed)
	}
}

// TestGenerateCertificate_TenantZeroFailsClosed: issuance without a tenant is
// rejected before any generation work.
func TestGenerateCertificate_TenantZeroFailsClosed(t *testing.T) {
	repo := &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: 42}}
	svc := newIssuanceTestService(t, repo)
	ctx := testutil.TestContext()

	_, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "cafecafecafecafe",
		ValidityDays: 365,
	})
	if err == nil {
		t.Fatal("expected tenant-zero rejection")
	}
	if tok, ok := pkggrpc.TokenOf(err); !ok || tok != pkggrpc.ErrTokenMissingTenantCtx {
		t.Fatalf("error = %q, want typed token %q", err.Error(), pkggrpc.ErrTokenMissingTenantCtx)
	}
}
