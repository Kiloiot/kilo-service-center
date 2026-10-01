package certificates

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

var errEventStoreDown = errors.New("event store down")

const testStationKey = "station key"

// captureDisclosures records the disclosure events, failing each with err when set.
type captureDisclosures struct {
	events []audit.Event
	err    error
}

func (c *captureDisclosures) RecordRequired(_ context.Context, ev audit.Event) error {
	if c.err != nil {
		return c.err
	}
	c.events = append(c.events, ev)
	return nil
}

// keyedService serves the private key sealed into repo, recording through disclosures.
func keyedService(t *testing.T, repo *mockBaseStationRepo, disclosures *captureDisclosures) (*Service, string) {
	t.Helper()
	tempDir := t.TempDir()
	svc, err := New(testutil.TestContext(), &config.Config{Certificates: config.CertificateConfig{TempDir: tempDir}}, &mockLogger{}, repo, &mockKeyEncryptor{}, ExecCertGen, clock.SystemClock{}, disclosures)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return svc, tempDir
}

func sealedRepo(key string) *mockBaseStationRepo {
	repo := tenantScopedRepo()
	sealed := base64.StdEncoding.EncodeToString([]byte(key))
	repo.bs.TLSKey = &sealed
	return repo
}

func TestPrivateKeyDownload_RecordsTheStationNeverTheKey(t *testing.T) {
	disclosures := &captureDisclosures{}
	svc, _ := keyedService(t, sealedRepo(testStationKey), disclosures)

	key, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, testStationEUI, CertTypeKey)

	if err != nil || string(key) != testStationKey {
		t.Fatalf("download: %q, %v", key, err)
	}
	if len(disclosures.events) != 1 {
		t.Fatalf("recorded %d events, want 1", len(disclosures.events))
	}
	ev := disclosures.events[0]
	if ev.EventType != models.EventTypeCertificatePrivateKeyDownloaded || ev.TenantID != testOwnerTenant || ev.SourceName != testBundleEUICompact {
		t.Fatalf("event = %+v", ev)
	}
	if ev.BaseStationID == nil || *ev.BaseStationID != 1 {
		t.Fatalf("event names station %v, want 1", ev.BaseStationID)
	}
	written := fmt.Sprintf("%v %s", ev.Details, ev.Description)
	if strings.Contains(written, testStationKey) || strings.Contains(written, base64.StdEncoding.EncodeToString([]byte(testStationKey))) {
		t.Fatalf("the key entered the event: %s", written)
	}
}

func TestPrivateKeyDownload_KeepsTheKeyWhenItCannotBeRecorded(t *testing.T) {
	repo := sealedRepo(testStationKey)
	disclosures := &captureDisclosures{err: errEventStoreDown}
	svc, _ := keyedService(t, repo, disclosures)

	key, _, err := svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, testStationEUI, CertTypeKey)

	if !errors.Is(err, ErrKeyDownloadNotRecorded) {
		t.Fatalf("error = %v, want %v", err, ErrKeyDownloadNotRecorded)
	}
	if key != nil {
		t.Fatal("an unrecorded download handed the key out")
	}
	if repo.bs.TLSKey == nil {
		t.Fatal("an unrecorded download lost the stored key")
	}

	disclosures.err = nil
	key, _, err = svc.GetStoredCertificate(testutil.TestContext(), testOwnerTenant, testStationEUI, CertTypeKey)
	if err != nil || string(key) != testStationKey {
		t.Fatalf("retry once recording works: %q, %v", key, err)
	}
}

func TestBundlePrivateKeyDownload_KeepsTheBundleKeyWhenItCannotBeRecorded(t *testing.T) {
	repo := sealedRepo(testBundleKey)
	disclosures := &captureDisclosures{err: errEventStoreDown}
	svc, tempDir := keyedService(t, repo, disclosures)
	id := writeBundle(t, tempDir, time.Now())

	if _, _, err := svc.DownloadCertificateByID(testutil.TestContext(), testOwnerTenant, CertTypeKey, id); !errors.Is(err, ErrKeyDownloadNotRecorded) {
		t.Fatalf("error = %v, want %v", err, ErrKeyDownloadNotRecorded)
	}
	if _, err := os.Stat(filepath.Join(tempDir, id, clientKeyFileName)); err != nil {
		t.Fatalf("the bundle's key file was removed: %v", err)
	}
	if repo.bs.TLSKey == nil {
		t.Fatal("the station's stored key was lost")
	}
}

func TestPrivateKeyDownload_RefusalsRecordNothing(t *testing.T) {
	cases := map[string]struct {
		repo   *mockBaseStationRepo
		tenant int64
	}{
		"another tenant": {repo: sealedRepo(testStationKey), tenant: testForeignTenant},
		"no stored key":  {repo: tenantScopedRepo(), tenant: testOwnerTenant},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			disclosures := &captureDisclosures{}
			svc, _ := keyedService(t, tc.repo, disclosures)

			if _, _, err := svc.GetStoredCertificate(testutil.TestContext(), tc.tenant, testStationEUI, CertTypeKey); err == nil {
				t.Fatal("the key was handed out")
			}
			if len(disclosures.events) != 0 {
				t.Fatalf("a refused download recorded %d events", len(disclosures.events))
			}
		})
	}
}
