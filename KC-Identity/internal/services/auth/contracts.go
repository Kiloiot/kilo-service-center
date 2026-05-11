// Package auth provides authentication and user management services.
package auth

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
)

// UserStore provides user persistence operations.
type UserStore interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	SetPasswordHash(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// OrganizationMembershipStore provides membership queries.
type OrganizationMembershipStore interface {
	ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error)
}

// RefreshTokenStore provides refresh token persistence.
type RefreshTokenStore interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	MarkReplaced(ctx context.Context, tokenID, replacedByID uuid.UUID) error
	RevokeByHash(ctx context.Context, tokenHash string) error
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error
}

// External auth interfaces and types.

// ExternalUserStore extends UserStore with external ID operations for OIDC/OAuth2.
type ExternalUserStore interface {
	UserStore
	GetByExternalID(ctx context.Context, externalID string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
}

// StateStore provides state/nonce/verifier storage for external auth flows.
// Implementation backed by Redis with TTL expiration.
// Renamed from AuthStateStore to avoid stuttering (auth.StateStore vs auth.AuthStateStore).
type StateStore interface {
	// StoreState stores a state token with associated data and TTL.
	StoreState(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// GetState retrieves and deletes state (atomic get-and-delete for replay prevention).
	GetState(ctx context.Context, key string) ([]byte, error)

	// DeleteState explicitly removes a state token (optional cleanup).
	DeleteState(ctx context.Context, key string) error
}

// State represents stored state data for external auth flows.
// Renamed from AuthState to avoid stuttering (auth.State vs auth.AuthState).
type State struct {
	Provider     string `json:"provider"`                // "oidc" or "oauth2"
	CreatedAt    int64  `json:"created_at"`              // Unix timestamp
	Nonce        string `json:"nonce,omitempty"`         // OIDC nonce parameter
	CodeVerifier string `json:"code_verifier,omitempty"` // PKCE code_verifier
}

// OIDCClient provides OIDC protocol operations.
type OIDCClient interface {
	// GetAuthorizationURL builds the OIDC authorization URL with state and nonce.
	GetAuthorizationURL(state, nonce string) string

	// ExchangeCode exchanges authorization code for tokens.
	ExchangeCode(ctx context.Context, code string) (*OIDCTokenResponse, error)

	// ValidateIDToken validates the ID token and extracts claims.
	// Returns typed claims AND raw claim map for custom claim extraction.
	ValidateIDToken(ctx context.Context, idToken, expectedNonce string) (claims *OIDCClaims, rawClaims map[string]any, err error)
}

// OIDCTokenResponse contains tokens returned from OIDC token endpoint.
type OIDCTokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// OIDCClaims contains standard OIDC ID token claims.
type OIDCClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`
	Nonce         string `json:"nonce,omitempty"`
	Issuer        string `json:"iss"`
	Audience      string `json:"aud"`
	ExpiresAt     int64  `json:"exp"`
	IssuedAt      int64  `json:"iat"`
}

// OAuth2Client provides OAuth2 PKCE protocol operations.
type OAuth2Client interface {
	// GetAuthorizationURL builds the OAuth2 authorization URL with state and PKCE.
	GetAuthorizationURL(state string) (authURL string, codeVerifier string)

	// ExchangeCode exchanges authorization code for tokens using PKCE verifier.
	ExchangeCode(ctx context.Context, code, codeVerifier string) (*OAuth2TokenResponse, error)

	// GetUserInfo retrieves user info from the userinfo endpoint.
	GetUserInfo(ctx context.Context, accessToken string) (*OAuth2UserInfo, error)
}

// OAuth2TokenResponse contains tokens returned from OAuth2 token endpoint.
type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// OAuth2UserInfo contains user information from OAuth2 userinfo endpoint.
type OAuth2UserInfo struct {
	ID            string `json:"id,omitempty"`
	Sub           string `json:"sub,omitempty"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`
}

// OrganizationResolver resolves external org claims to local organization IDs.
// Delegated to KC-Core/pkg/org for cross-service use.
type OrganizationResolver = org.OrganizationResolver
