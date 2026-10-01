package grpc

import (
	"context"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	certTestValidityDays = 30
	certTestFilePerm     = 0o600
	certTestFlagDir      = "-dir"
	certTestCertLifetime = 24 * time.Hour
	certTestCAFile       = "ca.crt"
	certTestCAKeyFile    = "ca.key"
	certTestClientCert   = "client.crt"
	certTestClientKey    = "client.key"
	certTestPEMCert      = "CERTIFICATE"
	certTestPEMKey       = "EC PRIVATE KEY"
	certTestStagedCA     = "staged"
	certTestTraversalID  = "../../etc"
	certTestNonUUIDID    = "abc123"
)

// ownedCertStations answers base station lookups for testOwnerTenant only,
// with the repository's wrapped not-found for every other tenant.
type ownedCertStations struct{}

func (ownedCertStations) GetByEUI(_ context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	if tenantID != testOwnerTenant {
		return nil, storage.ErrNotFound
	}
	bs := &models.BaseStation{ID: 1, TenantID: tenantID}
	copy(bs.EUI[:], eui)
	return bs, nil
}

func (ownedCertStations) Update(context.Context, int64, int64, map[string]interface{}) error {
	return nil
}

func (ownedCertStations) TakeTLSKey(context.Context, int64, []byte, func(string) error) error {
	return storage.ErrNotFound
}

type plainKeyEncryptor struct{}

func (plainKeyEncryptor) EncryptKey(key []byte) (string, error)       { return string(key), nil }
func (plainKeyEncryptor) DecryptKey(encrypted string) ([]byte, error) { return []byte(encrypted), nil }

// selfSignedCertGen writes a CA certificate and a client certificate and key
// into the -dir argument, standing in for the certgen binary.
func selfSignedCertGen(_ context.Context, _ string, args ...string) (string, string, error) {
	var dir string
	for i := 0; i+1 < len(args); i++ {
		if args[i] == certTestFlagDir {
			dir = args[i+1]
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: testOwnedBsEui}, NotAfter: time.Now().Add(certTestCertLifetime)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: certTestPEMCert, Bytes: der})
	files := map[string][]byte{
		certTestCAFile:     certPEM,
		certTestClientCert: certPEM,
		certTestClientKey:  pem.EncodeToMemory(&pem.Block{Type: certTestPEMKey, Bytes: keyDER}),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, certTestFilePerm); err != nil {
			return "", "", err
		}
	}
	return "", "", nil
}

func newCertificateHandlersOverRealService(t *testing.T) *CoreService {
	t.Helper()
	certsDir := t.TempDir()
	for _, name := range []string{certTestCAFile, certTestCAKeyFile} {
		require.NoError(t, os.WriteFile(filepath.Join(certsDir, name), []byte(certTestStagedCA), certTestFilePerm))
	}
	cfg := &config.Config{Certificates: config.CertificateConfig{CertsDir: certsDir, TempDir: filepath.Join(t.TempDir(), "bundles")}}
	certSvc, err := certificates.New(testutil.TestContext(), cfg, logger.NewNop(), ownedCertStations{}, plainKeyEncryptor{}, selfSignedCertGen, clock.SystemClock{}, &captureAuditRecorder{})
	require.NoError(t, err)
	return testCoreService(coreFields{certSvc: certSvc, log: logger.NewNop()})
}

func issueBundle(t *testing.T, svc *CoreService) string {
	t.Helper()
	resp, err := svc.GenerateCertificate(contextForTenant(testOwnerTenant), &pb.GenerateCertificateRequest{BsEui: testOwnedBsEui, ValidityDays: certTestValidityDays})
	require.NoError(t, err)
	id := resp.DownloadUrls["private_key"]
	require.NotEmpty(t, id)
	return id
}

func TestDownloadCertificate_ForeignTenantGetsNotFound(t *testing.T) {
	svc := newCertificateHandlersOverRealService(t)
	id := issueBundle(t, svc)

	for _, certType := range []string{certificates.CertTypeCA, certificates.CertTypeClient, certificates.CertTypeKey} {
		_, err := svc.DownloadCertificate(contextForTenant(testForeignTenant), &pb.DownloadCertificateRequest{Id: id, CertType: certType})
		require.Error(t, err, "another tenant must not download the %s of this bundle", certType)
		assert.Equal(t, codes.NotFound, status.Code(err))
	}

	resp, err := svc.DownloadCertificate(contextForTenant(testOwnerTenant), &pb.DownloadCertificateRequest{Id: id, CertType: certificates.CertTypeClient})
	require.NoError(t, err, "the issuing tenant downloads its own bundle")
	assert.NotEmpty(t, resp.Content)
}

func TestDownloadCertificate_RejectsIDsThatAreNotBundleIDs(t *testing.T) {
	svc := newCertificateHandlersOverRealService(t)
	issueBundle(t, svc)

	for _, id := range []string{certTestTraversalID, certTestNonUUIDID} {
		_, err := svc.DownloadCertificate(contextForTenant(testOwnerTenant), &pb.DownloadCertificateRequest{Id: id, CertType: certificates.CertTypeCA})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err), "id %q", id)
	}
}

func TestDownloadCertificate_RequiresTenant(t *testing.T) {
	svc := newCertificateHandlersOverRealService(t)
	id := issueBundle(t, svc)

	_, err := svc.DownloadCertificate(testutil.TestContext(), &pb.DownloadCertificateRequest{Id: id, CertType: certificates.CertTypeClient})
	require.Error(t, err)
	assert.NotEqual(t, codes.OK, status.Code(err))
}

func TestGenerateCertificate_BundleKeepsNoCAKey(t *testing.T) {
	certsDir := t.TempDir()
	for _, name := range []string{certTestCAFile, certTestCAKeyFile} {
		require.NoError(t, os.WriteFile(filepath.Join(certsDir, name), []byte(certTestStagedCA), certTestFilePerm))
	}
	bundles := filepath.Join(t.TempDir(), "bundles")
	cfg := &config.Config{Certificates: config.CertificateConfig{CertsDir: certsDir, TempDir: bundles}}
	certSvc, err := certificates.New(testutil.TestContext(), cfg, logger.NewNop(), ownedCertStations{}, plainKeyEncryptor{}, selfSignedCertGen, clock.SystemClock{}, &captureAuditRecorder{})
	require.NoError(t, err)
	svc := testCoreService(coreFields{certSvc: certSvc, log: logger.NewNop()})

	id := issueBundle(t, svc)
	_, err = os.Stat(filepath.Join(bundles, id, certTestCAKeyFile))
	assert.True(t, os.IsNotExist(err), "the CA private key must not stay in a downloadable bundle directory")
}
