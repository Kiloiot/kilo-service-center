package auth

import (
	"io"
	"strings"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	authsvc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingReader simulates an exhausted platform entropy source.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func entropyTestConfig() OAuth2ClientConfig {
	return OAuth2ClientConfig{
		AuthorizeURL: "https://idp.example/authorize",
		ClientID:     "client",
		RedirectURL:  "https://app.example/callback",
		Scopes:       []string{"openid"},
		PKCEMethod:   authsvc.PKCEMethodS256,
	}
}

func TestGetAuthorizationURL_EntropyFailure(t *testing.T) {
	client, err := newOAuth2ClientWithEntropy(entropyTestConfig(), logger.NewNop(), failingReader{})
	require.NoError(t, err)

	authURL, verifier, err := client.GetAuthorizationURL("state-token")

	require.ErrorIs(t, err, authsvc.ErrEntropyUnavailable)
	assert.Empty(t, authURL, "no authorization URL may be issued on entropy failure")
	assert.Empty(t, verifier, "no code verifier may be issued on entropy failure")
}

func TestGetAuthorizationURL_DefaultEntropy(t *testing.T) {
	client, err := NewOAuth2Client(entropyTestConfig(), logger.NewNop())
	require.NoError(t, err)

	authURL, verifier, err := client.GetAuthorizationURL("state-token")

	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.True(t, strings.HasPrefix(authURL, "https://idp.example/authorize?"))
	assert.Contains(t, authURL, "state=state-token")
	assert.Contains(t, authURL, "code_challenge=")
}

func TestNewOAuth2Client_RejectsUnknownPKCEMethod(t *testing.T) {
	cfg := entropyTestConfig()
	cfg.PKCEMethod = "sha1"
	_, err := NewOAuth2Client(cfg, logger.NewNop())
	require.ErrorIs(t, err, ErrInvalidPKCEMethod)

	cfg.PKCEMethod = authsvc.PKCEMethodPlain
	_, err = NewOAuth2Client(cfg, logger.NewNop())
	require.NoError(t, err)
}
