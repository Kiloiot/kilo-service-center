package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	extTestSubject = "idp-subject-1"
	extTestEmail   = "member@example.com"
	extTestCode    = "authorization-code"
	extTestState   = "state-token"
	extTestEnabled = true
)

// externalUsers is an external user store whose lookups miss with a wrapped
// storage.ErrRecordNotFound, like the real one; it counts every write.
type externalUsers struct {
	user   *models.User
	writes int
}

func (e *externalUsers) lookup(match bool, key string) (*models.User, error) {
	if !match {
		return nil, fmt.Errorf("user %s: %w", key, storage.ErrRecordNotFound)
	}
	copied := *e.user
	return &copied, nil
}

func (e *externalUsers) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	return e.lookup(id == e.user.ID, id.String())
}

func (e *externalUsers) GetByEmail(_ context.Context, email string) (*models.User, error) {
	return e.lookup(email == e.user.Email, email)
}

func (e *externalUsers) GetByExternalID(_ context.Context, externalID string) (*models.User, error) {
	return e.lookup(e.user.ExternalID != nil && *e.user.ExternalID == externalID, externalID)
}

func (e *externalUsers) SetPasswordHash(context.Context, uuid.UUID, string) error {
	e.writes++
	return nil
}

func (e *externalUsers) Create(context.Context, *models.User) error {
	e.writes++
	return nil
}

func (e *externalUsers) Update(context.Context, *models.User) error {
	e.writes++
	return nil
}

// providerState returns the stored state of one provider for any key.
type providerState struct{ provider string }

func (p providerState) StoreState(context.Context, string, []byte, time.Duration) error { return nil }
func (p providerState) GetState(context.Context, string) ([]byte, error) {
	return json.Marshal(State{Provider: p.provider})
}
func (p providerState) DeleteState(context.Context, string) error { return nil }

// verifiedOIDC authenticates extTestSubject with a verified extTestEmail.
type verifiedOIDC struct{}

func (verifiedOIDC) GetAuthorizationURL(string, string) string { return "" }
func (verifiedOIDC) ExchangeCode(context.Context, string) (*OIDCTokenResponse, error) {
	return &OIDCTokenResponse{IDToken: "id-token"}, nil
}
func (verifiedOIDC) ValidateIDToken(context.Context, string, string) (*OIDCClaims, map[string]any, error) {
	return &OIDCClaims{Sub: extTestSubject, Email: extTestEmail, EmailVerified: true}, nil, nil
}

// verifiedOAuth2 authenticates extTestSubject with a verified extTestEmail.
type verifiedOAuth2 struct{}

func (verifiedOAuth2) GetAuthorizationURL(string) (string, string, error) { return "", "", nil }
func (verifiedOAuth2) ExchangeCode(context.Context, string, string) (*OAuth2TokenResponse, error) {
	return &OAuth2TokenResponse{AccessToken: "provider-access"}, nil
}
func (verifiedOAuth2) GetUserInfo(context.Context, string) (*OAuth2UserInfo, error) {
	return &OAuth2UserInfo{Sub: extTestSubject, Email: extTestEmail, EmailVerified: true}, nil
}

// countingAccessIssuer counts every access token it issues.
type countingAccessIssuer struct{ issued int }

func (c *countingAccessIssuer) IssueAccessToken(uuid.UUID, *uuid.UUID) (string, error) {
	c.issued++
	return "local-access", nil
}
func (c *countingAccessIssuer) GetAccessTTL() int64 { return int64(time.Minute.Seconds()) }

type externalLogin func(svc *ExternalAuthService) error

func completeOIDC(svc *ExternalAuthService) error {
	_, err := svc.CompleteOIDCLogin(testutil.TestContext(), extTestCode, extTestState)
	return err
}

func completeOAuth2(svc *ExternalAuthService) error {
	_, err := svc.CompleteOAuth2Login(testutil.TestContext(), extTestCode, extTestState)
	return err
}

func newExternalTestService(users *externalUsers, issuer *countingAccessIssuer) *ExternalAuthService {
	return NewExternalAuthService(
		verifiedOIDC{}, verifiedOAuth2{},
		providerState{provider: ProviderOIDC}, providerState{provider: ProviderOAuth2},
		users, oneMembership{}, issuer, nil, nil,
		ExternalAuthServiceConfig{OIDCEnabled: extTestEnabled, OAuth2Enabled: extTestEnabled},
		logger.NewNop(),
	)
}

func TestExternalLogin_GatesExistingAccountsOnActivity(t *testing.T) {
	subject := extTestSubject
	providers := []struct {
		name  string
		login externalLogin
	}{
		{name: "oidc", login: completeOIDC},
		{name: "oauth2", login: completeOAuth2},
	}
	accounts := []struct {
		name       string
		externalID *string
		active     bool
		wantErr    error
		wantWrites int
		wantIssued int
	}{
		{name: "deactivated, found by external id", externalID: &subject, wantErr: ErrUserNotActive},
		{name: "deactivated, found by email", wantErr: ErrUserNotActive},
		{name: "active, found by external id", externalID: &subject, active: true, wantIssued: 1},
		{name: "active, found by email is linked", active: true, wantWrites: 1, wantIssued: 1},
	}
	for _, provider := range providers {
		for _, account := range accounts {
			t.Run(provider.name+"/"+account.name, func(t *testing.T) {
				users := &externalUsers{user: &models.User{ID: uuid.New(), Email: extTestEmail, ExternalID: account.externalID, IsActive: account.active}}
				issuer := &countingAccessIssuer{}

				err := provider.login(newExternalTestService(users, issuer))

				require.ErrorIs(t, err, account.wantErr)
				assert.Equal(t, account.wantWrites, users.writes, "a deactivated account must not be linked")
				assert.Equal(t, account.wantIssued, issuer.issued)
			})
		}
	}
}

func TestIssueLocalTokensWithOrg_RefusesADeactivatedUser(t *testing.T) {
	users := &externalUsers{user: &models.User{ID: uuid.New(), Email: extTestEmail}}
	issuer := &countingAccessIssuer{}

	result, err := newExternalTestService(users, issuer).issueLocalTokensWithOrg(testutil.TestContext(), users.user, uuid.Nil)

	require.ErrorIs(t, err, ErrUserNotActive)
	assert.Nil(t, result)
	assert.Zero(t, issuer.issued)
}
