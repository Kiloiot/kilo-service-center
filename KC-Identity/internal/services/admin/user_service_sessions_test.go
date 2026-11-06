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
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
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

// refreshTokenMemory honours the refresh token store contract: an unknown
// hash is (nil, nil), a session opens only for an active owner who still has
// the verified password, and Rotate claims only a usable token of an active
// owner.
type refreshTokenMemory struct {
	mu        sync.Mutex
	byHash    map[string]*models.RefreshToken
	owners    UserAdminStore
	revokeErr error
}

func newRefreshTokenMemory(owners UserAdminStore) *refreshTokenMemory {
	return &refreshTokenMemory{byHash: map[string]*models.RefreshToken{}, owners: owners}
}

func (m *refreshTokenMemory) CreateForPassword(ctx context.Context, token *models.RefreshToken, passwordHash string) error {
	owner, err := m.owners.GetByID(ctx, token.UserID)
	if err != nil || !owner.IsActive {
		return storage.ErrUserInactive
	}
	if owner.PasswordHash == nil || *owner.PasswordHash != passwordHash {
		return storage.ErrUserCredentialsChanged
	}
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
		return nil, nil
	}
	copied := *token
	return &copied, nil
}

func (m *refreshTokenMemory) Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error {
	owner, err := m.owners.GetByID(ctx, newToken.UserID)
	if err != nil || !owner.IsActive {
		return storage.ErrUserInactive
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, token := range m.byHash {
		if token.ID == oldTokenID && token.UserID == newToken.UserID && token.IsValid() {
			token.ReplacedBy = &newToken.ID
			stored := *newToken
			m.byHash[newToken.TokenHash] = &stored
			return nil
		}
	}
	return storage.ErrRefreshTokenNotClaimable
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

// oneUserStore holds a single account that follows SetPasswordHash and Update.
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
		updateFn: func(_ context.Context, updated *models.User) error {
			mu.Lock()
			defer mu.Unlock()
			applyStoredUpdate(user, updated)
			return nil
		},
	}, user.ID
}

// applyStoredUpdate copies the columns UserRepository.Update writes; the
// password hash is never among them.
func applyStoredUpdate(stored, updated *models.User) {
	stored.ExternalID = updated.ExternalID
	stored.Email = updated.Email
	stored.EmailVerified = updated.EmailVerified
	stored.IsAdmin = updated.IsAdmin
	stored.IsActive = updated.IsActive
	stored.IsTenantManager = updated.IsTenantManager
	stored.IsBaseStationManager = updated.IsBaseStationManager
	stored.IsEndpointManager = updated.IsEndpointManager
	stored.Note = updated.Note
	stored.FirstName = updated.FirstName
	stored.LastName = updated.LastName
	stored.CompanyName = updated.CompanyName
}

func sessionAuthService(users auth.UserStore, tokens auth.RefreshTokenStore) *auth.Service {
	issuer := auth.NewJWTTokenIssuer([]byte(sessionTestSecret), sessionTestTenantClaim, "", "", sessionTestAccessTTL, sessionTestRefreshTTL)
	return auth.NewService(users, singleMembership{}, tokens, issuer, sessionTestLocalLogin, sessionTestRefresh, newTestLogger())
}

func TestUpdatePassword_RefusesRefreshTokensIssuedBeforeTheReset(t *testing.T) {
	ctx := testutil.TestContext()
	users, userID := oneUserStore(t)
	tokens := newRefreshTokenMemory(users)
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
	tokens := newRefreshTokenMemory(users)
	tokens.revokeErr = errSessionRevokeUnavailable
	admin := NewUserAdminService(users, tokens, newTestLogger())

	err := admin.UpdatePassword(ctx, userID, sessionTestNewPassword)
	require.ErrorIs(t, err, errSessionRevokeUnavailable)
	require.ErrorIs(t, err, auth.ErrPasswordChangedSessionsKept, "the caller must learn that the new password was stored")
}

func TestUpdate_DeactivationEndsSessionsAndRefusesRefresh(t *testing.T) {
	ctx := testutil.TestContext()
	users, userID := oneUserStore(t)
	tokens := newRefreshTokenMemory(users)
	login := sessionAuthService(users, tokens)
	admin := NewUserAdminService(users, tokens, newTestLogger())

	session, err := login.Login(ctx, sessionTestEmail, sessionTestOldPassword)
	require.NoError(t, err)
	inactive := !sessionTestUserActive

	_, err = admin.Update(ctx, userID, &grpcservices.UserUpdateRequest{IsActive: &inactive})
	require.NoError(t, err)

	stored, err := tokens.GetByHash(ctx, auth.HashRefreshToken(session.Tokens.RefreshToken))
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.True(t, stored.IsRevoked(), "deactivation must revoke the refresh token")
	tokensAfter, err := login.RefreshTokens(ctx, session.Tokens.RefreshToken)
	require.ErrorIs(t, err, auth.ErrRefreshTokenRevoked)
	require.Nil(t, tokensAfter)
}

// countingRevoker records every session revocation.
type countingRevoker struct{ calls int }

func (c *countingRevoker) RevokeByUserID(context.Context, uuid.UUID) error {
	c.calls++
	return nil
}

func TestUpdate_RevokesSessionsOnEveryDeactivatingRequest(t *testing.T) {
	active, inactive := true, false
	note := "rotated on-call duty"
	tests := []struct {
		name        string
		startActive bool
		req         grpcservices.UserUpdateRequest
		wantRevokes int
	}{
		{name: "active user deactivated", startActive: true, req: grpcservices.UserUpdateRequest{IsActive: &inactive}, wantRevokes: 1},
		{name: "inactive user deactivated again", startActive: false, req: grpcservices.UserUpdateRequest{IsActive: &inactive}, wantRevokes: 1},
		{name: "active user confirmed active", startActive: true, req: grpcservices.UserUpdateRequest{IsActive: &active}},
		{name: "unrelated field on an active user", startActive: true, req: grpcservices.UserUpdateRequest{Note: &note}},
		{name: "unrelated field on an inactive user", startActive: false, req: grpcservices.UserUpdateRequest{Note: &note}},
		{name: "inactive user reactivated", startActive: false, req: grpcservices.UserUpdateRequest{IsActive: &active}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, userID := oneUserStore(t)
			if !tt.startActive {
				require.NoError(t, users.Update(testutil.TestContext(), &models.User{ID: userID, Email: sessionTestEmail}))
			}
			revoker := &countingRevoker{}
			admin := NewUserAdminService(users, revoker, newTestLogger())

			_, err := admin.Update(testutil.TestContext(), userID, &tt.req)

			require.NoError(t, err)
			require.Equal(t, tt.wantRevokes, revoker.calls)
		})
	}
}

func TestUpdate_ReportsADeactivationRevocationFailure(t *testing.T) {
	ctx := testutil.TestContext()
	users, userID := oneUserStore(t)
	tokens := newRefreshTokenMemory(users)
	tokens.revokeErr = errSessionRevokeUnavailable
	admin := NewUserAdminService(users, tokens, newTestLogger())
	inactive := !sessionTestUserActive

	_, err := admin.Update(ctx, userID, &grpcservices.UserUpdateRequest{IsActive: &inactive})

	require.ErrorIs(t, err, errSessionRevokeUnavailable)
}
