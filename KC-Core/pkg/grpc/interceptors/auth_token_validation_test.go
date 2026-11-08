package interceptors

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	validationTestIssuer      = "validation-issuer"
	validationTestAudience    = "validation-audience"
	validationTestOtherValue  = "someone-else"
	validationTestTenantClaim = "tenant_id"
	validationTestTenantID    = int64(42)
	validationTestMethod      = "/kc.Test/Method"
	validationTestOtherSecret = "another-secret-of-at-least-32-bytes!"
	validationTestKeyID       = "validation-key"
	validationTestRSABits     = 2048
	validationTestContentType = "application/json"
	validationTestHeaderType  = "Content-Type"
	validationTestNoneHeader  = `{"alg":"none","typ":"JWT"}`
	validationTestPartSep     = "."
)

func validationInterceptor(t *testing.T) *AuthInterceptor {
	t.Helper()
	ai, err := NewAuthInterceptor(AuthConfig{
		Enabled:     testAuthEnabled,
		HMACSecret:  testHMACSecret,
		Issuer:      validationTestIssuer,
		Audience:    validationTestAudience,
		TenantClaim: validationTestTenantClaim,
		Logger:      logger.Get(),
	})
	require.NoError(t, err)
	return ai
}

func validationClaims(t *testing.T) jwt.Token {
	t.Helper()
	token := jwt.New()
	require.NoError(t, token.Set(jwt.IssuerKey, validationTestIssuer))
	require.NoError(t, token.Set(jwt.AudienceKey, []string{validationTestAudience}))
	require.NoError(t, token.Set(jwt.ExpirationKey, time.Now().Add(time.Hour)))
	require.NoError(t, token.Set(validationTestTenantClaim, validationTestTenantID))
	return token
}

func signHS256(t *testing.T, token jwt.Token, secret string) string {
	t.Helper()
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), []byte(secret)))
	require.NoError(t, err)
	return string(signed)
}

func TestAuthenticateJWT_AcceptsAValidToken(t *testing.T) {
	ctx, err := validationInterceptor(t).authenticateJWT(testutil.TestContext(),
		signHS256(t, validationClaims(t), testHMACSecret), validationTestMethod)
	require.NoError(t, err)

	tenantID, err := pkgcontext.GetTenantID(ctx)
	require.NoError(t, err)
	assert.Equal(t, validationTestTenantID, tenantID)
}

func TestAuthenticateJWT_RefusesInvalidTokens(t *testing.T) {
	unsigned := func(t *testing.T) string {
		t.Helper()
		payload, err := json.Marshal(validationClaims(t))
		require.NoError(t, err)
		encode := base64.RawURLEncoding.EncodeToString
		return encode([]byte(validationTestNoneHeader)) + validationTestPartSep + encode(payload) + validationTestPartSep
	}
	withClaim := func(name string, value any) func(*testing.T) string {
		return func(t *testing.T) string {
			t.Helper()
			token := validationClaims(t)
			require.NoError(t, token.Set(name, value))
			return signHS256(t, token, testHMACSecret)
		}
	}

	cases := map[string]func(*testing.T) string{
		"expired":        withClaim(jwt.ExpirationKey, time.Now().Add(-time.Hour)),
		"not yet valid":  withClaim(jwt.NotBeforeKey, time.Now().Add(time.Hour)),
		"wrong issuer":   withClaim(jwt.IssuerKey, validationTestOtherValue),
		"wrong audience": withClaim(jwt.AudienceKey, []string{validationTestOtherValue}),
		"unsigned":       unsigned,
		"forged signature": func(t *testing.T) string {
			t.Helper()
			return signHS256(t, validationClaims(t), validationTestOtherSecret)
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := validationInterceptor(t).authenticateJWT(testutil.TestContext(), build(t), validationTestMethod)
			require.Error(t, err)
		})
	}
}

func TestAuthenticateJWT_UsesTheConfiguredKeySet(t *testing.T) {
	signingKey := newRSASigningKey(t)
	publicKey, err := jwk.PublicKeyOf(signingKey)
	require.NoError(t, err)
	set := jwk.NewSet()
	require.NoError(t, set.AddKey(publicKey))
	body, err := json.Marshal(set)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(validationTestHeaderType, validationTestContentType)
		_, writeErr := w.Write(body)
		assert.NoError(t, writeErr)
	}))
	t.Cleanup(server.Close)

	ai, err := NewAuthInterceptor(AuthConfig{
		Enabled:      testAuthEnabled,
		JWKSEndpoint: server.URL,
		TenantClaim:  validationTestTenantClaim,
		Logger:       logger.Get(),
	})
	require.NoError(t, err)

	signWith := func(key jwk.Key) string {
		signed, signErr := jwt.Sign(validationClaims(t), jwt.WithKey(jwa.RS256(), key))
		require.NoError(t, signErr)
		return string(signed)
	}

	_, err = ai.authenticateJWT(testutil.TestContext(), signWith(signingKey), validationTestMethod)
	require.NoError(t, err)

	_, err = ai.authenticateJWT(testutil.TestContext(), signWith(newRSASigningKey(t)), validationTestMethod)
	require.Error(t, err)
}

func newRSASigningKey(t *testing.T) jwk.Key {
	t.Helper()
	raw, err := rsa.GenerateKey(rand.Reader, validationTestRSABits)
	require.NoError(t, err)
	key, err := jwk.Import[jwk.Key](raw)
	require.NoError(t, err)
	require.NoError(t, key.Set(jwk.KeyIDKey, validationTestKeyID))
	require.NoError(t, key.Set(jwk.AlgorithmKey, jwa.RS256()))
	return key
}
