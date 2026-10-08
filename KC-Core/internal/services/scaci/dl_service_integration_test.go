package scaciservices

import (
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
)

func seedTenantWithOrganization(t *testing.T, db *postgres.DB, name string) (int64, uuid.UUID) {
	t.Helper()
	ctx := testutil.TestContext()
	var tenantID int64
	require.NoError(t, db.QueryRow(ctx, `INSERT INTO tenants (name) VALUES ($1) RETURNING id`, name).Scan(&tenantID))
	orgID := uuid.New()
	_, err := db.Exec(ctx, `INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, name+"-org")
	require.NoError(t, err)
	return tenantID, orgID
}

func applicationDownlink(tenantID int64, orgID uuid.UUID, acQueID uint64) *storage.DownlinkMessage {
	return &storage.DownlinkMessage{
		EPEUI:          "70B3D59CD0000400",
		TenantID:       strconv.FormatInt(tenantID, 10),
		OrganizationID: &orgID,
		Payload:        []byte{0x01},
		Status:         mioty.DLQueueStatusPending,
		ACQueID:        &acQueID,
	}
}

// TestDLService_ApplicationQueueIDsAreTenantScoped runs the dlDataQue
// persistence path against PostgreSQL: the same Application Center queue id
// is queued independently by two tenants under distinct service center ids,
// a duplicate inside one tenant is refused as that tenant's duplicate, and a
// service center id that is already taken is drawn again (SCACI §3.10.1).
func TestDLService_ApplicationQueueIDsAreTenantScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()
	ctx := testutil.TestContext()
	tenantA, orgA := seedTenantWithOrganization(t, db, "ac-queue-tenant-a")
	tenantB, orgB := seedTenantWithOrganization(t, db, "ac-queue-tenant-b")

	ids := &sequenceQueueIDs{ids: []int64{5_000_001, 5_000_001, 5_000_002, 5_000_003}}
	repos := postgres.NewRepositories(db)
	svc := newTestDLServiceWithIDs(t, new(mockDownlinkScheduler), repos.Downlinks, ids)

	storedA, err := svc.EnqueueDownlink(ctx, applicationDownlink(tenantA, orgA, 42))
	require.NoError(t, err)
	assert.Equal(t, int64(5_000_001), storedA.QueID)

	storedB, err := svc.EnqueueDownlink(ctx, applicationDownlink(tenantB, orgB, 42))
	require.NoError(t, err, "another tenant's Application Center may reuse the queue id")
	assert.Equal(t, int64(5_000_002), storedB.QueID, "the taken service center id is drawn again")

	_, err = svc.EnqueueDownlink(ctx, applicationDownlink(tenantA, orgA, 42))
	require.ErrorIs(t, err, storage.ErrDuplicateKey)

	rowA, err := repos.Downlinks.GetDownlinkByQueueID(ctx, 5_000_001, strconv.FormatInt(tenantA, 10))
	require.NoError(t, err)
	require.NotNil(t, rowA.ACQueID)
	assert.Equal(t, uint64(42), *rowA.ACQueID)
	_, err = repos.Downlinks.GetDownlinkByQueueID(ctx, 5_000_001, strconv.FormatInt(tenantB, 10))
	require.Error(t, err, "a tenant never reads another tenant's row by its service center id")
}
