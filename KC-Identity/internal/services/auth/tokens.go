package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwt"
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
	hmacSecret  []byte
	tenantClaim string
	issuer      string
	audience    string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewJWTTokenIssuer creates a new token issuer with the given configuration.
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

	return &JWTTokenIssuer{
		hmacSecret:  hmacSecret,
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

	tokenString, err := ti.sign(claims)
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

	tokenString, err := ti.sign(claims)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errPrefixIssueRefreshToken, err)
	}

	return tokenString, nil
}

// ParseRefreshToken validates and parses a refresh token.
// Returns the user ID if valid.
func (ti *JWTTokenIssuer) ParseRefreshToken(tokenString string) (uuid.UUID, error) {
	// Expiry is checked below so every refusal is the same ErrInvalidRefreshToken.
	token, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.HS256(), ti.hmacSecret), jwt.WithValidate(false))
	if err != nil {
		return uuid.Nil, ErrInvalidRefreshToken
	}

	tokenType, err := jwt.Get[string](token, claimTokenType)
	if err != nil || tokenType != tokenTypeRefresh {
		return uuid.Nil, ErrInvalidRefreshToken
	}

	exp, ok := token.Expiration()
	if !ok || time.Now().After(exp) {
		return uuid.Nil, ErrInvalidRefreshToken
	}

	subjectStr, ok := token.Subject()
	if !ok {
		return uuid.Nil, ErrInvalidRefreshToken
	}
	userID, err := uuid.Parse(subjectStr)
	if err != nil {
		return uuid.Nil, ErrInvalidRefreshToken
	}

	return userID, nil
}

// sign builds an HS256 token from the claims.
func (ti *JWTTokenIssuer) sign(claims map[string]interface{}) (string, error) {
	token := jwt.New()
	for name, value := range claims {
		if err := token.Set(name, value); err != nil {
			return "", fmt.Errorf("%s %s: %w", errPrefixSetClaim, name, err)
		}
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), ti.hmacSecret))
	if err != nil {
		return "", fmt.Errorf("%s: %w", errPrefixSignToken, err)
	}
	return string(signed), nil
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
