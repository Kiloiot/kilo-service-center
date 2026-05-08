package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// Token type claim values
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Standard JWT claims
const (
	ClaimSubject    = "sub"
	ClaimIssuer     = "iss"
	ClaimAudience   = "aud"
	ClaimIssuedAt   = "iat"
	ClaimExpiration = "exp"
	ClaimTokenType  = "typ"
	ClaimJWTID      = "jti"
)

// TokenIssuer handles JWT token creation for local authentication.
type TokenIssuer struct {
	jwtAuth     *jwtauth.JWTAuth
	tenantClaim string
	issuer      string
	audience    string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewTokenIssuer creates a new token issuer with the given configuration.
func NewTokenIssuer(
	hmacSecret []byte,
	tenantClaim string,
	issuer string,
	audience string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *TokenIssuer {
	// Apply defaults from config constants if not set
	if issuer == "" {
		issuer = config.AuthDefaultIssuer
	}
	if audience == "" {
		audience = config.AuthDefaultAudience
	}

	jwtAuth := jwtauth.New("HS256", hmacSecret, nil)

	return &TokenIssuer{
		jwtAuth:     jwtAuth,
		tenantClaim: tenantClaim,
		issuer:      issuer,
		audience:    audience,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

// IssueAccessToken creates a new access token for the given user and org.
func (ti *TokenIssuer) IssueAccessToken(userID uuid.UUID, orgID *uuid.UUID) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(ti.accessTTL)

	claims := map[string]interface{}{
		ClaimSubject:    userID.String(),
		ClaimIssuer:     ti.issuer,
		ClaimAudience:   ti.audience,
		ClaimIssuedAt:   now.Unix(),
		ClaimExpiration: exp.Unix(),
		ClaimTokenType:  TokenTypeAccess,
	}

	// Add org claim if present
	if orgID != nil && ti.tenantClaim != "" {
		claims[ti.tenantClaim] = orgID.String()
	}

	_, tokenString, err := ti.jwtAuth.Encode(claims)
	if err != nil {
		return "", fmt.Errorf("issue access token: %w", err)
	}

	return tokenString, nil
}

// IssueRefreshToken creates a new refresh token for the given user.
// Refresh tokens have longer TTL and are stored in the database for rotation.
func (ti *TokenIssuer) IssueRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(ti.refreshTTL)

	// Generate a unique token ID for tracking
	tokenID := uuid.New()

	claims := map[string]interface{}{
		ClaimSubject:    userID.String(),
		ClaimIssuer:     ti.issuer,
		ClaimAudience:   ti.audience,
		ClaimIssuedAt:   now.Unix(),
		ClaimExpiration: exp.Unix(),
		ClaimTokenType:  TokenTypeRefresh,
		ClaimJWTID:      tokenID.String(), // JWT ID for uniqueness
	}

	_, tokenString, err := ti.jwtAuth.Encode(claims)
	if err != nil {
		return "", fmt.Errorf("issue refresh token: %w", err)
	}

	return tokenString, nil
}

// ParseRefreshToken validates and parses a refresh token.
// Returns the user ID and token if valid.
func (ti *TokenIssuer) ParseRefreshToken(tokenString string) (uuid.UUID, jwt.Token, error) {
	token, err := ti.jwtAuth.Decode(tokenString)
	if err != nil {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}

	// Validate typ claim
	var typClaim interface{}
	if err := token.Get(ClaimTokenType, &typClaim); err != nil {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}
	if typStr, ok := typClaim.(string); !ok || typStr != TokenTypeRefresh {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}

	// Validate expiration
	exp, ok := token.Expiration()
	if !ok || time.Now().After(exp) {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}

	// Extract user ID from subject
	subjectStr, ok := token.Subject()
	if !ok {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}
	userID, err := uuid.Parse(subjectStr)
	if err != nil {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}

	return userID, token, nil
}

// GetAccessTTL returns the access token TTL in seconds.
func (ti *TokenIssuer) GetAccessTTL() int64 {
	return int64(ti.accessTTL.Seconds())
}

// GetRefreshTTL returns the refresh token TTL in seconds.
func (ti *TokenIssuer) GetRefreshTTL() int64 {
	return int64(ti.refreshTTL.Seconds())
}

// GetRefreshExpiresAt returns the expiration time for a new refresh token.
func (ti *TokenIssuer) GetRefreshExpiresAt() time.Time {
	return time.Now().UTC().Add(ti.refreshTTL)
}

// HashRefreshToken computes SHA256 hex hash of a refresh token.
// This hash is stored in the database instead of the raw token.
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
