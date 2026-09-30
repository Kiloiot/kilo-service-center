package admin_test

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	dbadapters "github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
)

func TestMain(m *testing.M) {
	os.Exit(testsupport.Main(m))
}

// TestOrganizationAdminService_MissingOrganizationIsNotFound wires the service
// to the PostgreSQL repository as the composition root does: a missing
// organization must surface as ErrOrganizationNotFound on every scoped path,
// never as an internal failure.
func TestOrganizationAdminService_MissingOrganizationIsNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := testsupport.SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	log := logger.Get()
	svc := admin.NewOrganizationAdminService(
		postgres.NewOrganizationRepository(db, log),
		adapters.NewTenantStoreAdapterWrapper(dbadapters.NewTenantStoreAdapter(postgres.NewTenantRepository(db))),
		log)
	ctx := testutil.TestContext()
	missing := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	const tenantID = int64(1)
	name := "renamed"

	_, err := svc.GetByID(ctx, missing, tenantID)
	assert.ErrorIs(t, err, admin.ErrOrganizationNotFound, "GetByID")
	_, err = svc.Update(ctx, missing, tenantID, &grpcservices.OrganizationUpdateRequest{Name: &name})
	assert.ErrorIs(t, err, admin.ErrOrganizationNotFound, "Update")
	assert.ErrorIs(t, svc.Delete(ctx, missing, tenantID), admin.ErrOrganizationNotFound, "Delete")
}
