package scaciservices

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/models"
)

// ============================================================================
// Test Helpers
// ============================================================================

// mockSCACISessionRepository implements interfaces.SCACISessionRepository for testing
type mockSCACISessionRepository struct {
	checkSessionResumableFunc func(ctx context.Context, tenantID int64, acUUID [16]byte, acOpId int64, scOpId int64) (*models.SCACISessionResumptionInfo, error)
	getSessionByAcUUIDFunc    func(ctx context.Context, tenantID int64, acUUID [16]byte) (*models.SCACISession, error)
}

func (m *mockSCACISessionRepository) CheckSessionResumable(ctx context.Context, tenantID int64, acUUID [16]byte, acOpId int64, scOpId int64) (*models.SCACISessionResumptionInfo, error) {
	if m.checkSessionResumableFunc != nil {
		return m.checkSessionResumableFunc(ctx, tenantID, acUUID, acOpId, scOpId)
	}
	return nil, nil
}

func (m *mockSCACISessionRepository) GetSessionByAcUUID(ctx context.Context, tenantID int64, acUUID [16]byte) (*models.SCACISession, error) {
	if m.getSessionByAcUUIDFunc != nil {
		return m.getSessionByAcUUIDFunc(ctx, tenantID, acUUID)
	}
	return nil, nil
}

// Stub methods for remaining interface methods (not used in these tests)
func (m *mockSCACISessionRepository) CreateSession(_ context.Context, _ *models.SCACISessionCreateRequest) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSCACISessionRepository) GetSessionByID(_ context.Context, _, _ int64) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSCACISessionRepository) GetActiveSessionByAcEUI(_ context.Context, _ int64, _ [8]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSCACISessionRepository) GetSessionByScUUID(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
	return nil, nil
}
func (m *mockSCACISessionRepository) UpdateSession(_ context.Context, _, _ int64, _ *models.SCACISessionUpdateRequest) error {
	return nil
}
func (m *mockSCACISessionRepository) UpdateOperationIDs(_ context.Context, _, _ int64, _, _ int64) error {
	return nil
}
func (m *mockSCACISessionRepository) UpdateHeartbeat(_ context.Context, _, _ int64) error {
	return nil
}
func (m *mockSCACISessionRepository) DisconnectSession(_ context.Context, _, _ int64) error {
	return nil
}
func (m *mockSCACISessionRepository) TerminateSession(_ context.Context, _, _ int64) error {
	return nil
}
func (m *mockSCACISessionRepository) TerminateAllSessions(_ context.Context, _ int64, _ [8]byte) error {
	return nil
}
func (m *mockSCACISessionRepository) ListSessions(_ context.Context, _ *models.SCACISessionFilter) ([]*models.SCACISession, int64, error) {
	return nil, 0, nil
}
func (m *mockSCACISessionRepository) GetSessionStatistics(_ context.Context, _ int64) (*models.SCACISessionStatistics, error) {
	return nil, nil
}
func (m *mockSCACISessionRepository) CleanupExpiredSessions(_ context.Context, _ int64) (int64, error) {
	return 0, nil
}

// mockOrgResolver implements org.Resolver for testing
type mockOrgResolver struct {
	lookupTenantFunc           func(ctx context.Context, orgUUID uuid.UUID) (int64, error)
	resolveCertFunc            func(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error)
	getDefaultOrgForTenantFunc func(ctx context.Context, tenantID int64) (uuid.UUID, error)
}

func (m *mockOrgResolver) LookupTenant(ctx context.Context, orgUUID uuid.UUID) (int64, error) {
	if m.lookupTenantFunc != nil {
		return m.lookupTenantFunc(ctx, orgUUID)
	}
	return 1, nil // Default tenant
}

func (m *mockOrgResolver) ResolveCert(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error) {
	if m.resolveCertFunc != nil {
		return m.resolveCertFunc(ctx, cert)
	}
	return uuid.Nil, 1, nil // Default: no org UUID, tenant 1
}

func (m *mockOrgResolver) GetDefaultOrgForTenant(ctx context.Context, tenantID int64) (uuid.UUID, error) {
	if m.getDefaultOrgForTenantFunc != nil {
		return m.getDefaultOrgForTenantFunc(ctx, tenantID)
	}
	return uuid.Nil, nil
}

// mockCertificateVerifier implements scaci.CertificateVerifier for testing
type mockCertificateVerifier struct {
	verifyCertificateFunc func(cert *x509.Certificate) string
}

func (m *mockCertificateVerifier) VerifyCertificate(cert *x509.Certificate) string {
	if m.verifyCertificateFunc != nil {
		return m.verifyCertificateFunc(cert)
	}
	return "" // Valid by default
}

// createValidTestCert creates a valid certificate for testing
func createValidTestCert(cn string) *x509.Certificate {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	now := time.Now()

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"TestOrg"},
		},
		NotBefore:   now.Add(-24 * time.Hour),
		NotAfter:    now.Add(365 * 24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	certDER, _ := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	cert, _ := x509.ParseCertificate(certDER)
	return cert
}

// ============================================================================
// Certificate Validation Tests
// ============================================================================

func TestValidateConnect_NilCertificate(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, nil)

	if errToken != scaci.ErrNilCertificate {
		t.Errorf("expected ErrNilCertificate, got: %s", errToken)
	}
	if session != nil || resp != nil {
		t.Error("expected nil session and response on nil certificate")
	}
}

func TestValidateConnect_CertificateExpired(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}

	// Mock certVerifier to return expired error
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return scaci.ErrCertExpired
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	cert := createValidTestCert("tenant-123")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	if errToken != scaci.ErrCertExpired {
		t.Errorf("expected ErrCertExpired, got: %s", errToken)
	}
	if session != nil || resp != nil {
		t.Error("expected nil session and response on expired certificate")
	}
}

func TestValidateConnect_CertificateNotYetValid(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}

	// Mock certVerifier to return not yet valid error
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return scaci.ErrCertNotYetValid
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	cert := createValidTestCert("tenant-123")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	if errToken != scaci.ErrCertNotYetValid {
		t.Errorf("expected ErrCertNotYetValid, got: %s", errToken)
	}
	if session != nil || resp != nil {
		t.Error("expected nil session and response on not yet valid certificate")
	}
}

func TestValidateConnect_CertificateMissingClientAuth(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}

	// Mock certVerifier to return missing ClientAuth error
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return scaci.ErrCertMissingClientAuth
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	cert := createValidTestCert("tenant-123")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	if errToken != scaci.ErrCertMissingClientAuth {
		t.Errorf("expected ErrCertMissingClientAuth, got: %s", errToken)
	}
	if session != nil || resp != nil {
		t.Error("expected nil session and response on certificate without ClientAuth")
	}
}

func TestValidateConnect_ValidCertificate_NewSession(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}

	// Mock org resolver to return tenant ID 42
	orgResolver := &mockOrgResolver{
		resolveCertFunc: func(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
			return uuid.Nil, 42, nil
		},
	}

	// Mock cert verifier to pass validation
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return "" // Valid
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	cert := createValidTestCert("tenant-42")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	if errToken != "" {
		t.Errorf("expected no error, got: %s", errToken)
	}
	if session == nil {
		t.Fatal("expected session to be created")
	}
	if resp == nil {
		t.Fatal("expected response to be created")
	}

	// Verify session properties
	if session.TenantID != 42 {
		t.Errorf("expected TenantID=42, got %d", session.TenantID)
	}
	if session.AcEui != 0xAABBCCDDEEFF1122 {
		t.Errorf("expected AcEui=0xAABBCCDDEEFF1122, got 0x%X", session.AcEui)
	}
	if session.Resumed {
		t.Error("expected new session (not resumed)")
	}

	// Verify response properties
	if resp.ScEui != 0x1122334455667788 {
		t.Errorf("expected ScEui=0x1122334455667788, got 0x%X", resp.ScEui)
	}
	if resp.SnResume {
		t.Error("expected SnResume=false for new session")
	}
	if resp.Version == nil || *resp.Version != "1.0.0" {
		t.Errorf("expected Version=1.0.0, got %v", resp.Version)
	}
}

// ============================================================================
// Strict Org Resolution Tests
// ============================================================================

// TestValidateConnect_StrictOrgResolution_FailsClosed verifies that strict mode
// fails closed when org resolution fails (production/managed cloud behavior).
func TestValidateConnect_StrictOrgResolution_FailsClosed(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}

	// Mock org resolver to fail
	orgResolver := &mockOrgResolver{
		resolveCertFunc: func(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
			return uuid.Nil, 0, context.DeadlineExceeded // Simulate resolution failure
		},
	}

	// Mock cert verifier to pass validation
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return "" // Valid
		},
	}

	// STRICT MODE: strictOrgResolution=true
	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, true, certVerifier, // strictOrgResolution=true
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	cert := createValidTestCert("unknown-org")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	// Strict mode should fail-closed
	if errToken != scaci.ErrCertificateTenantResolutionFailed {
		t.Errorf("expected ErrCertificateTenantResolutionFailed, got: %s", errToken)
	}
	if session != nil || resp != nil {
		t.Error("expected nil session and response on org resolution failure in strict mode")
	}
}

// TestValidateConnect_CommunityMode_FallsBack verifies that community mode
// falls back to default tenant when org resolution fails.
func TestValidateConnect_CommunityMode_FallsBack(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}

	defaultTenantID := int64(999)
	defaultOrgID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	// Mock org resolver to fail initial resolution but succeed on fallback
	orgResolver := &mockOrgResolver{
		resolveCertFunc: func(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
			return uuid.Nil, 0, context.DeadlineExceeded // Simulate resolution failure
		},
		getDefaultOrgForTenantFunc: func(_ context.Context, tenantID int64) (uuid.UUID, error) {
			if tenantID == defaultTenantID {
				return defaultOrgID, nil
			}
			return uuid.Nil, context.DeadlineExceeded
		},
	}

	// Mock cert verifier to pass validation
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return "" // Valid
		},
	}

	// COMMUNITY MODE: strictOrgResolution=false
	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, defaultTenantID, false, certVerifier, // strictOrgResolution=false
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	cert := createValidTestCert("unknown-org")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	// Community mode should fall back to default tenant
	if errToken != "" {
		t.Errorf("expected no error (community fallback), got: %s", errToken)
	}
	if session == nil {
		t.Fatal("expected session to be created")
	}
	if resp == nil {
		t.Fatal("expected response to be created")
	}

	// Verify session uses fallback tenant
	if session.TenantID != defaultTenantID {
		t.Errorf("expected TenantID=%d (fallback), got %d", defaultTenantID, session.TenantID)
	}
	if session.OrganizationID != defaultOrgID {
		t.Errorf("expected OrganizationID=%s (fallback), got %s", defaultOrgID, session.OrganizationID)
	}
}

// ============================================================================
// Version Negotiation Tests
// ============================================================================

func TestNegotiateVersion_ValidVersion(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	negotiated, errToken := svc.NegotiateVersion("1.0.0")

	if errToken != "" {
		t.Errorf("expected no error, got: %s", errToken)
	}
	if negotiated != "1.0.0" {
		t.Errorf("expected negotiated version 1.0.0, got: %s", negotiated)
	}
}

func TestNegotiateVersion_MajorVersionMismatch(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	negotiated, errToken := svc.NegotiateVersion("2.0.0")

	if errToken != scaci.ErrMajorVersionUnsupported {
		t.Errorf("expected ErrMajorVersionUnsupported, got: %s", errToken)
	}
	if negotiated != "" {
		t.Errorf("expected empty negotiated version on error, got: %s", negotiated)
	}
}

func TestNegotiateVersion_MinorVersionTooHigh(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	negotiated, errToken := svc.NegotiateVersion("1.5.0")

	if errToken != scaci.ErrMinorVersionUnsupported {
		t.Errorf("expected ErrMinorVersionUnsupported, got: %s", errToken)
	}
	if negotiated != "" {
		t.Errorf("expected empty negotiated version on error, got: %s", negotiated)
	}
}

func TestNegotiateVersion_InvalidFormat(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	negotiated, errToken := svc.NegotiateVersion("invalid-version")

	if errToken != scaci.ErrInvalidVersionFormat {
		t.Errorf("expected ErrInvalidVersionFormat, got: %s", errToken)
	}
	if negotiated != "" {
		t.Errorf("expected empty negotiated version on error, got: %s", negotiated)
	}
}

// TestNegotiateVersion_UsesPkgConstants verifies that NegotiateVersion
// returns the canonical ProtocolVersionString from pkg/scaci/constants.go
// per SCACI §§2.1-2.3 requirements.
func TestNegotiateVersion_UsesPkgConstants(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	// Request with version matching SupportedMajorVersion.SupportedMinorVersion
	negotiated, errToken := svc.NegotiateVersion("1.0.0")

	if errToken != "" {
		t.Fatalf("expected successful negotiation, got error: %s", errToken)
	}

	// Verify the negotiated version matches pkg/scaci/constants.go:ProtocolVersionString
	// This ensures we're using centralized constants, not hardcoded values
	if negotiated != scaci.ProtocolVersionString {
		t.Errorf("negotiated version %q does not match pkg constant scaci.ProtocolVersionString %q",
			negotiated, scaci.ProtocolVersionString)
	}

	// Also verify the constant itself has the expected value for documentation
	if scaci.ProtocolVersionString != "1.0.0" {
		t.Errorf("scaci.ProtocolVersionString should be '1.0.0', got %q", scaci.ProtocolVersionString)
	}
}

// ============================================================================
// Session Resumption Tests
// ============================================================================

func TestResolveResume_InvalidUUIDLength(t *testing.T) {
	logger := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	// Invalid UUID (only 8 bytes instead of 16)
	acUUID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 1, -1, "1.0.0")

	if canResume {
		t.Error("expected resume to fail with invalid UUID length")
	}
	if errToken != scaci.ErrSnAcUUIDZero {
		t.Errorf("expected ErrSnAcUUIDZero, got: %s", errToken)
	}
}

// TestResolveResume_ZeroUUID validates resume fails when SnAcUUID is all zeros
// Note: Zero UUID validation ideally happens at the session_validator layer (handler_connect.go:73),
// but if it reaches ResolveResume, a zero UUID (valid 16-byte length) will simply not match any session
func TestResolveResume_ZeroUUID(t *testing.T) {
	log := logger.NewNop()
	// Mock sessionRepo to return nil (no session found with zero UUID)
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return nil, nil // No session found
		},
	}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, log, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", scaci.ProtocolVersionString,
	).(*handshakeService)

	// All-zeros UUID (16 bytes of zeros) - valid length but non-existent
	acUUID := make([]byte, 16)

	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 1, -1, scaci.ProtocolVersionString)

	if canResume {
		t.Error("expected resume to fail with zero UUID")
	}
	// Zero UUID passes length check (16 bytes) but no session exists, so ErrNoActiveSession
	// (The ErrSnAcUUIDZero is returned by session_validator in the handler, before ResolveResume is called)
	if errToken != scaci.ErrNoActiveSession {
		t.Errorf("expected ErrNoActiveSession for zero UUID (no matching session), got: %s", errToken)
	}
}

func TestResolveResume_SessionNotResumable(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return non-resumable session
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:            false,
				ReasonIfNotResumable: "opId out of sequence",
				LastKnownAcOpId:      5,
				LastKnownScOpId:      -5,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 1, -1, "1.0.0")

	if canResume {
		t.Error("expected resume to fail when session is not resumable")
	}
	if errToken != scaci.ErrOpIdOutOfOrder {
		t.Errorf("expected ErrOpIdOutOfOrder, got: %s", errToken)
	}
}

func TestResolveResume_SuccessfulResume(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return resumable session
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:            true,
				ReasonIfNotResumable: "",
				LastKnownAcOpId:      10,
				LastKnownScOpId:      -10,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 10, -10, "1.0.0")

	if !canResume {
		t.Error("expected resume to succeed")
	}
	if errToken != "" {
		t.Errorf("expected no error, got: %s", errToken)
	}
}

// TestResolveResume_AcOpIdRegression validates SCACI-S.1-05:
// Resume must be rejected when AC opId has regressed (lower than stored value)
func TestResolveResume_AcOpIdRegression(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to validate AC opId and reject regression
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, acOpId int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			lastKnownAcOpId := int64(100)
			// AC opIds are positive and increment - provided must be >= last known
			if acOpId < lastKnownAcOpId {
				return &models.SCACISessionResumptionInfo{
					CanResume:            false,
					ReasonIfNotResumable: "AC opId regression: provided=50, expected>=100",
					LastKnownAcOpId:      lastKnownAcOpId,
					LastKnownScOpId:      -100,
				}, nil
			}
			return &models.SCACISessionResumptionInfo{
				CanResume:       true,
				LastKnownAcOpId: lastKnownAcOpId,
				LastKnownScOpId: -100,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Attempt resume with regressed AC opId (50 < 100)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 50, -100, "1.0.0")

	if canResume {
		t.Error("expected resume to fail when AC opId has regressed")
	}
	if errToken != scaci.ErrOpIdOutOfOrder {
		t.Errorf("expected ErrOpIdOutOfOrder for AC regression, got: %s", errToken)
	}
}

// TestResolveResume_ScOpIdRegression validates SCACI-3.2-03:
// Resume must be rejected when SC opId has regressed (higher than stored value, since SC opIds are negative/decrementing)
func TestResolveResume_ScOpIdRegression(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to validate SC opId and reject regression
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, scOpId int64) (*models.SCACISessionResumptionInfo, error) {
			lastKnownScOpId := int64(-100)
			// SC opIds are negative and decrement - provided must be <= last known
			if scOpId > lastKnownScOpId {
				return &models.SCACISessionResumptionInfo{
					CanResume:            false,
					ReasonIfNotResumable: "SC opId regression: provided=-50, expected<=-100",
					LastKnownAcOpId:      100,
					LastKnownScOpId:      lastKnownScOpId,
				}, nil
			}
			return &models.SCACISessionResumptionInfo{
				CanResume:       true,
				LastKnownAcOpId: 100,
				LastKnownScOpId: lastKnownScOpId,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Attempt resume with regressed SC opId (-50 > -100, regression for negative values)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 100, -50, "1.0.0")

	if canResume {
		t.Error("expected resume to fail when SC opId has regressed")
	}
	if errToken != scaci.ErrOpIdOutOfOrder {
		t.Errorf("expected ErrOpIdOutOfOrder for SC regression, got: %s", errToken)
	}
}

// TestResolveResume_BothOpIdsValid validates successful resumption when both opIds are valid
func TestResolveResume_BothOpIdsValid(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to validate both opIds and allow resume
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, acOpId int64, scOpId int64) (*models.SCACISessionResumptionInfo, error) {
			lastKnownAcOpId := int64(100)
			lastKnownScOpId := int64(-100)

			// AC opIds must be >= last known (positive, incrementing)
			if acOpId < lastKnownAcOpId {
				return &models.SCACISessionResumptionInfo{
					CanResume:            false,
					ReasonIfNotResumable: "AC opId regression",
					LastKnownAcOpId:      lastKnownAcOpId,
					LastKnownScOpId:      lastKnownScOpId,
				}, nil
			}
			// SC opIds must be <= last known (negative, decrementing)
			if scOpId > lastKnownScOpId {
				return &models.SCACISessionResumptionInfo{
					CanResume:            false,
					ReasonIfNotResumable: "SC opId regression",
					LastKnownAcOpId:      lastKnownAcOpId,
					LastKnownScOpId:      lastKnownScOpId,
				}, nil
			}
			return &models.SCACISessionResumptionInfo{
				CanResume:       true,
				LastKnownAcOpId: lastKnownAcOpId,
				LastKnownScOpId: lastKnownScOpId,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Resume with valid opIds: AC=100 (matches), SC=-100 (matches)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 100, -100, "1.0.0")

	if !canResume {
		t.Error("expected resume to succeed when both opIds are valid")
	}
	if errToken != "" {
		t.Errorf("expected no error, got: %s", errToken)
	}
}

// TestResolveResume_OpIdsAdvanced validates resumption accepts advanced opIds
// (AC higher than stored, SC lower than stored)
func TestResolveResume_OpIdsAdvanced(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to accept advanced opIds
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, acOpId int64, scOpId int64) (*models.SCACISessionResumptionInfo, error) {
			lastKnownAcOpId := int64(100)
			lastKnownScOpId := int64(-100)

			// AC opIds must be >= last known
			if acOpId < lastKnownAcOpId {
				return &models.SCACISessionResumptionInfo{
					CanResume:            false,
					ReasonIfNotResumable: "AC opId regression",
					LastKnownAcOpId:      lastKnownAcOpId,
					LastKnownScOpId:      lastKnownScOpId,
				}, nil
			}
			// SC opIds must be <= last known
			if scOpId > lastKnownScOpId {
				return &models.SCACISessionResumptionInfo{
					CanResume:            false,
					ReasonIfNotResumable: "SC opId regression",
					LastKnownAcOpId:      lastKnownAcOpId,
					LastKnownScOpId:      lastKnownScOpId,
				}, nil
			}
			return &models.SCACISessionResumptionInfo{
				CanResume:       true,
				LastKnownAcOpId: lastKnownAcOpId,
				LastKnownScOpId: lastKnownScOpId,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Resume with advanced opIds: AC=150 (> 100), SC=-150 (< -100)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 150, -150, "1.0.0")

	if !canResume {
		t.Error("expected resume to succeed when opIds have advanced")
	}
	if errToken != "" {
		t.Errorf("expected no error, got: %s", errToken)
	}
}

// ============================================================================
// Version Persistence and Resume Validation Tests (SCACI §§2.1-2.3)
// ============================================================================

// TestResolveResume_VersionMismatch validates SCACI §§2.1-2.3:
// Resume must be rejected when the request version differs from the stored negotiated version
func TestResolveResume_VersionMismatch(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return a session with NegotiatedVersion="1.0.0"
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   100,
				LastKnownScOpId:   -100,
				NegotiatedVersion: "1.0.0", // Stored version from original Connect
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Attempt resume with different version (1.1.0 vs stored 1.0.0)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 100, -100, "1.1.0")

	if canResume {
		t.Error("expected resume to fail when version mismatches stored negotiated version")
	}
	if errToken != scaci.ErrVersionMismatchOnResume {
		t.Errorf("expected ErrVersionMismatchOnResume, got: %s", errToken)
	}
}

// TestResolveResume_VersionMatch validates successful resume when versions match
func TestResolveResume_VersionMatch(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return a session with NegotiatedVersion="1.0.0"
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   100,
				LastKnownScOpId:   -100,
				NegotiatedVersion: "1.0.0", // Stored version matches request
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Resume with matching version
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 100, -100, "1.0.0")

	if !canResume {
		t.Error("expected resume to succeed when version matches stored negotiated version")
	}
	if errToken != "" {
		t.Errorf("expected no error, got: %s", errToken)
	}
}

// TestResolveResume_PatchDifferentAllowsResume validates SCACI §2.3:
// Resume must succeed when only the patch version differs (major.minor match)
func TestResolveResume_PatchDifferentAllowsResume(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return a session with NegotiatedVersion="1.0.0"
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   100,
				LastKnownScOpId:   -100,
				NegotiatedVersion: "1.0.0", // Stored version
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Resume with different patch version (1.0.5 vs stored 1.0.0)
	// Per SCACI §2.3: patch version MUST be ignored
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 100, -100, "1.0.5")

	if !canResume {
		t.Error("expected resume to succeed when only patch version differs (§2.3 requires patch ignored)")
	}
	if errToken != "" {
		t.Errorf("expected no error for patch difference per §2.3, got: %s", errToken)
	}
}

// TestResolveResume_MinorDifferentRejects validates SCACI §2.2:
// Resume must fail when minor version differs
func TestResolveResume_MinorDifferentRejects(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return a session with NegotiatedVersion="1.0.0"
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   100,
				LastKnownScOpId:   -100,
				NegotiatedVersion: "1.0.0", // Stored version
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Resume with different minor version (1.1.0 vs stored 1.0.0)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 100, -100, "1.1.0")

	if canResume {
		t.Error("expected resume to fail when minor version differs")
	}
	if errToken != scaci.ErrVersionMismatchOnResume {
		t.Errorf("expected ErrVersionMismatchOnResume for minor difference, got: %s", errToken)
	}
}

// TestResolveResume_MajorDifferentRejects validates SCACI §2.1:
// Resume must fail when major version differs
func TestResolveResume_MajorDifferentRejects(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return a session with NegotiatedVersion="1.0.0"
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   100,
				LastKnownScOpId:   -100,
				NegotiatedVersion: "1.0.0", // Stored version
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Resume with different major version (2.0.0 vs stored 1.0.0)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 100, -100, "2.0.0")

	if canResume {
		t.Error("expected resume to fail when major version differs")
	}
	if errToken != scaci.ErrVersionMismatchOnResume {
		t.Errorf("expected ErrVersionMismatchOnResume for major difference, got: %s", errToken)
	}
}

// TestResolveResume_EmptyStoredVersionAllowsResume validates backwards compatibility:
// If stored version is empty (previous session before migration), allow resume
func TestResolveResume_EmptyStoredVersionAllowsResume(t *testing.T) {
	logger := logger.NewNop()
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	// Mock sessionRepo to return a previous session with empty NegotiatedVersion
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   50,
				LastKnownScOpId:   -50,
				NegotiatedVersion: "", // Previous session - no version stored
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", "1.0.0",
	).(*handshakeService)

	acUUID := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}

	// Resume with any version when stored is empty (backwards compatibility)
	canResume, errToken := svc.ResolveResume(testutil.TestContext(), 1, acUUID, nil, 50, -50, "1.0.0")

	if !canResume {
		t.Error("expected resume to succeed when stored version is empty (previous session)")
	}
	if errToken != "" {
		t.Errorf("expected no error for previous session, got: %s", errToken)
	}
}

// ============================================================================
// Cross-Tenant Protection Tests (SCACI §3.3 Security)
// ============================================================================

// TestValidateConnect_CrossTenantResumeRejected validates defense-in-depth security:
// Resume must be rejected when certificate tenant differs from stored session tenant.
// This test covers the service-layer protection at handshake_service.go:223-230.
// See also: scaci_session_repository_test.go for repo-layer tenant filtering tests.
func TestValidateConnect_CrossTenantResumeRejected(t *testing.T) {
	log := logger.NewNop()
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return "" // Valid
		},
	}

	// Certificate resolves to tenant 2 (the attacking tenant)
	orgResolver := &mockOrgResolver{
		resolveCertFunc: func(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
			return uuid.Nil, 2, nil // Certificate belongs to tenant 2
		},
	}

	acUUID := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}
	acEUI := [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}

	// Session repo: CheckSessionResumable allows resume, but GetSessionByAcUUID
	// returns a session owned by tenant 1 (the victim tenant)
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			// Resume allowed at opId/version level
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   10,
				LastKnownScOpId:   -10,
				NegotiatedVersion: scaci.ProtocolVersionString,
			}, nil
		},
		getSessionByAcUUIDFunc: func(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
			// Return session owned by tenant 1 (different from certificate tenant 2)
			return &models.SCACISession{
				ID:         1,
				TenantID:   1, // Session belongs to tenant 1, but cert is tenant 2
				AcEUI:      acEUI,
				SnAcUUID:   acUUID,
				SnScUUID:   [16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00},
				Status:     "active",
				LastOpIDAc: 10,
				LastOpIDSc: -10,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, log, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", scaci.ProtocolVersionString,
	)

	// Build Connect request with resume fields pointing to tenant 1's session
	snAcOpId := int64(10)
	snScOpId := int64(-10)
	req := &scaci.Connect{
		Version:  scaci.ProtocolVersionString,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16(acUUID),
		SnAcOpId: &snAcOpId, // Resume fields present
		SnScOpId: &snScOpId,
	}

	cert := createValidTestCert("tenant-2") // Certificate for tenant 2

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	// Assert: Cross-tenant resume must be rejected with ErrNoActiveSession
	// (not a tenant-specific error to prevent tenant enumeration)
	if errToken != scaci.ErrNoActiveSession {
		t.Errorf("expected ErrNoActiveSession for cross-tenant resume, got: %q", errToken)
	}
	if session != nil {
		t.Error("expected nil session on cross-tenant resume rejection")
	}
	if resp != nil {
		t.Error("expected nil response on cross-tenant resume rejection")
	}
}

// TestValidateConnect_SameTenantResumeSucceeds verifies resume works when tenant matches
func TestValidateConnect_SameTenantResumeSucceeds(t *testing.T) {
	log := logger.NewNop()
	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(_ *x509.Certificate) string {
			return "" // Valid
		},
	}

	// Certificate resolves to tenant 1 (same as session)
	orgResolver := &mockOrgResolver{
		resolveCertFunc: func(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
			return uuid.Nil, 1, nil // Certificate belongs to tenant 1
		},
	}

	acUUID := [16]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
	}
	acEUI := [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}

	// Session repo: both functions confirm session belongs to tenant 1
	sessionRepo := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, _ int64, _ [16]byte, _ int64, _ int64) (*models.SCACISessionResumptionInfo, error) {
			return &models.SCACISessionResumptionInfo{
				CanResume:         true,
				LastKnownAcOpId:   10,
				LastKnownScOpId:   -10,
				NegotiatedVersion: scaci.ProtocolVersionString,
			}, nil
		},
		getSessionByAcUUIDFunc: func(_ context.Context, _ int64, _ [16]byte) (*models.SCACISession, error) {
			// Return session owned by tenant 1 (same as certificate)
			return &models.SCACISession{
				ID:         1,
				TenantID:   1, // Session belongs to tenant 1, cert is also tenant 1
				AcEUI:      acEUI,
				SnAcUUID:   acUUID,
				SnScUUID:   [16]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00},
				Status:     "active",
				LastOpIDAc: 10,
				LastOpIDSc: -10,
			}, nil
		},
	}

	svc := NewHandshakeService(
		sessionRepo, log, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", scaci.ProtocolVersionString,
	)

	// Build Connect request with resume fields
	snAcOpId := int64(10)
	snScOpId := int64(-10)
	req := &scaci.Connect{
		Version:  scaci.ProtocolVersionString,
		AcEui:    0xAABBCCDDEEFF1122,
		SnAcUUID: scaci.UUID16(acUUID),
		SnAcOpId: &snAcOpId,
		SnScOpId: &snScOpId,
	}

	cert := createValidTestCert("tenant-1")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	// Assert: Same-tenant resume should succeed
	if errToken != "" {
		t.Errorf("expected no error for same-tenant resume, got: %s", errToken)
	}
	if session == nil {
		t.Fatal("expected session to be returned on successful resume")
	}
	if !session.Resumed {
		t.Error("expected session.Resumed=true")
	}
	if session.TenantID != 1 {
		t.Errorf("expected TenantID=1, got %d", session.TenantID)
	}
	if resp == nil {
		t.Fatal("expected response to be returned on successful resume")
	}
	if !resp.SnResume {
		t.Error("expected SnResume=true in response")
	}
}

// ============================================================================
// Metadata Preservation Tests (SCACI §3.3.1 info field)
// ============================================================================

// TestSessionFromModel_PreservesNestedMetadata validates that sessionFromModel
// preserves non-string metadata values when reconstructing a session from DB.
// This is critical for SCACI §3.3.1 info field support (arbitrary key-value object).
// Ref: handshake_service.go:470-475 (sessionFromModel metadata handling)
func TestSessionFromModel_PreservesNestedMetadata(t *testing.T) {
	log := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, log, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", scaci.ProtocolVersionString,
	).(*handshakeService)

	// Build DB session with nested metadata (simulates info field per §3.3.1)
	dbSession := &models.SCACISession{
		ID:          123,
		TenantID:    42,
		AcEUI:       [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22},
		SnAcUUID:    [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID:    [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		ConnectedAt: time.Now(),
		LastOpIDAc:  100,
		LastOpIDSc:  -100,
		Status:      "active",
		Metadata: map[string]interface{}{
			"vendor":    "TestVendor",
			"model":     "TestModel",
			"name":      "TestAC",
			"swVersion": "2.0.0",
			// info field: nested object per SCACI §3.3.1
			"info": map[string]interface{}{
				"firmware":     "v1.2.3",
				"serialNo":     12345,
				"capabilities": []interface{}{"downlink", "multicast"},
				"nested": map[string]interface{}{
					"deep": "value",
				},
			},
		},
	}

	// Call sessionFromModel (private method exposed via handshakeService type)
	session := svc.sessionFromModel(dbSession)

	// Assert basic fields
	if session.ID != 123 {
		t.Errorf("expected ID=123, got %d", session.ID)
	}
	if session.TenantID != 42 {
		t.Errorf("expected TenantID=42, got %d", session.TenantID)
	}

	// Assert string metadata is preserved
	if session.Metadata["vendor"] != "TestVendor" {
		t.Errorf("expected vendor=TestVendor, got %v", session.Metadata["vendor"])
	}
	if session.Metadata["model"] != "TestModel" {
		t.Errorf("expected model=TestModel, got %v", session.Metadata["model"])
	}

	// Assert nested info object is preserved (CRITICAL for §3.3.1)
	info, ok := session.Metadata["info"]
	if !ok {
		t.Fatal("info field should be preserved in metadata")
	}
	infoMap, ok := info.(map[string]interface{})
	if !ok {
		t.Fatalf("info should be map[string]interface{}, got %T", info)
	}

	// Verify nested values
	if infoMap["firmware"] != "v1.2.3" {
		t.Errorf("expected info.firmware=v1.2.3, got %v", infoMap["firmware"])
	}
	// serialNo is numeric - verify it wasn't dropped
	if infoMap["serialNo"] != 12345 {
		t.Errorf("expected info.serialNo=12345, got %v", infoMap["serialNo"])
	}
	// capabilities is a slice - verify it wasn't dropped
	caps, ok := infoMap["capabilities"].([]interface{})
	if !ok || len(caps) != 2 {
		t.Errorf("expected info.capabilities to be slice with 2 elements, got %v", infoMap["capabilities"])
	}
	// nested object - verify deep nesting preserved
	nestedObj, ok := infoMap["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected info.nested to be map, got %T", infoMap["nested"])
	}
	if nestedObj["deep"] != "value" {
		t.Errorf("expected info.nested.deep=value, got %v", nestedObj["deep"])
	}
}

// TestSessionFromModel_EmptyMetadata validates sessionFromModel handles nil/empty metadata
func TestSessionFromModel_EmptyMetadata(t *testing.T) {
	log := logger.NewNop()
	sessionRepo := &mockSCACISessionRepository{}
	orgResolver := &mockOrgResolver{}
	certVerifier := &mockCertificateVerifier{}

	svc := NewHandshakeService(
		sessionRepo, log, orgResolver, 1, false, certVerifier,
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", scaci.ProtocolVersionString,
	).(*handshakeService)

	dbSession := &models.SCACISession{
		ID:          456,
		TenantID:    1,
		AcEUI:       [8]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22},
		SnAcUUID:    [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		SnScUUID:    [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20},
		ConnectedAt: time.Now(),
		Metadata:    nil, // No metadata
	}

	session := svc.sessionFromModel(dbSession)

	// Should not panic and should create empty map
	if session.Metadata == nil {
		t.Error("Metadata should be initialized to empty map, not nil")
	}
	if len(session.Metadata) != 0 {
		t.Errorf("expected empty metadata map, got %d entries", len(session.Metadata))
	}
}

// ============================================================================
// Integration Test: Full Connect Flow with Certificate Validation
// ============================================================================

func TestValidateConnect_FullFlow_WithCertificateValidation(t *testing.T) {
	logger := logger.NewNop()

	// Track which validations were called
	certVerifierCalled := false
	orgResolverCalled := false

	sessionRepo := &mockSCACISessionRepository{}

	orgResolver := &mockOrgResolver{
		resolveCertFunc: func(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
			orgResolverCalled = true
			// Simulate cert -> org UUID + tenant ID resolution
			// In real implementation, this would parse cert CN/SAN and query the database
			return uuid.Nil, 99, nil
		},
	}

	certVerifier := &mockCertificateVerifier{
		verifyCertificateFunc: func(cert *x509.Certificate) string {
			certVerifierCalled = true
			// Validate certificate properties
			if cert == nil {
				return scaci.ErrNilCertificate
			}
			// Simulate real validation checks
			now := time.Now()
			if now.Before(cert.NotBefore) {
				return scaci.ErrCertNotYetValid
			}
			if now.After(cert.NotAfter) {
				return scaci.ErrCertExpired
			}
			// Check ExtKeyUsage
			hasClientAuth := false
			for _, usage := range cert.ExtKeyUsage {
				if usage == x509.ExtKeyUsageClientAuth {
					hasClientAuth = true
					break
				}
			}
			if !hasClientAuth {
				return scaci.ErrCertMissingClientAuth
			}
			return "" // Valid
		},
	}

	svc := NewHandshakeService(
		sessionRepo, logger, orgResolver, 1, false, certVerifier,
		0x9999888877776666, "KiloCenter", "KC-2000", "prod-sc", "1.0.5",
	)

	req := &scaci.Connect{
		Version: "1.0.0",
		AcEui:   0xDEADBEEFCAFEBABE,
		SnAcUUID: scaci.UUID16{
			0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22,
			0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00,
		},
	}

	cert := createValidTestCert("tenant-99")

	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), req, cert)

	// Verify no errors
	if errToken != "" {
		t.Errorf("expected successful connect, got error: %s", errToken)
	}

	// Verify both validations were called in correct order
	if !certVerifierCalled {
		t.Error("certificate verifier was not called")
	}
	if !orgResolverCalled {
		t.Error("org resolver was not called")
	}

	// Verify session was created with correct tenant
	if session == nil {
		t.Fatal("expected session to be created")
	}
	if session.TenantID != 99 {
		t.Errorf("expected TenantID=99, got %d", session.TenantID)
	}
	if session.AcEui != 0xDEADBEEFCAFEBABE {
		t.Errorf("expected AcEui=0xDEADBEEFCAFEBABE, got 0x%X", session.AcEui)
	}

	// Verify response was built correctly
	if resp == nil {
		t.Fatal("expected response to be created")
	}
	if resp.ScEui != 0x9999888877776666 {
		t.Errorf("expected ScEui=0x9999888877776666, got 0x%X", resp.ScEui)
	}
	if resp.Vendor == nil || *resp.Vendor != "KiloCenter" {
		t.Errorf("expected Vendor=KiloCenter, got %v", resp.Vendor)
	}
	if resp.Model == nil || *resp.Model != "KC-2000" {
		t.Errorf("expected Model=KC-2000, got %v", resp.Model)
	}
}
