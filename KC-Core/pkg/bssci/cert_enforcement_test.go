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

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeEnforcementCert creates a real self-signed certificate and its PEM.
func makeEnforcementCert(t *testing.T, cn string) (*x509.Certificate, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
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

func newEnforcementServer(t *testing.T, directory *fakeBSDirectory) (*Server, *Session, *x509.Certificate, string) {
	t.Helper()
	log := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, _, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, nil, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.bsDirectory = directory

	cert, pemData := makeEnforcementCert(t, "CA-FE-CA-FE-CA-FE-CA-FE")
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:             "cert-enforcement",
			BaseStationEUI: 0xCAFECAFECAFECAFE,
		},
		ClientCert: cert,
	}
	return server, session, cert, pemData
}

// TestVerifyCertificateFingerprint_Match: the presented certificate matching
// the stored fingerprint passes.
func TestVerifyCertificateFingerprint_Match(t *testing.T) {
	directory := &fakeBSDirectory{}
	server, session, cert, _ := newEnforcementServer(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw),
	}

	assert.NoError(t, server.verifyCertificateFingerprint(context.Background(), session))
}

// TestVerifyCertificateFingerprint_ForgedSameCNRejected: a different
// certificate with the same CN is rejected by the fingerprint comparison.
func TestVerifyCertificateFingerprint_ForgedSameCNRejected(t *testing.T) {
	directory := &fakeBSDirectory{}
	server, session, _, _ := newEnforcementServer(t, directory)
	otherCert, _ := makeEnforcementCert(t, "CA-FE-CA-FE-CA-FE-CA-FE")
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(otherCert.Raw),
	}

	assert.Error(t, server.verifyCertificateFingerprint(context.Background(), session),
		"a forged certificate sharing the CN must be rejected")
}

// TestVerifyCertificateFingerprint_BlankBackfillsFromStoredPEM: a pre-upgrade
// row (blank fingerprint, stored PEM of the same certificate) is compared
// against the presented certificate first and then backfilled.
func TestVerifyCertificateFingerprint_BlankBackfillsFromStoredPEM(t *testing.T) {
	directory := &fakeBSDirectory{backfillResult: true}
	server, session, _, pemData := newEnforcementServer(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertificate: pemData,
	}

	require.NoError(t, server.verifyCertificateFingerprint(context.Background(), session))
	assert.Equal(t, 1, directory.backfillCalls, "the derived fingerprint is persisted")
}

// TestVerifyCertificateFingerprint_BlankStoredPEMOtherCertRejected: a blank
// fingerprint with a stored PEM of a DIFFERENT certificate rejects the
// connection and never backfills.
func TestVerifyCertificateFingerprint_BlankStoredPEMOtherCertRejected(t *testing.T) {
	directory := &fakeBSDirectory{backfillResult: true}
	server, session, _, _ := newEnforcementServer(t, directory)
	_, otherPEM := makeEnforcementCert(t, "CA-FE-CA-FE-CA-FE-CA-FE")
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertificate: otherPEM,
	}

	require.Error(t, server.verifyCertificateFingerprint(context.Background(), session))
	assert.Zero(t, directory.backfillCalls, "a mismatch must never be persisted")
}

// TestVerifyCertificateFingerprint_BlankNoStoredCertRejected: a blank
// fingerprint with no stored certificate has no verifiable identity.
func TestVerifyCertificateFingerprint_BlankNoStoredCertRejected(t *testing.T) {
	directory := &fakeBSDirectory{}
	server, session, _, _ := newEnforcementServer(t, directory)
	directory.station = RegisteredBaseStation{ID: 7, TenantID: 42}

	assert.Error(t, server.verifyCertificateFingerprint(context.Background(), session))
}

// TestVerifyCertificateFingerprint_BackfillRaceReloadsAndCompares: a lost
// backfill race (zero rows updated) reloads the row and compares against the
// concurrently written fingerprint.
func TestVerifyCertificateFingerprint_BackfillRaceReloadsAndCompares(t *testing.T) {
	directory := &fakeBSDirectory{backfillResult: false}
	server, session, cert, pemData := newEnforcementServer(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertificate: pemData,
	}
	directory.reloadStation = &RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw),
	}

	require.NoError(t, server.verifyCertificateFingerprint(context.Background(), session))
	assert.Equal(t, 2, directory.getCalls, "the lost race reloads the row")

	// A concurrent writer that stored a DIFFERENT fingerprint rejects
	directory2 := &fakeBSDirectory{backfillResult: false}
	server2, session2, _, pemData2 := newEnforcementServer(t, directory2)
	directory2.station = RegisteredBaseStation{ID: 7, TenantID: 42, TLSCertificate: pemData2}
	directory2.reloadStation = &RegisteredBaseStation{ID: 7, TenantID: 42, TLSCertFingerprint: "deadbeef"}
	assert.Error(t, server2.verifyCertificateFingerprint(context.Background(), session2))
}

// TestVerifyCertificateFingerprint_LookupFailureRejects: an unreadable
// registration rejects rather than skipping enforcement.
func TestVerifyCertificateFingerprint_LookupFailureRejects(t *testing.T) {
	directory := &fakeBSDirectory{getErr: errors.New("db down")}
	server, session, _, _ := newEnforcementServer(t, directory)

	assert.Error(t, server.verifyCertificateFingerprint(context.Background(), session))
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
		OrgEnforcementEnabled: true,
		ServiceCenterEUI:      TestBsEui02,
		Vendor:                "v", Model: "m", Name: "n", SoftwareVersion: "1.0.0",
	}
	server.RegisterHandlers()

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
