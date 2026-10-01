package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	scaciScopePacketCnt = uint32(7)
	scaciScopeOwnerQue  = int64(850001)
	scaciScopeOtherQue  = int64(850002)
)

// seedCounterDownlinkOf queues a pending downlink of the organization for the
// revoke endpoint, scheduled for the scope's packet counter.
func seedCounterDownlinkOf(t *testing.T, db *sqlx.DB, orgID uuid.UUID, queID int64) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, earliest_at, cnt_depend, packet_cnt)
		VALUES (decode('70b3d59cd0000341', 'hex'), $1, $2, '\x01', 'pending', $3, NULL, true, ARRAY[$4::bigint])`,
		revokeTestTenant, orgID, queID, int64(scaciScopePacketCnt))
	require.NoError(t, err)
}

func counterDownlinksOf(orgID *uuid.UUID) storage.PacketCounterDownlinks {
	return storage.PacketCounterDownlinks{
		TenantID:       revokeTestTenant,
		OrganizationID: orgID,
		EpEUI:          revokeTestEndpoint,
		PacketCnt:      scaciScopePacketCnt,
	}
}

func queueIDsOf(downlinks []*storage.DownlinkMessage) []int64 {
	ids := make([]int64, 0, len(downlinks))
	for _, dl := range downlinks {
		ids = append(ids, dl.QueID)
	}
	return ids
}

// TestSCACIRevoke_AnotherOrganizationKeepsTheDownlinkPending walks a dlDataRev
// of two Application Centers of different organizations in one tenant: each
// finds only its own organization's downlink for the counter, the other
// organization's revoke of a foreign queue id leaves it pending, and the
// owner's own revoke still ends it.
func TestSCACIRevoke_AnotherOrganizationKeepsTheDownlinkPending(t *testing.T) {
	downlinks, sqlxDB, _ := revokeFixture(t)
	owner := seedOrganization(t, sqlxDB, scopedRevokeOwnerName)
	other := seedOrganization(t, sqlxDB, scopedRevokeOtherName)
	seedCounterDownlinkOf(t, sqlxDB, owner, scaciScopeOwnerQue)
	seedCounterDownlinkOf(t, sqlxDB, other, scaciScopeOtherQue)
	endpoint := revokeTestEndpoint

	found, err := downlinks.ListPacketCounterDownlinks(t.Context(), counterDownlinksOf(&other))
	require.NoError(t, err)
	assert.Equal(t, []int64{scaciScopeOtherQue}, queueIDsOf(found), "another organization never finds the owner's downlink")

	foreign := storage.DownlinkRevocation{QueID: scaciScopeOwnerQue, TenantID: revokeTestTenant, OrganizationID: &other, EpEUI: &endpoint}
	revoked, err := downlinks.RevokeDownlink(t.Context(), foreign)
	require.NoError(t, err)
	assert.False(t, revoked, "another organization's revoke must not end the downlink")
	var status string
	require.NoError(t, sqlxDB.Get(&status, `SELECT status FROM downlink_queue WHERE que_id = $1`, scaciScopeOwnerQue))
	assert.Equal(t, "pending", status)

	found, err = downlinks.ListPacketCounterDownlinks(t.Context(), counterDownlinksOf(&owner))
	require.NoError(t, err)
	assert.Equal(t, []int64{scaciScopeOwnerQue}, queueIDsOf(found))
	own := storage.DownlinkRevocation{QueID: scaciScopeOwnerQue, TenantID: revokeTestTenant, OrganizationID: &owner, EpEUI: &endpoint}
	revoked, err = downlinks.RevokeDownlink(t.Context(), own)
	require.NoError(t, err)
	assert.True(t, revoked, "the owning organization still revokes its downlink")
}

// TestListPacketCounterDownlinks_WithoutOrganizationCoversTheTenant: a
// session without an organization addresses every organization's downlink.
func TestListPacketCounterDownlinks_WithoutOrganizationCoversTheTenant(t *testing.T) {
	downlinks, sqlxDB, _ := revokeFixture(t)
	seedCounterDownlinkOf(t, sqlxDB, seedOrganization(t, sqlxDB, scopedRevokeOwnerName), scaciScopeOwnerQue)
	seedCounterDownlinkOf(t, sqlxDB, seedOrganization(t, sqlxDB, scopedRevokeOtherName), scaciScopeOtherQue)

	found, err := downlinks.ListPacketCounterDownlinks(t.Context(), counterDownlinksOf(nil))
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{scaciScopeOwnerQue, scaciScopeOtherQue}, queueIDsOf(found))
}

// TestListInFlightDownlinks_ScopedToOrganization: the endpoint queue a
// deregistration revokes holds only the organization's downlinks.
func TestListInFlightDownlinks_ScopedToOrganization(t *testing.T) {
	downlinks, sqlxDB, _ := revokeFixture(t)
	owner := seedOrganization(t, sqlxDB, scopedRevokeOwnerName)
	other := seedOrganization(t, sqlxDB, scopedRevokeOtherName)
	seedCounterDownlinkOf(t, sqlxDB, owner, scaciScopeOwnerQue)
	seedCounterDownlinkOf(t, sqlxDB, other, scaciScopeOtherQue)
	endpoint := [8]byte(mioty.EUI64Bytes(revokeTestEndpoint))

	found, err := downlinks.ListInFlightDownlinks(t.Context(), revokeTestTenant, storage.DownlinkQueueFilter{EpEUI: &endpoint, OrganizationID: &other})
	require.NoError(t, err)
	assert.Equal(t, []int64{scaciScopeOtherQue}, queueIDsOf(found))

	found, err = downlinks.ListInFlightDownlinks(t.Context(), revokeTestTenant, storage.DownlinkQueueFilter{EpEUI: &endpoint})
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{scaciScopeOwnerQue, scaciScopeOtherQue}, queueIDsOf(found))
}
