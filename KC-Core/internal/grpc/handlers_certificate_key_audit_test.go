package grpc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const certTestStoredKey = "station private key"

// keyedCertStations owns one station of testOwnerTenant holding a stored
// private key, handed out once like the repository's row lock does.
type keyedCertStations struct {
	ownedCertStations
	mu  sync.Mutex
	key *string
}

func newKeyedCertStations() *keyedCertStations {
	key := certTestStoredKey
	return &keyedCertStations{key: &key}
}

func (s *keyedCertStations) GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	bs, err := s.ownedCertStations.GetByEUI(ctx, tenantID, eui)
	if err != nil {
		return nil, err
	}
	bs.ID = testOwnedBaseStationID
	return bs, nil
}

func (s *keyedCertStations) TakeTLSKey(_ context.Context, tenantID int64, _ []byte, open func(string) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tenantID != testOwnerTenant || s.key == nil {
		return storage.ErrNotFound
	}
	if err := open(*s.key); err != nil {
		return err
	}
	s.key = nil
	return nil
}

func (s *keyedCertStations) store(sealed string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key = &sealed
}

func (s *keyedCertStations) stored() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.key != nil
}

// keyDownloadService serves certificate RPCs over stations, recording through recorder.
func keyDownloadService(t *testing.T, stations *keyedCertStations, recorder *audit.Recorder) *CoreService {
	t.Helper()
	cfg := &config.Config{Certificates: config.CertificateConfig{CertsDir: t.TempDir(), TempDir: filepath.Join(t.TempDir(), "bundles")}}
	certSvc, err := certificates.New(testutil.TestContext(), cfg, logger.NewNop(), stations, plainKeyEncryptor{}, selfSignedCertGen, clock.SystemClock{}, recorder)
	require.NoError(t, err)
	return testCoreService(coreFields{certSvc: certSvc, audit: recorder, log: logger.NewNop()})
}

func keyDownloadCtx() context.Context {
	return pkgcontext.WithUserID(testutil.TestContextWithTenant(testOwnerTenant), auditTestActor)
}

func TestDownloadBaseStationPrivateKey_RecordsWhoDownloadedIt(t *testing.T) {
	recorder, capture := productionRecorder(t)
	svc := keyDownloadService(t, newKeyedCertStations(), recorder)

	resp, err := svc.DownloadBaseStationCertificate(keyDownloadCtx(), &pb.DownloadBaseStationCertificateRequest{BsEui: testOwnedBsEui, CertType: certificates.CertTypeKey})

	require.NoError(t, err)
	assert.Equal(t, certTestStoredKey, string(resp.GetContent()))
	require.Len(t, capture.events, 1, "the private key never leaves unrecorded")
	event := capture.events[0]
	assert.Equal(t, models.EventTypeCertificatePrivateKeyDownloaded, event.EventType)
	assert.Equal(t, auditTestActor, event.UserID, "the event names who downloaded the key")
	assert.Equal(t, testOwnerTenant, event.TenantID)
	require.NotNil(t, event.BaseStationID)
	assert.Equal(t, testOwnedBaseStationID, *event.BaseStationID)
	assert.NotContains(t, fmt.Sprintf("%v %s", event.Details, event.Description), certTestStoredKey, "the key never enters the event")
}

func TestDownloadBaseStationPrivateKey_ForeignTenantRecordsNothing(t *testing.T) {
	recorder, capture := productionRecorder(t)
	stations := newKeyedCertStations()
	svc := keyDownloadService(t, stations, recorder)
	foreign := pkgcontext.WithUserID(testutil.TestContextWithTenant(testForeignTenant), auditTestActor)

	resp, err := svc.DownloadBaseStationCertificate(foreign, &pb.DownloadBaseStationCertificateRequest{BsEui: testOwnedBsEui, CertType: certificates.CertTypeKey})

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Empty(t, capture.events)
	assert.True(t, stations.stored(), "another tenant's refusal leaves the key stored")
}

func TestDownloadBaseStationPrivateKey_IsRefusedAndKeptWhenItCannotBeRecorded(t *testing.T) {
	log := newCapturingLogger()
	stations := newKeyedCertStations()
	svc := keyDownloadService(t, stations, storeBackedRecorder(t, failingEventStore{}, log))

	resp, err := svc.DownloadBaseStationCertificate(keyDownloadCtx(), &pb.DownloadBaseStationCertificateRequest{BsEui: testOwnedBsEui, CertType: certificates.CertTypeKey})

	assert.Nil(t, resp, "the private key never leaves unrecorded")
	st := status.Convert(err)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError), st.Message())
	assert.True(t, stations.stored(), "a download that could not be recorded keeps the key for a retry")
	assert.NotContains(t, fmt.Sprintf("%s %v", st.Message(), log.entries), certTestStoredKey)

	recorder, capture := productionRecorder(t)
	retry := keyDownloadService(t, stations, recorder)
	resp, err = retry.DownloadBaseStationCertificate(keyDownloadCtx(), &pb.DownloadBaseStationCertificateRequest{BsEui: testOwnedBsEui, CertType: certificates.CertTypeKey})
	require.NoError(t, err, "once the event store is back the key is handed out")
	assert.Equal(t, certTestStoredKey, string(resp.GetContent()))
	assert.Len(t, capture.events, 1)
}

func TestDownloadBundlePrivateKey_IsRefusedAndKeptWhenItCannotBeRecorded(t *testing.T) {
	stations := newKeyedCertStations()
	certsDir := t.TempDir()
	for _, name := range []string{certTestCAFile, certTestCAKeyFile} {
		require.NoError(t, os.WriteFile(filepath.Join(certsDir, name), []byte(certTestStagedCA), certTestFilePerm))
	}
	bundles := filepath.Join(t.TempDir(), "bundles")
	cfg := &config.Config{Certificates: config.CertificateConfig{CertsDir: certsDir, TempDir: bundles}}
	certSvc, err := certificates.New(testutil.TestContext(), cfg, logger.NewNop(), stations, plainKeyEncryptor{}, selfSignedCertGen, clock.SystemClock{},
		storeBackedRecorder(t, failingEventStore{}, logger.NewNop()))
	require.NoError(t, err)
	svc := testCoreService(coreFields{certSvc: certSvc, log: logger.NewNop()})
	id := issueBundle(t, svc)
	bundleKey, err := os.ReadFile(filepath.Join(bundles, id, certTestClientKey))
	require.NoError(t, err)
	stations.store(string(bundleKey))

	resp, err := svc.DownloadCertificate(keyDownloadCtx(), &pb.DownloadCertificateRequest{Id: id, CertType: certificates.CertTypeKey})

	assert.Nil(t, resp)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), status.Code(err))
	assert.True(t, stations.stored(), "the station keeps its key")
	_, statErr := os.Stat(filepath.Join(bundles, id, certTestClientKey))
	assert.NoError(t, statErr, "the bundle keeps its key file for a retry")
}
