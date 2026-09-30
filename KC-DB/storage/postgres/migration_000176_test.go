package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const migrationGrandfathering uint = 176

type userSwitches struct {
	admin, tenantManager, baseStationManager, endpointManager bool
}

type membershipSwitches struct {
	orgAdmin, baseStationAdmin, endpointAdmin bool
}

// stepDown leaves the schema just before the migration last applied, whatever its number.
func (h *migrationHarness) stepDown() {
	h.t.Helper()
	require.NoError(h.t, h.m.Steps(-1))
}

func (h *migrationHarness) seedUser(email string, s userSwitches) uuid.UUID {
	h.t.Helper()
	id := uuid.New()
	h.exec(`INSERT INTO identity.users (id, email, email_verified, is_admin, is_active,
	            is_tenant_manager, is_base_station_manager, is_endpoint_manager)
	        VALUES ($1, $2, true, $3, true, $4, $5, $6)`,
		id, email, s.admin, s.tenantManager, s.baseStationManager, s.endpointManager)
	return id
}

func (h *migrationHarness) seedMembership(orgID, userID uuid.UUID, status string, s membershipSwitches) {
	h.t.Helper()
	h.exec(`INSERT INTO identity.organization_members (org_id, user_id, role, status, is_org_admin, is_base_station_admin, is_endpoint_admin)
	        VALUES ($1, $2, 'member', $3, $4, $5, $6)`, orgID, userID, status, s.orgAdmin, s.baseStationAdmin, s.endpointAdmin)
}

func (h *migrationHarness) userSwitches(id uuid.UUID) userSwitches {
	h.t.Helper()
	var s userSwitches
	require.NoError(h.t, h.db.QueryRow(`SELECT is_admin, is_tenant_manager, is_base_station_manager, is_endpoint_manager
	        FROM identity.users WHERE id = $1`, id).Scan(&s.admin, &s.tenantManager, &s.baseStationManager, &s.endpointManager))
	return s
}

func (h *migrationHarness) membershipSwitches(orgID, userID uuid.UUID) membershipSwitches {
	h.t.Helper()
	var s membershipSwitches
	require.NoError(h.t, h.db.QueryRow(`SELECT is_org_admin, is_base_station_admin, is_endpoint_admin
	        FROM identity.organization_members WHERE org_id = $1 AND user_id = $2`, orgID, userID).Scan(&s.orgAdmin, &s.baseStationAdmin, &s.endpointAdmin))
	return s
}

func TestMigration176GrandfathersExistingAccess(t *testing.T) {
	h := newMigrationHarness(t)
	h.migrateTo(migrationGrandfathering)
	h.stepDown()

	tenantID, _ := h.seedTenantAndBaseStation("70b3d59cd0000176")
	orgID := uuid.New()
	h.exec(`INSERT INTO identity.organizations (org_id, tenant_id, name) VALUES ($1, $2, 'grandfathered org')`, orgID, tenantID)

	plain := h.seedUser("plain@example.com", userSwitches{})
	partial := h.seedUser("partial@example.com", userSwitches{tenantManager: true, baseStationManager: true})
	admin := h.seedUser("admin@example.com", userSwitches{admin: true})
	invited := h.seedUser("invited@example.com", userSwitches{})
	orgAdmin := h.seedUser("orgadmin@example.com", userSwitches{})

	h.seedMembership(orgID, plain, "active", membershipSwitches{})
	h.seedMembership(orgID, orgAdmin, "active", membershipSwitches{orgAdmin: true, endpointAdmin: true})
	h.seedMembership(orgID, invited, "invited", membershipSwitches{})

	adminsBefore := h.queryInt(`SELECT count(*) FROM identity.users WHERE is_admin`)
	h.migrateTo(migrationGrandfathering)

	assert.Equal(t, userSwitches{baseStationManager: true, endpointManager: true}, h.userSwitches(plain),
		"a pre-upgrade account keeps base station and endpoint access and gains neither admin nor tenant manager")
	assert.Equal(t, userSwitches{tenantManager: true, baseStationManager: true, endpointManager: true}, h.userSwitches(partial))
	assert.Equal(t, userSwitches{admin: true}, h.userSwitches(admin), "administrators are left as they were")
	assert.Equal(t, membershipSwitches{baseStationAdmin: true, endpointAdmin: true}, h.membershipSwitches(orgID, plain))
	assert.Equal(t, membershipSwitches{orgAdmin: true, baseStationAdmin: true, endpointAdmin: true}, h.membershipSwitches(orgID, orgAdmin),
		"member management stays with the members that already held it")
	assert.Equal(t, membershipSwitches{}, h.membershipSwitches(orgID, invited), "an invitation that is not accepted grants nothing")
	assert.Equal(t, adminsBefore, h.queryInt(`SELECT count(*) FROM identity.users WHERE is_admin`), "no account becomes an administrator")

	later := h.seedUser("later@example.com", userSwitches{})
	h.seedMembership(orgID, later, "active", membershipSwitches{})
	assert.Equal(t, userSwitches{}, h.userSwitches(later), "an account created after the upgrade starts without roles")
	assert.Equal(t, membershipSwitches{}, h.membershipSwitches(orgID, later))

	h.exec(`DELETE FROM identity.organization_members WHERE user_id = $1`, later)
	h.exec(`DELETE FROM identity.users WHERE id = $1`, later)
	h.stepDown()

	assert.Equal(t, userSwitches{}, h.userSwitches(plain), "down restores the switches the migration turned on")
	assert.Equal(t, userSwitches{tenantManager: true, baseStationManager: true}, h.userSwitches(partial))
	assert.Equal(t, membershipSwitches{}, h.membershipSwitches(orgID, plain))
	assert.Equal(t, membershipSwitches{orgAdmin: true, endpointAdmin: true}, h.membershipSwitches(orgID, orgAdmin))
	assert.Equal(t, int64(0), h.queryInt(`SELECT count(*) FROM information_schema.tables WHERE table_schema = 'identity' AND table_name LIKE 'role_grandfathered_%'`))
}
