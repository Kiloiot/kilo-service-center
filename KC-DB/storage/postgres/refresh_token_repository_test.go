package postgres

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenRepository_CreateForPassword_OpensASessionForCurrentCredentials(t *testing.T) {
	const verifiedHash = "pbkdf2-verified-hash"
	tests := []struct {
		name       string
		active     bool
		storedHash *string
		wantErr    error
	}{
		{name: "active user with the verified password", active: true, storedHash: ptr(verifiedHash)},
		{name: "deactivated user", active: false, storedHash: ptr(verifiedHash), wantErr: storage.ErrUserInactive},
		{name: "password changed after verification", active: true, storedHash: ptr("pbkdf2-newer-hash"), wantErr: storage.ErrUserCredentialsChanged},
		{name: "password removed after verification", active: true, wantErr: storage.ErrUserCredentialsChanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRotationFixture(t, tt.active, time.Now().UTC().Add(testTokenTTL))
			_, err := f.db.Exec("UPDATE users SET password_hash = $2 WHERE id = $1", f.userID, tt.storedHash)
			require.NoError(t, err)
			token := rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL))

			err = f.repo.CreateForPassword(testutil.TestContext(), token, verifiedHash)

			require.ErrorIs(t, err, tt.wantErr)
			stored := f.stored(t, token)
			if tt.wantErr != nil {
				assert.Nil(t, stored, "a refused session must store nothing")
				return
			}
			require.NotNil(t, stored)
			assert.True(t, stored.IsValid())
			assert.False(t, token.CreatedAt.IsZero())
		})
	}
}

func TestRefreshTokenRepository_CreateForPassword_RefusesAMissingUser(t *testing.T) {
	f := newRotationFixture(t, true, time.Now().UTC().Add(testTokenTTL))
	token := rotationToken(uuid.New(), time.Now().UTC().Add(testTokenTTL))

	err := f.repo.CreateForPassword(testutil.TestContext(), token, "any-hash")

	require.ErrorIs(t, err, storage.ErrUserInactive)
	assert.Nil(t, f.stored(t, token))
}

// TestRefreshTokenRepository_GetByHash_Success tests successful token retrieval
func TestRefreshTokenRepository_GetByHash_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db, clock.SystemClock{})
	userRepo := NewUserRepository(db, clock.SystemClock{})
	ctx := testutil.TestContext()

	// Create a test user first
	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "refresh-getbyhash@example.com",
		IsActive: true,
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	tokenID := uuid.New()
	tokenHash := "gethash" + tokenID.String()[:8]
	token := &models.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: tokenHash,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(testTokenTTL),
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", tokenID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	_, err = db.NamedExec(insertRefreshTokenQuery, token)
	require.NoError(t, err)

	// Test: Get token by hash
	retrieved, err := repo.GetByHash(ctx, tokenHash)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, tokenID, retrieved.ID)
	assert.Equal(t, userID, retrieved.UserID)
	assert.Equal(t, tokenHash, retrieved.TokenHash)
}

// TestRefreshTokenRepository_GetByHash_NotFound tests token not found returns nil
func TestRefreshTokenRepository_GetByHash_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db, clock.SystemClock{})
	ctx := testutil.TestContext()

	// Test: Get non-existent token
	token, err := repo.GetByHash(ctx, "nonexistent-hash")
	require.NoError(t, err)
	assert.Nil(t, token) // Not found returns nil, not error
}

// TestRefreshTokenRepository_RevokeByUserID_Success tests family revocation
func TestRefreshTokenRepository_RevokeByUserID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db, clock.SystemClock{})
	userRepo := NewUserRepository(db, clock.SystemClock{})
	ctx := testutil.TestContext()

	// Create a test user first
	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "refresh-revokebyuser@example.com",
		IsActive: true,
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Create multiple tokens for the same user
	tokenIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		tokenIDs[i] = uuid.New()
		token := &models.RefreshToken{
			ID:        tokenIDs[i],
			UserID:    userID,
			TokenHash: "familyhash" + tokenIDs[i].String()[:8],
			IssuedAt:  time.Now().UTC(),
			ExpiresAt: time.Now().UTC().Add(testTokenTTL),
		}
		_, err := db.NamedExec(insertRefreshTokenQuery, token)
		require.NoError(t, err)
	}

	// Cleanup after test
	defer func() {
		for _, id := range tokenIDs {
			_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", id)
		}
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Test: Revoke all tokens for user
	err = repo.RevokeByUserID(ctx, userID)
	require.NoError(t, err)

	// Verify all tokens are revoked
	for _, id := range tokenIDs {
		var revokedAt *time.Time
		err := db.Get(&revokedAt, "SELECT revoked_at FROM refresh_tokens WHERE id = $1", id)
		require.NoError(t, err)
		assert.NotNil(t, revokedAt)
	}
}

// rotationFixture holds a repository, one user and one usable refresh token of that user.
type rotationFixture struct {
	db     *sqlx.DB
	repo   *RefreshTokenRepository
	userID uuid.UUID
	old    *models.RefreshToken
}

func newRotationFixture(t *testing.T, userActive bool, oldExpiresAt time.Time) rotationFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	repo := NewRefreshTokenRepository(db, clock.SystemClock{})
	ctx := testutil.TestContext()

	userID := uuid.New()
	user := &models.User{ID: userID, Email: "rotate-" + userID.String() + "@example.com", IsActive: userActive}
	require.NoError(t, NewUserRepository(db, clock.SystemClock{}).Create(ctx, user))
	f := rotationFixture{db: db, repo: repo, userID: userID}
	t.Cleanup(func() {
		f.deleteUser(t, userID)
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	})

	f.old = rotationToken(userID, oldExpiresAt)
	_, err := db.NamedExec(insertRefreshTokenQuery, f.old)
	require.NoError(t, err)
	return f
}

// deleteUser removes a test user and every refresh token of that user.
func (f rotationFixture) deleteUser(t *testing.T, userID uuid.UUID) {
	t.Helper()
	for _, query := range []string{"DELETE FROM refresh_tokens WHERE user_id = $1", "DELETE FROM users WHERE id = $1"} {
		if _, err := f.db.Exec(query, userID); err != nil {
			t.Logf("cleanup %q: %v", query, err)
		}
	}
}

func rotationToken(userID uuid.UUID, expiresAt time.Time) *models.RefreshToken {
	id := uuid.New()
	now := time.Now().UTC()
	return &models.RefreshToken{ID: id, UserID: userID, TokenHash: "rotate" + id.String(), IssuedAt: now, ExpiresAt: expiresAt, CreatedAt: now}
}

func (f rotationFixture) stored(t *testing.T, token *models.RefreshToken) *models.RefreshToken {
	t.Helper()
	got, err := f.repo.GetByHash(testutil.TestContext(), token.TokenHash)
	require.NoError(t, err)
	return got
}

func TestRefreshTokenRepository_Rotate_StoresTheSuccessorAndClaimsTheOldToken(t *testing.T) {
	f := newRotationFixture(t, true, time.Now().UTC().Add(testTokenTTL))
	successor := rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL))

	require.NoError(t, f.repo.Rotate(testutil.TestContext(), f.old.ID, successor))

	old := f.stored(t, f.old)
	require.NotNil(t, old)
	require.NotNil(t, old.ReplacedBy)
	assert.Equal(t, successor.ID, *old.ReplacedBy)
	stored := f.stored(t, successor)
	require.NotNil(t, stored)
	assert.True(t, stored.IsValid())
}

func TestRefreshTokenRepository_Rotate_RefusesAnUnclaimableOldToken(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	tests := []struct {
		name         string
		oldExpiresAt time.Time
		prepare      func(t *testing.T, f rotationFixture)
	}{
		{
			name:         "already rotated",
			oldExpiresAt: time.Now().UTC().Add(testTokenTTL),
			prepare: func(t *testing.T, f rotationFixture) {
				require.NoError(t, f.repo.Rotate(testutil.TestContext(), f.old.ID, rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL))))
			},
		},
		{
			name:         "revoked",
			oldExpiresAt: time.Now().UTC().Add(testTokenTTL),
			prepare: func(t *testing.T, f rotationFixture) {
				require.NoError(t, f.repo.RevokeByUserID(testutil.TestContext(), f.userID))
			},
		},
		{
			name:         "expired",
			oldExpiresAt: time.Now().UTC().Add(-time.Minute),
			prepare:      func(*testing.T, rotationFixture) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRotationFixture(t, true, tt.oldExpiresAt)
			tt.prepare(t, f)
			successor := rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL))

			err := f.repo.Rotate(testutil.TestContext(), f.old.ID, successor)

			require.ErrorIs(t, err, storage.ErrRefreshTokenNotClaimable)
			assert.Nil(t, f.stored(t, successor), "a refused rotation must not leave the successor behind")
		})
	}
}

func TestRefreshTokenRepository_Rotate_RefusesAnotherUsersToken(t *testing.T) {
	f := newRotationFixture(t, true, time.Now().UTC().Add(testTokenTTL))
	ctx := testutil.TestContext()
	otherID := uuid.New()
	require.NoError(t, NewUserRepository(f.db, clock.SystemClock{}).Create(ctx, &models.User{ID: otherID, Email: "rotate-" + otherID.String() + "@example.com", IsActive: true}))
	t.Cleanup(func() { f.deleteUser(t, otherID) })
	successor := rotationToken(otherID, time.Now().UTC().Add(testTokenTTL))

	err := f.repo.Rotate(ctx, f.old.ID, successor)

	require.ErrorIs(t, err, storage.ErrRefreshTokenNotClaimable)
	assert.Nil(t, f.stored(t, successor))
	assert.False(t, f.stored(t, f.old).IsReplaced())
}

func TestRefreshTokenRepository_RevokeByUserID_WaitsForAnInFlightRotation(t *testing.T) {
	f := newRotationFixture(t, true, time.Now().UTC().Add(testTokenTTL))
	ctx := testutil.TestContext()

	rotation, err := f.db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := rotation.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Logf("roll back rotation: %v", err)
		}
	})
	var isActive bool
	require.NoError(t, rotation.GetContext(ctx, &isActive, `SELECT is_active FROM users WHERE id = $1 FOR SHARE`, f.userID))
	successor := rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL))
	_, err = rotation.NamedExecContext(ctx, insertRefreshTokenQuery, successor)
	require.NoError(t, err)

	revoked := make(chan error, 1)
	go func() { revoked <- f.repo.RevokeByUserID(ctx, f.userID) }()
	select {
	case err := <-revoked:
		t.Fatalf("revocation finished while a rotation of the user was in flight: %v", err)
	case <-time.After(testRevokeNotBlockedWithin):
	}

	require.NoError(t, rotation.Commit())
	require.NoError(t, <-revoked)
	assert.True(t, f.stored(t, successor).IsRevoked(), "the revocation must cover the token the rotation minted")
}

func TestRefreshTokenRepository_Rotate_ConcurrentClaimsLetExactlyOneWin(t *testing.T) {
	const contenders = 8
	f := newRotationFixture(t, true, time.Now().UTC().Add(testTokenTTL))

	errs := make([]error, contenders)
	var wg sync.WaitGroup
	for i := range contenders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = f.repo.Rotate(testutil.TestContext(), f.old.ID, rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL)))
		}(i)
	}
	wg.Wait()

	won := 0
	for _, err := range errs {
		if err == nil {
			won++
			continue
		}
		require.ErrorIs(t, err, storage.ErrRefreshTokenNotClaimable)
	}
	assert.Equal(t, 1, won)
	var successors int
	require.NoError(t, f.db.Get(&successors, "SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND id <> $2", f.userID, f.old.ID))
	assert.Equal(t, 1, successors)
}

func TestRefreshTokenRepository_Rotate_RefusesAnInactiveUser(t *testing.T) {
	f := newRotationFixture(t, false, time.Now().UTC().Add(testTokenTTL))
	successor := rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL))

	err := f.repo.Rotate(testutil.TestContext(), f.old.ID, successor)

	require.ErrorIs(t, err, storage.ErrUserInactive)
	assert.Nil(t, f.stored(t, successor))
	old := f.stored(t, f.old)
	require.NotNil(t, old)
	assert.False(t, old.IsReplaced())
}

func TestRefreshTokenRepository_RevokeByUserID_RevokesARotatedSuccessor(t *testing.T) {
	f := newRotationFixture(t, true, time.Now().UTC().Add(testTokenTTL))
	successor := rotationToken(f.userID, time.Now().UTC().Add(testTokenTTL))
	require.NoError(t, f.repo.Rotate(testutil.TestContext(), f.old.ID, successor))

	require.NoError(t, f.repo.RevokeByUserID(testutil.TestContext(), f.userID))

	stored := f.stored(t, successor)
	require.NotNil(t, stored)
	assert.True(t, stored.IsRevoked())
}

func TestRefreshTokenRepository_RevokeByUserID_UnknownUserIsNoError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := SetupTestDB(t)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	})

	require.NoError(t, NewRefreshTokenRepository(db, clock.SystemClock{}).RevokeByUserID(testutil.TestContext(), uuid.New()))
}
