package postgres

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserRepository_Create_Success tests successful user creation
func TestUserRepository_Create_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test-create@example.com",
		EmailVerified: true,
		IsAdmin:       false,
		IsActive:      true,
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Test: Create user
	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

// TestUserRepository_GetByID_Success tests successful user retrieval by ID
func TestUserRepository_GetByID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test-getbyid@example.com",
		EmailVerified: true,
		IsAdmin:       true,
		IsActive:      true,
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Create user first
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Test: Get user by ID
	retrieved, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, retrieved.ID)
	assert.Equal(t, "test-getbyid@example.com", retrieved.Email)
	assert.True(t, retrieved.IsAdmin)
}

// TestUserRepository_GetByID_NotFound tests user not found scenario
func TestUserRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	// Test: Get non-existent user
	nonExistentID := uuid.New()
	_, err := repo.GetByID(ctx, nonExistentID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestUserRepository_GetByEmail_Success tests successful user retrieval by email
func TestUserRepository_GetByEmail_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	userID := uuid.New()
	email := "test-getbyemail@example.com"
	user := &models.User{
		ID:            userID,
		Email:         email,
		EmailVerified: false,
		IsAdmin:       false,
		IsActive:      true,
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Create user first
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Test: Get user by email
	retrieved, err := repo.GetByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, userID, retrieved.ID)
	assert.Equal(t, email, retrieved.Email)
}

// TestUserRepository_Update_Success tests successful user update
func TestUserRepository_Update_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test-update@example.com",
		EmailVerified: false,
		IsAdmin:       false,
		IsActive:      true,
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Create user first
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Update user
	user.EmailVerified = true
	user.IsAdmin = true
	user.Email = "test-update-new@example.com"

	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	assert.True(t, retrieved.EmailVerified)
	assert.True(t, retrieved.IsAdmin)
	assert.Equal(t, "test-update-new@example.com", retrieved.Email)
}

// TestUserRepository_SetPasswordHash_Success tests password hash update
func TestUserRepository_SetPasswordHash_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test-setpassword@example.com",
		EmailVerified: true,
		IsAdmin:       false,
		IsActive:      true,
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	// Create user first
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Set password hash (PHC format example)
	passwordHash := "$pbkdf2-sha512$i=100000$somesaltvalue$somehashvalue"
	err = repo.SetPasswordHash(ctx, userID, passwordHash)
	require.NoError(t, err)

	// Verify password was set
	retrieved, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, retrieved.PasswordHash)
	assert.Equal(t, passwordHash, *retrieved.PasswordHash)
}

// TestUserRepository_Delete_Success tests successful user deletion
func TestUserRepository_Delete_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test-delete@example.com",
		EmailVerified: true,
		IsAdmin:       false,
		IsActive:      true,
	}

	// Create user first
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Test: Delete user
	err = repo.Delete(ctx, userID)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestUserRepository_List_Success tests user listing with pagination
func TestUserRepository_List_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	// Create multiple test users
	userIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		userIDs[i] = uuid.New()
		user := &models.User{
			ID:            userIDs[i],
			Email:         "test-list-" + userIDs[i].String()[:8] + "@example.com",
			EmailVerified: true,
			IsAdmin:       false,
			IsActive:      true,
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	// Cleanup after test
	defer func() {
		for _, id := range userIDs {
			_, _ = db.Exec("DELETE FROM users WHERE id = $1", id)
		}
	}()

	// Test: List users with pagination
	users, err := repo.List(ctx, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 3)
}

// TestUserRepository_Count_Success tests user count
func TestUserRepository_Count_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewUserRepository(db)
	ctx := testutil.TestContext()

	// Get initial count
	initialCount, err := repo.Count(ctx)
	require.NoError(t, err)

	// Create a test user
	userID := uuid.New()
	user := &models.User{
		ID:            userID,
		Email:         "test-count@example.com",
		EmailVerified: true,
		IsAdmin:       false,
		IsActive:      true,
	}

	// Cleanup after test
	defer func() {
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	err = repo.Create(ctx, user)
	require.NoError(t, err)

	// Test: Count should increase by 1
	newCount, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, initialCount+1, newCount)
}
