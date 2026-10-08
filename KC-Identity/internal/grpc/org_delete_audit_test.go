package grpc

import (
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbadapters "github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

func TestMain(m *testing.M) { os.Exit(testsupport.Main(m)) }

// countingDrops counts the audit events the recorder reports as dropped.
type countingDrops struct{ byType map[string]int }

func (c *countingDrops) Inc(eventType string) { c.byType[eventType]++ }

const (
	orgDeleteTenant         int64 = 7201
	orgDeletePlatformTenant int64 = 1
	orgDeleteName                 = "audit-persistence-org"
)

// Deleting the last organization of a tenant removes the tenant in the same
// transaction; the deletion must still be on the audit trail.
func TestDeleteOrganization_PersistsTheDeletionWhenTheTenantGoesWithIt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := testsupport.SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	_, err := db.Exec(`INSERT INTO tenants (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())`, orgDeleteTenant, orgDeleteName, t.Name())
	require.NoError(t, err)
	orgID := uuid.New()
	_, err = db.Exec(`INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', NOW(), NOW())`, orgID, orgDeleteTenant, orgDeleteName)
	require.NoError(t, err)

	log := logger.NewNop()
	events := postgres.NewSystemEventStore(db.DB, clock.SystemClock{}, log)
	emitter, err := audit.NewEmitter(events, clock.SystemClock{})
	require.NoError(t, err)
	drops := &countingDrops{byType: map[string]int{}}
	recorder, err := audit.NewRecorder(emitter, log, drops)
	require.NoError(t, err)
	callerID := uuid.New()
	base, err := NewIdentityService(log, recorder, recorder)
	require.NoError(t, err)
	svc := base.
		WithAdminUserService(&mockAdminUserService{getByIDFunc: adminGetByIDFunc(callerID)}).
		WithOrganizationService(admin.NewOrganizationAdminService(
			postgres.NewOrganizationRepository(db, log),
			adapters.NewTenantStoreAdapterWrapper(dbadapters.NewTenantStoreAdapter(postgres.NewTenantRepository(db))),
			log)).
		WithPlatformTenantID(orgDeletePlatformTenant)
	ctx := pkgcontext.WithUserID(contextForTenant(orgDeleteTenant), callerID.String())

	_, err = svc.DeleteOrganization(ctx, &pb.DeleteOrganizationRequest{Id: orgID.String()})
	require.NoError(t, err)
	assert.Empty(t, drops.byType, "no audit event is dropped")

	var remaining int
	require.NoError(t, db.Get(&remaining, `SELECT COUNT(*) FROM tenants WHERE id = $1`, orgDeleteTenant))
	require.Zero(t, remaining, "the orphaned tenant is removed with its last organization")

	var rows []struct {
		TenantID sql.NullInt64 `db:"tenant_id"`
		UserID   string        `db:"user_id"`
		Data     []byte        `db:"data"`
	}
	require.NoError(t, db.Select(&rows, `SELECT tenant_id, COALESCE(user_id, '') AS user_id, data
		FROM system_events WHERE event_type = $1`, models.EventTypeOrgDeleted))
	require.Len(t, rows, 1, "the deletion is on the audit trail")
	assert.Equal(t, orgDeletePlatformTenant, rows[0].TenantID.Int64, "filed under the platform tenant, which outlives the deleted one")
	assert.Equal(t, callerID.String(), rows[0].UserID)
	var details map[string]any
	require.NoError(t, json.Unmarshal(rows[0].Data, &details))
	assert.Equal(t, orgID.String(), details[auditKeyOrgID])
	assert.EqualValues(t, orgDeleteTenant, details[models.EventDetailKeyTenantID], "the deleted organization's tenant is kept")
}
