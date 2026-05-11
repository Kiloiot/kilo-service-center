package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// apiKeyTestParams holds configurable fields for inserting a test API key.
type apiKeyTestParams struct {
	TenantID int64
	OrgID    uuid.UUID
	UserID   *uuid.UUID
	Name     string
	KeyType  string
	IsActive bool
}

// insertTestAPIKey inserts a test API key with the given parameters.
// Returns the inserted key's UUID.
func insertTestAPIKey(t *testing.T, db *sqlx.DB, p apiKeyTestParams) uuid.UUID {
	t.Helper()

	id := uuid.New()
	rawKey := fmt.Sprintf("kc_test_%s", id)
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])
	keyPrefix := rawKey[:8]

	_, err := db.Exec(`
		INSERT INTO api_keys (id, tenant_id, org_id, user_id, name, key_hash, key_prefix, key_type, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, id, p.TenantID, p.OrgID, p.UserID, p.Name, keyHash, keyPrefix, p.KeyType, p.IsActive, time.Now().UTC())
	require.NoError(t, err, "insert test API key %q", p.Name)

	return id
}

// setupAPIKeyTestData creates two orgs (A and B) within the same tenant,
// a real user in the users table, and returns the IDs for use in tests.
func setupAPIKeyTestData(t *testing.T, db *sqlx.DB) (tenantID int64, orgA, orgB uuid.UUID, userID uuid.UUID) {
	t.Helper()

	tenantID = 300
	createTestTenant(t, db, tenantID, "API Key Test Tenant")

	orgA = uuid.MustParse("11111111-2222-3333-4444-555555555555")
	orgB = uuid.MustParse("66666666-7777-8888-9999-aaaaaaaaaaaa")

	_, err := db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, can_have_base_stations, created_at, updated_at)
		VALUES ($1, $2, 'Org A', true, NOW(), NOW())
		ON CONFLICT (org_id) DO NOTHING
	`, orgA, tenantID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, can_have_base_stations, created_at, updated_at)
		VALUES ($1, $2, 'Org B', true, NOW(), NOW())
		ON CONFLICT (org_id) DO NOTHING
	`, orgB, tenantID)
	require.NoError(t, err)

	userID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	_, err = db.Exec(`
		INSERT INTO users (id, email, email_verified, is_admin, is_active, created_at, updated_at)
		VALUES ($1, 'apikey-test@example.com', true, false, true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, userID)
	require.NoError(t, err)

	return tenantID, orgA, orgB, userID
}

func TestAPIKeyRepository_List_AdminView(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, _, userID := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	// Insert keys: 2 user keys + 1 service account key
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "user-key-1", KeyType: models.KeyTypeUser, IsActive: true,
	})
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "user-key-2", KeyType: models.KeyTypeUser, IsActive: true,
	})
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: nil,
		Name: "sa-key-1", KeyType: models.KeyTypeServiceAccount, IsActive: true,
	})

	repo := NewAPIKeyRepository(db)

	// Admin view: nil userID returns all keys for org
	keys, err := repo.List(ctx, tenantID, orgA, nil, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, len(keys), "admin view should return all 3 keys")
}

func TestAPIKeyRepository_List_UserScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, _, userID := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	otherUserID := uuid.MustParse("bbbbbbbb-cccc-dddd-eeee-ffffffffffff")
	_, err := db.Exec(`
		INSERT INTO users (id, email, email_verified, is_admin, is_active, created_at, updated_at)
		VALUES ($1, 'other-user@example.com', true, false, true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, otherUserID)
	require.NoError(t, err)

	// Insert: 1 key for our user, 1 key for another user, 1 SA key
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "my-key", KeyType: models.KeyTypeUser, IsActive: true,
	})
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &otherUserID,
		Name: "other-key", KeyType: models.KeyTypeUser, IsActive: true,
	})
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: nil,
		Name: "sa-key", KeyType: models.KeyTypeServiceAccount, IsActive: true,
	})

	repo := NewAPIKeyRepository(db)

	// User-scoped: returns user's keys + SA keys (but NOT other user's keys)
	keys, err := repo.List(ctx, tenantID, orgA, &userID, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, len(keys), "user-scoped view should return own key + SA key")

	for _, k := range keys {
		isOwnOrSA := (k.UserID != nil && *k.UserID == userID) || k.UserID == nil
		assert.True(t, isOwnOrSA, "user-scoped list should only contain own keys or SA keys")
	}
}

func TestAPIKeyRepository_List_CrossOrgIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, orgB, userID := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	// Insert key in org A
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "orgA-key", KeyType: models.KeyTypeUser, IsActive: true,
	})

	// Insert key in org B
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgB, UserID: &userID,
		Name: "orgB-key", KeyType: models.KeyTypeUser, IsActive: true,
	})

	repo := NewAPIKeyRepository(db)

	// Query org A — should NOT see org B's keys
	keysA, err := repo.List(ctx, tenantID, orgA, nil, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, len(keysA), "org A should only see its own keys")
	assert.Equal(t, "orgA-key", keysA[0].Name)

	keysB, err := repo.List(ctx, tenantID, orgB, nil, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, len(keysB), "org B should only see its own keys")
	assert.Equal(t, "orgB-key", keysB[0].Name)
}

func TestAPIKeyRepository_List_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, _, _ := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	// Insert 5 SA keys
	for i := 0; i < 5; i++ {
		insertTestAPIKey(t, db, apiKeyTestParams{
			TenantID: tenantID, OrgID: orgA, UserID: nil,
			Name: fmt.Sprintf("paginate-key-%d", i), KeyType: models.KeyTypeServiceAccount, IsActive: true,
		})
	}

	repo := NewAPIKeyRepository(db)

	// Page 1: limit=2, offset=0
	page1, err := repo.List(ctx, tenantID, orgA, nil, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, len(page1), "first page should have 2 keys")

	// Page 2: limit=2, offset=2
	page2, err := repo.List(ctx, tenantID, orgA, nil, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(page2), "second page should have 2 keys")

	// Page 3: limit=2, offset=4
	page3, err := repo.List(ctx, tenantID, orgA, nil, 2, 4)
	require.NoError(t, err)
	assert.Equal(t, 1, len(page3), "third page should have 1 key")

	// Verify no overlap between pages
	allIDs := make(map[uuid.UUID]bool)
	for _, keys := range [][]*models.APIKey{page1, page2, page3} {
		for _, k := range keys {
			assert.False(t, allIDs[k.ID], "key %s should not appear in multiple pages", k.ID)
			allIDs[k.ID] = true
		}
	}
	assert.Equal(t, 5, len(allIDs), "all 5 keys should appear across pages")
}

func TestAPIKeyRepository_Count_AdminView(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, _, userID := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "count-user-key", KeyType: models.KeyTypeUser, IsActive: true,
	})
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: nil,
		Name: "count-sa-key", KeyType: models.KeyTypeServiceAccount, IsActive: true,
	})

	repo := NewAPIKeyRepository(db)

	count, err := repo.Count(ctx, tenantID, orgA, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "admin count should match total keys")
}

func TestAPIKeyRepository_Count_UserScoped(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, _, userID := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	otherUserID := uuid.MustParse("cccccccc-dddd-eeee-ffff-111111111111")
	_, err := db.Exec(`
		INSERT INTO users (id, email, email_verified, is_admin, is_active, created_at, updated_at)
		VALUES ($1, 'count-other@example.com', true, false, true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, otherUserID)
	require.NoError(t, err)

	// 1 key for our user, 1 key for other user, 1 SA key
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "my-counted-key", KeyType: models.KeyTypeUser, IsActive: true,
	})
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &otherUserID,
		Name: "other-counted-key", KeyType: models.KeyTypeUser, IsActive: true,
	})
	insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: nil,
		Name: "sa-counted-key", KeyType: models.KeyTypeServiceAccount, IsActive: true,
	})

	repo := NewAPIKeyRepository(db)

	count, err := repo.Count(ctx, tenantID, orgA, &userID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "user-scoped count should include own key + SA key")
}

func TestAPIKeyRepository_GetByIDAndOrg_WrongOrg(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, orgB, userID := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	keyID := insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "wrong-org-key", KeyType: models.KeyTypeUser, IsActive: true,
	})

	repo := NewAPIKeyRepository(db)

	// Should succeed with correct org
	key, err := repo.GetByIDAndOrg(ctx, keyID, orgA)
	require.NoError(t, err)
	assert.Equal(t, keyID, key.ID)

	// Should fail with wrong org
	_, err = repo.GetByIDAndOrg(ctx, keyID, orgB)
	require.Error(t, err)
	assert.True(t, errors.Is(err, interfaces.ErrRecordNotFound), "wrong org should return ErrRecordNotFound")
}

func TestAPIKeyRepository_DeleteByIDAndOrg_WrongOrg(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := SetupPostgresContainer(t)
	defer cleanup()

	tenantID, orgA, orgB, userID := setupAPIKeyTestData(t, db)
	ctx := testutil.TestContext()

	keyID := insertTestAPIKey(t, db, apiKeyTestParams{
		TenantID: tenantID, OrgID: orgA, UserID: &userID,
		Name: "delete-wrong-org-key", KeyType: models.KeyTypeUser, IsActive: true,
	})

	repo := NewAPIKeyRepository(db)

	// Should fail with wrong org
	err := repo.DeleteByIDAndOrg(ctx, keyID, orgB)
	require.Error(t, err)
	assert.True(t, errors.Is(err, interfaces.ErrRecordNotFound), "wrong org should return ErrRecordNotFound")

	// Key should still exist
	key, err := repo.GetByIDAndOrg(ctx, keyID, orgA)
	require.NoError(t, err)
	assert.Equal(t, keyID, key.ID, "key should still exist after failed cross-org delete")

	// Should succeed with correct org
	err = repo.DeleteByIDAndOrg(ctx, keyID, orgA)
	require.NoError(t, err)

	// Key should be gone
	_, err = repo.GetByIDAndOrg(ctx, keyID, orgA)
	require.Error(t, err)
	assert.True(t, errors.Is(err, interfaces.ErrRecordNotFound))
}
