package bssci

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

var errReconcileTestStore = errors.New("session store unavailable")

// recordingReconciler counts reconciliations and fails them on request.
type recordingReconciler struct {
	calls atomic.Int32
	err   error
}

func (r *recordingReconciler) ReconcileAbandonedSessions(context.Context) error {
	r.calls.Add(1)
	return r.err
}

// newFullyWiredServer builds a server whose every required collaborator is
// present; certificate files are absent, so Start defers the listener and
// never serves a connection.
func newFullyWiredServer(t *testing.T, reconciler AbandonedSessionReconciler) *Server {
	t.Helper()
	certDir := t.TempDir()
	cfg := &Config{
		ListenAddr:              "127.0.0.1:0",
		TLSCert:                 filepath.Join(certDir, "server.crt"),
		TLSKey:                  filepath.Join(certDir, "server.key"),
		TLSCACert:               filepath.Join(certDir, "ca.crt"),
		CertificatePollInterval: time.Hour,
		SocketWriteTimeout:      TestFrameWriteTimeout,
	}
	server, err := NewServer(cfg, logger.NewNop(), Dependencies{
		Clock: clock.SystemClock{},
		Protocol: ProtocolServices{
			Session:            struct{ SessionService }{},
			VersionNegotiator:  staticTestVersionNegotiator{},
			Downlink:           struct{ DownlinkService }{},
			Status:             struct{ StatusService }{},
			ConnectionRegistry: struct{ BaseStationConnectionRegistry }{},
			QueueSerializer:    struct{ QueueSerializer }{},
			AuditLogger:        struct{ AuditLogger }{},
			TenantResolver:     struct{ TenantResolver }{},
			SessionReconciler:  reconciler,
			ServingStations:    struct{ ServingStationLocator }{},
			AttachmentDecider:  struct{ AttachmentDecider }{},
			StationEvents:      struct{ StationEventRecorder }{},
		},
		Storage: StorageContracts{
			Events:            struct{ EventStore }{},
			BaseStations:      struct{ BaseStationStore }{},
			Endpoints:         struct{ EndpointDirectory }{},
			EndpointOwners:    struct{ EndpointOwnerResolver }{},
			AttachPersistence: struct{ EndpointAttachmentPersistence }{},
			SessionKeys:       struct{ NetworkSessionKeySource }{},
			ProtocolMessages:  struct{ ProtocolMessageStore }{},
			DLRXStatus:        struct{ DLRXStatusStore }{},
			BaseStationStatus: struct{ BaseStationStatusStore }{},
			DownlinkQueue:     struct{ DownlinkQueueStore }{},
			DownlinkRevoke:    struct{ DownlinkRevocationStore }{},
			PendingDownlinks:  struct{ PendingDownlinkLister }{},
		},
		Identity: IdentityResolvers{
			OrgDirectory:        struct{ OrganizationDirectory }{},
			CertIdentity:        struct{ CertificateIdentityResolver }{},
			StationCertificates: struct{ StationCertificateBinder }{},
		},
		Ingest: IngestPipeline{
			Uplink:            struct{ UplinkIngestService }{},
			Disposition:       struct{ IngressDispositionResolver }{},
			BlueprintDecoder:  struct{ BlueprintDecoder }{},
			BlueprintResolver: struct{ BlueprintResolver }{},
		},
	})
	require.NoError(t, err)
	require.NoError(t, server.ConfigureRuntime(newRuntimeDeps()))
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	return server
}

// Start returns the sessions a crashed previous process left active to the
// resumable state before it accepts a connection.
func TestStartReconcilesAbandonedSessions(t *testing.T) {
	reconciler := &recordingReconciler{}
	server := newFullyWiredServer(t, reconciler)

	require.NoError(t, server.Start())
	assert.Equal(t, int32(1), reconciler.calls.Load(), "Start reconciles exactly once")
}

// A reconciliation that cannot run fails Start instead of serving with
// sessions no base station can resume.
func TestStartFailsWhenReconciliationFails(t *testing.T) {
	reconciler := &recordingReconciler{err: errReconcileTestStore}
	server := newFullyWiredServer(t, reconciler)

	err := server.Start()
	require.ErrorIs(t, err, errReconcileTestStore)
	assert.Nil(t, server.listener, "no listener starts after a failed reconciliation")
}
