package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

const (
	revokeTestTenant   = int64(341)
	revokeTestEndpoint = uint64(0x70b3d59cd0000341)
	revokeTestStation  = uint64(0x70b3d59cd0000a01)
)

// answeredBy is the base station's answer to the dlDataRev for the downlink.
func answeredBy(queID int64, station uint64) storage.DownlinkRevocation {
	return storage.DownlinkRevocation{QueID: queID, TenantID: revokeTestTenant, Station: &station}
}

// holdAt records the base station holding the downlink.
func holdAt(t *testing.T, db *sqlx.DB, queID int64, station uint64) {
	t.Helper()
	_, err := db.Exec(`UPDATE downlink_queue SET bs_eui = $1 WHERE que_id = $2`, mioty.EUI64Bytes(station), queID)
	require.NoError(t, err)
}

func revokeFixture(t *testing.T) (*MIOTYDownlinkRepository, *sqlx.DB, func(queID int64, status string)) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, sqlxDB, revokeTestTenant, "RevokeTenant")
	orgID := uuid.New()
	_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, 'revoke-org')`, orgID, revokeTestTenant)
	require.NoError(t, err)
	seed := func(queID int64, status string) {
		_, err := sqlxDB.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at)
			VALUES (decode('70b3d59cd0000341', 'hex'), $1, $2, '\x01', $3, $4, NULL)`, revokeTestTenant, orgID, status, queID)
		require.NoError(t, err)
	}
	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	return NewRepositories(db).Downlinks, sqlxDB, seed
}

// TestRevokeDownlink_MarksTheRowRevoked pins the base station confirmed
// revoke (BSSCI §3.13): a row the station holds becomes revoked. The result
// column holds the dlDataRes outcome (sent, expired, invalid) and is left alone.
func TestRevokeDownlink_MarksTheRowRevoked(t *testing.T) {
	downlinks, sqlxDB, seed := revokeFixture(t)
	for queID, held := range map[int64]string{840001: "queued", 840002: "reserved"} {
		seed(queID, held)
		holdAt(t, sqlxDB, queID, revokeTestStation)

		revoked, err := downlinks.RevokeDownlink(t.Context(), answeredBy(queID, revokeTestStation))
		require.NoError(t, err, held)
		assert.True(t, revoked, held)

		var status string
		var result *string
		require.NoError(t, sqlxDB.QueryRow(`SELECT status, result FROM downlink_queue WHERE que_id = $1`, queID).Scan(&status, &result))
		assert.Equal(t, "revoked", status, held)
		assert.Nil(t, result, held)
	}
}

// TestRevokeDownlink_LeavesAFinishedDownlinkAlone: the confirmation of a
// revoke the service center sent after the downlink expired, or after it
// finished any other way, changes nothing and is not an error.
func TestRevokeDownlink_LeavesAFinishedDownlinkAlone(t *testing.T) {
	downlinks, sqlxDB, seed := revokeFixture(t)
	for queID, finished := range map[int64]string{840031: "expired", 840032: "transmitted", 840033: "revoked"} {
		seed(queID, finished)
		holdAt(t, sqlxDB, queID, revokeTestStation)

		revoked, err := downlinks.RevokeDownlink(t.Context(), answeredBy(queID, revokeTestStation))

		require.NoError(t, err, finished)
		assert.False(t, revoked, finished)
		var status string
		require.NoError(t, sqlxDB.Get(&status, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID))
		assert.Equal(t, finished, status)
	}
}

// TestRevokeDownlink_LateStationAnswerLeavesAnotherStationsDownlink pins
// BSSCI §3.13 and §3.17: a base station's answer to dlDataRev ends only the
// downlink it holds, so a row dispatched to another station since survives a
// late confirmation or refusal from the station that held it before, and the
// station holding it now still ends it.
func TestRevokeDownlink_LateStationAnswerLeavesAnotherStationsDownlink(t *testing.T) {
	downlinks, sqlxDB, seed := revokeFixture(t)
	const previousHolder, currentHolder = revokeTestStation, uint64(0x70b3d59cd0000a02)
	for queID, held := range map[int64]string{840051: "queued", 840052: "reserved"} {
		seed(queID, held)
		holdAt(t, sqlxDB, queID, currentHolder)

		revoked, err := downlinks.RevokeDownlink(t.Context(), answeredBy(queID, previousHolder))

		require.NoError(t, err, held)
		assert.False(t, revoked, held)
		var status string
		require.NoError(t, sqlxDB.Get(&status, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID))
		assert.Equal(t, held, status, "the downlink %s at another station survives", held)

		revoked, err = downlinks.RevokeDownlink(t.Context(), answeredBy(queID, currentHolder))
		require.NoError(t, err, held)
		assert.True(t, revoked, "the current holder's answer revokes the downlink %s", held)
	}
}

// TestUpdateDownlinkResult_LeavesAFinishedDownlinkAlone: a result a base
// station reports for a downlink that already expired or was revoked keeps
// the outcome its originators were told; an unknown one is not found.
func TestUpdateDownlinkResult_LeavesAFinishedDownlinkAlone(t *testing.T) {
	downlinks, sqlxDB, seed := revokeFixture(t)
	for queID, finished := range map[int64]string{840041: "expired", 840042: "revoked"} {
		seed(queID, finished)

		_, err := downlinks.UpdateDownlinkResult(t.Context(), revokeTestTenant, releaseStation,
			&mioty.DLDataResult{EpEui: revokeTestEndpoint, QueId: uint64(queID), Result: mioty.ResultSent})

		require.ErrorIs(t, err, storage.ErrDownlinkFinished, finished)
		var status string
		require.NoError(t, sqlxDB.Get(&status, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID))
		assert.Equal(t, finished, status)
	}
	_, err := downlinks.UpdateDownlinkResult(t.Context(), revokeTestTenant, releaseStation,
		&mioty.DLDataResult{EpEui: revokeTestEndpoint, QueId: 840049, Result: mioty.ResultSent})
	require.ErrorIs(t, err, storage.ErrDownlinkNotFound)
}

// TestRevokeDownlink_InTheQueueOnlyRevokesAPendingRow pins the revoke of a
// downlink no base station holds: a pending row becomes revoked, a row a
// dispatcher already reserved is left for the base station revoke.
func TestRevokeDownlink_InTheQueueOnlyRevokesAPendingRow(t *testing.T) {
	downlinks, sqlxDB, seed := revokeFixture(t)
	seed(840011, "pending")
	seed(840012, "reserved")
	inQueue := func(queID, tenantID int64) storage.DownlinkRevocation {
		return storage.DownlinkRevocation{QueID: queID, TenantID: tenantID}
	}

	revoked, err := downlinks.RevokeDownlink(t.Context(), inQueue(840011, revokeTestTenant+1))
	require.NoError(t, err)
	assert.False(t, revoked, "another tenant never revokes the row")

	revoked, err = downlinks.RevokeDownlink(t.Context(), inQueue(840011, revokeTestTenant))
	require.NoError(t, err)
	assert.True(t, revoked)

	revoked, err = downlinks.RevokeDownlink(t.Context(), inQueue(840012, revokeTestTenant))
	require.NoError(t, err)
	assert.False(t, revoked)

	status := func(queID int64) string {
		var s string
		require.NoError(t, sqlxDB.Get(&s, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID))
		return s
	}
	assert.Equal(t, "revoked", status(840011))
	assert.Equal(t, "reserved", status(840012))
}

// TestFailQueuedDownlink_RecordsTheStationError pins the BSSCI §3.17 outcome
// of a dlDataQue a base station answered with error: the row it held fails
// with the station's error, and no other row, another station's included,
// is touched.
func TestFailQueuedDownlink_RecordsTheStationError(t *testing.T) {
	const reason = "base station error 28: queue full"
	otherStation := revokeTestStation + 1
	downlinks, sqlxDB, seed := revokeFixture(t)
	seed(840021, "queued")
	seed(840022, "reserved")
	seed(840023, "pending")
	seed(840024, "queued")
	for _, queID := range []int64{840021, 840022, 840024} {
		holdAt(t, sqlxDB, queID, revokeTestStation)
	}

	_, err := downlinks.FailQueuedDownlink(t.Context(), 840024, revokeTestTenant, otherStation, reason)
	require.ErrorIs(t, err, storage.ErrDownlinkNotFound, "a station's error answer never fails another station's downlink")
	for _, queID := range []int64{840021, 840022} {
		failed, err := downlinks.FailQueuedDownlink(t.Context(), queID, revokeTestTenant, revokeTestStation, reason)
		require.NoError(t, err)
		assert.Equal(t, queID, failed.QueID, "the failed row is returned for its originators")
	}
	_, err = downlinks.FailQueuedDownlink(t.Context(), 840023, revokeTestTenant, revokeTestStation, reason)
	require.ErrorIs(t, err, storage.ErrDownlinkNotFound)
	_, err = downlinks.FailQueuedDownlink(t.Context(), 840021, revokeTestTenant+1, revokeTestStation, reason)
	require.ErrorIs(t, err, storage.ErrDownlinkNotFound)

	row := func(queID int64) (string, *string) {
		var status string
		var failure *string
		require.NoError(t, sqlxDB.QueryRow(`SELECT status, failure_reason FROM downlink_queue WHERE que_id = $1`, queID).Scan(&status, &failure))
		return status, failure
	}
	for _, queID := range []int64{840021, 840022} {
		status, failure := row(queID)
		assert.Equal(t, "failed", status)
		require.NotNil(t, failure)
		assert.Equal(t, reason, *failure)
	}
	status, failure := row(840023)
	assert.Equal(t, "pending", status)
	assert.Nil(t, failure)
	status, failure = row(840024)
	assert.Equal(t, "queued", status)
	assert.Nil(t, failure)
}
