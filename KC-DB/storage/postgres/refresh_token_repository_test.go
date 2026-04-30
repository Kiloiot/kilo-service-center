package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefreshTokenRepository_Create_Success tests successful token creation
func TestRefreshTokenRepository_Create_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db)
	userRepo := NewUserRepository(db)
	ctx := testutil.TestContext()

	// Create a test user first (FK constraint)
	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "refresh-create@example.com",
		IsActive: true,
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	tokenID := uuid.New()
	tokenHash := "testhash" + tokenID.String()[:8]
	token := &models.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: tokenHash,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", tokenID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Test: Create token
	err = repo.Create(ctx, token)
	require.NoError(t, err)
	assert.False(t, token.CreatedAt.IsZero())
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

	repo := NewRefreshTokenRepository(db)
	userRepo := NewUserRepository(db)
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
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", tokenID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Create token first
	err = repo.Create(ctx, token)
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

	repo := NewRefreshTokenRepository(db)
	ctx := testutil.TestContext()

	// Test: Get non-existent token
	token, err := repo.GetByHash(ctx, "nonexistent-hash")
	require.NoError(t, err)
	assert.Nil(t, token) // Not found returns nil, not error
}

// TestRefreshTokenRepository_RevokeByHash_Success tests successful token revocation
func TestRefreshTokenRepository_RevokeByHash_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db)
	userRepo := NewUserRepository(db)
	ctx := testutil.TestContext()

	// Create a test user first
	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "refresh-revoke@example.com",
		IsActive: true,
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	tokenID := uuid.New()
	tokenHash := "revokehash" + tokenID.String()[:8]
	token := &models.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: tokenHash,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", tokenID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Create token first
	err = repo.Create(ctx, token)
	require.NoError(t, err)

	// Test: Revoke token
	err = repo.RevokeByHash(ctx, tokenHash)
	require.NoError(t, err)

	// Verify revocation
	retrieved, err := repo.GetByHash(ctx, tokenHash)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.True(t, retrieved.IsRevoked())
}

// TestRefreshTokenRepository_RevokeByHash_NotFound tests revoke of non-existent token
func TestRefreshTokenRepository_RevokeByHash_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db)
	ctx := testutil.TestContext()

	// Test: Revoke non-existent token
	err := repo.RevokeByHash(ctx, "nonexistent-hash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found or already revoked")
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

	repo := NewRefreshTokenRepository(db)
	userRepo := NewUserRepository(db)
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
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		}
		err := repo.Create(ctx, token)
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

// TestRefreshTokenRepository_MarkReplaced_Success tests token replacement tracking
func TestRefreshTokenRepository_MarkReplaced_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db)
	userRepo := NewUserRepository(db)
	ctx := testutil.TestContext()

	// Create a test user first
	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "refresh-markreplaced@example.com",
		IsActive: true,
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Create old token
	oldTokenID := uuid.New()
	oldToken := &models.RefreshToken{
		ID:        oldTokenID,
		UserID:    userID,
		TokenHash: "oldhash" + oldTokenID.String()[:8],
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	err = repo.Create(ctx, oldToken)
	require.NoError(t, err)

	// Create new token
	newTokenID := uuid.New()
	newToken := &models.RefreshToken{
		ID:        newTokenID,
		UserID:    userID,
		TokenHash: "newhash" + newTokenID.String()[:8],
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	err = repo.Create(ctx, newToken)
	require.NoError(t, err)

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", oldTokenID)
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", newTokenID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Test: Mark old token as replaced by new token
	err = repo.MarkReplaced(ctx, oldTokenID, newTokenID)
	require.NoError(t, err)

	// Verify replacement is tracked
	retrieved, err := repo.GetByHash(ctx, oldToken.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.True(t, retrieved.IsReplaced())
	assert.Equal(t, newTokenID, *retrieved.ReplacedBy)
}

// TestRefreshTokenRepository_MarkReplaced_NotFound tests marking non-existent token
func TestRefreshTokenRepository_MarkReplaced_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db)
	ctx := testutil.TestContext()

	// Test: Mark non-existent token
	oldID := uuid.New()
	newID := uuid.New()
	err := repo.MarkReplaced(ctx, oldID, newID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestRefreshTokenRepository_DeleteExpired_Success tests expired token cleanup
func TestRefreshTokenRepository_DeleteExpired_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRefreshTokenRepository(db)
	userRepo := NewUserRepository(db)
	ctx := testutil.TestContext()

	// Create a test user first
	userID := uuid.New()
	user := &models.User{
		ID:       userID,
		Email:    "refresh-deleteexpired@example.com",
		IsActive: true,
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Create an expired token
	expiredTokenID := uuid.New()
	expiredToken := &models.RefreshToken{
		ID:        expiredTokenID,
		UserID:    userID,
		TokenHash: "expiredhash" + expiredTokenID.String()[:8],
		IssuedAt:  time.Now().UTC().Add(-48 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-24 * time.Hour), // Expired
	}
	err = repo.Create(ctx, expiredToken)
	require.NoError(t, err)

	// Create a valid token
	validTokenID := uuid.New()
	validToken := &models.RefreshToken{
		ID:        validTokenID,
		UserID:    userID,
		TokenHash: "validhash" + validTokenID.String()[:8],
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour), // Not expired
	}
	err = repo.Create(ctx, validToken)
	require.NoError(t, err)

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", expiredTokenID)
		_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", validTokenID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Test: Delete expired tokens
	deleted, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, deleted, int64(1)) // At least our expired token

	// Verify expired token is gone
	expiredRetrieved, err := repo.GetByHash(ctx, expiredToken.TokenHash)
	require.NoError(t, err)
	assert.Nil(t, expiredRetrieved)

	// Verify valid token still exists
	validRetrieved, err := repo.GetByHash(ctx, validToken.TokenHash)
	require.NoError(t, err)
	assert.NotNil(t, validRetrieved)
}
