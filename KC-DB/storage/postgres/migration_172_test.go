package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertApplicationDownlink inserts a downlink of the organization that its
// Application Center queued under acQueID.
func (h *migrationHarness) insertApplicationDownlink(tenantID int64, orgID uuid.UUID, queID, acQueID int64, status string) error {
	_, err := h.db.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, ac_que_id, earliest_at)
		VALUES (decode('70b3d59cd0000172', 'hex'), $1, $2, '\x01', $3, $4, $5, NULL)`, tenantID, orgID, status, queID, acQueID)
	return err
}

// TestMigration172ScopesTheApplicationQueueIDToInFlightDownlinks: from
// migration 172 an Application Center queue id is unique among the in-flight
// downlinks of an organization; the downgrade refuses while an id repeats
// within a tenant and restores one id per tenant once none does.
func TestMigration172ScopesTheApplicationQueueIDToInFlightDownlinks(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(171)
	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000172a")
	owner, other := h.seedOrganization(tenantID), h.seedOrganization(tenantID)
	require.NoError(t, h.insertApplicationDownlink(tenantID, owner, 17201, 42, "transmitted"))
	assert.ErrorContains(t, h.insertApplicationDownlink(tenantID, other, 17202, 42, "pending"), "idx_downlink_queue_tenant_ac_que_id",
		"before 172 an id is unique within the tenant for good")

	h.migrateTo(172)

	require.NoError(t, h.insertApplicationDownlink(tenantID, other, 17202, 42, "pending"), "another organization of the tenant uses the id")
	require.NoError(t, h.insertApplicationDownlink(tenantID, owner, 17203, 42, "pending"), "the id is free again once its downlink ended")
	assert.ErrorContains(t, h.insertApplicationDownlink(tenantID, owner, 17204, 42, "queued"), "idx_downlink_queue_org_ac_que_id_in_flight",
		"two in-flight downlinks of one organization never share the id")

	err := h.m.Migrate(171)
	require.ErrorContains(t, err, "downlink_queue has 1 Application Center queue id(s) repeated within a tenant")
	require.NoError(t, h.m.Force(172))

	h.exec(`DELETE FROM downlink_queue WHERE que_id IN (17201, 17202)`)
	h.migrateTo(171)
	assert.ErrorContains(t, h.insertApplicationDownlink(tenantID, other, 17205, 42, "transmitted"), "idx_downlink_queue_tenant_ac_que_id",
		"the downgrade restores one id per tenant")
	assert.Zero(t, h.queryInt(`SELECT count(*) FROM pg_proc WHERE proname = 'downlink_queue_in_flight'`))
	h.migrateTo(172)
}
