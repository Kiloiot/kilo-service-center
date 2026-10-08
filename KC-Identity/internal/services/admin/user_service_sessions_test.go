package admin

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
)

const (
	sessionTestSecret      = "admin-reset-session-test-secret"
	sessionTestTenantClaim = "org_id"
	sessionTestEmail       = "operator@example.com"
	sessionTestOldPassword = "compromised1"
	sessionTestNewPassword = "replacement2"
	sessionTestLocalLogin  = true
	sessionTestUserActive  = true
	sessionTestRefresh     = true
	sessionTestAccessTTL   = time.Minute
	sessionTestRefreshTTL  = time.Hour
)

var errSessionRevokeUnavailable = errors.New("refresh token store unavailable")

type refreshTokenMemory struct {
	mu        sync.Mutex
	byHash    map[string]*models.RefreshToken
	revokeErr error
}

func newRefreshTokenMemory() *refreshTokenMemory {
	return &refreshTokenMemory{byHash: map[string]*models.RefreshToken{}}
}

func (m *refreshTokenMemory) Create(_ context.Context, token *models.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *token
	m.byHash[token.TokenHash] = &stored
	return nil
}

func (m *refreshTokenMemory) GetByHash(_ context.Context, tokenHash string) (*models.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	token, ok := m.byHash[tokenHash]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	copied := *token
	return &copied, nil
}

func (m *refreshTokenMemory) MarkReplaced(_ context.Context, tokenID, replacedByID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, token := range m.byHash {
		if token.ID == tokenID {
			token.ReplacedBy = &replacedByID
		}
	}
	return nil
}

func (m *refreshTokenMemory) RevokeByUserID(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.revokeErr != nil {
		return m.revokeErr
	}
	now := time.Now().UTC()
	for _, token := range m.byHash {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &now
		}
	}
	return nil
}

type singleMembership struct{}

func (singleMembership) ListUserMemberships(context.Context, uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	return []*models.OrganizationMembershipWithOrg{{OrgID: uuid.New(), Role: models.OrganizationRoleMember}}, nil
}

// oneUserStore holds a single account whose password hash follows SetPasswordHash.
func oneUserStore(t *testing.T) (*mockUserStore, uuid.UUID) {
	t.Helper()
	salt := make([]byte, config.AuthPBKDF2SaltLength)
	hash := auth.HashPassword(sessionTestOldPassword, salt, config.AuthPBKDF2Iterations)
	user := &models.User{ID: uuid.New(), Email: sessionTestEmail, IsActive: sessionTestUserActive, PasswordHash: &hash}
	var mu sync.Mutex
	current := func() (*models.User, error) {
		mu.Lock()
		defer mu.Unlock()
		copied := *user
		return &copied, nil
	}
	return &mockUserStore{
		getByIDFn:    func(context.Context, uuid.UUID) (*models.User, error) { return current() },
		getByEmailFn: func(context.Context, string) (*models.User, error) { return current() },
		setPasswordFn: func(_ context.Context, _ uuid.UUID, newHash string) error {
			mu.Lock()
			defer mu.Unlock()
			user.PasswordHash = &newHash
			return nil
		},
	}, user.ID
}

func sessionAuthService(users auth.UserStore, tokens auth.RefreshTokenStore) *auth.Service {
	issuer := auth.NewJWTTokenIssuer([]byte(sessionTestSecret), sessionTestTenantClaim, "", "", sessionTestAccessTTL, sessionTestRefreshTTL)
	return auth.NewService(users, singleMembership{}, tokens, issuer, sessionTestLocalLogin, sessionTestRefresh, newTestLogger())
}

func TestUpdatePassword_RefusesRefreshTokensIssuedBeforeTheReset(t *testing.T) {
	ctx := testutil.TestContext()
	users, userID := oneUserStore(t)
	tokens := newRefreshTokenMemory()
	login := sessionAuthService(users, tokens)
	admin := NewUserAdminService(users, tokens, newTestLogger())

	session, err := login.Login(ctx, sessionTestEmail, sessionTestOldPassword)
	require.NoError(t, err)
	require.NotEmpty(t, session.Tokens.RefreshToken)

	require.NoError(t, admin.UpdatePassword(ctx, userID, sessionTestNewPassword))

	_, err = login.RefreshTokens(ctx, session.Tokens.RefreshToken)
	require.ErrorIs(t, err, auth.ErrRefreshTokenRevoked)
}

func TestUpdatePassword_ReportsASessionRevocationFailure(t *testing.T) {
	ctx := testutil.TestContext()
	users, userID := oneUserStore(t)
	tokens := newRefreshTokenMemory()
	tokens.revokeErr = errSessionRevokeUnavailable
	admin := NewUserAdminService(users, tokens, newTestLogger())

	err := admin.UpdatePassword(ctx, userID, sessionTestNewPassword)
	require.ErrorIs(t, err, errSessionRevokeUnavailable)
}
