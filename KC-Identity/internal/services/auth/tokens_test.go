package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tokensTestTenantClaim = "org_id"
	tokensTestIssuer      = "tokens-issuer"
	tokensTestAudience    = "tokens-audience"
	tokensTestTTL         = time.Hour
	tokensTestPartSep     = "."
	tokensTestPartCount   = 3
	tokensTestNoneHeader  = `{"alg":"none","typ":"JWT"}`
)

var (
	tokensTestSecret      = []byte("tokens-test-secret-of-32-bytes-long!")
	tokensTestOtherSecret = []byte("another-tokens-secret-of-32-bytes-ok")
)

func tokensTestIssuerWith(secret []byte, ttl time.Duration) *JWTTokenIssuer {
	return NewJWTTokenIssuer(secret, tokensTestTenantClaim, tokensTestIssuer, tokensTestAudience, ttl, ttl)
}

func TestParseRefreshToken_AcceptsItsOwnToken(t *testing.T) {
	issuer := tokensTestIssuerWith(tokensTestSecret, tokensTestTTL)
	userID := uuid.New()

	token, err := issuer.IssueRefreshToken(userID)
	require.NoError(t, err)

	got, err := issuer.ParseRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, got)
}

func TestParseRefreshToken_RefusesAnythingElse(t *testing.T) {
	issuer := tokensTestIssuerWith(tokensTestSecret, tokensTestTTL)
	userID := uuid.New()

	access, err := issuer.IssueAccessToken(userID, nil)
	require.NoError(t, err)
	forged, err := tokensTestIssuerWith(tokensTestOtherSecret, tokensTestTTL).IssueRefreshToken(userID)
	require.NoError(t, err)
	expired, err := tokensTestIssuerWith(tokensTestSecret, -tokensTestTTL).IssueRefreshToken(userID)
	require.NoError(t, err)
	valid, err := issuer.IssueRefreshToken(userID)
	require.NoError(t, err)
	parts := strings.Split(valid, tokensTestPartSep)
	require.Len(t, parts, tokensTestPartCount)
	unsigned := base64.RawURLEncoding.EncodeToString([]byte(tokensTestNoneHeader)) + tokensTestPartSep + parts[1] + tokensTestPartSep

	for name, token := range map[string]string{
		"access token":     access,
		"forged signature": forged,
		"expired":          expired,
		"unsigned":         unsigned,
	} {
		t.Run(name, func(t *testing.T) {
			_, parseErr := issuer.ParseRefreshToken(token)
			require.ErrorIs(t, parseErr, ErrInvalidRefreshToken)
		})
	}
}

// Clients read these claims, so their wire form must not change with the JWT library.
func TestIssueAccessToken_ClaimsKeepTheirWireForm(t *testing.T) {
	issuer := tokensTestIssuerWith(tokensTestSecret, tokensTestTTL)
	userID := uuid.New()
	orgID := uuid.New()

	token, err := issuer.IssueAccessToken(userID, &orgID)
	require.NoError(t, err)

	parts := strings.Split(token, tokensTestPartSep)
	require.Len(t, parts, tokensTestPartCount)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	assert.Equal(t, userID.String(), claims[claimSubject])
	assert.Equal(t, tokensTestIssuer, claims[claimIssuer])
	assert.Equal(t, []any{tokensTestAudience}, claims[claimAudience])
	assert.Equal(t, tokenTypeAccess, claims[claimTokenType])
	assert.Equal(t, orgID.String(), claims[tokensTestTenantClaim])
	assert.IsType(t, float64(0), claims[claimExpiration])
	assert.IsType(t, float64(0), claims[claimIssuedAt])
}
