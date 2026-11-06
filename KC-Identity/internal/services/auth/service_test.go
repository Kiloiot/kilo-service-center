package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	svcTestSecret      = "auth-service-test-secret"
	svcTestTenantClaim = "org_id"
	svcTestEmail       = "operator@example.com"
	svcTestPassword    = "correct-horse1"
	svcTestAccessTTL   = time.Minute
	svcTestRefreshTTL  = time.Hour
	svcTestLocalLogin  = true
	svcTestRefresh     = true
)

var errSvcTestStoreDown = errors.New("refresh token store unavailable")

// userMemory is a user store whose lookups miss with a wrapped
// storage.ErrRecordNotFound, like the real one.
type userMemory struct {
	mu    sync.Mutex
	users map[uuid.UUID]*models.User
}

func (m *userMemory) GetByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user %s: %w", id, storage.ErrRecordNotFound)
	}
	copied := *user
	return &copied, nil
}

func (m *userMemory) GetByEmail(_ context.Context, email string) (*models.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, user := range m.users {
		if user.Email == email {
			copied := *user
			return &copied, nil
		}
	}
	return nil, fmt.Errorf("user %s: %w", email, storage.ErrRecordNotFound)
}

func (m *userMemory) SetPasswordHash(_ context.Context, id uuid.UUID, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[id].PasswordHash = &hash
	return nil
}

// tokenMemory honours the refresh token store contract: an unknown hash is
// (nil, nil) and Rotate claims only a usable token of an active user.
// rotateErr, when set, replaces the outcome of Rotate.
type tokenMemory struct {
	mu        sync.Mutex
	byHash    map[string]*models.RefreshToken
	owners    UserStore
	rotateErr error
	revokes   int
}

func (m *tokenMemory) Create(_ context.Context, token *models.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored := *token
	m.byHash[token.TokenHash] = &stored
	return nil
}

func (m *tokenMemory) GetByHash(_ context.Context, tokenHash string) (*models.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	token, ok := m.byHash[tokenHash]
	if !ok {
		return nil, nil
	}
	copied := *token
	return &copied, nil
}

func (m *tokenMemory) Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error {
	if m.rotateErr != nil {
		return m.rotateErr
	}
	owner, err := m.owners.GetByID(ctx, newToken.UserID)
	if err != nil || !owner.IsActive {
		return storage.ErrRefreshTokenUserInactive
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, token := range m.byHash {
		if token.ID == oldTokenID && token.IsValid() {
			token.ReplacedBy = &newToken.ID
			stored := *newToken
			m.byHash[newToken.TokenHash] = &stored
			return nil
		}
	}
	return storage.ErrRefreshTokenNotClaimable
}

func (m *tokenMemory) RevokeByUserID(_ context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revokes++
	now := time.Now().UTC()
	for _, token := range m.byHash {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &now
		}
	}
	return nil
}

func (m *tokenMemory) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byHash)
}

type oneMembership struct{}

func (oneMembership) ListUserMemberships(context.Context, uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	return []*models.OrganizationMembershipWithOrg{{OrgID: uuid.New(), Role: models.OrganizationRoleMember}}, nil
}

// authFixture wires a real token issuer to in-memory stores holding one user.
type authFixture struct {
	svc    *Service
	issuer *JWTTokenIssuer
	users  *userMemory
	tokens *tokenMemory
	user   *models.User
}

func newAuthFixture(t *testing.T, active bool) authFixture {
	t.Helper()
	salt := make([]byte, config.AuthPBKDF2SaltLength)
	_, err := rand.Read(salt)
	require.NoError(t, err)
	hash := HashPassword(svcTestPassword, salt, config.AuthPBKDF2Iterations)
	user := &models.User{ID: uuid.New(), Email: svcTestEmail, IsActive: active, PasswordHash: &hash}
	users := &userMemory{users: map[uuid.UUID]*models.User{user.ID: user}}
	tokens := &tokenMemory{byHash: map[string]*models.RefreshToken{}, owners: users}
	issuer := NewJWTTokenIssuer([]byte(svcTestSecret), svcTestTenantClaim, "", "", svcTestAccessTTL, svcTestRefreshTTL)
	svc := NewService(users, oneMembership{}, tokens, issuer, svcTestLocalLogin, svcTestRefresh, logger.NewNop())
	return authFixture{svc: svc, issuer: issuer, users: users, tokens: tokens, user: user}
}

// seedRefreshToken stores a refresh token of the fixture user shaped by edit
// and returns the raw token a client would present.
func (f authFixture) seedRefreshToken(t *testing.T, edit func(*models.RefreshToken)) string {
	t.Helper()
	raw, err := f.issuer.IssueRefreshToken(f.user.ID)
	require.NoError(t, err)
	token := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    f.user.ID,
		TokenHash: HashRefreshToken(raw),
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(svcTestRefreshTTL),
	}
	edit(token)
	require.NoError(t, f.tokens.Create(testutil.TestContext(), token))
	return raw
}

func (f authFixture) stored(t *testing.T, raw string) *models.RefreshToken {
	t.Helper()
	token, err := f.tokens.GetByHash(testutil.TestContext(), HashRefreshToken(raw))
	require.NoError(t, err)
	require.NotNil(t, token)
	return token
}

func keepToken(*models.RefreshToken) {}

func TestLogin_IssuesTokensOnlyToActiveUsers(t *testing.T) {
	tests := []struct {
		name       string
		active     bool
		wantErr    error
		wantTokens int
	}{
		{name: "active user", active: true, wantTokens: 1},
		{name: "deactivated user", active: false, wantErr: ErrUserNotActive},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t, tt.active)

			result, err := f.svc.Login(testutil.TestContext(), svcTestEmail, svcTestPassword)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantTokens, f.tokens.count())
			if tt.wantErr != nil {
				assert.Nil(t, result)
				return
			}
			assert.NotEmpty(t, result.Tokens.AccessToken)
			assert.NotEmpty(t, result.Tokens.RefreshToken)
		})
	}
}

func TestRefreshTokens_RotatesAUsableToken(t *testing.T) {
	f := newAuthFixture(t, true)
	raw := f.seedRefreshToken(t, keepToken)

	tokens, err := f.svc.RefreshTokens(testutil.TestContext(), raw)

	require.NoError(t, err)
	require.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	require.NotEqual(t, raw, tokens.RefreshToken)
	successor := f.stored(t, tokens.RefreshToken)
	assert.True(t, successor.IsValid())
	old := f.stored(t, raw)
	require.NotNil(t, old.ReplacedBy)
	assert.Equal(t, successor.ID, *old.ReplacedBy)
	assert.Zero(t, f.tokens.revokes)
}

func TestRefreshTokens_RefusesUnusableTokens(t *testing.T) {
	past := time.Now().UTC().Add(-time.Minute)
	successor := uuid.New()
	tests := []struct {
		name        string
		edit        func(*models.RefreshToken)
		unknown     bool
		wantErr     error
		wantRevokes int
	}{
		{name: "revoked", edit: func(tok *models.RefreshToken) { tok.RevokedAt = &past }, wantErr: ErrRefreshTokenRevoked},
		{name: "replaced is reuse and ends every session", edit: func(tok *models.RefreshToken) { tok.ReplacedBy = &successor }, wantErr: ErrRefreshTokenReused, wantRevokes: 1},
		{name: "expired", edit: func(tok *models.RefreshToken) { tok.ExpiresAt = past }, wantErr: ErrInvalidRefreshToken},
		{name: "unknown to the store", edit: keepToken, unknown: true, wantErr: ErrInvalidRefreshToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t, true)
			raw := f.seedRefreshToken(t, tt.edit)
			if tt.unknown {
				raw = f.seedRefreshToken(t, func(tok *models.RefreshToken) { tok.TokenHash = "other" })
			}
			before := f.tokens.count()

			tokens, err := f.svc.RefreshTokens(testutil.TestContext(), raw)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, tokens)
			assert.Equal(t, before, f.tokens.count(), "a refused refresh must not store a token")
			assert.Equal(t, tt.wantRevokes, f.tokens.revokes)
		})
	}
}

func TestRefreshTokens_RefusesADeactivatedUserAndEndsTheirSessions(t *testing.T) {
	f := newAuthFixture(t, false)
	raw := f.seedRefreshToken(t, keepToken)

	tokens, err := f.svc.RefreshTokens(testutil.TestContext(), raw)

	require.ErrorIs(t, err, ErrUserNotActive)
	assert.Nil(t, tokens)
	assert.Equal(t, 1, f.tokens.count(), "no successor may be stored")
	assert.Equal(t, 1, f.tokens.revokes)
	assert.True(t, f.stored(t, raw).IsRevoked())
}

func TestRefreshTokens_MapsARefusedRotation(t *testing.T) {
	tests := []struct {
		name      string
		rotateErr error
		wantErr   error
	}{
		{name: "another request claimed the token first", rotateErr: storage.ErrRefreshTokenNotClaimable, wantErr: ErrInvalidRefreshToken},
		{name: "user deactivated during the rotation", rotateErr: storage.ErrRefreshTokenUserInactive, wantErr: ErrUserNotActive},
		{name: "store failure", rotateErr: errSvcTestStoreDown, wantErr: errSvcTestStoreDown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t, true)
			raw := f.seedRefreshToken(t, keepToken)
			f.tokens.rotateErr = tt.rotateErr

			tokens, err := f.svc.RefreshTokens(testutil.TestContext(), raw)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, tokens)
			assert.Zero(t, f.tokens.revokes, "a lost race is not reuse: no session may be revoked")
		})
	}
}

func TestRefreshTokens_PresentingARotatedTokenAgainIsReuse(t *testing.T) {
	f := newAuthFixture(t, true)
	raw := f.seedRefreshToken(t, keepToken)
	first, err := f.svc.RefreshTokens(testutil.TestContext(), raw)
	require.NoError(t, err)

	_, err = f.svc.RefreshTokens(testutil.TestContext(), raw)

	require.ErrorIs(t, err, ErrRefreshTokenReused)
	assert.True(t, f.stored(t, first.RefreshToken).IsRevoked(), "reuse must revoke the successor too")
}
