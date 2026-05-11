package postgres

import (
	"errors"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupOrgTestDB connects to the test database using testcontainers
func setupOrgTestDB(t *testing.T) *sqlx.DB {
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	return db
}

// setupOrgTestData inserts test organizations and members for testing
func setupOrgTestData(t *testing.T, db *sqlx.DB) (tenantID int64, orgID uuid.UUID, userID uuid.UUID) {
	orgID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	userID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// Create test tenant using shared helper
	tenantID = 100
	createTestTenant(t, db, tenantID, "Test Tenant")

	// Clean up any existing test data for this org UUID and user UUID
	_, _ = db.Exec("DELETE FROM organization_members WHERE org_id = $1", orgID)
	_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
	_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)

	// Insert test user first (FK constraint from organization_members)
	var err error
	_, err = db.Exec(`
		INSERT INTO users (id, email, email_verified, is_admin, is_active, created_at, updated_at)
		VALUES ($1, 'test@example.com', true, false, true, NOW(), NOW())
	`, userID)
	require.NoError(t, err)

	// Insert test organization
	_, err = db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, 'Test Organization', 'active', NOW(), NOW())
	`, orgID, tenantID)
	require.NoError(t, err)

	// Insert test member
	_, err = db.Exec(`
		INSERT INTO organization_members (org_id, user_id, role, status, created_at, updated_at)
		VALUES ($1, $2, 'member', 'active', NOW(), NOW())
	`, orgID, userID)
	require.NoError(t, err)

	return tenantID, orgID, userID
}

// TestGetTenantByOrgID_Success verifies UUID→tenant resolution
func TestGetTenantByOrgID_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	tenantID, orgID, _ := setupOrgTestData(t, db)
	defer func() {
		if _, err := db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID); err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
	}()

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	// Test: Resolve org UUID to tenant ID
	resolvedTenantID, err := repo.GetTenantByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, tenantID, resolvedTenantID)
}

// TestGetTenantByOrgID_NotFound verifies error for invalid UUID
func TestGetTenantByOrgID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	// Test: Non-existent org UUID should return error
	nonExistentOrgID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	_, err := repo.GetTenantByOrgID(ctx, nonExistentOrgID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetOrgByTenantID_MultipleOrgs verifies deterministic ordering
func TestGetOrgByTenantID_MultipleOrgs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	tenantID := int64(2)
	createTestTenant(t, db, tenantID, "Tenant 2")

	org1ID := uuid.MustParse("22222222-2222-2222-2222-222222222221")
	org2ID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// Clean up
	defer func() {
		if _, err := db.Exec("DELETE FROM organizations WHERE org_id IN ($1, $2)", org1ID, org2ID); err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
	}()

	// Insert two orgs for same tenant (org1 created first)
	_, err := db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, 'First Org', 'active', NOW() - INTERVAL '1 hour', NOW())
	`, org1ID, tenantID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, 'Second Org', 'active', NOW(), NOW())
	`, org2ID, tenantID)
	require.NoError(t, err)

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	// Test: Should return first org (oldest created_at)
	org, err := repo.GetOrgByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, org1ID, org.OrgID)
	assert.Equal(t, "First Org", org.Name)
}

// TestCheckUserMembership_ValidMember verifies active membership check
func TestCheckUserMembership_ValidMember(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	_, orgID, userID := setupOrgTestData(t, db)
	defer func() {
		if _, err := db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID); err != nil {
			t.Fatalf("cleanup failed: %v", err)
		}
	}()

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	// Test: Active member should return true
	isMember, err := repo.CheckUserMembership(ctx, orgID, userID)
	require.NoError(t, err)
	assert.True(t, isMember)
}

// TestCheckUserMembership_RemovedMember verifies removed status returns false
func TestCheckUserMembership_RemovedMember(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	_, orgID, userID := setupOrgTestData(t, db)
	defer func() {
		if _, err := db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	}()

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	// Update member status to 'removed'
	_, err := db.Exec(`
		UPDATE organization_members
		SET status = 'removed'
		WHERE org_id = $1 AND user_id = $2
	`, orgID, userID)
	require.NoError(t, err)

	// Test: Removed member should return false
	isMember, err := repo.CheckUserMembership(ctx, orgID, userID)
	require.NoError(t, err)
	assert.False(t, isMember)
}

// TestUpsertOrg_CreateAndUpdate tests both create and update paths
func TestUpsertOrg_CreateAndUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	tenantID := int64(3)
	createTestTenant(t, db, tenantID, "Tenant 3")

	newOrgID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	defer func() {
		if _, err := db.Exec("DELETE FROM organizations WHERE org_id = $1", newOrgID); err != nil {
			t.Logf("cleanup failed: %v", err)
		}
	}()

	// Test CREATE path: Upsert non-existent org
	newOrg := &models.Organization{
		OrgID:    newOrgID,
		TenantID: tenantID,
		Name:     "New Kilo Cloud Org",
		State:    "active",
	}

	err := repo.UpsertOrg(ctx, newOrg)
	require.NoError(t, err)

	// Verify org was created
	created, err := repo.GetByID(ctx, newOrgID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "New Kilo Cloud Org", created.Name)
	assert.Equal(t, "active", created.State)
	assert.Equal(t, tenantID, created.TenantID)

	// Test UPDATE path: Upsert existing org with changed name
	newOrg.Name = "Updated Kilo Cloud Org"
	newOrg.State = "suspended"
	newOrg.TenantID = 999 // Try to change immutable field

	err = repo.UpsertOrg(ctx, newOrg)
	require.NoError(t, err)

	// Verify org was updated but tenant_id preserved
	updated, err := repo.GetByID(ctx, newOrgID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Kilo Cloud Org", updated.Name)
	assert.Equal(t, "suspended", updated.State)
	assert.Equal(t, tenantID, updated.TenantID, "tenant_id should be immutable and preserved from existing record")
}

// ============================================================================
// ListOrgMembersWithEmail Integration Tests
// ============================================================================

// setupOrgMembersTestData creates an org with multiple members of varied statuses for pagination tests.
// Returns orgID plus three userIDs (active1, active2, invited).
func setupOrgMembersTestData(t *testing.T, db *sqlx.DB) (orgID uuid.UUID, userIDs [3]uuid.UUID) {
	t.Helper()

	tenantID := int64(200)
	createTestTenant(t, db, tenantID, "Pagination Tenant")

	orgID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userIDs = [3]uuid.UUID{
		uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb01"),
		uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb02"),
		uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbb03"),
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM organization_members WHERE org_id = $1", orgID)
	_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
	for _, uid := range userIDs {
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", uid)
	}

	// Insert users
	for i, uid := range userIDs {
		_, err := db.Exec(`
			INSERT INTO users (id, email, email_verified, is_admin, is_active, created_at, updated_at)
			VALUES ($1, $2, true, false, true, NOW(), NOW())
		`, uid, "member"+string(rune('A'+i))+"@example.com")
		require.NoError(t, err)
	}

	// Insert organization
	_, err := db.Exec(`
		INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, 'Pagination Org', 'active', NOW(), NOW())
	`, orgID, tenantID)
	require.NoError(t, err)

	// Insert members: 2 active, 1 invited
	statuses := []string{models.OrganizationMemberStatusActive, models.OrganizationMemberStatusActive, "invited"}
	for i, uid := range userIDs {
		_, err := db.Exec(`
			INSERT INTO organization_members (org_id, user_id, role, status, created_at, updated_at)
			VALUES ($1, $2, 'member', $3, NOW(), NOW())
		`, orgID, uid, statuses[i])
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM organization_members WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
		for _, uid := range userIDs {
			_, _ = db.Exec("DELETE FROM users WHERE id = $1", uid)
		}
	})

	return orgID, userIDs
}

func TestListOrgMembersWithEmail_NoStatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	orgID, _ := setupOrgMembersTestData(t, db)

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	members, totalCount, err := repo.ListOrgMembersWithEmail(ctx, orgID, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), totalCount, "empty status should return all members")
	assert.Len(t, members, 3)
	// Verify emails populated
	for _, m := range members {
		assert.NotEmpty(t, m.Email, "email should be populated via JOIN")
	}
}

func TestListOrgMembersWithEmail_ActiveFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	orgID, _ := setupOrgMembersTestData(t, db)

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	members, totalCount, err := repo.ListOrgMembersWithEmail(ctx, orgID, "active", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), totalCount, "active filter should return 2 members")
	assert.Len(t, members, 2)
	for _, m := range members {
		assert.Equal(t, models.OrganizationMemberStatusActive, m.Status)
	}
}

func TestListOrgMembersWithEmail_InactiveFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	orgID, _ := setupOrgMembersTestData(t, db)

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	members, totalCount, err := repo.ListOrgMembersWithEmail(ctx, orgID, "inactive", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), totalCount, "inactive filter should return non-active members")
	assert.Len(t, members, 1)
	assert.NotEqual(t, models.OrganizationMemberStatusActive, members[0].Status)
}

func TestListOrgMembersWithEmail_LimitOffset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	orgID, _ := setupOrgMembersTestData(t, db)

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	// First page: limit=2, offset=0
	page1, total1, err := repo.ListOrgMembersWithEmail(ctx, orgID, "", 2, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total1)
	assert.Len(t, page1, 2)

	// Second page: limit=2, offset=2
	page2, total2, err := repo.ListOrgMembersWithEmail(ctx, orgID, "", 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total2)
	assert.Len(t, page2, 1)
}

func TestListOrgMembersWithEmail_TotalCountConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	orgID, _ := setupOrgMembersTestData(t, db)

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	// totalCount should remain stable across different pages
	_, total1, err := repo.ListOrgMembersWithEmail(ctx, orgID, "", 1, 0)
	require.NoError(t, err)

	_, total2, err := repo.ListOrgMembersWithEmail(ctx, orgID, "", 1, 1)
	require.NoError(t, err)

	_, total3, err := repo.ListOrgMembersWithEmail(ctx, orgID, "", 1, 2)
	require.NoError(t, err)

	assert.Equal(t, total1, total2, "totalCount should be stable across pages")
	assert.Equal(t, total2, total3, "totalCount should be stable across pages")
}

func TestGetOrgMemberWithEmail_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupOrgTestDB(t)
	orgID, _ := setupOrgMembersTestData(t, db)

	repo := NewOrganizationRepository(db)
	ctx := testutil.TestContext()

	nonExistentUser := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	_, err := repo.GetOrgMemberWithEmail(ctx, orgID, nonExistentUser)
	require.Error(t, err)
	assert.True(t, errors.Is(err, storage.ErrNotFound), "expected storage.ErrNotFound for non-existent member")
}
