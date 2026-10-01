package postgres

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// validateDownlinkRevoking: migration 000190 adds the revoking state, keeps
// it in flight and indexes a station's revocations.
func validateDownlinkRevoking(t *testing.T, db *sql.DB) {
	var check string
	require.NoError(t, db.QueryRow(`SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = 'downlink_queue_status_check'`).Scan(&check))
	assert.Contains(t, check, "'revoking'")
	require.True(t, mioty.DLQueueStatusRevoking.Known())
	require.False(t, mioty.DLQueueStatusRevoking.Terminal())
	var inFlight bool
	require.NoError(t, db.QueryRow(`SELECT downlink_queue_in_flight($1)`, string(mioty.DLQueueStatusRevoking)).Scan(&inFlight))
	assert.True(t, inFlight, "a downlink being revoked is still in flight")
	var index string
	require.NoError(t, db.QueryRow(`SELECT indexdef FROM pg_indexes WHERE indexname = 'idx_downlink_queue_revoking_station'`).Scan(&index))
	assert.Contains(t, index, "WHERE ((status)::text = 'revoking'::text)")
	var askedAt string
	require.NoError(t, db.QueryRow(`SELECT data_type FROM information_schema.columns WHERE table_name = 'downlink_queue' AND column_name = 'revoke_asked_at'`).Scan(&askedAt))
	assert.Equal(t, "timestamp with time zone", askedAt, "when the holder was last asked to drop it")
}

// TestMigration190KeepsARevokingDownlinkInFlight: a revoking downlink keeps
// its Application Center queue id taken like any in-flight one; the downgrade
// returns it to queued at its holder, which the previous release expires and
// revokes again, and refuses the state.
func TestMigration190KeepsARevokingDownlinkInFlight(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(189)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000190a")
	org := h.seedOrganization(tenantID)
	assert.Error(t, h.insertApplicationDownlink(tenantID, org, 19001, 90, "revoking"), "before 190 there is no revoking")

	h.migrateTo(190)

	require.NoError(t, h.insertApplicationDownlink(tenantID, org, 19001, 90, "revoking"))
	assert.ErrorContains(t, h.insertApplicationDownlink(tenantID, org, 19002, 90, "pending"), "idx_downlink_queue_org_ac_que_id_in_flight",
		"a downlink being revoked keeps its Application Center queue id")

	h.migrateTo(189)
	assert.Equal(t, []string{"queued"}, h.queryStrings(`SELECT status FROM downlink_queue WHERE que_id = 19001`))
	assert.Error(t, h.insertApplicationDownlink(tenantID, org, 19003, 91, "revoking"), "the downgrade refuses the state")
	h.migrateTo(190)
}
