package postgres

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateDownlinkCommandRefUnique: migration 000189 lets a ref name one
// downlink of an organization's endpoint, in flight or finished.
func validateDownlinkCommandRefUnique(t *testing.T, db *sql.DB) {
	var index string
	require.NoError(t, db.QueryRow(`SELECT indexdef FROM pg_indexes
		WHERE indexname = 'uq_downlink_queue_command_ref'`).Scan(&index))
	assert.Contains(t, index, "UNIQUE INDEX")
	assert.Contains(t, index, "(tenant_id, organization_id, ep_eui, ref)")
	assert.Contains(t, index, "WHERE (ref IS NOT NULL)")
}

// insertCommandDownlink inserts a downlink of the organization queued for the
// endpoint under ref, NULL for none.
func (h *migrationHarness) insertCommandDownlink(tenantID int64, orgID uuid.UUID, endpoint string, queID int64, status string, ref interface{}) error {
	_, err := h.db.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at, ref)
		VALUES (decode($1, 'hex'), $2, $3, '\x01', $4, $5, NULL, $6)`, endpoint, tenantID, orgID, status, queID, ref)
	return err
}

// TestMigration189KeepsTheEarliestRefAndRefusesARepeat: a ref repeated before
// the upgrade stays on its earliest downlink only; afterwards a ref names one
// downlink of the endpoint, finished or not, while another endpoint, another
// organization and a downlink without a ref are free; the downgrade lets a
// ref repeat again.
func TestMigration189KeepsTheEarliestRefAndRefusesARepeat(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(188)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000189a")
	owner, other := h.seedOrganization(tenantID), h.seedOrganization(tenantID)
	const endpoint, otherEndpoint = "70b3d59cd0000189", "70b3d59cd0000190"
	require.NoError(t, h.insertCommandDownlink(tenantID, owner, endpoint, 189001, "expired", "order-1"))
	require.NoError(t, h.insertCommandDownlink(tenantID, owner, endpoint, 189002, "pending", "order-1"))
	require.NoError(t, h.insertCommandDownlink(tenantID, owner, endpoint, 189003, "pending", "order-1"))

	h.migrateTo(189)

	assert.Equal(t, []string{"189001:order-1"}, h.queryStrings(
		`SELECT que_id || ':' || ref FROM downlink_queue WHERE ref IS NOT NULL ORDER BY que_id`),
		"the earliest downlink keeps the ref, the later receptions lose it")
	assert.Equal(t, int64(3), h.queryInt(`SELECT count(*) FROM downlink_queue`), "no downlink is removed")
	assert.ErrorContains(t, h.insertCommandDownlink(tenantID, owner, endpoint, 189004, "pending", "order-1"),
		"uq_downlink_queue_command_ref", "a finished downlink still holds its ref")
	require.NoError(t, h.insertCommandDownlink(tenantID, owner, otherEndpoint, 189005, "pending", "order-1"), "another endpoint")
	require.NoError(t, h.insertCommandDownlink(tenantID, other, endpoint, 189006, "pending", "order-1"), "another organization")
	require.NoError(t, h.insertCommandDownlink(tenantID, owner, endpoint, 189007, "pending", nil), "a downlink without a ref")
	require.NoError(t, h.insertCommandDownlink(tenantID, owner, endpoint, 189008, "pending", nil), "a second downlink without a ref")

	h.migrateTo(188)
	require.NoError(t, h.insertCommandDownlink(tenantID, owner, endpoint, 189009, "pending", "order-1"), "the downgrade lets a ref repeat")
	h.exec(`DELETE FROM downlink_queue WHERE que_id = 189009`)
	h.migrateTo(189)
}
