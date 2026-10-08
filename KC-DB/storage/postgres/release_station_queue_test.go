package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Stations of the queue release test: the reconnecting one and a neighbor.
const (
	releaseStation  uint64 = 0x70B3D59CD0000B01
	neighborStation uint64 = 0x70B3D59CD0000B02
)

func seedHeldDownlink(t *testing.T, db *sqlx.DB, tenantID int64, orgID uuid.UUID, queID int64, status mioty.DLQueueStatus, bsEUI uint64) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, organization_id, payload, status, priority, bs_eui, attempts, earliest_at, latest_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7, 0, NULL, NULL)`,
		queID, mioty.EUI64Bytes(uint64(queID)), tenantID, orgID, []byte{0x01}, status, mioty.EUI64Bytes(bsEUI))
	require.NoError(t, err)
}

type heldRow struct {
	Status   string `db:"status"`
	BsEUI    []byte `db:"bs_eui"`
	Attempts int    `db:"attempts"`
}

func heldDownlink(t *testing.T, db *sqlx.DB, queID int64) heldRow {
	t.Helper()
	var row heldRow
	require.NoError(t, db.Get(&row, `SELECT status, bs_eui, attempts FROM downlink_queue WHERE que_id = $1`, queID))
	return row
}

// TestReleaseStationQueue_ReturnsEveryTenantsQueuedRowsToPending pins BSSCI
// §1: a session that is not resumed discards what the station held, so the
// downlinks queued at it, a roaming tenant's included, go back to pending
// without a holder, while other stations' rows and every other state stay.
func TestReleaseStationQueue_ReturnsEveryTenantsQueuedRowsToPending(t *testing.T) {
	downlinks, db, orgs := applicationQueueIDFixture(t)
	seedHeldDownlink(t, db, 321, orgs[321], 820001, mioty.DLQueueStatusQueued, releaseStation)
	seedHeldDownlink(t, db, 322, orgs[322], 820002, mioty.DLQueueStatusQueued, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 820003, mioty.DLQueueStatusQueued, neighborStation)
	seedHeldDownlink(t, db, 321, orgs[321], 820004, mioty.DLQueueStatusReserved, releaseStation)
	seedHeldDownlink(t, db, 321, orgs[321], 820005, mioty.DLQueueStatusTransmitted, releaseStation)

	released, err := downlinks.ReleaseStationQueue(t.Context(), releaseStation)

	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{820001, 820002}, releasedQueueIDs(released))
	for _, queID := range []int64{820001, 820002} {
		row := heldDownlink(t, db, queID)
		assert.Equal(t, string(mioty.DLQueueStatusPending), row.Status, "queue id %d", queID)
		assert.Nil(t, row.BsEUI, "the discarding station no longer holds %d", queID)
		assert.Equal(t, 1, row.Attempts)
	}
	assert.Equal(t, string(mioty.DLQueueStatusQueued), heldDownlink(t, db, 820003).Status, "a neighbor's queue is untouched")
	assert.Equal(t, string(mioty.DLQueueStatusReserved), heldDownlink(t, db, 820004).Status, "reservations have their own reclaim")
	assert.Equal(t, string(mioty.DLQueueStatusTransmitted), heldDownlink(t, db, 820005).Status)
}
