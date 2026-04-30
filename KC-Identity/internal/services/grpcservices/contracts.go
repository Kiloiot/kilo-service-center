// Package grpcservices defines service interfaces and DTOs for the identity gRPC layer.
package grpcservices

import (
	"context"
	"time"

	"github.com/google/uuid"
	grpcpkg "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-DB/storage/models"
)

// AuthLoginResult contains tokens and profile for login/exchange operations
type AuthLoginResult struct {
	Tokens  *AuthTokens
	Profile *UserProfile
}

// AuthService handles authentication operations for gRPC layer
type AuthService interface {
	Login(ctx context.Context, email, password string) (*AuthLoginResult, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*AuthTokens, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
	GetAuthSettings(ctx context.Context) (*AuthSettings, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
}

// ExternalAuthService handles OIDC/OAuth2 exchanges
type ExternalAuthService interface {
	InitiateOIDCLogin(ctx context.Context) (string, error)
	CompleteOIDCLogin(ctx context.Context, code, state string) (*AuthLoginResult, error)

	InitiateOAuth2Login(ctx context.Context) (string, error)
	CompleteOAuth2Login(ctx context.Context, code, state string) (*AuthLoginResult, error)
}

// AdminUserService handles user administration operations
type AdminUserService interface {
	Create(ctx context.Context, user *UserCreateRequest) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, id uuid.UUID, req *UserUpdateRequest) (*models.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.User, int64, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, newPassword string) error
}

// OrganizationService handles organization CRUD operations
type OrganizationService interface {
	Create(ctx context.Context, org *OrganizationCreateRequest) (*models.Organization, error)
	GetByID(ctx context.Context, id uuid.UUID, tenantID int64) (*models.Organization, error)
	GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	Update(ctx context.Context, id uuid.UUID, tenantID int64, req *OrganizationUpdateRequest) (*models.Organization, error)
	Delete(ctx context.Context, id uuid.UUID, tenantID int64) error
	List(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Organization, int64, error)
	ListAll(ctx context.Context, limit, offset int) ([]*models.Organization, int64, error)
}

// MembershipService handles organization membership operations
type MembershipService interface {
	AddUser(ctx context.Context, orgID, userID uuid.UUID, role string) error
	GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*OrganizationMember, error)
	UpdateRole(ctx context.Context, orgID, userID uuid.UUID, role string) error
	UpdatePermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error
	RemoveUser(ctx context.Context, orgID, userID uuid.UUID) error
	ListMembers(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*OrganizationMember, int64, error)
	ListUserOrganizations(ctx context.Context, userID uuid.UUID, tenantID int64) ([]OrganizationMembership, error)
}

// APIKeyService handles API key management operations
type APIKeyService interface {
	Create(ctx context.Context, req *APIKeyCreateRequest) (*APIKeyCreateResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, int64, error)
	GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error)
	DeleteByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) error
}

// EventWriter writes system events to the event store.
type EventWriter = grpcpkg.EventWriter

// AuthTokens contains access and refresh tokens
type AuthTokens struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresIn  int64
	RefreshExpiresIn int64
}

// UserProfile contains user profile information
type UserProfile struct {
	ID           uuid.UUID
	Email        string
	IsAdmin      bool
	HasPassword  bool
	FirstName    string
	LastName     string
	Memberships  []OrganizationMembership
	DefaultOrgID *uuid.UUID
}

// AuthSettings contains authentication settings
type AuthSettings struct {
	Enabled             bool
	LocalLoginEnabled   bool
	LoginURL            string
	LoginLabel          string
	LoginRedirect       bool
	LogoutURL           string
	RefreshTokenEnabled bool
	OIDCEnabled         bool
	OIDCProviderURL     string
}

// UserCreateRequest contains fields for creating a new user
type UserCreateRequest struct {
	Email                string
	Password             string
	IsAdmin              bool
	IsActive             bool
	EmailVerified        bool
	IsTenantManager      bool
	IsBaseStationManager bool
	IsEndpointManager    bool
	Note                 string
	FirstName            string
	LastName             string
	CompanyName          string
}

// UserUpdateRequest contains fields for updating a user
type UserUpdateRequest struct {
	Email                *string
	IsAdmin              *bool
	IsActive             *bool
	EmailVerified        *bool
	IsTenantManager      *bool
	IsBaseStationManager *bool
	IsEndpointManager    *bool
	Note                 *string
}

// OrganizationCreateRequest contains fields for creating an organization
type OrganizationCreateRequest struct {
	Name        string
	Description string
	TenantID    int64
}

// OrganizationUpdateRequest contains fields for updating an organization
type OrganizationUpdateRequest struct {
	Name        *string
	Description *string
}

// OrganizationMember represents a user's membership in an organization
type OrganizationMember struct {
	UserID             uuid.UUID
	OrgID              uuid.UUID
	Role               string
	Status             string
	IsOrgAdmin         bool
	IsBaseStationAdmin bool
	IsEndpointAdmin    bool
	JoinedAt           time.Time
	UpdatedAt          time.Time
	UserEmail          string
}

// OrganizationMembership represents an organization in a user's profile
type OrganizationMembership struct {
	OrgID              uuid.UUID
	OrgName            string
	Role               string
	Status             string
	DisplayName        string
	IsOrgAdmin         bool
	IsBaseStationAdmin bool
	IsEndpointAdmin    bool
}

// APIKeyCreateRequest contains fields for creating an API key
type APIKeyCreateRequest struct {
	TenantID  int64
	OrgID     uuid.UUID
	UserID    *uuid.UUID
	Name      string
	KeyType   string
	ExpiresAt *time.Time
}

// APIKeyCreateResponse contains the created API key (returned only once)
type APIKeyCreateResponse struct {
	Key    string
	APIKey *models.APIKey
}

// RegistrationService handles self-service account registration.
type RegistrationService interface {
	RegisterAccount(ctx context.Context, req *RegisterAccountRequest) (*AuthLoginResult, error)
}

// RegisterAccountRequest contains fields for self-service account registration.
type RegisterAccountRequest struct {
	Email       string
	Password    string
	FirstName   string
	LastName    string
	CompanyName string
}
