package postgres

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAccount_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRegistrationRepository(db)
	ctx := testutil.TestContext()

	userID := uuid.New()
	firstName := "John"
	lastName := "Doe"
	companyName := "Acme Corp"
	passwordHash := "hashed-password-123"

	user := &models.User{
		ID:           userID,
		Email:        "register-test@example.com",
		PasswordHash: &passwordHash,
		IsAdmin:      false,
		IsActive:     true,
		FirstName:    &firstName,
		LastName:     &lastName,
		CompanyName:  &companyName,
	}

	defer func() {
		_, _ = db.Exec("DELETE FROM organization_members WHERE user_id = $1", userID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", userID)
	}()

	result, err := repo.RegisterAccount(ctx, &interfaces.RegistrationParams{
		User:        user,
		CompanyName: companyName,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify user
	assert.Equal(t, userID, result.User.ID)
	assert.False(t, result.User.IsAdmin, "registered user must not be server admin")
	assert.False(t, result.User.CreatedAt.IsZero())
	assert.False(t, result.User.UpdatedAt.IsZero())

	// Verify organization
	require.NotNil(t, result.Organization)
	assert.Equal(t, companyName, result.Organization.Name)
	assert.Equal(t, models.OrganizationStateActive, result.Organization.State)
	assert.False(t, result.Organization.CreatedAt.IsZero())

	// Verify tenant
	assert.Greater(t, result.TenantID, int64(0))
	assert.Equal(t, result.TenantID, result.Organization.TenantID)

	// Verify membership exists with owner role and admin permissions
	var role string
	var isOrgAdmin, isBSAdmin, isEPAdmin bool
	err = db.QueryRow(
		"SELECT role, is_org_admin, is_base_station_admin, is_endpoint_admin FROM organization_members WHERE org_id = $1 AND user_id = $2",
		result.Organization.OrgID, userID,
	).Scan(&role, &isOrgAdmin, &isBSAdmin, &isEPAdmin)
	require.NoError(t, err)
	assert.Equal(t, models.OrganizationRoleOwner, role)
	assert.True(t, isOrgAdmin, "registered user must be org admin")
	assert.True(t, isBSAdmin, "registered user must be base station admin")
	assert.True(t, isEPAdmin, "registered user must be endpoint admin")

	// Cleanup org and tenant created by registration
	_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", result.Organization.OrgID)
	_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", result.TenantID)
}

func TestRegisterAccount_DuplicateEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRegistrationRepository(db)
	ctx := testutil.TestContext()

	email := "duplicate-reg@example.com"
	passwordHash := "hashed-password"

	// First registration
	user1ID := uuid.New()
	firstName := "First"
	lastName := "User"
	company := "Company A"
	user1 := &models.User{
		ID:           user1ID,
		Email:        email,
		PasswordHash: &passwordHash,
		IsAdmin:      false,
		IsActive:     true,
		FirstName:    &firstName,
		LastName:     &lastName,
		CompanyName:  &company,
	}

	result1, err := repo.RegisterAccount(ctx, &interfaces.RegistrationParams{
		User:        user1,
		CompanyName: company,
	})
	require.NoError(t, err)

	defer func() {
		_, _ = db.Exec("DELETE FROM organization_members WHERE user_id = $1", user1ID)
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", result1.Organization.OrgID)
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", user1ID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", result1.TenantID)
	}()

	// Second registration with same email must fail
	user2ID := uuid.New()
	user2 := &models.User{
		ID:           user2ID,
		Email:        email,
		PasswordHash: &passwordHash,
		IsAdmin:      false,
		IsActive:     true,
		FirstName:    &firstName,
		LastName:     &lastName,
		CompanyName:  &company,
	}

	result2, err := repo.RegisterAccount(ctx, &interfaces.RegistrationParams{
		User:        user2,
		CompanyName: company,
	})

	require.Error(t, err, "duplicate email must fail")
	assert.Nil(t, result2)
	assert.Contains(t, err.Error(), "insert user")

	// Verify no partial state: user2 should not exist
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users WHERE id = $1", user2ID).Scan(&count)
	assert.Equal(t, 0, count, "rollback must prevent orphaned user")
}

func TestRegisterCEAccount_FirstUser_BecomesOwner(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRegistrationRepository(db)
	ctx := testutil.TestContext()

	// Create a fresh CE org for this test
	orgID := uuid.New()
	var tenantID int64
	err := db.QueryRow("INSERT INTO tenants (name) VALUES ($1) RETURNING id", "ce-test-tenant").Scan(&tenantID)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO organizations (org_id, tenant_id, name, state) VALUES ($1, $2, $3, $4)",
		orgID, tenantID, "CE Test Org", models.OrganizationStateActive)
	require.NoError(t, err)

	defer func() {
		_, _ = db.Exec("DELETE FROM organization_members WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM users WHERE email LIKE 'ce-owner-test%'")
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	userID := uuid.New()
	firstName := "First"
	lastName := "Owner"
	passwordHash := "hashed-pw"
	user := &models.User{
		ID:           userID,
		Email:        "ce-owner-test@example.com",
		PasswordHash: &passwordHash,
		IsAdmin:      false,
		IsActive:     true,
		FirstName:    &firstName,
		LastName:     &lastName,
	}

	result, err := repo.RegisterCEAccount(ctx, &interfaces.CERegistrationParams{
		User:     user,
		OrgID:    orgID,
		TenantID: tenantID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// First user must be owner with full admin rights
	var role string
	var isOrgAdmin, isBSAdmin, isEPAdmin bool
	err = db.QueryRow(
		"SELECT role, is_org_admin, is_base_station_admin, is_endpoint_admin FROM organization_members WHERE org_id = $1 AND user_id = $2",
		orgID, userID,
	).Scan(&role, &isOrgAdmin, &isBSAdmin, &isEPAdmin)
	require.NoError(t, err)
	assert.Equal(t, models.OrganizationRoleOwner, role, "first CE user must be owner")
	assert.True(t, isOrgAdmin, "first CE user must be org admin")
	assert.True(t, isBSAdmin, "first CE user must be base station admin")
	assert.True(t, isEPAdmin, "first CE user must be endpoint admin")
}

func TestRegisterCEAccount_SecondUser_BecomesMember(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := SetupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close db: %v", err)
		}
	}()

	repo := NewRegistrationRepository(db)
	ctx := testutil.TestContext()

	// Create a fresh CE org
	orgID := uuid.New()
	var tenantID int64
	err := db.QueryRow("INSERT INTO tenants (name) VALUES ($1) RETURNING id", "ce-test-tenant-2").Scan(&tenantID)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO organizations (org_id, tenant_id, name, state) VALUES ($1, $2, $3, $4)",
		orgID, tenantID, "CE Test Org 2", models.OrganizationStateActive)
	require.NoError(t, err)

	defer func() {
		_, _ = db.Exec("DELETE FROM organization_members WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM users WHERE email LIKE 'ce-member-test%'")
		_, _ = db.Exec("DELETE FROM organizations WHERE org_id = $1", orgID)
		_, _ = db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	}()

	// Register first user (becomes owner)
	user1ID := uuid.New()
	firstName1 := "First"
	lastName1 := "Owner"
	pw1 := "hashed-pw-1"
	user1 := &models.User{
		ID:           user1ID,
		Email:        "ce-member-test-1@example.com",
		PasswordHash: &pw1,
		IsAdmin:      false,
		IsActive:     true,
		FirstName:    &firstName1,
		LastName:     &lastName1,
	}
	_, err = repo.RegisterCEAccount(ctx, &interfaces.CERegistrationParams{
		User: user1, OrgID: orgID, TenantID: tenantID,
	})
	require.NoError(t, err)

	// Register second user (becomes member)
	user2ID := uuid.New()
	firstName2 := "Second"
	lastName2 := "Member"
	pw2 := "hashed-pw-2"
	user2 := &models.User{
		ID:           user2ID,
		Email:        "ce-member-test-2@example.com",
		PasswordHash: &pw2,
		IsAdmin:      false,
		IsActive:     true,
		FirstName:    &firstName2,
		LastName:     &lastName2,
	}
	result, err := repo.RegisterCEAccount(ctx, &interfaces.CERegistrationParams{
		User: user2, OrgID: orgID, TenantID: tenantID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Second user must be member, not owner
	var role string
	var isOrgAdmin, isBSAdmin, isEPAdmin bool
	err = db.QueryRow(
		"SELECT role, is_org_admin, is_base_station_admin, is_endpoint_admin FROM organization_members WHERE org_id = $1 AND user_id = $2",
		orgID, user2ID,
	).Scan(&role, &isOrgAdmin, &isBSAdmin, &isEPAdmin)
	require.NoError(t, err)
	assert.Equal(t, models.OrganizationRoleMember, role, "second CE user must be member")
	assert.False(t, isOrgAdmin, "second CE user must not be org admin")
	assert.False(t, isBSAdmin, "second CE user must not be base station admin")
	assert.False(t, isEPAdmin, "second CE user must not be endpoint admin")
}
