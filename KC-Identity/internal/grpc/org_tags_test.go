package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	dbadapters "github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// orgTagsFixture is the identity service over the PostgreSQL organization
// store, as the composition root wires it, with an admin and a non-admin caller.
type orgTagsFixture struct {
	admin    *IdentityService
	member   *IdentityService
	adminCtx context.Context
	insert   func(name string) uuid.UUID
}

func newOrgTagsFixture(t *testing.T) *orgTagsFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := testsupport.SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	log := logger.NewNop()
	orgs := admin.NewOrganizationAdminService(postgres.NewOrganizationRepository(db, log),
		adapters.NewTenantStoreAdapterWrapper(dbadapters.NewTenantStoreAdapter(postgres.NewTenantRepository(db))), log)
	callerID := uuid.New()
	recorder, err := audit.NewRecorder(&captureAudit{}, log, discardDrops{})
	require.NoError(t, err)
	service := func(users *mockAdminUserService) *IdentityService {
		base, err := NewIdentityService(log, recorder, recorder)
		require.NoError(t, err)
		return base.WithAdminUserService(users).WithOrganizationService(orgs)
	}
	member := func(_ context.Context, id uuid.UUID) (*models.User, error) {
		return &models.User{ID: id, IsActive: true}, nil
	}
	return &orgTagsFixture{
		admin:    service(&mockAdminUserService{getByIDFunc: adminGetByIDFunc(callerID)}),
		member:   service(&mockAdminUserService{getByIDFunc: member}),
		adminCtx: pkgcontext.WithUserID(testutil.TestContext(), callerID.String()),
		insert: func(name string) uuid.UUID {
			var tenantID int64
			require.NoError(t, db.QueryRow(`INSERT INTO tenants (name, status, created_at, updated_at)
				VALUES ($1, 'active', NOW(), NOW()) RETURNING id`, name).Scan(&tenantID))
			orgID := uuid.New()
			_, err := db.Exec(`INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
				VALUES ($1, $2, $3, 'active', NOW(), NOW())`, orgID, tenantID, name)
			require.NoError(t, err)
			return orgID
		},
	}
}

var orgTagsOfConnection = map[string]string{"kilo_org_id": "0b7c", "kilo_connection_id": "c-17"}

// TestCreateOrganization_TagsRoundTrip: the tags an admin creates an
// organization with are stored, with its description, and read back by
// GetOrganization and ListOrganizations.
func TestCreateOrganization_TagsRoundTrip(t *testing.T) {
	f := newOrgTagsFixture(t)

	created, err := f.admin.CreateOrganization(f.adminCtx, &pb.CreateOrganizationRequest{
		Name: "kilo-connection", Description: "created by the platform", Tags: orgTagsOfConnection,
	})
	require.NoError(t, err)
	id := created.GetOrganization().GetId()

	got, err := f.admin.GetOrganization(f.adminCtx, &pb.GetOrganizationRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, orgTagsOfConnection, got.GetOrganization().GetTags())
	assert.Equal(t, "created by the platform", got.GetOrganization().GetDescription())
	listed, err := f.admin.ListOrganizations(f.adminCtx, &pb.ListOrganizationsRequest{})
	require.NoError(t, err)
	listedTags := map[string]map[string]string{}
	for _, org := range listed.GetOrganizations() {
		listedTags[org.GetId()] = org.GetTags()
	}
	assert.Equal(t, orgTagsOfConnection, listedTags[id])
}

// TestOrganizationTags_AreFixedAtCreation: no caller changes the tags
// afterwards - an admin's update carrying tags is refused and changes
// nothing, and a non-admin can neither create nor update an organization.
func TestOrganizationTags_AreFixedAtCreation(t *testing.T) {
	f := newOrgTagsFixture(t)
	created, err := f.admin.CreateOrganization(f.adminCtx, &pb.CreateOrganizationRequest{Name: "kilo-connection", Tags: orgTagsOfConnection})
	require.NoError(t, err)
	id := created.GetOrganization().GetId()
	forged := map[string]string{"kilo_connection_id": "c-other"}

	_, err = f.admin.UpdateOrganization(f.adminCtx, &pb.UpdateOrganizationRequest{Id: id, Name: "renamed", Tags: forged})
	st := status.Convert(err)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgTagsImmutable), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgTagsImmutable), st.Message())
	memberCtx := pkgcontext.WithUserID(testutil.TestContext(), uuid.New().String())
	_, err = f.member.UpdateOrganization(memberCtx, &pb.UpdateOrganizationRequest{Id: id, Tags: forged})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), status.Code(err))
	_, err = f.member.CreateOrganization(memberCtx, &pb.CreateOrganizationRequest{Name: "forged", Tags: orgTagsOfConnection})
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), status.Code(err))

	got, err := f.admin.GetOrganization(f.adminCtx, &pb.GetOrganizationRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, orgTagsOfConnection, got.GetOrganization().GetTags())
	assert.Equal(t, "kilo-connection", got.GetOrganization().GetName(), "the refused update changed nothing")
}

// TestOrganizationTags_OrganizationsWithoutTagsReadEmpty: an organization
// stored before tags were kept, or created without any, reads with none.
func TestOrganizationTags_OrganizationsWithoutTagsReadEmpty(t *testing.T) {
	f := newOrgTagsFixture(t)
	stored := f.insert("stored-before")
	created, err := f.admin.CreateOrganization(f.adminCtx, &pb.CreateOrganizationRequest{Name: "untagged"})
	require.NoError(t, err)

	for _, id := range []string{stored.String(), created.GetOrganization().GetId()} {
		got, err := f.admin.GetOrganization(f.adminCtx, &pb.GetOrganizationRequest{Id: id})
		require.NoError(t, err)
		assert.Empty(t, got.GetOrganization().GetTags(), id)
	}
}
