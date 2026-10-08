package auth

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failingReader always fails, simulating an exhausted platform entropy source.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// recordingStateStore records every stored state token.
type recordingStateStore struct {
	stored int
}

func (r *recordingStateStore) StoreState(context.Context, string, []byte, time.Duration) error {
	r.stored++
	return nil
}
func (r *recordingStateStore) GetState(context.Context, string) ([]byte, error) { return nil, nil }
func (r *recordingStateStore) DeleteState(context.Context, string) error        { return nil }

// recordingOIDCClient records authorization URL builds.
type recordingOIDCClient struct {
	OIDCClient
	urlBuilds int
}

func (r *recordingOIDCClient) GetAuthorizationURL(string, string) string {
	r.urlBuilds++
	return "https://idp.example/authorize"
}

// recordingOAuth2Client records authorization URL builds.
type recordingOAuth2Client struct {
	OAuth2Client
	urlBuilds int
	err       error
}

func (r *recordingOAuth2Client) GetAuthorizationURL(string) (string, string, error) {
	if r.err != nil {
		return "", "", r.err
	}
	r.urlBuilds++
	return "https://idp.example/authorize", "verifier", nil
}

// Provider toggles and the fake entropy failure shared by the entropy tests.
const entropyTestProvidersEnabled = true

var errVerifierEntropy = errors.New("verifier entropy failed")

func newEntropyTestService(oidc *recordingOIDCClient, oauth2 *recordingOAuth2Client, oidcStore, oauth2Store StateStore) *ExternalAuthService {
	return NewExternalAuthService(
		oidc, oauth2, oidcStore, oauth2Store,
		nil, nil, nil, nil, nil,
		ExternalAuthServiceConfig{OIDCEnabled: entropyTestProvidersEnabled, OAuth2Enabled: entropyTestProvidersEnabled},
		logger.NewNop(),
	)
}

func TestInitiateOIDCLogin_EntropyFailure(t *testing.T) {
	oidc := &recordingOIDCClient{}
	store := &recordingStateStore{}
	svc := newEntropyTestService(oidc, &recordingOAuth2Client{}, store, &recordingStateStore{}).
		WithEntropy(failingReader{})

	url, err := svc.InitiateOIDCLogin(testutil.TestContext())

	require.ErrorIs(t, err, ErrEntropyUnavailable)
	assert.Empty(t, url, "no authorization URL may be issued on entropy failure")
	assert.Zero(t, store.stored, "no state record may be stored on entropy failure")
	assert.Zero(t, oidc.urlBuilds, "no nonce-bearing URL may be built on entropy failure")
}

func TestInitiateOAuth2Login_StateEntropyFailure(t *testing.T) {
	oauth2 := &recordingOAuth2Client{}
	store := &recordingStateStore{}
	svc := newEntropyTestService(&recordingOIDCClient{}, oauth2, &recordingStateStore{}, store).
		WithEntropy(failingReader{})

	url, err := svc.InitiateOAuth2Login(testutil.TestContext())

	require.ErrorIs(t, err, ErrEntropyUnavailable)
	assert.Empty(t, url)
	assert.Zero(t, store.stored, "no state record may be stored on entropy failure")
	assert.Zero(t, oauth2.urlBuilds, "no verifier-bearing URL may be built on entropy failure")
}

func TestInitiateOAuth2Login_VerifierEntropyFailure(t *testing.T) {
	cause := errVerifierEntropy
	oauth2 := &recordingOAuth2Client{err: cause}
	store := &recordingStateStore{}
	svc := newEntropyTestService(&recordingOIDCClient{}, oauth2, &recordingStateStore{}, store)

	url, err := svc.InitiateOAuth2Login(testutil.TestContext())

	require.ErrorIs(t, err, cause)
	assert.Empty(t, url)
	assert.Zero(t, store.stored, "no state record may be stored when the PKCE verifier fails")
}

func TestInitiateLogins_SucceedWithDefaultEntropy(t *testing.T) {
	oidc := &recordingOIDCClient{}
	oauth2 := &recordingOAuth2Client{}
	oidcStore := &recordingStateStore{}
	oauth2Store := &recordingStateStore{}
	svc := newEntropyTestService(oidc, oauth2, oidcStore, oauth2Store)

	oidcURL, err := svc.InitiateOIDCLogin(testutil.TestContext())
	require.NoError(t, err)
	assert.NotEmpty(t, oidcURL)
	assert.Equal(t, 1, oidcStore.stored)

	oauthURL, err := svc.InitiateOAuth2Login(testutil.TestContext())
	require.NoError(t, err)
	assert.NotEmpty(t, oauthURL)
	assert.Equal(t, 1, oauth2Store.stored)
}
