package certificates

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	testServiceCAPEM = "service-center-ca"
	testStaleCAPEM   = "stale-per-station-copy"
)

var testStationEUI = []byte{0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE, 0xCA, 0xFE}

// newStoredCertService serves station downloads for repo from a certificate
// directory holding caPEM as the service center CA, or no CA when empty.
func newStoredCertService(t *testing.T, repo *mockBaseStationRepo, caPEM string) *Service {
	t.Helper()
	certsDir := t.TempDir()
	if caPEM != "" {
		if err := os.WriteFile(filepath.Join(certsDir, caCertFileName), []byte(caPEM), testFilePerm); err != nil {
			t.Fatalf("stage CA: %v", err)
		}
	}
	cfg := &config.Config{Certificates: config.CertificateConfig{CertsDir: certsDir, TempDir: t.TempDir()}}
	svc, err := New(testutil.TestContext(), cfg, &mockLogger{}, repo, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, &captureDisclosures{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return svc
}

func TestGetStoredCertificate_CAIsTheServiceCenterCA(t *testing.T) {
	stale := testStaleCAPEM
	for name, station := range map[string]*models.BaseStation{
		"no stored copy":    {ID: 1, TenantID: testOwnerTenant},
		"stale stored copy": {ID: 1, TenantID: testOwnerTenant, TLSCACertificate: &stale},
	} {
		t.Run(name, func(t *testing.T) {
			repo := &mockBaseStationRepo{bs: station, ownerTenant: testOwnerTenant}
			svc := newStoredCertService(t, repo, testServiceCAPEM)

			data, filename, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, testStationEUI, CertTypeCA)
			if err != nil {
				t.Fatalf("CA download: %v", err)
			}
			if string(data) != testServiceCAPEM {
				t.Fatalf("content = %q, want the service center CA %q", data, testServiceCAPEM)
			}
			if want := fmt.Sprintf(downloadNameFmtBSCACert, testBundleEUICompact); filename != want {
				t.Fatalf("filename = %q, want %q", filename, want)
			}
		})
	}
}

func TestGetStoredCertificate_CAOfAForeignStationIsNotFound(t *testing.T) {
	svc := newStoredCertService(t, tenantScopedRepo(), testServiceCAPEM)

	data, _, err := svc.GetStoredCertificate(testutil.TestContext(), testForeignTenant, testStationEUI, CertTypeCA)
	if !errors.Is(err, ErrBaseStationNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrBaseStationNotFound)
	}
	if data != nil {
		t.Fatalf("a foreign tenant received %q", data)
	}
}

func TestGetStoredCertificate_CAUnreadableIsACARead(t *testing.T) {
	svc := newStoredCertService(t, tenantScopedRepo(), "")

	_, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, testStationEUI, CertTypeCA)
	if !errors.Is(err, ErrCACertRead) {
		t.Fatalf("error = %v, want %v", err, ErrCACertRead)
	}
}

func TestGetStoredCertificate_ClientWithoutStoredCopyIsNotStored(t *testing.T) {
	svc := newStoredCertService(t, tenantScopedRepo(), testServiceCAPEM)

	_, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, testStationEUI, CertTypeClient)
	if !errors.Is(err, ErrCertificateNotStored) {
		t.Fatalf("error = %v, want %v", err, ErrCertificateNotStored)
	}
}

func TestGetStoredCertificate_StoredClientCertificateIsServed(t *testing.T) {
	stored := "station-client-cert"
	repo := &mockBaseStationRepo{bs: &models.BaseStation{ID: 1, TenantID: testOwnerTenant, TLSCertificate: &stored}, ownerTenant: testOwnerTenant}
	svc := newStoredCertService(t, repo, testServiceCAPEM)

	data, filename, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, testStationEUI, CertTypeClient)
	if err != nil {
		t.Fatalf("client download: %v", err)
	}
	if string(data) != stored {
		t.Fatalf("content = %q, want %q", data, stored)
	}
	if want := fmt.Sprintf(downloadNameFmtClientCert, testBundleEUICompact); filename != want {
		t.Fatalf("filename = %q, want %q", filename, want)
	}
}
