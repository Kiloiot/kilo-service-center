package bssciservices

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
)

// BSSCI §3.13, §3.17 over PostgreSQL: a station's error answer to a dlDataRev
// means it does not hold the downlink. A downlink still queued there ends
// revoked. One the expiry sweep already expired, or a late dlDataRes already
// finished, keeps its outcome. None of them is reported to its originators
// again.
func TestProcessRevokeRefusal_EndsOnlyADownlinkStillInFlight(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	repos := postgres.NewRepositories(db)
	f := newReporterFixture(t)
	resolver := NewTenantResolver(nil)
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: resolver, Outcomes: repos.Downlinks, Holders: repos.Downlinks,
		Results: f.reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)

	const tenantID = int64(1)
	held := map[int64]mioty.DLQueueStatus{
		2000011: mioty.DLQueueStatusQueued,
		2000012: mioty.DLQueueStatusExpired,
		2000013: mioty.DLQueueStatusTransmitted,
	}
	for queID, status := range held {
		_, err := db.Exec(ctx, `
			INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority,
				organization_id, bs_eui, attempts, created_at, updated_at, earliest_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, NOW(), NOW(), NULL)`,
			queID, []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x03, 0x41}, tenantID, []byte("payload"),
			status, 5, uuid.New(), []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6})
		require.NoError(t, err, "seed downlink %d", queID)
		resolver.RegisterQueueTenant(queID, strconv.FormatInt(tenantID, 10))
	}
	status := func(queID int64) mioty.DLQueueStatus {
		var s string
		require.NoError(t, db.QueryRow(ctx, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID).Scan(&s))
		return mioty.DLQueueStatus(s)
	}
	station := &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: 0x70B3D59CD00009E6}}

	for queID, want := range map[int64]struct {
		revoked bool
		status  mioty.DLQueueStatus
	}{
		2000011: {revoked: true, status: mioty.DLQueueStatusRevoked},
		2000012: {status: mioty.DLQueueStatusExpired},
		2000013: {status: mioty.DLQueueStatusTransmitted},
	} {
		revoked, err := svc.ProcessRevokeRefusal(ctx, station, bssci.RevokeRefusal{
			QueueID: queID, EndpointEUI: 0x70B3D59CD0000341, Code: bssci.POSIX_ENOTSUP, Message: "no matching DL data found",
		})
		require.NoError(t, err)
		assert.Equal(t, want.revoked, revoked, "queue id %d", queID)
		assert.Equal(t, want.status, status(queID), "queue id %d", queID)
	}
	f.stop(t)
	assert.Empty(t, f.acs.delivered, "no Application Center is told a result again")
	assert.Empty(t, f.mqtt.published, "no MQTT result is published again")
	assert.Nil(t, f.events.lastEvent, "no result event is recorded again")
}

// BSSCI §3.13, §3.17 over PostgreSQL: a base station's answer to a dlDataRev
// ends only the downlink it holds. A downlink dispatched to another station
// since survives a late confirmation or refusal from the station that held it
// before, and the station holding it now still ends it.
func TestRevokeAnswers_EndOnlyTheDownlinkTheStationHolds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	repos := postgres.NewRepositories(db)
	resolver := NewTenantResolver(repos.DownlinkQueueReader)
	svc, err := NewDownlinkService(DownlinkServiceDeps{
		Logger: logger.NewNop(), Tenants: resolver, Outcomes: repos.Downlinks, Holders: repos.Downlinks,
		Results: newReporterFixture(t).reporter, Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
	})
	require.NoError(t, err)

	const tenantID = int64(1)
	previous := &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: 0x70B3D59CD00009E6}}
	current := &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: 0x70B3D59CD00009E2}}
	currentHolder := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE2}
	for _, queID := range []int64{2000021, 2000022} {
		_, err := db.Exec(ctx, `
			INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority,
				organization_id, bs_eui, attempts, created_at, updated_at, earliest_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, NOW(), NOW(), NULL)`,
			queID, []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x03, 0x41}, tenantID, []byte("payload"),
			mioty.DLQueueStatusQueued, 5, uuid.New(), currentHolder)
		require.NoError(t, err, "seed downlink %d", queID)
	}
	status := func(queID int64) mioty.DLQueueStatus {
		var s string
		require.NoError(t, db.QueryRow(ctx, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID).Scan(&s))
		return mioty.DLQueueStatus(s)
	}

	_, revoked, err := svc.ProcessRevokeResponse(ctx, previous, -21, 2000021, 0x70B3D59CD0000341)
	require.NoError(t, err)
	assert.False(t, revoked, "a late confirmation from the previous holder")
	revoked, err = svc.ProcessRevokeRefusal(ctx, previous, bssci.RevokeRefusal{
		QueueID: 2000022, EndpointEUI: 0x70B3D59CD0000341, Code: bssci.POSIX_ENOTSUP, Message: "no matching DL data found",
	})
	require.NoError(t, err)
	assert.False(t, revoked, "a late refusal from the previous holder")
	assert.Equal(t, mioty.DLQueueStatusQueued, status(2000021), "the downlink queued at the current holder survives the confirmation")
	assert.Equal(t, mioty.DLQueueStatusQueued, status(2000022), "the downlink queued at the current holder survives the refusal")

	_, revoked, err = svc.ProcessRevokeResponse(ctx, current, -22, 2000021, 0x70B3D59CD0000341)
	require.NoError(t, err)
	assert.True(t, revoked, "the current holder's confirmation revokes it")
	assert.Equal(t, mioty.DLQueueStatusRevoked, status(2000021))
}
