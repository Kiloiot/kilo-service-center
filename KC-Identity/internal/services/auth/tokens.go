package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// Token type claim values
const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

// Standard JWT claims
const (
	claimSubject    = "sub"
	claimIssuer     = "iss"
	claimAudience   = "aud"
	claimIssuedAt   = "iat"
	claimExpiration = "exp"
	claimTokenType  = "typ"
	claimJWTID      = "jti"
)

// JWTTokenIssuer handles JWT token creation for local authentication.
type JWTTokenIssuer struct {
	jwtAuth     *jwtauth.JWTAuth
	tenantClaim string
	issuer      string
	audience    string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

const jwtAlgHS256 = "HS256"

// NewJWTTokenIssuer creates a new token issuer with the given configuration.
// jwtAlgHS256 is the HMAC-SHA256 JWT signing algorithm identifier.
func NewJWTTokenIssuer(
	hmacSecret []byte,
	tenantClaim string,
	issuer string,
	audience string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *JWTTokenIssuer {
	// Apply defaults from config constants if not set
	if issuer == "" {
		issuer = config.AuthDefaultIssuer
	}
	if audience == "" {
		audience = config.AuthDefaultAudience
	}

	jwtAuth := jwtauth.New(jwtAlgHS256, hmacSecret, nil)

	return &JWTTokenIssuer{
		jwtAuth:     jwtAuth,
		tenantClaim: tenantClaim,
		issuer:      issuer,
		audience:    audience,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

// IssueAccessToken creates a new access token for the given user and org.
func (ti *JWTTokenIssuer) IssueAccessToken(userID uuid.UUID, orgID *uuid.UUID) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(ti.accessTTL)

	claims := map[string]interface{}{
		claimSubject:    userID.String(),
		claimIssuer:     ti.issuer,
		claimAudience:   ti.audience,
		claimIssuedAt:   now.Unix(),
		claimExpiration: exp.Unix(),
		claimTokenType:  tokenTypeAccess,
	}

	// Add org claim if present
	if orgID != nil && ti.tenantClaim != "" {
		claims[ti.tenantClaim] = orgID.String()
	}

	_, tokenString, err := ti.jwtAuth.Encode(claims)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errPrefixIssueAccessToken, err)
	}

	return tokenString, nil
}

// IssueRefreshToken creates a new refresh token for the given user.
// Refresh tokens have longer TTL and are stored in the database for rotation.
func (ti *JWTTokenIssuer) IssueRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now().UTC()
	exp := now.Add(ti.refreshTTL)

	// Generate a unique token ID for tracking
	tokenID := uuid.New()

	claims := map[string]interface{}{
		claimSubject:    userID.String(),
		claimIssuer:     ti.issuer,
		claimAudience:   ti.audience,
		claimIssuedAt:   now.Unix(),
		claimExpiration: exp.Unix(),
		claimTokenType:  tokenTypeRefresh,
		claimJWTID:      tokenID.String(), // JWT ID for uniqueness
	}

	_, tokenString, err := ti.jwtAuth.Encode(claims)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errPrefixIssueRefreshToken, err)
	}

	return tokenString, nil
}

// ParseRefreshToken validates and parses a refresh token.
// Returns the user ID and token if valid.
func (ti *JWTTokenIssuer) ParseRefreshToken(tokenString string) (uuid.UUID, jwt.Token, error) {
	token, err := ti.jwtAuth.Decode(tokenString)
	if err != nil {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}

	// Validate typ claim
	var typClaim interface{}
	if err := token.Get(claimTokenType, &typClaim); err != nil {
		return uuid.Nil, nil, ErrInvalidRefreshToken
	}
	if typStr, ok := typClaim.(string); !ok || typStr != tokenTypeRefresh {
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
func (ti *JWTTokenIssuer) GetAccessTTL() int64 {
	return int64(ti.accessTTL.Seconds())
}

// GetRefreshTTL returns the refresh token TTL in seconds.
func (ti *JWTTokenIssuer) GetRefreshTTL() int64 {
	return int64(ti.refreshTTL.Seconds())
}

// GetRefreshExpiresAt returns the expiration time for a new refresh token.
func (ti *JWTTokenIssuer) GetRefreshExpiresAt() time.Time {
	return time.Now().UTC().Add(ti.refreshTTL)
}

// HashRefreshToken computes SHA256 hex hash of a refresh token.
// This hash is stored in the database instead of the raw token.
func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
