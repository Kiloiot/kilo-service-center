package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (h *migrationHarness) seedDownlinkRow(tenantID, queID int64, status string) {
	h.t.Helper()
	orgID := uuid.New()
	h.exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, "org-"+orgID.String())
	h.exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at)
	        VALUES (decode('70b3d59cd0000160', 'hex'), $1, $2, '\x01', $3, $4, NULL)`, tenantID, orgID, status, queID)
}

func TestMigration160KeepsExistingRowsAddressable(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(159)

	tenantA, _ := h.seedTenantAndBaseStation("70b3d59cd000160a")
	tenantB, _ := h.seedTenantAndBaseStation("70b3d59cd000160b")
	h.seedDownlinkRow(tenantA, 42, "pending")
	h.seedDownlinkRow(tenantB, 43, "transmitted")

	h.migrateTo(160)

	assert.Equal(t, int64(42), h.queryInt(`SELECT ac_que_id FROM downlink_queue WHERE que_id = 42`),
		"an existing row keeps the queue id its Application Center already knows")
	assert.Equal(t, int64(43), h.queryInt(`SELECT ac_que_id FROM downlink_queue WHERE que_id = 43`))

	// The same Application Center id is independent across tenants and
	// unique inside one.
	h.exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, ac_que_id, earliest_at)
	        SELECT ep_eui, $1, organization_id, payload, 'pending', 9001, 42, NULL FROM downlink_queue WHERE que_id = 43`, tenantB)
	_, err := h.db.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, ac_que_id, earliest_at)
	        SELECT ep_eui, tenant_id, organization_id, payload, 'pending', 9002, 42, NULL FROM downlink_queue WHERE que_id = 42`)
	assert.ErrorContains(t, err, "idx_downlink_queue_tenant_ac_que_id")

	// A terminal row has already reported its result and does not block the
	// downgrade.
	h.exec(`UPDATE downlink_queue SET status = 'transmitted' WHERE que_id = 9001`)
	h.migrateTo(159)
	assert.False(t, h.columnExists("downlink_queue", "ac_que_id"))
}

func TestMigration160DowngradeRefusesDivergedInFlightRows(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(160)

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000160c")
	h.seedDownlinkRow(tenantID, 777, "pending")
	h.exec(`UPDATE downlink_queue SET ac_que_id = 5 WHERE que_id = 777`)

	err := h.m.Migrate(159)
	assert.ErrorContains(t, err, "in-flight row(s) whose Application Center queue id differs from que_id")
	assert.True(t, h.columnExists("downlink_queue", "ac_que_id"))
}

func TestMigration160GuardRejectsNonPositiveQueueIDs(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(159)

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd000160d")
	h.seedDownlinkRow(tenantID, 0, "pending")

	h.migrateExpectingGuard(160, "que_id is not positive")
	h.exec(`DELETE FROM downlink_queue WHERE que_id = 0`)
	h.migrateTo(160)
	assert.True(t, h.columnExists("downlink_queue", "ac_que_id"))
}
