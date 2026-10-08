package grpc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	statusTestCASubject     = "KiloCenter Test CA"
	statusTestServerSubject = "sc.kilocenter.test"
	statusTestServerFile    = "server.crt"
	statusTestCADays        = 7300
	statusTestServerDays    = 10
	statusTestCASerial      = 1
	statusTestServerSerial  = 2
)

var statusTestNow = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

type statusTestCert struct {
	subject   string
	issuer    string
	notBefore time.Time
	notAfter  time.Time
}

// writeServingCertificates writes a CA and a server certificate it signed into dir.
func writeServingCertificates(t *testing.T, dir string, ca, server statusTestCert) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(statusTestCASerial),
		Subject:               pkix.Name{CommonName: ca.subject},
		NotBefore:             ca.notBefore,
		NotAfter:              ca.notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(statusTestServerSerial),
		Subject:      pkix.Name{CommonName: server.subject},
		NotBefore:    server.notBefore,
		NotAfter:     server.notAfter,
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	require.NoError(t, err)

	for name, der := range map[string][]byte{certTestCAFile: caDER, statusTestServerFile: serverDER} {
		data := pem.EncodeToMemory(&pem.Block{Type: certTestPEMCert, Bytes: der})
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, certTestFilePerm))
	}
}

func assertCertificateStatus(t *testing.T, want statusTestCert, days int32, valid bool, got *pb.CertificateStatus) {
	t.Helper()
	require.NotNil(t, got)
	assert.Equal(t, want.subject, got.Subject)
	assert.Equal(t, want.issuer, got.Issuer)
	assert.True(t, want.notBefore.Equal(got.NotBefore.AsTime()), "not before: %v", got.NotBefore.AsTime())
	assert.True(t, want.notAfter.Equal(got.NotAfter.AsTime()), "not after: %v", got.NotAfter.AsTime())
	assert.Equal(t, days, got.DaysUntilExpiry)
	assert.Equal(t, valid, got.IsValid, "valid exactly while now is inside the validity window")
}

func TestGetServerCertificateStatus_ReportsBothCertificatesFromTheInjectedClock(t *testing.T) {
	certsDir := t.TempDir()
	issued := statusTestNow.Add(-certTestCertLifetime)
	ca := statusTestCert{subject: statusTestCASubject, issuer: statusTestCASubject, notBefore: issued, notAfter: statusTestNow.AddDate(0, 0, statusTestCADays)}
	server := statusTestCert{subject: statusTestServerSubject, issuer: statusTestCASubject, notBefore: issued, notAfter: statusTestNow.AddDate(0, 0, statusTestServerDays)}
	writeServingCertificates(t, certsDir, ca, server)

	resp := certificateStatusAt(t, certsDir, statusTestNow)

	assertCertificateStatus(t, ca, statusTestCADays, true, resp.CaCert)
	assertCertificateStatus(t, server, statusTestServerDays, true, resp.ServerCert)
}

// The status names what a renewal would issue, so the admin sees it before confirming.
func TestGetServerCertificateStatus_NamesWhatARenewalIssues(t *testing.T) {
	certsDir := t.TempDir()
	issued := statusTestNow.Add(-certTestCertLifetime)
	ca := statusTestCert{subject: statusTestCASubject, issuer: statusTestCASubject, notBefore: issued, notAfter: statusTestNow.AddDate(0, 0, statusTestCADays)}
	server := statusTestCert{subject: statusTestServerSubject, issuer: statusTestCASubject, notBefore: issued, notAfter: statusTestNow.AddDate(0, 0, statusTestServerDays)}
	writeServingCertificates(t, certsDir, ca, server)

	resp := certificateStatusAt(t, certsDir, statusTestNow)

	assert.Equal(t, []string{statusTestServerSubject}, resp.GetRenewalNames())
}

func TestGetServerCertificateStatus_AnExpiredCertificateIsInvalid(t *testing.T) {
	certsDir := t.TempDir()
	issued := statusTestNow.AddDate(0, 0, -statusTestCADays)
	ca := statusTestCert{subject: statusTestCASubject, issuer: statusTestCASubject, notBefore: issued, notAfter: statusTestNow.AddDate(0, 0, statusTestCADays)}
	server := statusTestCert{subject: statusTestServerSubject, issuer: statusTestCASubject, notBefore: issued, notAfter: statusTestNow.AddDate(0, 0, -statusTestServerDays)}
	writeServingCertificates(t, certsDir, ca, server)

	resp := certificateStatusAt(t, certsDir, statusTestNow)

	assertCertificateStatus(t, server, -statusTestServerDays, false, resp.ServerCert)
}

func certificateStatusAt(t *testing.T, certsDir string, now time.Time) *pb.GetServerCertificateStatusResponse {
	t.Helper()
	cfg := &config.Config{Certificates: config.CertificateConfig{CertsDir: certsDir, TempDir: t.TempDir()}}
	certSvc, err := certificates.New(testutil.TestContext(), cfg, logger.NewNop(), ownedCertStations{}, plainKeyEncryptor{}, certificates.ExecCertGen, fixedHandlerClock{now: now}, &captureAuditRecorder{})
	require.NoError(t, err)
	svc := testCoreService(coreFields{certSvc: certSvc, log: logger.NewNop()})
	resp, err := svc.GetServerCertificateStatus(contextForTenant(testOwnerTenant), &pb.GetServerCertificateStatusRequest{})
	require.NoError(t, err)
	return resp
}
