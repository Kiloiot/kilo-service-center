package postgres

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

const (
	scopedRevokeQueID     = int64(840071)
	scopedRevokeOtherEUI  = uint64(0x70b3d59cd0000342)
	scopedRevokeOwnerName = "revoke-owner"
	scopedRevokeOtherName = "revoke-other"
)

// seedOrganization adds an organization to the revoke tenant.
func seedOrganization(t *testing.T, db *sqlx.DB, name string) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	_, err := db.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, revokeTestTenant, name)
	require.NoError(t, err)
	return orgID
}

// seedPendingDownlinkOf queues a pending downlink of the organization for the revoke endpoint.
func seedPendingDownlinkOf(t *testing.T, db *sqlx.DB, orgID uuid.UUID, queID int64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at)
		VALUES (decode('70b3d59cd0000341', 'hex'), $1, $2, '\x01', 'pending', $3, NULL)`, revokeTestTenant, orgID, queID)
	require.NoError(t, err)
}

func revocationBy(orgID uuid.UUID, epEUI uint64) storage.DownlinkRevocation {
	return storage.DownlinkRevocation{QueID: scopedRevokeQueID, TenantID: revokeTestTenant, OrganizationID: &orgID, EpEUI: &epEUI}
}

// TestRevokeDownlink_AnotherOrganizationRevokesNothing: organization B of the
// same tenant names organization A's queue id; the row stays pending and the
// revoke lookup finds no row, so B cannot tell the id from a missing one.
func TestRevokeDownlink_AnotherOrganizationRevokesNothing(t *testing.T) {
	downlinks, sqlxDB, _ := revokeFixture(t)
	owner := seedOrganization(t, sqlxDB, scopedRevokeOwnerName)
	other := seedOrganization(t, sqlxDB, scopedRevokeOtherName)
	seedPendingDownlinkOf(t, sqlxDB, owner, scopedRevokeQueID)

	revoked, err := downlinks.RevokeDownlink(t.Context(), revocationBy(other, revokeTestEndpoint))
	require.NoError(t, err)
	assert.False(t, revoked, "another organization's revoke must not end the downlink")
	_, err = downlinks.GetDownlinkByRevocation(t.Context(), revocationBy(other, revokeTestEndpoint))
	assert.ErrorIs(t, err, sql.ErrNoRows, "another organization's lookup must not find the downlink")

	var status string
	require.NoError(t, sqlxDB.Get(&status, `SELECT status FROM downlink_queue WHERE que_id = $1`, scopedRevokeQueID))
	assert.Equal(t, "pending", status)

	revoked, err = downlinks.RevokeDownlink(t.Context(), revocationBy(owner, revokeTestEndpoint))
	require.NoError(t, err)
	assert.True(t, revoked, "the owning organization still revokes it")
}

// TestRevokeDownlink_AnotherEndpointRevokesNothing: the queue id of one
// endpoint's downlink named under another endpoint of the same organization
// revokes nothing.
func TestRevokeDownlink_AnotherEndpointRevokesNothing(t *testing.T) {
	downlinks, sqlxDB, _ := revokeFixture(t)
	owner := seedOrganization(t, sqlxDB, scopedRevokeOwnerName)
	seedPendingDownlinkOf(t, sqlxDB, owner, scopedRevokeQueID)

	revoked, err := downlinks.RevokeDownlink(t.Context(), revocationBy(owner, scopedRevokeOtherEUI))
	require.NoError(t, err)
	assert.False(t, revoked)
	_, err = downlinks.GetDownlinkByRevocation(t.Context(), revocationBy(owner, scopedRevokeOtherEUI))
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
