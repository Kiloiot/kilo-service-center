package bssciservices

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgbssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	endpointstate "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
)

// BSSCI §1, §3.9: a resumed station is sent the detachments decided while it
// was away. A detach that found the endpoint already detached decides
// nothing, so a station that was connected at the real detachment, and was
// sent it then, is not sent it again on resume.
func TestReconcileSendsOnlyDetachmentsDecidedWhileTheStationWasAway(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	endpointID := seedConcurrentAttachEndpoint(t, db, nil)
	endpoints := postgres.NewRepositories(db).Endpoints
	changed, err := endpoints.TransitionEndpointStatus(ctx, concurrentAttachTenant, endpointID, pkgbssci.EndpointStatusAttached)
	require.NoError(t, err)
	require.True(t, changed)

	detached, err := endpointstate.DetachEndpoint(ctx, endpoints, concurrentAttachTenant, endpointID, nil)
	require.NoError(t, err)
	require.True(t, detached, "the detachment")
	stationLeft := time.Now()
	detached, err = endpointstate.DetachEndpoint(ctx, endpoints, concurrentAttachTenant, endpointID, nil)
	require.NoError(t, err)
	require.False(t, detached, "a detach of an endpoint already detached changes nothing")

	stations := &mockAttachPropagateSender{}
	svc := NewPropagationService(endpoints, stations, logger.NewNop())
	require.NoError(t, svc.ReconcileBaseStation(ctx,
		propagation.BaseStationSession{ID: "station-connected-at-the-detachment", TenantID: concurrentAttachTenant, Resumed: true, DisconnectedAt: &stationLeft}, nil))

	assert.Empty(t, stations.detaches, "the station was sent the detachment when it was decided")
}

// BSSCI §1, §3.8: a resumed station keeps the attachments it held and the
// downlinks it queued for them, so an endpoint attached before it left is not
// sent again. An over-the-air attach another station heard meanwhile gives the
// endpoint a new attachment without changing its status; the resumed station
// holds the old one and is sent the new one.
func TestReconcileSendsAResumedStationTheAttachmentsDecidedWhileItWasAway(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	endpointID := seedConcurrentAttachEndpoint(t, db, nil)
	endpoints := postgres.NewRepositories(db).Endpoints
	decider, err := NewEndpointAttachmentDecider(endpoints, &recordingNotifier{})
	require.NoError(t, err)
	attach := func(report *pkgbssci.OverTheAirReport) {
		_, err := decider.Decide(ctx, pkgbssci.AttachmentDecision{
			TenantID: concurrentAttachTenant, EndpointID: endpointID, EpEUI: concurrentAttachEUI,
			Status: pkgbssci.EndpointStatusAttached, OverTheAir: report,
		})
		require.NoError(t, err)
	}
	attach(nil)
	stationLeft := time.Now()
	stations := &mockAttachPropagateSender{}
	svc := NewPropagationService(endpoints, stations, logger.NewNop())
	resumed := propagation.BaseStationSession{ID: "station-away", TenantID: concurrentAttachTenant, Resumed: true, DisconnectedAt: &stationLeft}

	require.NoError(t, svc.ReconcileBaseStation(ctx, resumed, nil))
	assert.Empty(t, stations.calls, "the station holds the attachment decided before it left")

	attach(&pkgbssci.OverTheAirReport{BaseStationEUI: reconcileHeardByOtherStation})
	require.NoError(t, svc.ReconcileBaseStation(ctx, resumed, nil))
	require.Len(t, stations.calls, 1, "the attachment decided while the station was away is sent to it")
	assert.Equal(t, endpointID, stations.calls[0].endpoint.ID)
}

// completingStation is a base station that takes every attach propagate it is
// sent: the service center reclaims the endpoint's downlinks the station
// queued before, as the attPrp completion does.
type completingStation struct {
	eui       uint64
	reclaimer pkgbssci.DownlinkReclaimer
	attached  []int64
}

func (s *completingStation) SendAttachPropagateBySessionID(ctx context.Context, _ string, endpoint *models.EndPoint) error {
	s.attached = append(s.attached, endpoint.ID)
	_, err := s.reclaimer.ReclaimEndpointQueue(ctx, endpoint.TenantID, endpoint.EUI.ToUint64(), s.eui, time.Now())
	return err
}

func (s *completingStation) SendDetachPropagate(string, uint64) error { return nil }

// BSSCI §1, §3.8: a station that resumes its session without the endpoint
// having been attached again keeps the attachment and the downlink it queued
// for it, so the downlink goes out at the endpoint's next window. Once the
// endpoint attaches again over the air while the station is away, the
// resumed station is sent the new attachment and the downlink it held, which
// that attPrp discards, returns to pending.
func TestResumeWithoutReattachKeepsHeldDownlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	endpointID := seedConcurrentAttachEndpoint(t, db, nil)
	repos := postgres.NewRepositories(db)
	decider, err := NewEndpointAttachmentDecider(repos.Endpoints, &recordingNotifier{})
	require.NoError(t, err)
	attach := func(report *pkgbssci.OverTheAirReport) {
		_, err := decider.Decide(ctx, pkgbssci.AttachmentDecision{
			TenantID: concurrentAttachTenant, EndpointID: endpointID, EpEUI: concurrentAttachEUI,
			Status: pkgbssci.EndpointStatusAttached, OverTheAir: report,
		})
		require.NoError(t, err)
	}
	const (
		station = uint64(0x70B3D59CD00009E6)
		queueID = int64(2000003)
	)
	attach(nil)
	_, err = db.Exec(ctx, `
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority,
			organization_id, bs_eui, attempts, tx_time, created_at, updated_at, earliest_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, NOW(), NOW(), NULL)`,
		queueID, []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x0B}, concurrentAttachTenant, []byte("payload"),
		pkgbssci.DLQueueStatusQueued, 5, uuid.New(), []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6}, time.Now().UnixNano())
	require.NoError(t, err, "seed the downlink the station holds")
	stationLeft := time.Now()
	stations := &completingStation{eui: station, reclaimer: mustReclaimer(t, repos.Downlinks, &requeueRecorder{})}
	svc := NewPropagationService(repos.Endpoints, stations, logger.NewNop())
	resumed := propagation.BaseStationSession{ID: "station-resuming", TenantID: concurrentAttachTenant, Resumed: true, DisconnectedAt: &stationLeft}
	status := func() string {
		var s string
		require.NoError(t, db.QueryRow(ctx, `SELECT status FROM downlink_queue WHERE que_id = $1`, queueID).Scan(&s))
		return s
	}

	require.NoError(t, svc.ReconcileBaseStation(ctx, resumed, nil))
	assert.Empty(t, stations.attached, "the station keeps the attachment it held")
	assert.Equal(t, string(pkgbssci.DLQueueStatusQueued), status(), "and the downlink it queued for the endpoint")

	attach(&pkgbssci.OverTheAirReport{BaseStationEUI: reconcileHeardByOtherStation})
	require.NoError(t, svc.ReconcileBaseStation(ctx, resumed, nil))
	assert.Equal(t, []int64{endpointID}, stations.attached, "the new attachment is sent to the station")
	assert.Equal(t, string(pkgbssci.DLQueueStatusPending), status(), "the downlink its attPrp discarded is dispatched again")
}
