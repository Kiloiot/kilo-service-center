package bssci

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// enforcementCertNotAfter is the expiry of every enforcement test certificate.
var enforcementCertNotAfter = time.Date(2036, time.January, 2, 3, 4, 5, 0, time.UTC)

// makeEnforcementCert creates a real self-signed certificate and its PEM.
func makeEnforcementCert(t *testing.T, cn string) (*x509.Certificate, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotAfter:     enforcementCertNotAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return cert, string(pemData)
}

// fakeBSDirectory implements RegisteredBaseStationDirectory for enforcement tests.
type fakeBSDirectory struct {
	station        RegisteredBaseStation
	getErr         error
	backfillResult bool
	backfillErr    error
	backfillCalls  int
	// reloadStation is returned by the GetGlobal call after a lost backfill race
	reloadStation *RegisteredBaseStation
	getCalls      int
	expiryErr     error
	// expiries are the certificate expiries the binder backfilled
	expiries []time.Time
}

func (d *fakeBSDirectory) GetGlobal(_ context.Context, _ uint64) (RegisteredBaseStation, error) {
	d.getCalls++
	if d.getErr != nil {
		return RegisteredBaseStation{}, d.getErr
	}
	if d.getCalls > 1 && d.reloadStation != nil {
		return *d.reloadStation, nil
	}
	return d.station, nil
}

func (d *fakeBSDirectory) BackfillFingerprintIfBlank(_ context.Context, _, _ int64, _ string) (bool, error) {
	d.backfillCalls++
	return d.backfillResult, d.backfillErr
}

func (d *fakeBSDirectory) BackfillCertExpiryIfBlank(_ context.Context, _, _ int64, expiresAt time.Time) (bool, error) {
	if d.expiryErr != nil {
		return false, d.expiryErr
	}
	d.expiries = append(d.expiries, expiresAt)
	return true, nil
}

// newEnforcementBinder binds a station claim whose certificate names no
// station, such as an organization certificate, over the directory.
func newEnforcementBinder(t *testing.T, directory *fakeBSDirectory) (StationCertificateBinder, StationCertificateClaim, *x509.Certificate, string) {
	t.Helper()
	binder, err := NewStationCertificateBinder(directory, logger.NewNop())
	require.NoError(t, err)
	cert, pemData := makeEnforcementCert(t, "CA-FE-CA-FE-CA-FE-CA-FE")
	return binder, StationCertificateClaim{BaseStationEUI: 0xCAFECAFECAFECAFE, Certificate: cert}, cert, pemData
}

// TestBindStationCertificate_Match: the presented certificate matching
// the stored fingerprint passes.
func TestBindStationCertificate_Match(t *testing.T) {
	directory := &fakeBSDirectory{}
	binder, claim, cert, _ := newEnforcementBinder(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw),
	}

	assert.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
}

// TestBindStationCertificate_ForgedSameCNRejected: a different
// certificate with the same CN is rejected by the fingerprint comparison.
func TestBindStationCertificate_ForgedSameCNRejected(t *testing.T) {
	directory := &fakeBSDirectory{}
	binder, claim, _, _ := newEnforcementBinder(t, directory)
	otherCert, _ := makeEnforcementCert(t, "CA-FE-CA-FE-CA-FE-CA-FE")
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(otherCert.Raw),
	}

	assert.Error(t, binder.BindStationCertificate(testutil.TestContext(), claim),
		"a forged certificate sharing the CN must be rejected")
}

// TestBindStationCertificate_BlankBackfillsFromStoredPEM: a pre-upgrade
// row (blank fingerprint, stored PEM of the same certificate) is compared
// against the presented certificate first and then backfilled.
func TestBindStationCertificate_BlankBackfillsFromStoredPEM(t *testing.T) {
	directory := &fakeBSDirectory{backfillResult: true}
	binder, claim, _, pemData := newEnforcementBinder(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertificate: pemData,
	}

	require.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Equal(t, 1, directory.backfillCalls, "the derived fingerprint is persisted")
}

// TestBindStationCertificate_BlankStoredPEMOtherCertRejected: a blank
// fingerprint with a stored PEM of a DIFFERENT certificate rejects the
// connection and never backfills.
func TestBindStationCertificate_BlankStoredPEMOtherCertRejected(t *testing.T) {
	directory := &fakeBSDirectory{backfillResult: true}
	binder, claim, _, _ := newEnforcementBinder(t, directory)
	_, otherPEM := makeEnforcementCert(t, "CA-FE-CA-FE-CA-FE-CA-FE")
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertificate: otherPEM,
	}

	require.Error(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Zero(t, directory.backfillCalls, "a mismatch must never be persisted")
}

// TestBindStationCertificate_BlankNoStoredCertRejected: a blank
// fingerprint with no stored certificate has no verifiable identity.
func TestBindStationCertificate_BlankNoStoredCertRejected(t *testing.T) {
	directory := &fakeBSDirectory{}
	binder, claim, _, _ := newEnforcementBinder(t, directory)
	directory.station = RegisteredBaseStation{ID: 7, TenantID: 42}

	assert.Error(t, binder.BindStationCertificate(testutil.TestContext(), claim))
}

// TestBindStationCertificate_BackfillRaceReloadsAndCompares: a lost
// backfill race (zero rows updated) reloads the row and compares against the
// concurrently written fingerprint.
func TestBindStationCertificate_BackfillRaceReloadsAndCompares(t *testing.T) {
	directory := &fakeBSDirectory{backfillResult: false}
	binder, claim, cert, pemData := newEnforcementBinder(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertificate: pemData,
	}
	directory.reloadStation = &RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw),
	}

	require.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Equal(t, 2, directory.getCalls, "the lost race reloads the row")

	// A concurrent writer that stored a DIFFERENT fingerprint rejects
	directory2 := &fakeBSDirectory{backfillResult: false}
	binder2, claim2, _, pemData2 := newEnforcementBinder(t, directory2)
	directory2.station = RegisteredBaseStation{ID: 7, TenantID: 42, TLSCertificate: pemData2}
	directory2.reloadStation = &RegisteredBaseStation{ID: 7, TenantID: 42, TLSCertFingerprint: "deadbeef"}
	assert.Error(t, binder2.BindStationCertificate(testutil.TestContext(), claim2))
}

// TestBindStationCertificate_LookupFailureRejects: an unreadable
// registration rejects rather than skipping enforcement.
func TestBindStationCertificate_LookupFailureRejects(t *testing.T) {
	directory := &fakeBSDirectory{getErr: errDbDown}
	binder, claim, _, _ := newEnforcementBinder(t, directory)

	assert.Error(t, binder.BindStationCertificate(testutil.TestContext(), claim))
}

// TestBindStationCertificate_NoCertificateRefused: a connection that
// presented no client certificate cannot be bound to a station.
func TestBindStationCertificate_NoCertificateRefused(t *testing.T) {
	directory := &fakeBSDirectory{station: RegisteredBaseStation{ID: 7, TenantID: 42}}
	binder, claim, _, _ := newEnforcementBinder(t, directory)
	claim.Certificate = nil

	assert.Error(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Zero(t, directory.getCalls, "the registration is not read")
}

// TestBindStationCertificate_FirstCertificateNamingTheStationPinned: a
// station registered with no certificate is pinned to the first certificate
// that names it; the pin is kept when a concurrent connect pinned the same
// certificate and refused when it pinned another.
func TestBindStationCertificate_FirstCertificateNamingTheStationPinned(t *testing.T) {
	station := uint64(0xCAFECAFECAFECAFE)
	directory := &fakeBSDirectory{station: RegisteredBaseStation{ID: 7, TenantID: 42}, backfillResult: true}
	binder, claim, cert, _ := newEnforcementBinder(t, directory)
	claim.SubjectEUI = &station

	require.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Equal(t, 1, directory.backfillCalls)

	raced := &fakeBSDirectory{station: RegisteredBaseStation{ID: 7, TenantID: 42},
		reloadStation: &RegisteredBaseStation{ID: 7, TenantID: 42, TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw)}}
	binder, _, _, _ = newEnforcementBinder(t, raced)
	require.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim), "the same certificate was pinned concurrently")

	raced.getCalls, raced.reloadStation = 0, &RegisteredBaseStation{ID: 7, TenantID: 42, TLSCertFingerprint: "deadbeef"}
	assert.Error(t, binder.BindStationCertificate(testutil.TestContext(), claim), "another certificate was pinned concurrently")
}

// TestBindStationCertificate_NilCollaboratorsRefused: the binder needs its
// directory and logger.
func TestBindStationCertificate_NilCollaboratorsRefused(t *testing.T) {
	_, err := NewStationCertificateBinder(nil, logger.NewNop())
	assert.ErrorIs(t, err, errNilRegisteredStationDirectory)
	_, err = NewStationCertificateBinder(&fakeBSDirectory{}, nil)
	assert.ErrorIs(t, err, errNilStationCertificateLogger)
}

// TestConnectStrictMode_CertSubjectEUIMismatchRejected: in strict mode an
// EUI-CN certificate bound to a different station than the connect bsEui is
// rejected indistinguishably from an unregistered station.
func TestConnectStrictMode_CertSubjectEUIMismatchRejected(t *testing.T) {
	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, interopConnectionService{}, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{
		MessageEncoding:       EncodingJSON,
		OrgEnforcementEnabled: orgEnforcementOn,
		ServiceCenterEUI:      TestBsEui02,
		Vendor:                "v", Model: "m", Name: "n", SoftwareVersion: "1.0.0",
	}
	bindStationCertificates(t, server, &fakeBSDirectory{})

	certEUI := uint64(0x1111111111111111) // bound to a DIFFERENT station
	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "strict-eui-mismatch",
			Encoding:         EncodingJSON,
			ResolvedTenantID: 1, // matches the registered tenant
			ConnectState:     ConnectStateAwaitingConnect,
		},
		Conn:           conn,
		certSubjectEUI: &certEUI,
	}

	payload := connectPayload("1.0.0", uint64(TestBsEui01))
	msg := &Message{Command: payload["command"].(string), OpId: 0, Data: payload}

	require.NoError(t, server.CallHandleConnect(session, msg, payload))

	require.GreaterOrEqual(t, conn.MessageCount(), 1)
	errFrame := conn.GetMessage(conn.MessageCount() - 1)
	assert.Equal(t, "error", errFrame["command"],
		"an EUI-CN certificate bound to another station must be rejected")
	assert.Equal(t, ConnectStateAwaitingConnectErrorAck, session.ConnectState)
}

// Sentinel errors returned by this package; callers match them with errors.Is.
var (
	errDbDown = errors.New("db down")
)
