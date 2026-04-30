package grpc

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/kilocenter/pkg/context"
)

// ============================================================================
// Mock Implementations (shared with identity_service_tenant_isolation_test.go)
// ============================================================================

// mockLogger implements logger.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(_ string, _ ...interface{})                           {}
func (m *mockLogger) Info(_ string, _ ...interface{})                            {}
func (m *mockLogger) Warn(_ string, _ ...interface{})                            {}
func (m *mockLogger) Error(_ string, _ ...interface{})                           {}
func (m *mockLogger) Fatal(_ string, _ ...interface{})                           {}
func (m *mockLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *mockLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *mockLogger) WithField(_ string, _ interface{}) logger.Logger            { return m }
func (m *mockLogger) WithFields(_ map[string]interface{}) logger.Logger          { return m }

// contextForTenant creates a test context with the given tenant ID, a random org ID, and a random user ID.
func contextForTenant(tenantID int64) context.Context {
	ctx := testutil.TestContextWithTenant(tenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())
	ctx = pkgcontext.WithUserID(ctx, uuid.New().String())
	return ctx
}

// mockAPIKeyService implements grpcservices.APIKeyService for testing
type mockAPIKeyService struct {
	createFunc           func(ctx context.Context, req *grpcservices.APIKeyCreateRequest) (*grpcservices.APIKeyCreateResponse, error)
	getByIDAndOrgFunc    func(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error)
	deleteByIDAndOrgFunc func(ctx context.Context, id, orgID uuid.UUID) error
	listFunc             func(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, int64, error)
}

func (m *mockAPIKeyService) Create(ctx context.Context, req *grpcservices.APIKeyCreateRequest) (*grpcservices.APIKeyCreateResponse, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAPIKeyService) GetByID(_ context.Context, _ uuid.UUID) (*models.APIKey, error) {
	return nil, nil
}

func (m *mockAPIKeyService) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockAPIKeyService) List(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, tenantID, orgID, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockAPIKeyService) GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error) {
	if m.getByIDAndOrgFunc != nil {
		return m.getByIDAndOrgFunc(ctx, id, orgID)
	}
	return nil, nil
}

func (m *mockAPIKeyService) DeleteByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) error {
	if m.deleteByIDAndOrgFunc != nil {
		return m.deleteByIDAndOrgFunc(ctx, id, orgID)
	}
	return nil
}

// mockMembershipService implements grpcservices.MembershipService for testing
type mockMembershipService struct {
	addUserFunc               func(ctx context.Context, orgID, userID uuid.UUID, role string) error
	getMembershipFunc         func(ctx context.Context, orgID, userID uuid.UUID) (*grpcservices.OrganizationMember, error)
	updateRoleFunc            func(ctx context.Context, orgID, userID uuid.UUID, role string) error
	updatePermissionsFunc     func(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error
	removeUserFunc            func(ctx context.Context, orgID, userID uuid.UUID) error
	listMembersFunc           func(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*grpcservices.OrganizationMember, int64, error)
	listUserOrganizationsFunc func(ctx context.Context, userID uuid.UUID, tenantID int64) ([]grpcservices.OrganizationMembership, error)
}

func (m *mockMembershipService) AddUser(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	if m.addUserFunc != nil {
		return m.addUserFunc(ctx, orgID, userID, role)
	}
	return nil
}

func (m *mockMembershipService) GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*grpcservices.OrganizationMember, error) {
	if m.getMembershipFunc != nil {
		return m.getMembershipFunc(ctx, orgID, userID)
	}
	return nil, nil
}

func (m *mockMembershipService) UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	if m.updateRoleFunc != nil {
		return m.updateRoleFunc(ctx, orgID, userID, role)
	}
	return nil
}

func (m *mockMembershipService) UpdatePermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	if m.updatePermissionsFunc != nil {
		return m.updatePermissionsFunc(ctx, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin)
	}
	return nil
}

func (m *mockMembershipService) RemoveUser(ctx context.Context, orgID, userID uuid.UUID) error {
	if m.removeUserFunc != nil {
		return m.removeUserFunc(ctx, orgID, userID)
	}
	return nil
}

func (m *mockMembershipService) ListMembers(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*grpcservices.OrganizationMember, int64, error) {
	if m.listMembersFunc != nil {
		return m.listMembersFunc(ctx, orgID, status, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockMembershipService) ListUserOrganizations(ctx context.Context, userID uuid.UUID, tenantID int64) ([]grpcservices.OrganizationMembership, error) {
	if m.listUserOrganizationsFunc != nil {
		return m.listUserOrganizationsFunc(ctx, userID, tenantID)
	}
	return nil, nil
}

// ============================================================================
// API Key & Membership Validation Tests
// ============================================================================

func TestCreateApiKey_InvalidKeyType(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		apiKeySvc: &mockAPIKeyService{},
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(10)
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	invalidKeyType := models.KeyTypeUser + "_invalid"
	req := &pb.CreateApiKeyRequest{
		Name:    "test-key",
		KeyType: invalidKeyType,
	}

	_, err := svc.CreateApiKey(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidApiKeyType), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidApiKeyType), st.Message())
}

func TestCreateApiKey_NonAdminRejected(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		apiKeySvc: &mockAPIKeyService{},
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				if id == callerID {
					return &models.User{ID: callerID, Email: "nonadmin@example.com", IsAdmin: false, IsActive: true}, nil
				}
				return nil, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContextWithTenant(10)
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	req := &pb.CreateApiKeyRequest{
		Name:    "test-key",
		KeyType: models.KeyTypeUser,
	}

	_, err := svc.CreateApiKey(ctx, req)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), st.Code())
}

func TestGetApiKey_OrgScoped(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	keyID := uuid.New()

	svc := &IdentityService{
		apiKeySvc: &mockAPIKeyService{
			getByIDAndOrgFunc: func(_ context.Context, id, org uuid.UUID) (*models.APIKey, error) {
				if org != orgID {
					return nil, interfaces.ErrRecordNotFound
				}
				return &models.APIKey{
					ID:    id,
					OrgID: org,
				}, nil
			},
		},
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		log: &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	ctx = pkgcontext.WithOrganizationID(ctx, orgID)
	resp, err := svc.GetApiKey(ctx, &pb.GetApiKeyRequest{Id: keyID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.ApiKey)
	assert.Equal(t, keyID.String(), resp.ApiKey.Id)

	otherCtx := adminTestContext(callerID)
	otherCtx = pkgcontext.WithOrganizationID(otherCtx, uuid.New())
	_, err = svc.GetApiKey(otherCtx, &pb.GetApiKeyRequest{Id: keyID.String()})
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenApiKeyNotFound), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenApiKeyNotFound), st.Message())
}

func TestListOrganizationUsers_StatusFilterInactive(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	var capturedStatus string

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			listMembersFunc: func(_ context.Context, id uuid.UUID, status string, _, _ int) ([]*grpcservices.OrganizationMember, int64, error) {
				capturedStatus = status
				assert.Equal(t, orgID, id)
				return []*grpcservices.OrganizationMember{}, 0, nil
			},
		},
		log: &mockLogger{},
	}

	req := &pb.ListOrganizationUsersRequest{
		OrgId:  orgID.String(),
		Status: models.OrganizationMemberStatusFilterInactive,
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	resp, err := svc.ListOrganizationUsers(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, models.OrganizationMemberStatusFilterInactive, capturedStatus)
}

func TestAddOrganizationUser_InvalidRole(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			addUserFunc: func(_ context.Context, _, _ uuid.UUID, _ string) error {
				t.Fatalf("AddUser should not be called for invalid roles")
				return nil
			},
		},
		log: &mockLogger{},
	}

	invalidRole := models.OrganizationRoleMember + "_invalid"
	req := &pb.AddOrganizationUserRequest{
		OrgId:  uuid.New().String(),
		UserId: uuid.New().String(),
		Role:   invalidRole,
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.AddOrganizationUser(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidOrgRole), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidOrgRole), st.Message())
}

// ============================================================================
// Cross-Tenant Org Access Rejection Tests
// ============================================================================

// mockOrgService implements grpcservices.OrganizationService for testing.
type mockOrgService struct {
	getByIDFunc         func(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error)
	getByIDUnscopedFunc func(ctx context.Context, id uuid.UUID) (*models.Organization, error)
}

func (m *mockOrgService) Create(_ context.Context, _ *grpcservices.OrganizationCreateRequest) (*models.Organization, error) {
	return nil, nil
}
func (m *mockOrgService) GetByID(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id, tenantID)
	}
	return nil, nil
}
func (m *mockOrgService) Update(_ context.Context, _ uuid.UUID, _ int64, _ *grpcservices.OrganizationUpdateRequest) (*models.Organization, error) {
	return nil, nil
}
func (m *mockOrgService) Delete(_ context.Context, _ uuid.UUID, _ int64) error { return nil }
func (m *mockOrgService) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	if m.getByIDUnscopedFunc != nil {
		return m.getByIDUnscopedFunc(ctx, id)
	}
	return &models.Organization{OrgID: id, TenantID: 100, Name: "Test Org"}, nil
}
func (m *mockOrgService) List(_ context.Context, _ int64, _, _ int) ([]*models.Organization, int64, error) {
	return nil, 0, nil
}
func (m *mockOrgService) ListAll(_ context.Context, _, _ int) ([]*models.Organization, int64, error) {
	return nil, 0, nil
}

// crossTenantOrgService returns a mock that rejects all org lookups (simulates cross-tenant denial).
func crossTenantOrgService() *mockOrgService {
	return &mockOrgService{
		getByIDFunc: func(_ context.Context, _ uuid.UUID, _ int64) (*models.Organization, error) {
			return nil, interfaces.ErrRecordNotFound
		},
		getByIDUnscopedFunc: func(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}
}

// sameTenantOrgService returns a mock that accepts all GetByID calls within the tenant.
func sameTenantOrgService() *mockOrgService {
	return &mockOrgService{
		getByIDFunc: func(_ context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error) {
			return &models.Organization{OrgID: id, TenantID: tenantID, Name: "Test Org"}, nil
		},
	}
}

func TestCrossTenantOrg_AddOrgUser(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc:        crossTenantOrgService(),
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	req := &pb.AddOrganizationUserRequest{
		OrgId:  uuid.New().String(),
		UserId: uuid.New().String(),
		Role:   "member",
	}

	_, err := svc.AddOrganizationUser(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

func TestCrossTenantOrg_GetOrgUser(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc:        crossTenantOrgService(),
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	req := &pb.GetOrganizationUserRequest{
		OrgId:  uuid.New().String(),
		UserId: uuid.New().String(),
	}

	_, err := svc.GetOrganizationUser(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

func TestCrossTenantOrg_UpdateOrgUser(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc:        crossTenantOrgService(),
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	req := &pb.UpdateOrganizationUserRequest{
		OrgId:  uuid.New().String(),
		UserId: uuid.New().String(),
		Role:   "admin",
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"role"},
		},
	}

	_, err := svc.UpdateOrganizationUser(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

func TestCrossTenantOrg_RemoveOrgUser(t *testing.T) {
	callerID := uuid.New()
	targetUserID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc:        crossTenantOrgService(),
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	req := &pb.RemoveOrganizationUserRequest{
		OrgId:  uuid.New().String(),
		UserId: targetUserID.String(),
	}

	_, err := svc.RemoveOrganizationUser(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

func TestCrossTenantOrg_ListOrgUsers(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc:        crossTenantOrgService(),
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	req := &pb.ListOrganizationUsersRequest{
		OrgId: uuid.New().String(),
	}

	_, err := svc.ListOrganizationUsers(ctx, req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgNotFound), st.Code())
}

// ============================================================================
// API Key Tests
// ============================================================================

func TestListApiKeys_AdminOnly(t *testing.T) {
	callerID := uuid.New()
	nonAdminUserSvc := &mockAdminUserService{
		getByIDFunc: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
			return &models.User{ID: callerID, IsAdmin: false, IsActive: true}, nil
		},
	}
	svc := &IdentityService{
		apiKeySvc:    &mockAPIKeyService{},
		adminUserSvc: nonAdminUserSvc,
	}
	ctx := adminTestContext(callerID)
	_, err := svc.ListApiKeys(ctx, &pb.ListApiKeysRequest{})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}

func TestListApiKeys_OrgAndTenantScoped(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	var capturedTenantID int64
	var capturedOrgID uuid.UUID

	svc := &IdentityService{
		apiKeySvc: &mockAPIKeyService{
			listFunc: func(_ context.Context, tID int64, oID uuid.UUID, _ *uuid.UUID, _, _ int) ([]*models.APIKey, int64, error) {
				capturedTenantID = tID
				capturedOrgID = oID
				return nil, 0, nil
			},
		},
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: func(_ context.Context, _ uuid.UUID) (*models.User, error) {
				return &models.User{ID: callerID, IsAdmin: true, IsActive: true}, nil
			},
		},
	}
	// Build context with tenant + user + org (mirrors production interceptor chain)
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), int64(42))
	ctx = pkgcontext.WithUserID(ctx, callerID.String())
	ctx = pkgcontext.WithOrganizationID(ctx, orgID)

	_, err := svc.ListApiKeys(ctx, &pb.ListApiKeysRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTenantID != 42 {
		t.Errorf("expected tenantID 42 forwarded to service, got %v", capturedTenantID)
	}
	if capturedOrgID != orgID {
		t.Errorf("expected orgID %v forwarded to service, got %v", orgID, capturedOrgID)
	}
}
