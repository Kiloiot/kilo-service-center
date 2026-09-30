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
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// Sentinels and fixtures shared by the certificate service test doubles.
var (
	errNotImplemented  = errors.New("not implemented")
	errTenantMismatch  = errors.New("not found for tenant")
	errMissingDirFlag  = errors.New("missing -dir")
	errRepoUnavailable = errors.New("db down")
)

const (
	testMissingCertGenPath = "/nonexistent/certgen"
	testCertValidity       = 24 * time.Hour
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
	decryptErr error
}

func (m *mockKeyEncryptor) EncryptKey(key []byte) (string, error) {
	if m.encryptErr != nil {
		return "", m.encryptErr
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func (m *mockKeyEncryptor) DecryptKey(encrypted string) ([]byte, error) {
	if m.decryptErr != nil {
		return nil, m.decryptErr
	}
	return base64.StdEncoding.DecodeString(encrypted)
}

const (
	testBundleKey           = "key"
	testConcurrentDownloads = 16
)

var errTestDecrypt = errors.New("wrong master key")

const (
	testOwnerTenant      = int64(42)
	testForeignTenant    = int64(99)
	testBundleEUI        = "CA-FE-CA-FE-CA-FE-CA-FE"
	testBundleEUICompact = "CAFECAFECAFECAFE"
	testDirPerm          = 0o750
	testFilePerm         = 0o600
)

var testBundleTime = time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)

// tenantScopedRepo owns its base station for testOwnerTenant only.
func tenantScopedRepo() *mockBaseStationRepo {
	return &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: testOwnerTenant}, ownerTenant: testOwnerTenant}
}

// writeBundle stages a generated bundle for testBundleEUI and returns its id.
func writeBundle(t *testing.T, tempDir string, modTime time.Time) string {
	t.Helper()
	id := uuid.New().String()
	dir := filepath.Join(tempDir, id)
	if err := os.MkdirAll(dir, testDirPerm); err != nil {
		t.Fatalf("stage bundle: %v", err)
	}
	info, err := json.Marshal(map[string]string{"bsEui": testBundleEUI})
	if err != nil {
		t.Fatalf("marshal info: %v", err)
	}
	for name, data := range map[string][]byte{certInfoFileName: info, clientCertFileName: []byte("cert"), clientKeyFileName: []byte(testBundleKey), caCertFileName: []byte("ca")} {
		if err := os.WriteFile(filepath.Join(dir, name), data, testFilePerm); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	if err := os.Chtimes(dir, modTime, modTime); err != nil {
		t.Fatalf("age bundle: %v", err)
	}
	return id
}

func newBundleService(t *testing.T, clk clock.Clock) (*Service, string) {
	t.Helper()
	tempDir := t.TempDir()
	cfg := &config.Config{Certificates: config.CertificateConfig{TempDir: tempDir}}
	svc, err := New(testutil.TestContext(), cfg, &mockLogger{}, tenantScopedRepo(), &mockKeyEncryptor{}, ExecCertGen, clk, &captureDisclosures{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return svc, tempDir
}

func TestDownloadCertificateByID_OwnerGetsTheBundleFile(t *testing.T) {
	svc, tempDir := newBundleService(t, clock.SystemClock{})
	id := writeBundle(t, tempDir, time.Now())

	data, filename, err := svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeClient, id)
	if err != nil {
		t.Fatalf("owner download: %v", err)
	}
	if string(data) != "cert" {
		t.Fatalf("content = %q", data)
	}
	if want := fmt.Sprintf(downloadNameFmtClientCert, testBundleEUICompact); filename != want {
		t.Fatalf("filename = %q, want %q", filename, want)
	}
}

func TestDownloadCertificateByID_ForeignTenantGetsNotFound(t *testing.T) {
	svc, tempDir := newBundleService(t, clock.SystemClock{})
	id := writeBundle(t, tempDir, time.Now())

	for _, certType := range []string{CertTypeCA, CertTypeClient, CertTypeKey} {
		_, _, err := svc.DownloadCertificateByID(testutil.TestContext(), testForeignTenant, certType, id)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("%s for another tenant: err = %v, want ErrNotFound", certType, err)
		}
	}
}

func TestDownloadCertificateByID_RejectsIDsOutsideTheBundleStore(t *testing.T) {
	svc, tempDir := newBundleService(t, clock.SystemClock{})
	writeBundle(t, tempDir, time.Now())

	for _, id := range []string{"abc123", "../" + filepath.Base(tempDir), uuid.New().String()} {
		_, _, err := svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeCA, id)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("id %q: err = %v, want ErrNotFound", id, err)
		}
	}
}

func TestCleanupExpiredCertificates_RemovesOnlyBundlesPastTheDownloadWindow(t *testing.T) {
	svc, tempDir := newBundleService(t, clock.SystemClock{})
	expired := writeBundle(t, tempDir, testBundleTime.Add(-certDownloadWindow-time.Minute))
	fresh := writeBundle(t, tempDir, testBundleTime.Add(-certDownloadWindow+time.Minute))

	svc.CleanupExpiredCertificates(testutil.TestContext(), testBundleTime)

	if _, err := os.Stat(filepath.Join(tempDir, expired)); !os.IsNotExist(err) {
		t.Fatalf("expired bundle still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, fresh)); err != nil {
		t.Fatalf("fresh bundle removed: %v", err)
	}
}

func TestNew_RefusesMissingCollaborators(t *testing.T) {
	cfg := &config.Config{Certificates: config.CertificateConfig{TempDir: t.TempDir()}}
	ctx := testutil.TestContext()
	for name, build := range map[string]func() (*Service, error){
		"config": func() (*Service, error) {
			return New(ctx, nil, &mockLogger{}, tenantScopedRepo(), &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
		},
		"logger": func() (*Service, error) {
			return New(ctx, cfg, nil, tenantScopedRepo(), &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
		},
		"station store": func() (*Service, error) {
			return New(ctx, cfg, &mockLogger{}, nil, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
		},
		"key encryptor": func() (*Service, error) {
			return New(ctx, cfg, &mockLogger{}, tenantScopedRepo(), nil, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
		},
		"generator": func() (*Service, error) {
			return New(ctx, cfg, &mockLogger{}, tenantScopedRepo(), &mockKeyEncryptor{}, nil, clock.SystemClock{}, &captureDisclosures{})
		},
		"clock": func() (*Service, error) {
			return New(ctx, cfg, &mockLogger{}, tenantScopedRepo(), &mockKeyEncryptor{}, ExecCertGen, nil, &captureDisclosures{})
		},
		"disclosure recorder": func() (*Service, error) {
			return New(ctx, cfg, &mockLogger{}, tenantScopedRepo(), &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, nil)
		},
	} {
		if _, err := build(); !errors.Is(err, ErrServiceNotConfigured) {
			t.Fatalf("missing %s: err = %v, want ErrServiceNotConfigured", name, err)
		}
	}
}

// newGenerateTestService builds a Service whose certgen binary is absent so that
// GenerateCertificate proceeds past EUI validation but fails at the generator check.
func newGenerateTestService(t *testing.T) *Service {
	t.Helper()
	tmpDir := t.TempDir()
	for _, name := range []string{"ca.crt", "ca.key"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("staged"), 0o600); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	return &Service{
		clock:        clock.SystemClock{},
		logger:       &mockLogger{},
		bsRepo:       &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: 42}},
		keyEncryptor: &mockKeyEncryptor{},
		certGen:      ExecCertGen,
		settings: settings{
			certGenPath:        filepath.Join(tmpDir, "certgen-missing"),
			certsDir:           tmpDir,
			tempDir:            filepath.Join(tmpDir, "temp"),
			serverValidityDays: config.DefaultCertificatesServerValidityDays,
			protocol:           &config.ProtocolConfig{},
		},
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
	if !errors.Is(err, ErrInvalidBaseStationEUI) {
		t.Errorf("error = %q, want ErrInvalidBaseStationEUI", err.Error())
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
	if errors.Is(err, ErrInvalidBaseStationEUI) {
		t.Errorf("dashed high-bit EUI was rejected as invalid: %v", err)
	}
	if !errors.Is(err, ErrGeneratorNotFound) {
		t.Errorf("error = %q, want ErrGeneratorNotFound", err.Error())
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
	if errors.Is(err, ErrInvalidBaseStationEUI) {
		t.Errorf("plain 16-hex EUI was rejected as invalid: %v", err)
	}
}

func TestGetStoredCertificate_InvalidCertType(t *testing.T) {
	svc := &Service{
		clock:  clock.SystemClock{},
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
	if !errors.Is(err, ErrTypeRequired) {
		t.Fatalf("error = %q, want sentinel %q", err.Error(), ErrTypeRequired)
	}
}

func TestPersistCertsToBaseStation_MissingCACert(t *testing.T) {
	tempDir := t.TempDir()
	svc := &Service{
		clock:  clock.SystemClock{},
		logger: &mockLogger{},
		bsRepo: &mockBaseStationRepo{
			bs: &models.BaseStation{
				ID: 1,
			},
		},
	}

	ctx := testutil.TestContext()
	err := svc.persistCertsToBaseStation(ctx, tempDir, 1, 1, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrCACertRead) {
		t.Fatalf("error = %q, want sentinel %q", err.Error(), ErrCACertRead)
	}
}

type mockBaseStationRepo struct {
	bs          *models.BaseStation
	getErr      error
	updateErr   error
	updateID    int64
	updateArgs  map[string]interface{}
	ownerTenant int64
	keyMu       sync.Mutex
}

func (m *mockBaseStationRepo) Create(_ context.Context, _ *models.BaseStation) error {
	return errNotImplemented
}

func (m *mockBaseStationRepo) GetByID(_ context.Context, _ int64, _ int64) (*models.BaseStation, error) {
	return nil, errNotImplemented
}

func (m *mockBaseStationRepo) GetByEUI(_ context.Context, tenantID int64, _ []byte) (*models.BaseStation, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.ownerTenant != 0 && tenantID != m.ownerTenant {
		return nil, errTenantMismatch
	}
	return m.bs, nil
}

func (m *mockBaseStationRepo) Update(_ context.Context, _ int64, id int64, updates map[string]interface{}) error {
	m.updateID = id
	m.updateArgs = updates
	return m.updateErr
}

// TakeTLSKey hands the stored key to its owner once and keeps it when open
// refuses it, serialized like the repository's row lock.
func (m *mockBaseStationRepo) TakeTLSKey(_ context.Context, tenantID int64, _ []byte, open func(string) error) error {
	m.keyMu.Lock()
	defer m.keyMu.Unlock()
	if m.bs == nil || m.bs.TLSKey == nil || (m.ownerTenant != 0 && tenantID != m.ownerTenant) {
		return storage.ErrNotFound
	}
	if err := open(*m.bs.TLSKey); err != nil {
		return err
	}
	m.bs.TLSKey = nil
	return nil
}

func (m *mockBaseStationRepo) Delete(_ context.Context, _ int64, _ int64) error {
	return errNotImplemented
}

func (m *mockBaseStationRepo) List(_ context.Context, _ *models.BaseStationFilter) ([]*models.BaseStation, int64, error) {
	return nil, 0, errNotImplemented
}

func (m *mockBaseStationRepo) UpdateConnectionStatus(_ context.Context, _ int64, _ int64, _ bool, _ *string) error {
	return errNotImplemented
}

func (m *mockBaseStationRepo) GetStatistics(_ context.Context, _ int64) (*models.BaseStationStatistics, error) {
	return nil, errNotImplemented
}

func (m *mockBaseStationRepo) UpdateEUI(_ context.Context, _ int64, _ []byte, _ []byte) (*models.BaseStation, error) {
	return nil, errNotImplemented
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
	if err := os.WriteFile(filepath.Join(tmpDir, "server.crt"), []byte("dummy"), 0o600); err != nil {
		t.Fatalf("failed to write dummy server.crt: %v", err)
	}

	svc := &Service{
		clock:  clock.SystemClock{},
		logger: &mockLogger{},
		settings: settings{
			certsDir:           tmpDir,
			certGenPath:        testMissingCertGenPath,
			serverValidityDays: 365,
			protocol:           &config.ProtocolConfig{},
		},
	}

	ctx := testutil.TestContext()
	err := svc.RenewServerCertificates(ctx)
	if err == nil {
		t.Fatal("expected error when CA files are missing, got nil")
	}

	if !errors.Is(err, ErrCACertRead) && !errors.Is(err, ErrCAKeyRead) {
		t.Errorf("expected ErrCACertRead or ErrCAKeyRead, got: %s", err.Error())
	}
}

func TestRenewServerCertificates_FailsWhenNoServerCert(t *testing.T) {
	tmpDir := t.TempDir()

	svc := &Service{
		clock:  clock.SystemClock{},
		logger: &mockLogger{},
		settings: settings{
			certsDir:           tmpDir,
			certGenPath:        testMissingCertGenPath,
			serverValidityDays: 365,
			protocol:           &config.ProtocolConfig{},
		},
	}

	ctx := testutil.TestContext()
	err := svc.RenewServerCertificates(ctx)
	if err == nil {
		t.Fatal("expected error when server.crt is missing, got nil")
	}

	if !errors.Is(err, ErrNoCertificatesToRenew) {
		t.Errorf("expected ErrNoCertificatesToRenew, got: %s", err.Error())
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

	svc, err := New(testutil.TestContext(), cfg, &mockLogger{}, &mockBaseStationRepo{}, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if svc.settings.serverValidityDays != 730 {
		t.Errorf("expected serverValidityDays=730, got %d", svc.settings.serverValidityDays)
	}
}

func TestPlannedServerNames_WildcardExternalURL(t *testing.T) {
	svc := &Service{
		clock: clock.SystemClock{},
		settings: settings{
			certsDir: t.TempDir(),
			protocol: &config.ProtocolConfig{
				BSCIExternalURL: "tls://0.0.0.0:5000",
				BSCIHost:        "",
			},
		},
	}

	hostname := svc.plannedServerNames(testutil.TestContext()).subject
	if hostname == "0.0.0.0" {
		t.Errorf("the certificate subject should not return wildcard address, got %q", hostname)
	}
	if hostname != config.DefaultCertificatesHostname {
		t.Errorf("expected %q, got %q", config.DefaultCertificatesHostname, hostname)
	}
}

func TestPlannedServerNames_ValidExternalURL(t *testing.T) {
	svc := &Service{
		clock: clock.SystemClock{},
		settings: settings{
			certsDir: t.TempDir(),
			protocol: &config.ProtocolConfig{
				BSCIExternalURL: "tls://bssci.example.com:5000",
			},
		},
	}

	hostname := svc.plannedServerNames(testutil.TestContext()).subject
	if hostname != "bssci.example.com" {
		t.Errorf("expected 'bssci.example.com', got %q", hostname)
	}
}

func TestPlannedServerNames_FallbackToBSCIHost(t *testing.T) {
	svc := &Service{
		clock: clock.SystemClock{},
		settings: settings{
			certsDir: t.TempDir(),
			protocol: &config.ProtocolConfig{
				BSCIHost: "192.168.1.10",
			},
		},
	}

	hostname := svc.plannedServerNames(testutil.TestContext()).subject
	if hostname != "192.168.1.10" {
		t.Errorf("expected '192.168.1.10', got %q", hostname)
	}
}

func TestPlannedServerNames_WildcardBSCIHost(t *testing.T) {
	svc := &Service{
		clock: clock.SystemClock{},
		settings: settings{
			certsDir: t.TempDir(),
			protocol: &config.ProtocolConfig{
				BSCIHost: "0.0.0.0",
			},
		},
	}

	hostname := svc.plannedServerNames(testutil.TestContext()).subject
	if hostname != config.DefaultCertificatesHostname {
		t.Errorf("expected %q, got %q", config.DefaultCertificatesHostname, hostname)
	}
}

// newGenerateTestServiceWithRepo mirrors newGenerateTestService but injects a
// base-station repository so the tenant-ownership check runs.
func newGenerateTestServiceWithRepo(t *testing.T, repo *mockBaseStationRepo) *Service {
	t.Helper()
	tmpDir := t.TempDir()
	for _, name := range []string{"ca.crt", "ca.key"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("staged"), 0o600); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	return &Service{
		clock:        clock.SystemClock{},
		logger:       &mockLogger{},
		bsRepo:       repo,
		keyEncryptor: &mockKeyEncryptor{},
		certGen:      ExecCertGen,
		settings: settings{
			certGenPath:        filepath.Join(tmpDir, "certgen-missing"),
			certsDir:           tmpDir,
			tempDir:            filepath.Join(tmpDir, "temp"),
			serverValidityDays: config.DefaultCertificatesServerValidityDays,
			protocol:           &config.ProtocolConfig{},
		},
	}
}

// TestGenerateCertificate_CrossTenantDenied verifies a certificate cannot be
// minted for an EUI the requesting tenant does not own: the ownership lookup
// fails, so issuance is rejected before any certgen work.
func TestGenerateCertificate_CrossTenantDenied(t *testing.T) {
	svc := newGenerateTestServiceWithRepo(t, &mockBaseStationRepo{getErr: errTenantMismatch})
	ctx := testutil.TestContext()

	_, err := svc.GenerateCertificate(ctx, &grpcservices.CertificateRequest{
		BsEUI:        "cafecafecafecafe",
		ValidityDays: 365,
		TenantID:     42,
	})
	if err == nil {
		t.Fatal("expected cross-tenant issuance to be denied")
	}
	if !errors.Is(err, ErrBaseStationNotFound) {
		t.Errorf("error = %q, want base-station-not-found ownership rejection", err.Error())
	}
	// Must fail BEFORE reaching the (missing) certgen binary.
	if errors.Is(err, ErrGeneratorNotFound) {
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
	if errors.Is(err, ErrBaseStationNotFound) {
		t.Errorf("ownership check must pass for an owned EUI, got %q", err.Error())
	}
	if !errors.Is(err, ErrGeneratorNotFound) {
		t.Errorf("error = %q, want it to reach the certgen step", err.Error())
	}
}

func (m *mockBaseStationRepo) UpdateTLSFingerprintIfBlank(_ context.Context, _, _ int64, _ string) (bool, error) {
	return true, nil
}

func (m *mockBaseStationRepo) UpdateTLSCertExpiryIfBlank(_ context.Context, _, _ int64, _ time.Time) (bool, error) {
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
			return "", "no -dir argument", errMissingDirFlag
		}

		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return "", err.Error(), err
		}
		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: "CA-FE-CA-FE-CA-FE-CA-FE"},
			NotAfter:     time.Now().Add(testCertValidity),
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
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
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
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("staged"), 0o600); err != nil {
			t.Fatalf("stage %s: %v", name, err)
		}
	}
	return &Service{
		clock:        clock.SystemClock{},
		logger:       &mockLogger{},
		bsRepo:       repo,
		keyEncryptor: &mockKeyEncryptor{},
		certGen:      fakeCertGen(t),
		settings: settings{
			certGenPath:        filepath.Join(tmpDir, "certgen-unused"),
			certsDir:           tmpDir,
			tempDir:            filepath.Join(tmpDir, "temp"),
			serverValidityDays: config.DefaultCertificatesServerValidityDays,
			protocol:           &config.ProtocolConfig{},
		},
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
	if resp.BaseStationID != repo.bs.ID || repo.updateID != repo.bs.ID {
		t.Fatalf("issued for station %d, stored on %d, want %d", resp.BaseStationID, repo.updateID, repo.bs.ID)
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
		updateErr: errRepoUnavailable,
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
	if !errors.Is(err, ErrPersistenceFailed) {
		t.Fatalf("error = %q, want sentinel %q", err.Error(), ErrPersistenceFailed)
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
	if !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("error = %q, want sentinel %q", err.Error(), ErrTenantRequired)
	}
}

// TestStoredPrivateKey_LeavesTheServiceCenterOnce: a base station's private key
// is handed out once, whichever download path takes it first.
func TestStoredPrivateKey_LeavesTheServiceCenterOnce(t *testing.T) {
	sealed := base64.StdEncoding.EncodeToString([]byte("station key"))
	eui := []byte{0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE}

	t.Run("stored download", func(t *testing.T) {
		repo := tenantScopedRepo()
		repo.bs.TLSKey = &sealed
		svc, err := New(testutil.TestContext(), &config.Config{Certificates: config.CertificateConfig{TempDir: t.TempDir()}}, &mockLogger{}, repo, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		if _, _, err := svc.GetStoredCertificate(testutil.TestContext(), testForeignTenant, eui, CertTypeKey); err == nil {
			t.Fatal("another tenant received the key")
		}
		key, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, eui, CertTypeKey)
		if err != nil || string(key) != "station key" {
			t.Fatalf("first download: %q, %v", key, err)
		}
		if _, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, eui, CertTypeKey); !errors.Is(err, ErrNotFound) {
			t.Fatalf("second download: %v, want ErrNotFound", err)
		}
	})

	t.Run("bundle download retires the stored copy", func(t *testing.T) {
		repo := tenantScopedRepo()
		bundleSealed := base64.StdEncoding.EncodeToString([]byte(testBundleKey))
		repo.bs.TLSKey = &bundleSealed
		tempDir := t.TempDir()
		svc, err := New(testutil.TestContext(), &config.Config{Certificates: config.CertificateConfig{TempDir: tempDir}}, &mockLogger{}, repo, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		id := writeBundle(t, tempDir, time.Now())

		if _, _, err := svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeKey, id); err != nil {
			t.Fatalf("bundle key download: %v", err)
		}
		if _, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, eui, CertTypeKey); !errors.Is(err, ErrNotFound) {
			t.Fatalf("stored key after the bundle download: %v, want ErrNotFound", err)
		}
		if _, _, err := svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeKey, id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("second bundle key download: %v, want ErrNotFound", err)
		}
	})

	t.Run("a superseded bundle key is refused and the stored key kept", func(t *testing.T) {
		repo := tenantScopedRepo()
		repo.bs.TLSKey = &sealed
		tempDir := t.TempDir()
		svc, err := New(testutil.TestContext(), &config.Config{Certificates: config.CertificateConfig{TempDir: tempDir}}, &mockLogger{}, repo, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		id := writeBundle(t, tempDir, time.Now())

		if _, _, err := svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeKey, id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("superseded bundle key: %v, want ErrNotFound", err)
		}
		if repo.bs.TLSKey == nil {
			t.Fatal("the station's current key was lost")
		}
	})
}

// TestBundleKeyRefusal_RemovesTheBundlesKeyFile: a bundle whose key was taken
// or superseded can never serve it again, so a refused download deletes the
// bundle's plaintext copy and leaves the station's stored key alone; a
// transient failure keeps the file for a retry.
func TestBundleKeyRefusal_RemovesTheBundlesKeyFile(t *testing.T) {
	eui := []byte{0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE}
	bundleSealed := base64.StdEncoding.EncodeToString([]byte(testBundleKey))
	currentSealed := base64.StdEncoding.EncodeToString([]byte("station key"))
	cases := []struct {
		name        string
		stored      *string
		takeFirst   bool
		encryptor   *mockKeyEncryptor
		fileRemains bool
		storedAfter *string
	}{
		{name: "already taken", stored: &bundleSealed, takeFirst: true, encryptor: &mockKeyEncryptor{}},
		{name: "superseded", stored: &currentSealed, encryptor: &mockKeyEncryptor{}, storedAfter: &currentSealed},
		{name: "transient failure", stored: &bundleSealed, encryptor: &mockKeyEncryptor{decryptErr: errTestDecrypt}, fileRemains: true, storedAfter: &bundleSealed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := tenantScopedRepo()
			stored := *tc.stored
			repo.bs.TLSKey = &stored
			tempDir := t.TempDir()
			svc, err := New(testutil.TestContext(), &config.Config{Certificates: config.CertificateConfig{TempDir: tempDir}}, &mockLogger{}, repo, tc.encryptor, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			id := writeBundle(t, tempDir, time.Now())
			keyFile := filepath.Join(tempDir, id, clientKeyFileName)
			if tc.takeFirst {
				if _, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, eui, CertTypeKey); err != nil {
					t.Fatalf("stored key download: %v", err)
				}
			}

			if _, _, err := svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeKey, id); err == nil {
				t.Fatal("the bundle key was served")
			}

			_, statErr := os.Stat(keyFile)
			if tc.fileRemains != (statErr == nil) {
				t.Fatalf("bundle key file present = %v, want %v", statErr == nil, tc.fileRemains)
			}
			switch {
			case tc.storedAfter == nil && repo.bs.TLSKey != nil:
				t.Fatal("a stored key appeared")
			case tc.storedAfter != nil && (repo.bs.TLSKey == nil || *repo.bs.TLSKey != *tc.storedAfter):
				t.Fatal("the station's stored key changed")
			}
		})
	}
}

// TestPrivateKey_ConcurrentDownloadsHandItOutOnce: concurrent downloads of a
// bundle key and of the stored key yield the key exactly once.
func TestPrivateKey_ConcurrentDownloadsHandItOutOnce(t *testing.T) {
	sealed := base64.StdEncoding.EncodeToString([]byte(testBundleKey))
	repo := tenantScopedRepo()
	repo.bs.TLSKey = &sealed
	tempDir := t.TempDir()
	svc, err := New(testutil.TestContext(), &config.Config{Certificates: config.CertificateConfig{TempDir: tempDir}}, &mockLogger{}, repo, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	id := writeBundle(t, tempDir, time.Now())
	eui := []byte{0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE}

	var served atomic.Int32
	var wg sync.WaitGroup
	for i := range testConcurrentDownloads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var key []byte
			var err error
			if i%2 == 0 {
				key, _, err = svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeKey, id)
			} else {
				key, _, err = svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, eui, CertTypeKey)
			}
			if err == nil && len(key) > 0 {
				served.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := served.Load(); got != 1 {
		t.Fatalf("the private key was served %d times, want exactly once", got)
	}
}

// TestStoredPrivateKey_SurvivesADecryptFailure: a key that fails to decrypt
// (a wrong master key) is refused and stays stored for a later download.
func TestStoredPrivateKey_SurvivesADecryptFailure(t *testing.T) {
	sealed := base64.StdEncoding.EncodeToString([]byte("station key"))
	eui := []byte{0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE}
	repo := tenantScopedRepo()
	repo.bs.TLSKey = &sealed
	encryptor := &mockKeyEncryptor{decryptErr: errTestDecrypt}
	svc, err := New(testutil.TestContext(), &config.Config{Certificates: config.CertificateConfig{TempDir: t.TempDir()}}, &mockLogger{}, repo, encryptor, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, eui, CertTypeKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("undecryptable key: %v, want ErrNotFound", err)
	}
	if repo.bs.TLSKey == nil {
		t.Fatal("a decrypt failure lost the stored key")
	}

	encryptor.decryptErr = nil
	key, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, eui, CertTypeKey)
	if err != nil || string(key) != "station key" {
		t.Fatalf("download after the master key is fixed: %q, %v", key, err)
	}
}
