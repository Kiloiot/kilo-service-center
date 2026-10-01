package bssciservices

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/workers/downlinkexpiry"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
)

const (
	revokingTenant   = int64(1)
	revokingEndpoint = uint64(0x70B3D59CD0000341)
	revokingStation  = uint64(0x70B3D59CD00009E6)
	revokingBatch    = 10
	// revokingSweepInterval never elapses in a test, which sweeps by hand.
	revokingSweepInterval = time.Hour
)

// askedStations records every overdue downlink the sweep asks its station to
// drop, and names the stations connected.
type askedStations struct {
	asked     []int64
	connected []uint64
}

func (a *askedStations) ConnectedStations() []uint64 {
	return a.connected
}

func (a *askedStations) RevokeHeldDownlink(_ context.Context, downlink *storage.DownlinkMessage) error {
	a.asked = append(a.asked, downlink.QueID)
	return nil
}

// revokingStack is the downlink lifecycle of one KC-Core over a database: the
// expiry sweep, the station answers and the reclaim of a fresh session. A
// restart builds a new one over the same database.
type revokingStack struct {
	reporter  *reporterFixture
	svc       bssci.DownlinkService
	sweep     *downlinkexpiry.Worker
	asked     *askedStations
	reclaimer *DownlinkReclaimer
}

func newRevokingStack(t *testing.T, repos *postgres.Repositories) *revokingStack {
	t.Helper()
	f := newReporterFixture(t)
	resolver := NewTenantResolver(repos.DownlinkQueueReader)
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: resolver, Outcomes: repos.Downlinks, Holders: repos.Downlinks,
		Results: f.reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	}, newRevokeAnswers(t, logger.NewNop(), resolver, repos.Downlinks, f.reporter))
	require.NoError(t, err)
	asked := &askedStations{}
	sweep, err := downlinkexpiry.NewWorker(downlinkexpiry.Dependencies{
		Queue: repos.Downlinks, Reporter: f.reporter, Revoker: asked, QueueTenants: resolver, Logger: logger.NewNop(),
	}, downlinkexpiry.Config{Interval: revokingSweepInterval, BatchSize: revokingBatch})
	require.NoError(t, err)
	reclaimer, err := NewDownlinkReclaimer(DownlinkReclaimerDeps{
		Store: repos.Downlinks, Revocations: repos.Downlinks, Events: &requeueRecorder{}, Expiries: f.reporter, Tenants: resolver, Logger: logger.NewNop(),
	})
	require.NoError(t, err)
	return &revokingStack{reporter: f, svc: svc, sweep: sweep, asked: asked, reclaimer: reclaimer}
}

// published lists the results reported on MQTT, waiting for the background delivery.
func (s *revokingStack) published(t *testing.T) []mioty.DLDataResult {
	t.Helper()
	s.reporter.stop(t)
	results := make([]mioty.DLDataResult, len(s.reporter.mqtt.published))
	for i, p := range s.reporter.mqtt.published {
		results[i] = p.result
	}
	return results
}

// seedOverdueQueued stores downlinks the station holds queued whose lifetime ended.
func seedOverdueQueued(t *testing.T, db *postgres.DB, queIDs ...int64) {
	t.Helper()
	ctx := testutil.TestContext()
	for _, queID := range queIDs {
		_, err := db.Exec(ctx, `
			INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority, organization_id, bs_eui,
				attempts, created_at, updated_at, earliest_at, latest_at)
			VALUES ($1, $2, $3, $4, $5, 0, $6, $7, 0, NOW(), NOW(), NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour')`,
			queID, mioty.EUI64Bytes(revokingEndpoint), revokingTenant, []byte{0x01}, mioty.DLQueueStatusQueued,
			uuid.New(), mioty.EUI64Bytes(revokingStation))
		require.NoError(t, err, "seed downlink %d", queID)
	}
}

func revokingStatus(t *testing.T, db *postgres.DB, queID int64) mioty.DLQueueStatus {
	t.Helper()
	var status string
	require.NoError(t, db.QueryRow(testutil.TestContext(), `SELECT status FROM downlink_queue WHERE que_id = $1`, queID).Scan(&status))
	return mioty.DLQueueStatus(status)
}

func revokingSession() *bssci.Session {
	return &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: revokingStation}}
}

// BSSCI §3.13, §3.14 over PostgreSQL: a downlink the station holds past its
// lifetime is not reported expired by the sweep; the station is asked to drop
// it and the downlink stays revoking. Its confirmation then ends it expired,
// reported once; a repeated confirmation changes nothing.
func TestRevokingDownlink_ExpiresOnlyWhenTheStationConfirms(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	repos := postgres.NewRepositories(db)
	stack := newRevokingStack(t, repos)
	seedOverdueQueued(t, db, 3100001)
	ctx := testutil.TestContext()

	stack.sweep.SweepOnce(ctx)
	assert.Equal(t, []int64{3100001}, stack.asked.asked, "the holding station is asked to drop it")
	assert.Equal(t, mioty.DLQueueStatusRevoking, revokingStatus(t, db, 3100001))
	stack.sweep.SweepOnce(ctx)
	assert.Len(t, stack.asked.asked, 1, "a downlink being revoked is not asked again by the sweep")

	_, revoked, err := stack.svc.ProcessRevokeResponse(ctx, revokingSession(), -31, 3100001, revokingEndpoint)
	require.NoError(t, err)
	assert.False(t, revoked, "it ends expired, not revoked")
	_, _, err = stack.svc.ProcessRevokeResponse(ctx, revokingSession(), -31, 3100001, revokingEndpoint)
	require.NoError(t, err)

	assert.Equal(t, mioty.DLQueueStatusExpired, revokingStatus(t, db, 3100001))
	published := stack.published(t)
	require.Len(t, published, 1, "expired is reported once, after the station confirmed")
	assert.Equal(t, mioty.ResultExpired, published[0].Result)
}

// BSSCI §3.14.1 over PostgreSQL: a dlDataRes "sent" that arrives while the
// downlink is being revoked is recorded and reported as sent; the station's
// revoke confirmation and refusal that follow report nothing, and a repeated
// result changes nothing.
func TestRevokingDownlink_ASentResultRacingTheRevokeIsReportedSent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	repos := postgres.NewRepositories(db)
	stack := newRevokingStack(t, repos)
	seedOverdueQueued(t, db, 3100011)
	ctx := testutil.TestContext()
	stack.sweep.SweepOnce(ctx)
	packetCnt := uint32(12)
	txTime := dispatchTestNow.UnixNano()
	sent := &mioty.DLDataResult{EpEui: revokingEndpoint, QueId: 3100011, Result: mioty.ResultSent, PacketCnt: &packetCnt, TxTime: &txTime}

	_, err := stack.svc.ProcessDLDataResult(ctx, revokingSession(), sent)
	require.NoError(t, err)
	_, err = stack.svc.ProcessDLDataResult(ctx, revokingSession(), sent)
	require.NoError(t, err)
	_, revoked, err := stack.svc.ProcessRevokeResponse(ctx, revokingSession(), -32, 3100011, revokingEndpoint)
	require.NoError(t, err)
	assert.False(t, revoked)
	revoked, err = stack.svc.ProcessRevokeRefusal(ctx, revokingSession(), bssci.RevokeRefusal{
		QueueID: 3100011, EndpointEUI: revokingEndpoint, Code: bssci.POSIX_ENOENT,
	})
	require.NoError(t, err)
	assert.False(t, revoked)

	assert.Equal(t, mioty.DLQueueStatusTransmitted, revokingStatus(t, db, 3100011))
	published := stack.published(t)
	require.Len(t, published, 1)
	assert.Equal(t, mioty.ResultSent, published[0].Result, "the transmission is the outcome")
}

// Over PostgreSQL: a KC-Core that restarts while a downlink is revoking knows
// nothing of it in memory; the database carries the holder and the owner, so
// the station's answer to the reissued dlDataRev, handled by the new process,
// ends it expired and reports it.
func TestRevokingDownlink_SurvivesARestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	repos := postgres.NewRepositories(db)
	seedOverdueQueued(t, db, 3100021)
	before := newRevokingStack(t, repos)
	before.sweep.SweepOnce(testutil.TestContext())
	assert.Empty(t, before.published(t))

	after := newRevokingStack(t, postgres.NewRepositories(db))
	_, _, err := after.svc.ProcessRevokeResponse(testutil.TestContext(), revokingSession(), -33, 3100021, revokingEndpoint)
	require.NoError(t, err)

	assert.Equal(t, mioty.DLQueueStatusExpired, revokingStatus(t, db, 3100021))
	published := after.published(t)
	require.Len(t, published, 1)
	assert.Equal(t, mioty.ResultExpired, published[0].Result)
}

// BSSCI §1 over PostgreSQL: a station unreachable when the downlink's lifetime
// ended leaves it revoking. A session of that station that is not resumed
// discarded it, so it ends expired and is reported; the queued downlinks of
// the station return to pending as before.
func TestRevokingDownlink_AFreshSessionOfTheHolderExpiresIt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	repos := postgres.NewRepositories(db)
	stack := newRevokingStack(t, repos)
	seedOverdueQueued(t, db, 3100031)
	ctx := testutil.TestContext()
	stack.sweep.SweepOnce(ctx)

	_, err := stack.reclaimer.ReclaimReservations(ctx, revokingStation, nil)
	require.NoError(t, err)
	assert.Equal(t, mioty.DLQueueStatusRevoking, revokingStatus(t, db, 3100031), "a resumed session keeps the revoke open")

	_, err = stack.reclaimer.ReclaimDiscardedQueue(ctx, revokingStation)
	require.NoError(t, err)
	assert.Equal(t, mioty.DLQueueStatusExpired, revokingStatus(t, db, 3100031))
	published := stack.published(t)
	require.Len(t, published, 1)
	assert.Equal(t, mioty.ResultExpired, published[0].Result)
}

// BSSCI §3.13, §3.17 over PostgreSQL: a connected holder that refused the
// revoke with a code that proves nothing is asked again once its last ask is
// a sweep interval old, and not before; the downlink stays revoking and
// nothing is reported, however often it is asked.
func TestRevokingDownlink_AnUnansweredRevokeIsAskedAgainOncePerInterval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	stack := newRevokingStack(t, postgres.NewRepositories(db))
	stack.asked.connected = []uint64{revokingStation}
	seedOverdueQueued(t, db, 3100041)
	ctx := testutil.TestContext()

	stack.sweep.SweepOnce(ctx)
	revoked, err := stack.svc.ProcessRevokeRefusal(ctx, revokingSession(), bssci.RevokeRefusal{
		QueueID: 3100041, EndpointEUI: revokingEndpoint, Code: bssci.POSIX_EIO,
	})
	require.NoError(t, err)
	assert.False(t, revoked)
	stack.sweep.SweepOnce(ctx)
	assert.Equal(t, []int64{3100041}, stack.asked.asked, "not asked again inside the interval")

	_, err = db.Exec(ctx, `UPDATE downlink_queue SET revoke_asked_at = NOW() - INTERVAL '2 hours' WHERE que_id = 3100041`)
	require.NoError(t, err)
	stack.sweep.SweepOnce(ctx)
	stack.sweep.SweepOnce(ctx)

	assert.Equal(t, []int64{3100041, 3100041}, stack.asked.asked, "asked again once the interval passed, once")
	assert.Equal(t, mioty.DLQueueStatusRevoking, revokingStatus(t, db, 3100041), "time passing ends nothing")
	assert.Empty(t, stack.published(t))
}

// Over PostgreSQL: a deleted base station can never transmit what it held.
// The downlink it was asked to drop ends expired and is reported, and the
// downlink it held queued returns to pending for another station.
func TestRevokingDownlink_ADeletedHolderExpiresIt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	stack := newRevokingStack(t, postgres.NewRepositories(db))
	seedOverdueQueued(t, db, 3100051, 3100052)
	ctx := testutil.TestContext()
	stack.sweep.SweepOnce(ctx)
	_, err := db.Exec(ctx, `UPDATE downlink_queue SET status = $1, latest_at = NOW() + INTERVAL '1 hour' WHERE que_id = 3100052`,
		mioty.DLQueueStatusQueued)
	require.NoError(t, err)

	stack.reclaimer.ReleaseDeletedStation(ctx, revokingStation)

	assert.Equal(t, mioty.DLQueueStatusExpired, revokingStatus(t, db, 3100051))
	assert.Equal(t, mioty.DLQueueStatusPending, revokingStatus(t, db, 3100052))
	published := stack.published(t)
	require.Len(t, published, 1)
	assert.Equal(t, uint64(3100051), published[0].QueId)
	assert.Equal(t, mioty.ResultExpired, published[0].Result)
}
