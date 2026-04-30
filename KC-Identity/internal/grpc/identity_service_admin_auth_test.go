package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/admin"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/kilocenter/pkg/context"
)

// ============================================================================
// requireServerAdmin Tests (tested via ListUsers which calls requireServerAdmin)
// ============================================================================

func TestRequireServerAdmin_Success(t *testing.T) {
	callerID := uuid.New()
	now := time.Now()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
			listFunc: func(_ context.Context, _, _ int) ([]*models.User, int64, error) {
				return []*models.User{
					{ID: uuid.New(), Email: "u@example.com", IsActive: true, CreatedAt: now, UpdatedAt: now},
				}, 1, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	resp, err := svc.ListUsers(ctx, &pb.ListUsersRequest{PageSize: 10})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 1)
}

func TestRequireServerAdmin_NonAdmin(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, IsAdmin: false, IsActive: true}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	_, err := svc.ListUsers(ctx, &pb.ListUsersRequest{PageSize: 10})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired), st.Message())
}

// ============================================================================
// requireOrgAdmin Tests (tested via ListOrganizationUsers which calls resolveOrgAccess)
// ============================================================================

func TestRequireOrgAdmin_Success(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			// Non-admin caller so requireServerAdmin fails and falls through to requireOrgAdmin
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, IsAdmin: false, IsActive: true}, nil
			},
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				if oID == orgID && uID == callerID {
					return &grpcservices.OrganizationMember{OrgID: oID, UserID: uID, IsOrgAdmin: true, Role: "admin"}, nil
				}
				return nil, fmt.Errorf("not found")
			},
			listMembersFunc: func(_ context.Context, _ uuid.UUID, _ string, _, _ int) ([]*grpcservices.OrganizationMember, int64, error) {
				return []*grpcservices.OrganizationMember{}, 0, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	resp, err := svc.ListOrganizationUsers(ctx, &pb.ListOrganizationUsersRequest{
		OrgId:    orgID.String(),
		PageSize: 10,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestRequireOrgAdmin_WrongOrg(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	wrongOrgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, IsAdmin: false, IsActive: true}, nil
			},
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			// Caller is org admin of orgID but not wrongOrgID
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				if oID == orgID && uID == callerID {
					return &grpcservices.OrganizationMember{OrgID: oID, UserID: uID, IsOrgAdmin: true, Role: "admin"}, nil
				}
				return nil, fmt.Errorf("not found")
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.ListOrganizationUsers(ctx, &pb.ListOrganizationUsersRequest{
		OrgId:    wrongOrgID.String(),
		PageSize: 10,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgAdminRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgAdminRequired), st.Message())
}

func TestRequireOrgAdmin_NonAdmin(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, IsAdmin: false, IsActive: true}, nil
			},
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			// Caller is a member but not an org admin
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				if oID == orgID && uID == callerID {
					return &grpcservices.OrganizationMember{OrgID: oID, UserID: uID, IsOrgAdmin: false, Role: "member"}, nil
				}
				return nil, fmt.Errorf("not found")
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.ListOrganizationUsers(ctx, &pb.ListOrganizationUsersRequest{
		OrgId:    orgID.String(),
		PageSize: 10,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgAdminRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgAdminRequired), st.Message())
}

// ============================================================================
// requireServerOrOrgAdmin Tests (tested via AddOrganizationUser which calls resolveOrgAccess)
// ============================================================================

func TestRequireServerOrOrgAdmin_ServerAdmin(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	targetUserID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc: &mockOrgService{
			getByIDUnscopedFunc: func(_ context.Context, id uuid.UUID) (*models.Organization, error) {
				return &models.Organization{OrgID: id, TenantID: 42, Name: "Test Org"}, nil
			},
		},
		membershipSvc: &mockMembershipService{
			addUserFunc: func(_ context.Context, _, _ uuid.UUID, _ string) error {
				return nil
			},
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				return &grpcservices.OrganizationMember{
					OrgID:  oID,
					UserID: uID,
					Role:   models.OrganizationRoleMember,
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	resp, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId:  orgID.String(),
		UserId: targetUserID.String(),
		Role:   models.OrganizationRoleMember,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Member)
}

func TestRequireServerOrOrgAdmin_OrgAdmin(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	targetUserID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			// Non-admin caller; requireServerAdmin will fail, falling through to requireOrgAdmin
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, IsAdmin: false, IsActive: true}, nil
			},
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				// Return org admin for the caller, and the newly added member for the target
				if oID == orgID && uID == callerID {
					return &grpcservices.OrganizationMember{OrgID: oID, UserID: uID, IsOrgAdmin: true, Role: "admin"}, nil
				}
				if oID == orgID && uID == targetUserID {
					return &grpcservices.OrganizationMember{OrgID: oID, UserID: uID, Role: models.OrganizationRoleMember}, nil
				}
				return nil, fmt.Errorf("not found")
			},
			addUserFunc: func(_ context.Context, _, _ uuid.UUID, _ string) error {
				return nil
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	resp, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId:  orgID.String(),
		UserId: targetUserID.String(),
		Role:   models.OrganizationRoleMember,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Member)
}

func TestRequireServerOrOrgAdmin_Neither(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, IsAdmin: false, IsActive: true}, nil
			},
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			// Caller is a regular member, not org admin
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				if oID == orgID && uID == callerID {
					return &grpcservices.OrganizationMember{OrgID: oID, UserID: uID, IsOrgAdmin: false, Role: "member"}, nil
				}
				return nil, fmt.Errorf("not found")
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId:  orgID.String(),
		UserId: uuid.New().String(),
		Role:   models.OrganizationRoleMember,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenOrgAdminRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenOrgAdminRequired), st.Message())
}

// ============================================================================
// AddOrganizationUser Tests — add-by-email and one-of validation
// ============================================================================

func TestAddOrgUser_ByUserId_ServerAdmin(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	targetUserID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc: &mockOrgService{
			getByIDUnscopedFunc: func(_ context.Context, id uuid.UUID) (*models.Organization, error) {
				return &models.Organization{OrgID: id, TenantID: 42, Name: "Test Org"}, nil
			},
		},
		membershipSvc: &mockMembershipService{
			addUserFunc: func(_ context.Context, oID, uID uuid.UUID, role string) error {
				assert.Equal(t, orgID, oID)
				assert.Equal(t, targetUserID, uID)
				assert.Equal(t, models.OrganizationRoleMember, role)
				return nil
			},
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				return &grpcservices.OrganizationMember{
					OrgID:  oID,
					UserID: uID,
					Role:   models.OrganizationRoleMember,
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	resp, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId:  orgID.String(),
		UserId: targetUserID.String(),
		Role:   models.OrganizationRoleMember,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Member)
	assert.Equal(t, targetUserID.String(), resp.Member.UserId)
}

func TestAddOrgUser_ByEmail_OrgAdmin(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()
	targetUserID := uuid.New()
	targetEmail := "target@example.com"

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			// Non-admin caller; falls through to org admin check
			getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
				return &models.User{ID: id, IsAdmin: false, IsActive: true}, nil
			},
			getByEmailFunc: func(_ context.Context, email string) (*models.User, error) {
				if email == targetEmail {
					return &models.User{ID: targetUserID, Email: targetEmail, IsActive: true}, nil
				}
				return nil, fmt.Errorf("not found")
			},
		},
		orgSvc: sameTenantOrgService(),
		membershipSvc: &mockMembershipService{
			getMembershipFunc: func(_ context.Context, oID, uID uuid.UUID) (*grpcservices.OrganizationMember, error) {
				if oID == orgID && uID == callerID {
					return &grpcservices.OrganizationMember{OrgID: oID, UserID: uID, IsOrgAdmin: true, Role: "admin"}, nil
				}
				if oID == orgID && uID == targetUserID {
					return &grpcservices.OrganizationMember{
						OrgID:  oID,
						UserID: uID,
						Role:   models.OrganizationRoleMember,
					}, nil
				}
				return nil, fmt.Errorf("not found")
			},
			addUserFunc: func(_ context.Context, oID, uID uuid.UUID, role string) error {
				assert.Equal(t, orgID, oID)
				assert.Equal(t, targetUserID, uID)
				assert.Equal(t, models.OrganizationRoleMember, role)
				return nil
			},
		},
		log: &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	resp, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId: orgID.String(),
		Email: targetEmail,
		Role:  models.OrganizationRoleMember,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Member)
	assert.Equal(t, targetUserID.String(), resp.Member.UserId)
}

func TestAddOrgUser_ByEmail_UserNotFound(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
			getByEmailFunc: func(_ context.Context, _ string) (*models.User, error) {
				return nil, admin.ErrUserNotFound
			},
		},
		orgSvc: &mockOrgService{
			getByIDUnscopedFunc: func(_ context.Context, id uuid.UUID) (*models.Organization, error) {
				return &models.Organization{OrgID: id, TenantID: 42, Name: "Test Org"}, nil
			},
		},
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId: orgID.String(),
		Email: "unknown@example.com",
		Role:  models.OrganizationRoleMember,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserNotFound), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserNotFound), st.Message())
}

func TestAddOrgUser_BothUserIdAndEmail(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc: &mockOrgService{
			getByIDUnscopedFunc: func(_ context.Context, id uuid.UUID) (*models.Organization, error) {
				return &models.Organization{OrgID: id, TenantID: 42, Name: "Test Org"}, nil
			},
		},
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId:  orgID.String(),
		UserId: uuid.New().String(),
		Email:  "user@example.com",
		Role:   models.OrganizationRoleMember,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserEmailRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserEmailRequired), st.Message())
}

func TestAddOrgUser_NeitherUserIdNorEmail(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		orgSvc: &mockOrgService{
			getByIDUnscopedFunc: func(_ context.Context, id uuid.UUID) (*models.Organization, error) {
				return &models.Organization{OrgID: id, TenantID: 42, Name: "Test Org"}, nil
			},
		},
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId: orgID.String(),
		Role:  models.OrganizationRoleMember,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUserEmailRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUserEmailRequired), st.Message())
}

func TestAddOrgUser_ByEmail_StoreError(t *testing.T) {
	callerID := uuid.New()
	orgID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
			getByEmailFunc: func(_ context.Context, _ string) (*models.User, error) {
				return nil, fmt.Errorf("connection refused")
			},
		},
		orgSvc: &mockOrgService{
			getByIDUnscopedFunc: func(_ context.Context, id uuid.UUID) (*models.Organization, error) {
				return &models.Organization{OrgID: id, TenantID: 42, Name: "Test Org"}, nil
			},
		},
		membershipSvc: &mockMembershipService{},
		log:           &mockLogger{},
	}

	ctx := contextForTenant(42)
	ctx = pkgcontext.WithUserID(ctx, callerID.String())

	_, err := svc.AddOrganizationUser(ctx, &pb.AddOrganizationUserRequest{
		OrgId: orgID.String(),
		Email: "user@example.com",
		Role:  models.OrganizationRoleMember,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError), st.Message())
}
