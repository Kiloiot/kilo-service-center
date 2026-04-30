package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/kilocenter/pkg/context"
)

// ============================================================================
// Tenant Isolation Mock: Organization Service
// ============================================================================

// mockOrgSvcIsolation captures tenantID to verify tenant propagation.
type mockOrgSvcIsolation struct {
	capturedTenantIDs   []int64
	listAllCallCount    int
	getByIDFunc         func(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error)
	getByIDUnscopedFunc func(ctx context.Context, id uuid.UUID) (*models.Organization, error)
}

func (m *mockOrgSvcIsolation) Create(_ context.Context, _ *grpcservices.OrganizationCreateRequest) (*models.Organization, error) {
	return &models.Organization{
		OrgID:     uuid.New(),
		Name:      "Test Org",
		TenantID:  100,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockOrgSvcIsolation) GetByID(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id, tenantID)
	}
	return &models.Organization{OrgID: id, TenantID: tenantID, Name: "Test Org", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (m *mockOrgSvcIsolation) Update(_ context.Context, _ uuid.UUID, _ int64, _ *grpcservices.OrganizationUpdateRequest) (*models.Organization, error) {
	return nil, nil
}

func (m *mockOrgSvcIsolation) Delete(_ context.Context, _ uuid.UUID, _ int64) error {
	return nil
}

func (m *mockOrgSvcIsolation) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	if m.getByIDUnscopedFunc != nil {
		return m.getByIDUnscopedFunc(ctx, id)
	}
	return &models.Organization{OrgID: id, TenantID: 100, Name: "Test Org", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

func (m *mockOrgSvcIsolation) List(_ context.Context, tenantID int64, _, _ int) ([]*models.Organization, int64, error) {
	m.capturedTenantIDs = append(m.capturedTenantIDs, tenantID)
	return []*models.Organization{}, 0, nil
}

func (m *mockOrgSvcIsolation) ListAll(_ context.Context, _, _ int) ([]*models.Organization, int64, error) {
	m.listAllCallCount++
	return []*models.Organization{}, 0, nil
}

// ============================================================================
// Tenant Isolation Mock: Admin User Service (for requireAdmin)
// ============================================================================

// mockAdminUserSvcIsolation always returns an admin user for the given caller.
type mockAdminUserSvcIsolation struct{}

func (m *mockAdminUserSvcIsolation) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	return &models.User{ID: id, IsAdmin: true}, nil
}

func (m *mockAdminUserSvcIsolation) GetByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcIsolation) GetByExternalID(_ context.Context, _ string) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcIsolation) Create(_ context.Context, _ *grpcservices.UserCreateRequest) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcIsolation) Update(_ context.Context, _ uuid.UUID, _ *grpcservices.UserUpdateRequest) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcIsolation) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockAdminUserSvcIsolation) List(_ context.Context, _, _ int) ([]*models.User, int64, error) {
	return nil, 0, nil
}

func (m *mockAdminUserSvcIsolation) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// mockAdminUserSvcNonAdmin returns a non-admin user for requireAdmin tests.
type mockAdminUserSvcNonAdmin struct{}

func (m *mockAdminUserSvcNonAdmin) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	return &models.User{ID: id, IsAdmin: false}, nil
}

func (m *mockAdminUserSvcNonAdmin) GetByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcNonAdmin) GetByExternalID(_ context.Context, _ string) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcNonAdmin) Create(_ context.Context, _ *grpcservices.UserCreateRequest) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcNonAdmin) Update(_ context.Context, _ uuid.UUID, _ *grpcservices.UserUpdateRequest) (*models.User, error) {
	return nil, nil
}

func (m *mockAdminUserSvcNonAdmin) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockAdminUserSvcNonAdmin) List(_ context.Context, _, _ int) ([]*models.User, int64, error) {
	return nil, 0, nil
}

func (m *mockAdminUserSvcNonAdmin) UpdatePassword(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// ============================================================================
// Helper: Create IdentityService with tenant-capturing mocks
// ============================================================================

type identityIsolationTestServices struct {
	svc     *IdentityService
	orgMock *mockOrgSvcIsolation
}

func newIdentityIsolationTestService() *identityIsolationTestServices {
	orgMock := &mockOrgSvcIsolation{}
	svc := &IdentityService{
		orgSvc:       orgMock,
		adminUserSvc: &mockAdminUserSvcIsolation{},
		log:          &mockLogger{},
	}
	return &identityIsolationTestServices{
		svc:     svc,
		orgMock: orgMock,
	}
}

// ============================================================================
// Category 1: Organizations — Tenant Scoped List
// ============================================================================

func TestTenantIsolation_ListOrganizations_UsesListAll(t *testing.T) {
	ts := newIdentityIsolationTestService()

	// Server admin from tenant 42 lists orgs
	ctx42 := contextForTenant(42)
	_, err := ts.svc.ListOrganizations(ctx42, &pb.ListOrganizationsRequest{PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, ts.orgMock.listAllCallCount)

	// Server admin from tenant 99 lists orgs
	ctx99 := contextForTenant(99)
	_, err = ts.svc.ListOrganizations(ctx99, &pb.ListOrganizationsRequest{PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, ts.orgMock.listAllCallCount)
}

func TestTenantIsolation_OrgCRUD_RequiresAdmin(t *testing.T) {
	ts := newIdentityIsolationTestService()

	// Set adminUserSvc to return non-admin user
	ts.svc.adminUserSvc = &mockAdminUserSvcNonAdmin{}

	ctx := contextForTenant(42)

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "CreateOrganization",
			call: func() error {
				_, err := ts.svc.CreateOrganization(ctx, &pb.CreateOrganizationRequest{Name: "Test"})
				return err
			},
		},
		{
			name: "DeleteOrganization",
			call: func() error {
				_, err := ts.svc.DeleteOrganization(ctx, &pb.DeleteOrganizationRequest{Id: uuid.New().String()})
				return err
			},
		},
		{
			name: "ListOrganizations",
			call: func() error {
				_, err := ts.svc.ListOrganizations(ctx, &pb.ListOrganizationsRequest{PageSize: 10})
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), st.Code(),
				"non-admin should be rejected for %s", tc.name)
		})
	}
}

// ============================================================================
// Category 2: Organization CRUD — Cross-Tenant Org Access Rejection
// ============================================================================

func TestTenantIsolation_GetOrganization_RejectsCrossTenantOrg(t *testing.T) {
	ts := newIdentityIsolationTestService()
	crossTenantOrgID := uuid.New()

	// Mock GetByID to reject: org does not exist in tenant 42
	ts.orgMock.getByIDFunc = func(_ context.Context, _ uuid.UUID, _ int64) (*models.Organization, error) {
		return nil, storage.ErrNotFound
	}

	ctx := testutil.TestContextWithTenant(42)
	ctx = pkgcontext.WithUserID(ctx, uuid.New().String())

	_, err := ts.svc.GetOrganization(ctx, &pb.GetOrganizationRequest{Id: crossTenantOrgID.String()})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

func TestTenantIsolation_UpdateOrganization_RejectsCrossTenantOrg(t *testing.T) {
	ts := newIdentityIsolationTestService()
	crossTenantOrgID := uuid.New()

	// Override GetByIDUnscoped since UpdateOrganization now uses validateOrgAccessUnscoped
	ts.orgMock.getByIDUnscopedFunc = func(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
		return nil, storage.ErrNotFound
	}

	ctx := testutil.TestContextWithTenant(42)
	ctx = pkgcontext.WithUserID(ctx, uuid.New().String())

	_, err := ts.svc.UpdateOrganization(ctx, &pb.UpdateOrganizationRequest{Id: crossTenantOrgID.String(), Name: "new"})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

func TestTenantIsolation_DeleteOrganization_RejectsCrossTenantOrg(t *testing.T) {
	ts := newIdentityIsolationTestService()
	crossTenantOrgID := uuid.New()

	// Override GetByIDUnscoped since DeleteOrganization now uses validateOrgAccessUnscoped
	ts.orgMock.getByIDUnscopedFunc = func(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
		return nil, storage.ErrNotFound
	}

	ctx := testutil.TestContextWithTenant(42)
	ctx = pkgcontext.WithUserID(ctx, uuid.New().String())

	_, err := ts.svc.DeleteOrganization(ctx, &pb.DeleteOrganizationRequest{Id: crossTenantOrgID.String()})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

// ============================================================================
// Category 3: API Keys — Tenant Propagation
// ============================================================================

func TestTenantIsolation_ListApiKeys_TenantPropagation(t *testing.T) {
	var capturedTenantID int64
	mockAPIKey := &mockAPIKeyService{
		listFunc: func(_ context.Context, tenantID int64, _ uuid.UUID, _ *uuid.UUID, _, _ int) ([]*models.APIKey, int64, error) {
			capturedTenantID = tenantID
			return []*models.APIKey{}, 0, nil
		},
	}

	svc := &IdentityService{
		apiKeySvc:    mockAPIKey,
		adminUserSvc: &mockAdminUserSvcIsolation{},
		log:          &mockLogger{},
	}

	ctx42 := contextForTenant(42)
	_, err := svc.ListApiKeys(ctx42, &pb.ListApiKeysRequest{PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(42), capturedTenantID)

	ctx99 := contextForTenant(99)
	_, err = svc.ListApiKeys(ctx99, &pb.ListApiKeysRequest{PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(99), capturedTenantID)
}
