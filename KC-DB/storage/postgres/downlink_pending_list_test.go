package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// TestListPendingDownlinks pins the candidates a connecting base station may
// take ahead of time: every tenant's unexpired pending rows, grouped by
// endpoint with the higher priority and then the older row first; rows a
// station holds and rows whose lifetime elapsed are left out.
func TestListPendingDownlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	const (
		tenant      = int64(351)
		otherTenant = int64(352)
	)
	createTestTenant(t, sqlxDB, tenant, "PendingListTenant")
	createTestTenant(t, sqlxDB, otherTenant, "PendingListOtherTenant")
	orgs := map[int64]uuid.UUID{tenant: uuid.New(), otherTenant: uuid.New()}
	for tenantID, orgID := range orgs {
		_, err := sqlxDB.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, "pending-list-org-"+orgID.String())
		require.NoError(t, err)
	}
	now := time.Date(2027, time.April, 1, 9, 0, 0, 0, time.UTC)
	endpoint := []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x03, 0x51}
	otherEndpoint := []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x03, 0x52}
	seed := func(tenantID, queID int64, epEUI []byte, status string, priority float32, createdAt, latestAt time.Time) {
		_, err := sqlxDB.Exec(`INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, status, que_id, priority, created_at, earliest_at, latest_at)
			VALUES ($1, $2, $3, '\x01', $4, $5, $6, $7, $7, $8)`,
			epEUI, tenantID, orgs[tenantID], status, queID, priority, createdAt, latestAt)
		require.NoError(t, err)
	}
	later := now.Add(time.Hour)
	seed(tenant, 850001, endpoint, "pending", 1, now.Add(-2*time.Minute), later)
	seed(tenant, 850002, endpoint, "pending", 5, now.Add(-time.Minute), later)
	seed(tenant, 850003, endpoint, "pending", 1, now.Add(-3*time.Minute), later)
	seed(otherTenant, 850004, otherEndpoint, "pending", 1, now.Add(-time.Minute), later)
	seed(tenant, 850005, endpoint, "queued", 9, now.Add(-time.Minute), later)
	seed(tenant, 850006, endpoint, "pending", 9, now.Add(-time.Hour), now.Add(-time.Minute))

	logger.Initialize("error", "json")
	db := &DB{clock: testutil.NewFakeClock(now), conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	pending, err := NewRepositories(db).Downlinks.ListPendingDownlinks(t.Context())
	require.NoError(t, err)

	var ours []storage.PendingDownlink
	for _, downlink := range pending {
		if downlink.TenantID == tenant || downlink.TenantID == otherTenant {
			ours = append(ours, downlink)
		}
	}
	assert.Equal(t, []storage.PendingDownlink{
		{QueID: 850002, TenantID: tenant, OrganizationID: orgs[tenant], EpEUI: 0x70B3D56770110351},
		{QueID: 850003, TenantID: tenant, OrganizationID: orgs[tenant], EpEUI: 0x70B3D56770110351},
		{QueID: 850001, TenantID: tenant, OrganizationID: orgs[tenant], EpEUI: 0x70B3D56770110351},
		{QueID: 850004, TenantID: otherTenant, OrganizationID: orgs[otherTenant], EpEUI: 0x70B3D56770110352},
	}, ours)
}
