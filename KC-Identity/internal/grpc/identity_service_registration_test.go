package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
)

// mockRegistrationService implements grpcservices.RegistrationService for testing.
type mockRegistrationService struct {
	registerAccountFunc func(ctx context.Context, req *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error)
}

func (m *mockRegistrationService) RegisterAccount(ctx context.Context, req *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
	if m.registerAccountFunc != nil {
		return m.registerAccountFunc(ctx, req)
	}
	return nil, nil
}

// ============================================================================
// RegisterAccount Tests
// ============================================================================

func TestRegisterAccount_Handler_Success(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	mockReg := &mockRegistrationService{
		registerAccountFunc: func(_ context.Context, _ *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
			return &grpcservices.AuthLoginResult{
				Tokens: &grpcservices.AuthTokens{
					AccessToken:      "access-token-123",
					RefreshToken:     "refresh-token-456",
					AccessExpiresIn:  3600,
					RefreshExpiresIn: 86400,
				},
				Profile: &grpcservices.UserProfile{
					ID:           userID,
					Email:        "newuser@example.com",
					IsAdmin:      false,
					HasPassword:  true,
					FirstName:    "John",
					LastName:     "Doe",
					DefaultOrgID: &orgID,
					Memberships: []grpcservices.OrganizationMembership{
						{
							OrgID:              orgID,
							OrgName:            "Acme Corp",
							Role:               "owner",
							IsOrgAdmin:         true,
							IsBaseStationAdmin: true,
							IsEndpointAdmin:    true,
							DisplayName:        "Acme Corp",
						},
					},
				},
			}, nil
		},
	}

	svc := &IdentityService{
		registrationSvc: mockReg,
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "newuser@example.com",
		Password:    "securepassword123",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Tokens)
	assert.Equal(t, "access-token-123", resp.Tokens.AccessToken)
	assert.Equal(t, "refresh-token-456", resp.Tokens.RefreshToken)
	assert.Equal(t, int64(3600), resp.Tokens.AccessExpiresIn)
	assert.Equal(t, int64(86400), resp.Tokens.RefreshExpiresIn)
	require.NotNil(t, resp.User)
	assert.Equal(t, userID.String(), resp.User.Id)
	assert.Equal(t, "newuser@example.com", resp.User.Email)
	assert.False(t, resp.User.IsAdmin)
	assert.True(t, resp.User.HasPassword)
	assert.Equal(t, "John", resp.User.FirstName)
	assert.Equal(t, "Doe", resp.User.LastName)
	assert.Equal(t, orgID.String(), resp.User.DefaultOrgId)
	require.Len(t, resp.User.Memberships, 1)
	assert.Equal(t, orgID.String(), resp.User.Memberships[0].OrgId)
	assert.Equal(t, "Acme Corp", resp.User.Memberships[0].OrgName)
	assert.Equal(t, "owner", resp.User.Memberships[0].Role)
	assert.True(t, resp.User.Memberships[0].IsOrgAdmin)
	assert.True(t, resp.User.Memberships[0].IsBaseStationAdmin)
	assert.True(t, resp.User.Memberships[0].IsEndpointAdmin)
}

func TestRegisterAccount_Handler_MissingEmail(t *testing.T) {
	svc := &IdentityService{
		registrationSvc: &mockRegistrationService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "",
		Password:    "securepassword123",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenEmailRequired), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailRequired))
}

func TestRegisterAccount_Handler_MissingPassword(t *testing.T) {
	svc := &IdentityService{
		registrationSvc: &mockRegistrationService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "newuser@example.com",
		Password:    "",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenPasswordRequired), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenPasswordRequired))
}

func TestRegisterAccount_Handler_MissingFirstName(t *testing.T) {
	svc := &IdentityService{
		registrationSvc: &mockRegistrationService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "newuser@example.com",
		Password:    "securepassword123",
		FirstName:   "",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenFirstNameRequired), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenFirstNameRequired))
}

func TestRegisterAccount_Handler_MissingLastName(t *testing.T) {
	svc := &IdentityService{
		registrationSvc: &mockRegistrationService{},
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "newuser@example.com",
		Password:    "securepassword123",
		FirstName:   "John",
		LastName:    "",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLastNameRequired), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLastNameRequired))
}

func TestRegisterAccount_Handler_MissingCompanyName(t *testing.T) {
	mockReg := &mockRegistrationService{
		registerAccountFunc: func(_ context.Context, _ *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
			return nil, fmt.Errorf("company_name: %s", grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCompanyNameRequired))
		},
	}

	svc := &IdentityService{
		registrationSvc: mockReg,
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "newuser@example.com",
		Password:    "securepassword123",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCompanyNameRequired), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCompanyNameRequired))
}

func TestRegisterAccount_Handler_ServiceNotConfigured(t *testing.T) {
	svc := &IdentityService{
		registrationSvc: nil,
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "newuser@example.com",
		Password:    "securepassword123",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenRegistrationDisabled), st.Code())
	assert.Equal(t, grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenRegistrationDisabled), st.Message())
}

func TestRegisterAccount_Handler_ServiceError_DuplicateEmail(t *testing.T) {
	mockReg := &mockRegistrationService{
		registerAccountFunc: func(_ context.Context, _ *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
			return nil, fmt.Errorf("email: %s", grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenEmailAlreadyExists))
		},
	}

	svc := &IdentityService{
		registrationSvc: mockReg,
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "test@example.com",
		Password:    "securepassword123",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	// Duplicate email returns generic InvalidArgument to prevent email enumeration
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidArgument), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidArgument))
}

func TestRegisterAccount_Handler_ServiceError_WeakPassword(t *testing.T) {
	mockReg := &mockRegistrationService{
		registerAccountFunc: func(_ context.Context, _ *grpcservices.RegisterAccountRequest) (*grpcservices.AuthLoginResult, error) {
			return nil, fmt.Errorf("password: %s", grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenWeakPassword))
		},
	}

	svc := &IdentityService{
		registrationSvc: mockReg,
		log:             &mockLogger{},
	}

	ctx := testutil.TestContext()
	req := &pb.RegisterAccountRequest{
		Email:       "newuser@example.com",
		Password:    "weak",
		FirstName:   "John",
		LastName:    "Doe",
		CompanyName: "Acme Corp",
	}

	resp, err := svc.RegisterAccount(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenWeakPassword), st.Code())
	assert.Contains(t, st.Message(), grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenWeakPassword))
}
