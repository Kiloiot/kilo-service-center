package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/grpc/interceptors"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
	pkgcontext "github.com/kilocenter/pkg/context"
)

// ============================================================================
// Mock Implementations for Auth Tests
// ============================================================================

// mockAuthService implements grpcservices.AuthService for testing
type mockAuthService struct {
	loginFunc           func(ctx context.Context, email, password string) (*grpcservices.AuthLoginResult, error)
	refreshTokensFunc   func(ctx context.Context, refreshToken string) (*grpcservices.AuthTokens, error)
	getProfileFunc      func(ctx context.Context, userID uuid.UUID) (*grpcservices.UserProfile, error)
	logoutFunc          func(ctx context.Context, userID uuid.UUID) error
	changePasswordFunc  func(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	getAuthSettingsFunc func(ctx context.Context) (*grpcservices.AuthSettings, error)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*grpcservices.AuthLoginResult, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, email, password)
	}
	return nil, nil
}

func (m *mockAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*grpcservices.AuthTokens, error) {
	if m.refreshTokensFunc != nil {
		return m.refreshTokensFunc(ctx, refreshToken)
	}
	return nil, nil
}

func (m *mockAuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*grpcservices.UserProfile, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockAuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, userID)
	}
	return nil
}

func (m *mockAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if m.changePasswordFunc != nil {
		return m.changePasswordFunc(ctx, userID, currentPassword, newPassword)
	}
	return nil
}

func (m *mockAuthService) GetAuthSettings(ctx context.Context) (*grpcservices.AuthSettings, error) {
	if m.getAuthSettingsFunc != nil {
		return m.getAuthSettingsFunc(ctx)
	}
	return nil, nil
}

// mockAdminUserService implements grpcservices.AdminUserService for testing
type mockAdminUserService struct {
	createFunc         func(ctx context.Context, req *grpcservices.UserCreateRequest) (*models.User, error)
	getByIDFunc        func(ctx context.Context, id uuid.UUID) (*models.User, error)
	getByEmailFunc     func(ctx context.Context, email string) (*models.User, error)
	updateFunc         func(ctx context.Context, id uuid.UUID, req *grpcservices.UserUpdateRequest) (*models.User, error)
	deleteFunc         func(ctx context.Context, id uuid.UUID) error
	listFunc           func(ctx context.Context, limit, offset int) ([]*models.User, int64, error)
	updatePasswordFunc func(ctx context.Context, id uuid.UUID, newPassword string) error
}

func (m *mockAdminUserService) Create(ctx context.Context, req *grpcservices.UserCreateRequest) (*models.User, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockAdminUserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAdminUserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *mockAdminUserService) Update(ctx context.Context, id uuid.UUID, req *grpcservices.UserUpdateRequest) (*models.User, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *mockAdminUserService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockAdminUserService) List(ctx context.Context, limit, offset int) ([]*models.User, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockAdminUserService) UpdatePassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, id, newPassword)
	}
	return nil
}

// ============================================================================
// Login Tests
// ============================================================================

func TestLogin_Success(t *testing.T) {
	mockAuth := &mockAuthService{
		loginFunc: func(_ context.Context, _ string, _ string) (*grpcservices.AuthLoginResult, error) {
			return &grpcservices.AuthLoginResult{
				Tokens: &grpcservices.AuthTokens{
					AccessToken:      "access-token-123",
					RefreshToken:     "refresh-token-456",
					AccessExpiresIn:  3600,
					RefreshExpiresIn: 86400,
				},
				Profile: &grpcservices.UserProfile{
					ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email: "test@example.com",
				},
			}, nil
		},
	}

	svc := &IdentityService{
		authSvc: mockAuth,
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	resp, err := svc.Login(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "access-token-123", resp.Tokens.AccessToken)
	assert.Equal(t, "refresh-token-456", resp.Tokens.RefreshToken)
	assert.Equal(t, int64(3600), resp.Tokens.AccessExpiresIn)
	assert.Equal(t, int64(86400), resp.Tokens.RefreshExpiresIn)
}

func TestLogin_MissingEmail(t *testing.T) {
	svc := &IdentityService{
		authSvc: &mockAuthService{},
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.LoginRequest{
		Email:    "",
		Password: "password123",
	}

	resp, err := svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "email is required")
}

func TestLogin_MissingPassword(t *testing.T) {
	svc := &IdentityService{
		authSvc: &mockAuthService{},
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "",
	}

	resp, err := svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "password is required")
}

func TestLogin_InvalidCredentials(t *testing.T) {
	mockAuth := &mockAuthService{
		loginFunc: func(_ context.Context, _, _ string) (*grpcservices.AuthLoginResult, error) {
			return nil, assert.AnError
		},
	}

	svc := &IdentityService{
		authSvc: mockAuth,
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	resp, err := svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestLogin_ServiceNotConfigured(t *testing.T) {
	svc := &IdentityService{
		authSvc: nil,
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	resp, err := svc.Login(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unimplemented, st.Code())
}

// ============================================================================
// RefreshTokens Tests
// ============================================================================

func TestRefreshTokens_Success(t *testing.T) {
	mockAuth := &mockAuthService{
		refreshTokensFunc: func(_ context.Context, _ string) (*grpcservices.AuthTokens, error) {
			return &grpcservices.AuthTokens{
				AccessToken:      "new-access-token",
				RefreshToken:     "new-refresh-token",
				AccessExpiresIn:  3600,
				RefreshExpiresIn: 86400,
			}, nil
		},
	}

	svc := &IdentityService{
		authSvc: mockAuth,
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RefreshTokensRequest{
		RefreshToken: "old-refresh-token",
	}

	resp, err := svc.RefreshTokens(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "new-access-token", resp.Tokens.AccessToken)
	assert.Equal(t, "new-refresh-token", resp.Tokens.RefreshToken)
}

func TestRefreshTokens_MissingToken(t *testing.T) {
	svc := &IdentityService{
		authSvc: &mockAuthService{},
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RefreshTokensRequest{
		RefreshToken: "",
	}

	resp, err := svc.RefreshTokens(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// ============================================================================
// GetProfile Tests
// ============================================================================

func TestGetProfile_Success(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	mockAuth := &mockAuthService{
		getProfileFunc: func(_ context.Context, _ uuid.UUID) (*grpcservices.UserProfile, error) {
			return &grpcservices.UserProfile{
				ID:           userID,
				Email:        "test@example.com",
				IsAdmin:      false,
				HasPassword:  true,
				DefaultOrgID: &orgID,
				Memberships: []grpcservices.OrganizationMembership{
					{OrgID: orgID, OrgName: "Test Org", Role: "admin"},
				},
			}, nil
		},
	}

	svc := &IdentityService{
		authSvc: mockAuth,
		log:     &mockLogger{},
	}

	ctx := pkgcontext.WithUserID(testutil.TestContext(), userID.String())
	req := &pb.GetProfileRequest{}

	resp, err := svc.GetProfile(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userID.String(), resp.User.Id)
	assert.Equal(t, "test@example.com", resp.User.Email)
	assert.Equal(t, orgID.String(), resp.User.DefaultOrgId)
	assert.Len(t, resp.User.Memberships, 1)
}

// ============================================================================
// Admin User Tests
// ============================================================================

// adminTestContext returns a context with a user ID for requireAdmin to extract.
func adminTestContext(callerID uuid.UUID) context.Context {
	return pkgcontext.WithUserID(testutil.TestContext(), callerID.String())
}

// adminGetByIDFunc returns a mock getByIDFunc that returns an admin user for the given caller.
func adminGetByIDFunc(callerID uuid.UUID) func(context.Context, uuid.UUID) (*models.User, error) {
	return func(_ context.Context, id uuid.UUID) (*models.User, error) {
		if id == callerID {
			return &models.User{ID: callerID, Email: "admin-caller@example.com", IsAdmin: true, IsActive: true}, nil
		}
		return nil, nil
	}
}

func TestCreateUser_Success(t *testing.T) {
	callerID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mockAdmin := &mockAdminUserService{
		getByIDFunc: adminGetByIDFunc(callerID),
		createFunc: func(_ context.Context, req *grpcservices.UserCreateRequest) (*models.User, error) {
			return &models.User{
				ID:        userID,
				Email:     req.Email,
				IsAdmin:   req.IsAdmin,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	svc := &IdentityService{
		adminUserSvc: mockAdmin,
		log:          &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	req := &pb.CreateUserRequest{
		Email:    "newuser@example.com",
		Password: "securepassword123",
		IsAdmin:  false,
		IsActive: true,
	}

	resp, err := svc.CreateUser(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userID.String(), resp.User.Id)
	assert.Equal(t, "newuser@example.com", resp.User.Email)
	assert.False(t, resp.User.IsAdmin)
}

func TestCreateUser_MissingEmail(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		log: &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	req := &pb.CreateUserRequest{
		Email:    "",
		Password: "securepassword123",
	}

	resp, err := svc.CreateUser(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "email is required")
}

func TestGetUser_Success(t *testing.T) {
	callerID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mockAdmin := &mockAdminUserService{
		getByIDFunc: func(_ context.Context, id uuid.UUID) (*models.User, error) {
			if id == callerID {
				return &models.User{ID: callerID, Email: "admin@example.com", IsAdmin: true, IsActive: true}, nil
			}
			return &models.User{
				ID:        id,
				Email:     "test@example.com",
				IsAdmin:   false,
				IsActive:  true,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	}

	svc := &IdentityService{
		adminUserSvc: mockAdmin,
		log:          &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	req := &pb.GetUserRequest{
		Id: userID.String(),
	}

	resp, err := svc.GetUser(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, userID.String(), resp.User.Id)
	assert.Equal(t, "test@example.com", resp.User.Email)
}

func TestGetUser_InvalidIDFormat(t *testing.T) {
	callerID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
		},
		log: &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	req := &pb.GetUserRequest{
		Id: "not-a-uuid",
	}

	resp, err := svc.GetUser(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid id format")
}

func TestListUsers_Success(t *testing.T) {
	callerID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()
	now := time.Now()

	mockAdmin := &mockAdminUserService{
		getByIDFunc: adminGetByIDFunc(callerID),
		listFunc: func(_ context.Context, _ int, _ int) ([]*models.User, int64, error) {
			return []*models.User{
				{ID: userID1, Email: "user1@example.com", IsAdmin: false, IsActive: true, CreatedAt: now, UpdatedAt: now},
				{ID: userID2, Email: "user2@example.com", IsAdmin: true, IsActive: true, CreatedAt: now, UpdatedAt: now},
			}, 2, nil
		},
	}

	svc := &IdentityService{
		adminUserSvc: mockAdmin,
		log:          &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	req := &pb.ListUsersRequest{
		PageSize: 10,
	}

	resp, err := svc.ListUsers(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, int32(2), resp.TotalCount)
}

func TestDeleteUser_Success(t *testing.T) {
	callerID := uuid.New()
	userID := uuid.New()

	mockAdmin := &mockAdminUserService{
		getByIDFunc: adminGetByIDFunc(callerID),
		deleteFunc: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}

	svc := &IdentityService{
		adminUserSvc: mockAdmin,
		log:          &mockLogger{},
	}

	ctx := adminTestContext(callerID)
	req := &pb.DeleteUserRequest{
		Id: userID.String(),
	}

	resp, err := svc.DeleteUser(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

// ============================================================================
// GetAuthSettings Tests
// ============================================================================

func TestGetAuthSettings_Success(t *testing.T) {
	mockAuth := &mockAuthService{
		getAuthSettingsFunc: func(_ context.Context) (*grpcservices.AuthSettings, error) {
			return &grpcservices.AuthSettings{
				Enabled:             true,
				LocalLoginEnabled:   true,
				LoginURL:            "/login",
				LoginLabel:          "Sign In",
				LoginRedirect:       true,
				LogoutURL:           "/logout",
				RefreshTokenEnabled: true,
				OIDCEnabled:         false,
			}, nil
		},
	}

	svc := &IdentityService{
		authSvc: mockAuth,
		log:     &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.GetAuthSettingsRequest{}

	resp, err := svc.GetAuthSettings(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Settings.Enabled)
	assert.True(t, resp.Settings.LocalLoginEnabled)
	assert.Equal(t, "/login", resp.Settings.LoginUrl)
}

func TestGetAuthSettings_RegistrationEnabled_WhenServiceWired(t *testing.T) {
	mockAuth := &mockAuthService{
		getAuthSettingsFunc: func(_ context.Context) (*grpcservices.AuthSettings, error) {
			return &grpcservices.AuthSettings{Enabled: true, LocalLoginEnabled: true}, nil
		},
	}

	svc := &IdentityService{
		authSvc:         mockAuth,
		registrationSvc: &mockRegistrationService{},
		log:             &mockLogger{},
	}

	resp, err := svc.GetAuthSettings(testutil.TestContext(), &pb.GetAuthSettingsRequest{})
	require.NoError(t, err)
	assert.True(t, resp.Settings.RegistrationEnabled, "registration_enabled should be true when registrationSvc is wired")
}

func TestGetAuthSettings_RegistrationDisabled_WhenServiceNil(t *testing.T) {
	mockAuth := &mockAuthService{
		getAuthSettingsFunc: func(_ context.Context) (*grpcservices.AuthSettings, error) {
			return &grpcservices.AuthSettings{Enabled: true, LocalLoginEnabled: true}, nil
		},
	}

	svc := &IdentityService{
		authSvc:         mockAuth,
		registrationSvc: nil,
		log:             &mockLogger{},
	}

	resp, err := svc.GetAuthSettings(testutil.TestContext(), &pb.GetAuthSettingsRequest{})
	require.NoError(t, err)
	assert.False(t, resp.Settings.RegistrationEnabled, "registration_enabled should be false when registrationSvc is nil")
}

// ============================================================================
// Logout Tests
// ============================================================================

func TestLogout_Success(t *testing.T) {
	userID := uuid.New()

	mockAuth := &mockAuthService{
		logoutFunc: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}

	svc := &IdentityService{
		authSvc: mockAuth,
		log:     &mockLogger{},
	}

	ctx := pkgcontext.WithUserID(testutil.TestContext(), userID.String())
	req := &pb.LogoutRequest{}

	resp, err := svc.Logout(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

// ============================================================================
// ChangePassword Tests
// ============================================================================

func TestChangePassword_Success(t *testing.T) {
	userID := uuid.New()

	mockAuth := &mockAuthService{
		changePasswordFunc: func(_ context.Context, _ uuid.UUID, _, _ string) error {
			return nil
		},
	}

	svc := &IdentityService{
		authSvc: mockAuth,
		log:     &mockLogger{},
	}

	ctx := pkgcontext.WithUserID(testutil.TestContext(), userID.String())
	req := &pb.ChangePasswordRequest{
		CurrentPassword: "oldpassword",
		NewPassword:     "newpassword123",
	}

	resp, err := svc.ChangePassword(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestChangePassword_MissingCurrentPassword(t *testing.T) {
	userID := uuid.New()

	svc := &IdentityService{
		authSvc: &mockAuthService{},
		log:     &mockLogger{},
	}

	ctx := pkgcontext.WithUserID(testutil.TestContext(), userID.String())
	req := &pb.ChangePasswordRequest{
		CurrentPassword: "",
		NewPassword:     "newpassword123",
	}

	resp, err := svc.ChangePassword(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "current_password is required")
}

// ============================================================================
// Auth Interceptor Tests
// ============================================================================

// ============================================================================
// ExchangeOIDC Tests
// ============================================================================

func TestExchangeOIDC_MissingCode(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOIDCRequest{
		Code:  "",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOIDC(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "code is required")
}

func TestExchangeOIDC_MissingState(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOIDCRequest{
		Code:  "valid-code",
		State: "",
	}

	resp, err := svc.ExchangeOIDC(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "state is required")
}

func TestExchangeOIDC_ServiceNotConfigured(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: nil,
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOIDCRequest{
		Code:  "valid-code",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOIDC(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	// Handler uses ErrTokenAuthOIDCDisabled which maps to FailedPrecondition
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOIDCDisabled), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOIDCDisabled), st.Message())
}

func TestExchangeOIDC_Success(t *testing.T) {
	testUserID := uuid.New()
	testOrgID := uuid.New()

	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{
			completeOIDCLoginFunc: func(_ context.Context, _, _ string) (*grpcservices.AuthLoginResult, error) {
				return &grpcservices.AuthLoginResult{
					Tokens: &grpcservices.AuthTokens{
						AccessToken:      "test-access-token",
						RefreshToken:     "test-refresh-token",
						AccessExpiresIn:  3600,
						RefreshExpiresIn: 86400,
					},
					Profile: &grpcservices.UserProfile{
						ID:          testUserID,
						Email:       "test@example.com",
						IsAdmin:     false,
						HasPassword: false,
						Memberships: []grpcservices.OrganizationMembership{
							{OrgID: testOrgID, OrgName: "Test Org", Role: "member"},
						},
						DefaultOrgID: &testOrgID,
					},
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOIDCRequest{
		Code:  "valid-code",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOIDC(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Tokens)
	assert.Equal(t, "test-access-token", resp.Tokens.AccessToken)
	assert.Equal(t, "test-refresh-token", resp.Tokens.RefreshToken)
	require.NotNil(t, resp.User)
	assert.Equal(t, testUserID.String(), resp.User.Id)
	assert.Equal(t, "test@example.com", resp.User.Email)
}

func TestExchangeOIDC_NilTokensReturnsInternalError(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{
			completeOIDCLoginFunc: func(_ context.Context, _, _ string) (*grpcservices.AuthLoginResult, error) {
				return &grpcservices.AuthLoginResult{
					Tokens: nil, // Nil tokens should trigger internal error
					Profile: &grpcservices.UserProfile{
						ID:    uuid.New(),
						Email: "test@example.com",
					},
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOIDCRequest{
		Code:  "valid-code",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOIDC(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), st.Code())
}

func TestExchangeOIDC_NilProfileReturnsInternalError(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{
			completeOIDCLoginFunc: func(_ context.Context, _, _ string) (*grpcservices.AuthLoginResult, error) {
				return &grpcservices.AuthLoginResult{
					Tokens: &grpcservices.AuthTokens{
						AccessToken:  "test-access-token",
						RefreshToken: "test-refresh-token",
					},
					Profile: nil, // Nil profile should trigger internal error
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOIDCRequest{
		Code:  "valid-code",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOIDC(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError), st.Code())
}

// ============================================================================
// ExchangeOAuth2 Tests
// ============================================================================

func TestExchangeOAuth2_MissingCode(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOAuth2Request{
		Code:  "",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOAuth2(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "code is required")
}

func TestExchangeOAuth2_MissingState(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOAuth2Request{
		Code:  "valid-code",
		State: "",
	}

	resp, err := svc.ExchangeOAuth2(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "state is required")
}

func TestExchangeOAuth2_ServiceNotConfigured(t *testing.T) {
	svc := &IdentityService{
		externalAuthSvc: nil,
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOAuth2Request{
		Code:  "valid-code",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOAuth2(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	// Handler uses ErrTokenAuthOAuth2Disabled which maps to FailedPrecondition
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAuthOAuth2Disabled), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAuthOAuth2Disabled), st.Message())
}

func TestExchangeOAuth2_Success(t *testing.T) {
	testUserID := uuid.New()
	testOrgID := uuid.New()

	svc := &IdentityService{
		externalAuthSvc: &mockExternalAuthService{
			completeOAuth2LoginFunc: func(_ context.Context, _, _ string) (*grpcservices.AuthLoginResult, error) {
				return &grpcservices.AuthLoginResult{
					Tokens: &grpcservices.AuthTokens{
						AccessToken:      "test-access-token",
						RefreshToken:     "test-refresh-token",
						AccessExpiresIn:  3600,
						RefreshExpiresIn: 86400,
					},
					Profile: &grpcservices.UserProfile{
						ID:          testUserID,
						Email:       "oauth2-user@example.com",
						IsAdmin:     false,
						HasPassword: false,
						Memberships: []grpcservices.OrganizationMembership{
							{OrgID: testOrgID, OrgName: "Test Org", Role: "member"},
						},
						DefaultOrgID: &testOrgID,
					},
				}, nil
			},
		},
		log: &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.ExchangeOAuth2Request{
		Code:  "valid-code",
		State: "valid-state",
	}

	resp, err := svc.ExchangeOAuth2(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Tokens)
	assert.Equal(t, "test-access-token", resp.Tokens.AccessToken)
	assert.Equal(t, "test-refresh-token", resp.Tokens.RefreshToken)
	require.NotNil(t, resp.User)
	assert.Equal(t, testUserID.String(), resp.User.Id)
	assert.Equal(t, "oauth2-user@example.com", resp.User.Email)
}

// mockExternalAuthService implements grpcservices.ExternalAuthService for testing
type mockExternalAuthService struct {
	initiateOIDCLoginFunc   func(ctx context.Context) (string, error)
	completeOIDCLoginFunc   func(ctx context.Context, code, state string) (*grpcservices.AuthLoginResult, error)
	initiateOAuth2LoginFunc func(ctx context.Context) (string, error)
	completeOAuth2LoginFunc func(ctx context.Context, code, state string) (*grpcservices.AuthLoginResult, error)
}

func (m *mockExternalAuthService) InitiateOIDCLogin(ctx context.Context) (string, error) {
	if m.initiateOIDCLoginFunc != nil {
		return m.initiateOIDCLoginFunc(ctx)
	}
	return "", nil
}

func (m *mockExternalAuthService) CompleteOIDCLogin(ctx context.Context, code, state string) (*grpcservices.AuthLoginResult, error) {
	if m.completeOIDCLoginFunc != nil {
		return m.completeOIDCLoginFunc(ctx, code, state)
	}
	return nil, nil
}

func (m *mockExternalAuthService) InitiateOAuth2Login(ctx context.Context) (string, error) {
	if m.initiateOAuth2LoginFunc != nil {
		return m.initiateOAuth2LoginFunc(ctx)
	}
	return "", nil
}

func (m *mockExternalAuthService) CompleteOAuth2Login(ctx context.Context, code, state string) (*grpcservices.AuthLoginResult, error) {
	if m.completeOAuth2LoginFunc != nil {
		return m.completeOAuth2LoginFunc(ctx, code, state)
	}
	return nil, nil
}

// ============================================================================
// Auth Interceptor Tests
// ============================================================================

func TestAuthInterceptor_UserClaimExtracted(t *testing.T) {
	secret := []byte("test-secret")

	userID := uuid.New()
	token := jwt.New()
	require.NoError(t, token.Set("tenant_id", int64(42)))
	require.NoError(t, token.Set("sub", userID.String()))

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), secret))
	require.NoError(t, err)

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{
		Enabled:     true,
		TenantClaim: "tenant_id",
		HMACSecret:  string(secret),
	})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(testutil.TestContext(),
		metadata.Pairs("authorization", "Bearer "+string(signed)))

	var capturedCtx context.Context
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		capturedCtx = ctx
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/kilocenter.api.v1.CoreService/ListEndPoints"}
	_, err = ai.UnaryInterceptor()(ctx, nil, info, handler)
	require.NoError(t, err)

	gotUserID, err := pkgcontext.GetUserID(capturedCtx)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), gotUserID)
}

func TestUpdateUser_EmptyEmailInMask_Rejected(t *testing.T) {
	callerID := uuid.New()
	targetID := uuid.New()

	svc := &IdentityService{
		adminUserSvc: &mockAdminUserService{
			getByIDFunc: adminGetByIDFunc(callerID),
			updateFunc: func(_ context.Context, _ uuid.UUID, _ *grpcservices.UserUpdateRequest) (*models.User, error) {
				t.Fatal("Update should not be called with empty email")
				return nil, nil
			},
		},
		log: &mockLogger{},
	}

	req := &pb.UpdateUserRequest{
		Id:    targetID.String(),
		Email: "",
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"email"},
		},
	}

	_, err := svc.UpdateUser(adminTestContext(callerID), req)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired), st.Message())
}
