package registration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/logger"

	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/auth"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations using function pointer pattern.

type mockRegistrationRepo struct {
	registerAccountFn func(ctx context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error)
}

func (m *mockRegistrationRepo) RegisterAccount(ctx context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
	return m.registerAccountFn(ctx, params)
}

func (m *mockRegistrationRepo) RegisterCEAccount(_ context.Context, _ *interfaces.CERegistrationParams) (*interfaces.RegistrationResult, error) {
	return nil, errors.New("not implemented")
}

type mockUserStore struct {
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*models.User, error)
	getByEmailFn      func(ctx context.Context, email string) (*models.User, error)
	setPasswordHashFn func(ctx context.Context, userID uuid.UUID, hash string) error
}

func (m *mockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockUserStore) SetPasswordHash(ctx context.Context, userID uuid.UUID, hash string) error {
	return m.setPasswordHashFn(ctx, userID, hash)
}

type mockRefreshTokenStore struct {
	createFn       func(ctx context.Context, token *models.RefreshToken) error
	getByHashFn    func(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	markReplacedFn func(ctx context.Context, tokenID, replacedByID uuid.UUID) error
	revokeByHashFn func(ctx context.Context, tokenHash string) error
	revokeByUserFn func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockRefreshTokenStore) Create(ctx context.Context, token *models.RefreshToken) error {
	return m.createFn(ctx, token)
}

func (m *mockRefreshTokenStore) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	return m.getByHashFn(ctx, tokenHash)
}

func (m *mockRefreshTokenStore) MarkReplaced(ctx context.Context, tokenID, replacedByID uuid.UUID) error {
	return m.markReplacedFn(ctx, tokenID, replacedByID)
}

func (m *mockRefreshTokenStore) RevokeByHash(ctx context.Context, tokenHash string) error {
	return m.revokeByHashFn(ctx, tokenHash)
}

func (m *mockRefreshTokenStore) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	return m.revokeByUserFn(ctx, userID)
}

type mockMembershipStore struct {
	listUserMembershipsFn func(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error)
}

func (m *mockMembershipStore) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	return m.listUserMembershipsFn(ctx, userID)
}

// newTestTokenIssuer creates a real TokenIssuer with test-appropriate configuration.
func newTestTokenIssuer() *auth.TokenIssuer {
	return auth.NewTokenIssuer(
		[]byte("test-secret-key-for-unit-tests"),
		"org_id",
		"kilocenter-test",
		"kilocenter-test-api",
		15*time.Minute,
		24*time.Hour,
	)
}

func newTestLogger() logger.Logger {
	return logger.NewNop()
}

// validRequest returns a RegisterAccountRequest with all fields populated and a valid password.
func validRequest() *grpcservices.RegisterAccountRequest {
	return &grpcservices.RegisterAccountRequest{
		Email:       "test@example.com",
		Password:    "securepassword123",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}
}

func TestRegisterAccount_Success(t *testing.T) {
	orgID := uuid.New()

	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			assert.Equal(t, "test@example.com", params.User.Email)
			assert.Equal(t, "Acme Corp", params.CompanyName)
			assert.NotNil(t, params.User.PasswordHash, "password hash should be set")
			assert.False(t, params.User.IsAdmin, "new user must not be admin")
			assert.True(t, params.User.IsActive, "new user must be active")

			// Return the user as-is with an org
			return &interfaces.RegistrationResult{
				User:         params.User,
				Organization: &models.Organization{OrgID: orgID, TenantID: 1, Name: "Acme Corp"},
				TenantID:     1,
			}, nil
		},
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	var storedRefreshToken *models.RefreshToken
	refreshStore := &mockRefreshTokenStore{
		createFn: func(_ context.Context, token *models.RefreshToken) error {
			storedRefreshToken = token
			return nil
		},
	}

	membershipStore := &mockMembershipStore{
		listUserMembershipsFn: func(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
			return []*models.OrganizationMembershipWithOrg{
				{
					OrgID:              orgID,
					OrgName:            "Acme Corp",
					Role:               "owner",
					IsOrgAdmin:         true,
					IsBaseStationAdmin: true,
					IsEndpointAdmin:    true,
				},
			}, nil
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(), refreshStore, membershipStore,
		true, // registrationEnabled
		true, // localLoginEnabled
		true, // refreshTokenEnabled
		newTestLogger(),
	)

	result, err := svc.RegisterAccount(testutil.TestContext(), validRequest())
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify tokens
	require.NotNil(t, result.Tokens)
	assert.NotEmpty(t, result.Tokens.AccessToken, "access token should be issued")
	assert.Greater(t, result.Tokens.AccessExpiresIn, int64(0), "access TTL should be positive")
	assert.NotEmpty(t, result.Tokens.RefreshToken, "refresh token should be issued when enabled")
	assert.Greater(t, result.Tokens.RefreshExpiresIn, int64(0), "refresh TTL should be positive")

	// Verify refresh token was stored
	require.NotNil(t, storedRefreshToken, "refresh token should be persisted")
	assert.NotEqual(t, uuid.Nil, storedRefreshToken.UserID)
	assert.NotEmpty(t, storedRefreshToken.TokenHash)

	// Verify profile
	require.NotNil(t, result.Profile)
	assert.Equal(t, "test@example.com", result.Profile.Email)
	assert.Equal(t, "John", result.Profile.FirstName)
	assert.Equal(t, "Doe", result.Profile.LastName)
	assert.False(t, result.Profile.IsAdmin)
	assert.True(t, result.Profile.HasPassword)
	assert.NotNil(t, result.Profile.DefaultOrgID)
	assert.Equal(t, orgID, *result.Profile.DefaultOrgID)

	// Verify memberships populated in profile
	require.Len(t, result.Profile.Memberships, 1)
	assert.Equal(t, orgID, result.Profile.Memberships[0].OrgID)
	assert.Equal(t, "Acme Corp", result.Profile.Memberships[0].OrgName)
	assert.Equal(t, "owner", result.Profile.Memberships[0].Role)
	assert.True(t, result.Profile.Memberships[0].IsOrgAdmin)
}

func TestRegisterAccount_Success_RefreshDisabled(t *testing.T) {
	orgID := uuid.New()

	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			return &interfaces.RegistrationResult{
				User:         params.User,
				Organization: &models.Organization{OrgID: orgID, TenantID: 1, Name: "Acme Corp"},
				TenantID:     1,
			}, nil
		},
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	refreshStore := &mockRefreshTokenStore{
		createFn: func(_ context.Context, _ *models.RefreshToken) error {
			t.Fatal("refresh token store should not be called when disabled")
			return nil
		},
	}

	membershipStore := &mockMembershipStore{
		listUserMembershipsFn: func(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
			return nil, nil
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(), refreshStore, membershipStore,
		true,  // registrationEnabled
		true,  // localLoginEnabled
		false, // refreshTokenEnabled
		newTestLogger(),
	)

	result, err := svc.RegisterAccount(testutil.TestContext(), validRequest())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.Tokens.AccessToken)
	assert.Empty(t, result.Tokens.RefreshToken, "refresh token should be empty when disabled")
	assert.Zero(t, result.Tokens.RefreshExpiresIn, "refresh TTL should be zero when disabled")
}

func TestRegisterAccount_RegistrationDisabled(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{},
		&mockUserStore{},
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		false, // registrationEnabled
		true,  // localLoginEnabled
		true,  // refreshTokenEnabled
		newTestLogger(),
	)

	result, err := svc.RegisterAccount(testutil.TestContext(), validRequest())
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errRegistrationDisabled)
	assert.Contains(t, err.Error(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationDisabled))
}

func TestRegisterAccount_EmptyEmail(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{},
		&mockUserStore{},
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		true, // registrationEnabled
		true, // localLoginEnabled
		true, // refreshTokenEnabled
		newTestLogger(),
	)

	req := validRequest()
	req.Email = ""

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errEmailRequired)
}

func TestRegisterAccount_EmptyFirstName(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{},
		&mockUserStore{},
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		true, // registrationEnabled
		true, // localLoginEnabled
		true, // refreshTokenEnabled
		newTestLogger(),
	)

	req := validRequest()
	req.FirstName = ""

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errFirstNameRequired)
}

func TestRegisterAccount_EmptyLastName(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{},
		&mockUserStore{},
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		true, // registrationEnabled
		true, // localLoginEnabled
		true, // refreshTokenEnabled
		newTestLogger(),
	)

	req := validRequest()
	req.LastName = ""

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errLastNameRequired)
}

func TestRegisterAccount_EmptyCompanyName(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{},
		&mockUserStore{},
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		true, // registrationEnabled
		true, // localLoginEnabled
		true, // refreshTokenEnabled
		newTestLogger(),
	)

	req := validRequest()
	req.CompanyName = ""

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errCompanyNameRequired)
}

func TestRegisterAccount_WeakPassword(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{},
		&mockUserStore{},
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		true, // registrationEnabled
		true, // localLoginEnabled
		true, // refreshTokenEnabled
		newTestLogger(),
	)

	req := validRequest()
	req.Password = "short"

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errWeakPassword)
}

func TestRegisterAccount_DuplicateEmail(t *testing.T) {
	existingUser := &models.User{
		ID:    uuid.New(),
		Email: "existing@example.com",
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, email string) (*models.User, error) {
			if email == "existing@example.com" {
				return existingUser, nil
			}
			return nil, interfaces.ErrRecordNotFound
		},
	}

	svc := NewService(
		&mockRegistrationRepo{},
		userStore,
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		true, true, true,
		newTestLogger(),
	)

	req := validRequest()
	req.Email = "existing@example.com"

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errEmailExists)
	assert.NotContains(t, err.Error(), "existing@example.com", "error must not leak email address (PII)")
}

func TestRegisterAccount_EmailCheckStoreError(t *testing.T) {
	regRepoCalled := false
	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, _ *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			regRepoCalled = true
			return nil, errors.New("should not be reached")
		},
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, errors.New("connection refused")
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(),
		&mockRefreshTokenStore{}, &mockMembershipStore{},
		true, true, true, newTestLogger(),
	)

	result, err := svc.RegisterAccount(testutil.TestContext(), validRequest())
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errRegistrationFailed)
	assert.False(t, regRepoCalled, "registration repo must not be called when email check fails")
}

func TestRegisterAccount_RepoFailure(t *testing.T) {
	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, _ *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			return nil, errors.New("database connection failed")
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(),
		&mockRefreshTokenStore{}, &mockMembershipStore{},
		true, true, true, newTestLogger(),
	)

	result, err := svc.RegisterAccount(testutil.TestContext(), validRequest())
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errRegistrationFailed)
}

func TestRegisterAccount_MembershipLoadFailure_StillSucceeds(t *testing.T) {
	orgID := uuid.New()

	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			return &interfaces.RegistrationResult{
				User:         params.User,
				Organization: &models.Organization{OrgID: orgID, TenantID: 1, Name: "Test Corp"},
				TenantID:     1,
			}, nil
		},
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	refreshStore := &mockRefreshTokenStore{
		createFn: func(_ context.Context, _ *models.RefreshToken) error {
			return nil
		},
	}

	membershipStore := &mockMembershipStore{
		listUserMembershipsFn: func(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
			return nil, errors.New("membership query timeout")
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(), refreshStore, membershipStore,
		true, true, true,
		newTestLogger(),
	)

	result, err := svc.RegisterAccount(testutil.TestContext(), validRequest())
	require.NoError(t, err, "membership failure should not block registration")
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Tokens.AccessToken)
	assert.Empty(t, result.Profile.Memberships, "memberships should be empty on load failure")
}

func TestRegisterAccount_RefreshTokenStoreFailure_StillSucceeds(t *testing.T) {
	orgID := uuid.New()

	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			return &interfaces.RegistrationResult{
				User:         params.User,
				Organization: &models.Organization{OrgID: orgID, TenantID: 1, Name: "Test Corp"},
				TenantID:     1,
			}, nil
		},
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	refreshStore := &mockRefreshTokenStore{
		createFn: func(_ context.Context, _ *models.RefreshToken) error {
			return errors.New("redis unavailable")
		},
	}

	membershipStore := &mockMembershipStore{
		listUserMembershipsFn: func(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
			return nil, nil
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(), refreshStore, membershipStore,
		true, true, true,
		newTestLogger(),
	)

	result, err := svc.RegisterAccount(testutil.TestContext(), validRequest())
	require.NoError(t, err, "refresh token store failure should not block registration")
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Tokens.AccessToken)
	assert.Empty(t, result.Tokens.RefreshToken, "refresh token should be empty when store fails")
}

func TestRegisterAccount_ValidationOrder(t *testing.T) {
	// Verifies that validation checks execute in the documented order:
	// loginEnabled -> email -> firstName -> lastName -> companyName -> password

	svc := NewService(
		&mockRegistrationRepo{},
		&mockUserStore{},
		newTestTokenIssuer(),
		&mockRefreshTokenStore{},
		&mockMembershipStore{},
		true, true, true,
		newTestLogger(),
	)

	// All fields empty except email: should fail on first_name first
	req := &grpcservices.RegisterAccountRequest{
		Email:    "test@example.com",
		Password: "securepassword123",
	}
	_, err := svc.RegisterAccount(testutil.TestContext(), req)
	require.Error(t, err)
	assert.ErrorIs(t, err, errFirstNameRequired, "should fail on first_name before last_name/company_name")

	// FirstName set, LastName empty: should fail on last_name
	req.FirstName = "John"
	_, err = svc.RegisterAccount(testutil.TestContext(), req)
	require.Error(t, err)
	assert.ErrorIs(t, err, errLastNameRequired, "should fail on last_name before company_name")

	// LastName set, CompanyName empty: should fail on company_name
	req.LastName = "Doe"
	_, err = svc.RegisterAccount(testutil.TestContext(), req)
	require.Error(t, err)
	assert.ErrorIs(t, err, errCompanyNameRequired, "should fail on company_name")
}

func TestRegisterAccount_PasswordBoundary(t *testing.T) {
	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	orgID := uuid.New()
	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			return &interfaces.RegistrationResult{
				User:         params.User,
				Organization: &models.Organization{OrgID: orgID, TenantID: 1, Name: "Acme Corp"},
				TenantID:     1,
			}, nil
		},
	}

	refreshStore := &mockRefreshTokenStore{
		createFn: func(_ context.Context, _ *models.RefreshToken) error { return nil },
	}

	membershipStore := &mockMembershipStore{
		listUserMembershipsFn: func(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
			return nil, nil
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(), refreshStore, membershipStore,
		true, true, true,
		newTestLogger(),
	)

	// Exactly 7 characters: should fail (min is 8)
	req := validRequest()
	req.Password = "abcde1x"
	_, err := svc.RegisterAccount(testutil.TestContext(), req)
	require.Error(t, err)
	assert.ErrorIs(t, err, errWeakPassword)

	// Exactly 8 characters with letter+digit: should pass
	req.Password = "abcdef1x"
	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestRegisterAccount_UserModelFields(t *testing.T) {
	// Verifies the user model is correctly populated before being passed to the repository.
	var capturedParams *interfaces.RegistrationParams

	orgID := uuid.New()
	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			capturedParams = params
			return &interfaces.RegistrationResult{
				User:         params.User,
				Organization: &models.Organization{OrgID: orgID, TenantID: 1, Name: "Acme Corp"},
				TenantID:     1,
			}, nil
		},
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, _ string) (*models.User, error) {
			return nil, interfaces.ErrRecordNotFound
		},
	}

	refreshStore := &mockRefreshTokenStore{
		createFn: func(_ context.Context, _ *models.RefreshToken) error { return nil },
	}

	membershipStore := &mockMembershipStore{
		listUserMembershipsFn: func(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
			return nil, nil
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(), refreshStore, membershipStore,
		true, true, true,
		newTestLogger(),
	)

	req := validRequest()
	_, err := svc.RegisterAccount(testutil.TestContext(), req)
	require.NoError(t, err)
	require.NotNil(t, capturedParams)

	user := capturedParams.User
	assert.NotEqual(t, uuid.Nil, user.ID, "user ID should be generated")
	assert.Equal(t, "test@example.com", user.Email)
	assert.False(t, user.EmailVerified, "new user email should not be verified")
	assert.NotNil(t, user.PasswordHash, "password hash should be set")
	assert.NotEmpty(t, *user.PasswordHash, "password hash should not be empty")
	assert.False(t, user.IsAdmin, "new user must not be admin")
	assert.True(t, user.IsActive, "new user must be active")
	require.NotNil(t, user.FirstName)
	assert.Equal(t, "John", *user.FirstName)
	require.NotNil(t, user.LastName)
	assert.Equal(t, "Doe", *user.LastName)
	require.NotNil(t, user.CompanyName)
	assert.Equal(t, "Acme Corp", *user.CompanyName)

	assert.Equal(t, "Acme Corp", capturedParams.CompanyName)
}

func TestRegisterAccount_EmailNormalization(t *testing.T) {
	var capturedEmail string

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, email string) (*models.User, error) {
			capturedEmail = email
			return nil, interfaces.ErrRecordNotFound
		},
	}

	orgID := uuid.New()
	regRepo := &mockRegistrationRepo{
		registerAccountFn: func(_ context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
			return &interfaces.RegistrationResult{
				User:         params.User,
				Organization: &models.Organization{OrgID: orgID, TenantID: 1, Name: "Acme Corp"},
				TenantID:     1,
			}, nil
		},
	}

	refreshStore := &mockRefreshTokenStore{
		createFn: func(_ context.Context, _ *models.RefreshToken) error { return nil },
	}

	membershipStore := &mockMembershipStore{
		listUserMembershipsFn: func(_ context.Context, _ uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
			return nil, nil
		},
	}

	svc := NewService(
		regRepo, userStore, newTestTokenIssuer(), refreshStore, membershipStore,
		true, true, true, newTestLogger(),
	)

	req := validRequest()
	req.Email = "Test@Example.Com"

	_, err := svc.RegisterAccount(testutil.TestContext(), req)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", capturedEmail, "email should be lowercased and trimmed before store lookup")
}

func TestRegisterAccount_InvalidEmailFormat(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{}, &mockUserStore{}, newTestTokenIssuer(),
		&mockRefreshTokenStore{}, &mockMembershipStore{},
		true, true, true, newTestLogger(),
	)

	req := validRequest()
	req.Email = "not-an-email"

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errEmailInvalidFormat)
}

func TestRegisterAccount_EmailTooLong(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{}, &mockUserStore{}, newTestTokenIssuer(),
		&mockRefreshTokenStore{}, &mockMembershipStore{},
		true, true, true, newTestLogger(),
	)

	req := validRequest()
	req.Email = strings.Repeat("a", 245) + "@example.com" // 257 characters total

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errFieldTooLong)
}

func TestRegisterAccount_FieldTooLong(t *testing.T) {
	svc := NewService(
		&mockRegistrationRepo{}, &mockUserStore{}, newTestTokenIssuer(),
		&mockRefreshTokenStore{}, &mockMembershipStore{},
		true, true, true, newTestLogger(),
	)

	longValue := strings.Repeat("a", 257)

	t.Run("firstName exceeds max length", func(t *testing.T) {
		req := validRequest()
		req.FirstName = longValue
		result, err := svc.RegisterAccount(testutil.TestContext(), req)
		assert.Nil(t, result)
		require.Error(t, err)
		assert.ErrorIs(t, err, errFieldTooLong)
	})

	t.Run("lastName exceeds max length", func(t *testing.T) {
		req := validRequest()
		req.LastName = longValue
		result, err := svc.RegisterAccount(testutil.TestContext(), req)
		assert.Nil(t, result)
		require.Error(t, err)
		assert.ErrorIs(t, err, errFieldTooLong)
	})

	t.Run("companyName exceeds max length", func(t *testing.T) {
		req := validRequest()
		req.CompanyName = longValue
		result, err := svc.RegisterAccount(testutil.TestContext(), req)
		assert.Nil(t, result)
		require.Error(t, err)
		assert.ErrorIs(t, err, errFieldTooLong)
	})
}

func TestRegisterAccount_DuplicateEmail_NoPII(t *testing.T) {
	existingUser := &models.User{
		ID:    uuid.New(),
		Email: "sensitive@example.com",
	}

	userStore := &mockUserStore{
		getByEmailFn: func(_ context.Context, email string) (*models.User, error) {
			if email == "sensitive@example.com" {
				return existingUser, nil
			}
			return nil, interfaces.ErrRecordNotFound
		},
	}

	svc := NewService(
		&mockRegistrationRepo{}, userStore, newTestTokenIssuer(),
		&mockRefreshTokenStore{}, &mockMembershipStore{},
		true, true, true, newTestLogger(),
	)

	req := validRequest()
	req.Email = "sensitive@example.com"

	result, err := svc.RegisterAccount(testutil.TestContext(), req)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, errEmailExists)
	assert.NotContains(t, err.Error(), "sensitive@example.com", "error message must not leak the email address (PII)")
}
