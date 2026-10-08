package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migration171AcEui is the Application Center EUI the migration 171 test's
// sessions share.
const migration171AcEui = "70b3d59cd000171a"

func (h *migrationHarness) seedOrganization(tenantID int64) uuid.UUID {
	h.t.Helper()
	orgID := uuid.New()
	h.exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, orgID.String())
	return orgID
}

// insertLiveSCACISession inserts an active session of the migration 171
// Application Center in the organization, nil for none.
func (h *migrationHarness) insertLiveSCACISession(tenantID int64, orgID *uuid.UUID, seed byte) error {
	uuidOf := func(offset byte) []byte {
		id := make([]byte, 16)
		for i := range id {
			id[i] = seed + offset + byte(i)
		}
		return id
	}
	_, err := h.db.Exec(`INSERT INTO scaci_sessions (tenant_id, organization_id, ac_eui, sn_ac_uuid, sn_sc_uuid, status)
		VALUES ($1, $2, decode($3, 'hex'), $4, $5, 'active')`, tenantID, orgID, migration171AcEui, uuidOf(0), uuidOf(0x40))
	return err
}

// TestMigration171ScopesTheLiveSessionToTheOrganization: from migration 171
// each organization of a tenant, and the sessions without one, keep one live
// session of an Application Center; the downgrade refuses while an acEui is
// live in more than one of them and restores the tenant-wide rule once not.
func TestMigration171ScopesTheLiveSessionToTheOrganization(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(169)
	tenantID, _ := h.seedTenantAndBaseStation(migration171AcEui)
	owner, other := h.seedOrganization(tenantID), h.seedOrganization(tenantID)
	require.NoError(t, h.insertLiveSCACISession(tenantID, &owner, 0x10))
	assert.ErrorContains(t, h.insertLiveSCACISession(tenantID, &other, 0x20), "idx_scaci_sessions_unique_active",
		"before 171 one live session per tenant and acEui")

	h.migrateTo(171)

	require.NoError(t, h.insertLiveSCACISession(tenantID, &other, 0x20))
	require.NoError(t, h.insertLiveSCACISession(tenantID, nil, 0x30))
	for name, orgID := range map[string]*uuid.UUID{"same organization": &owner, "no organization": nil} {
		assert.ErrorContains(t, h.insertLiveSCACISession(tenantID, orgID, 0x50), "idx_scaci_sessions_unique_active",
			"%s: one live session per Application Center", name)
	}

	err := h.m.Migrate(169)
	require.ErrorContains(t, err, "scaci_sessions has 1 acEui(s) live in more than one organization")
	require.NoError(t, h.m.Force(171))

	h.exec(`UPDATE scaci_sessions SET status = 'terminated' WHERE organization_id IS DISTINCT FROM $1`, owner)
	h.migrateTo(169)
	assert.ErrorContains(t, h.insertLiveSCACISession(tenantID, &other, 0x60), "idx_scaci_sessions_unique_active",
		"the downgrade restores one live session per tenant and acEui")
	h.migrateTo(171)
}
