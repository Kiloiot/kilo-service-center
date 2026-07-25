package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// OrganizationRepository implements the OrganizationRepository interface for PostgreSQL
type OrganizationRepository struct {
	db *sqlx.DB
}

// NewOrganizationRepository creates a new PostgreSQL Organization repository
func NewOrganizationRepository(db *sqlx.DB) interfaces.OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// GetTenantByOrgID resolves organization UUID to numeric tenant ID
// Hot path for org.Resolver.LookupTenant - optimized with unique index
func (r *OrganizationRepository) GetTenantByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var tenantID int64
	query := `SELECT tenant_id FROM organizations WHERE org_id = $1`

	err := r.db.GetContext(ctx, &tenantID, query, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("organization %s not found", orgID)
		}
		return 0, fmt.Errorf("resolve organization to tenant: %w", err)
	}

	return tenantID, nil
}

// GetOrgByTenantID returns the default organization for a tenant
// Used for fallback when no org header provided (community mode)
func (r *OrganizationRepository) GetOrgByTenantID(ctx context.Context, tenantID int64) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       can_have_base_stations, max_base_station_count, max_endpoint_count,
		       tags, created_at, updated_at
		FROM organizations
		WHERE tenant_id = $1
		ORDER BY created_at ASC
		LIMIT 1`

	err := r.db.GetContext(ctx, &org, query, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no organization found for tenant %d", tenantID)
		}
		return nil, fmt.Errorf("get organization by tenant: %w", err)
	}

	return &org, nil
}

// UpsertOrg creates or updates organization (Kilo Cloud sync path).
// The organization ID is the global sync identity, so the existence check is
// by org_id alone; the tenant binding of an existing organization is immutable
// and always preserved, so a payload carrying a different tenant can neither
// move the organization nor read anything through this path.
func (r *OrganizationRepository) UpsertOrg(ctx context.Context, org *models.Organization) error {
	existing, err := r.getByOrgID(ctx, org.OrgID)
	if err != nil {
		// Only create if org truly doesn't exist
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			return r.Create(ctx, org)
		}
		// Propagate other errors (connection failures, query errors, etc.)
		return fmt.Errorf("upsert pre-check failed: %w", err)
	}

	// Organization exists, update it
	updates := map[string]interface{}{
		"name":  org.Name,
		"state": org.State,
	}

	// Preserve tenant_id from existing record (immutable after creation)
	org.TenantID = existing.TenantID

	return r.Update(ctx, org.OrgID, org.TenantID, updates)
}

// getByOrgID looks an organization up by its global identity for the sync
// upsert path only; tenant-scoped reads go through GetByID.
func (r *OrganizationRepository) getByOrgID(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, created_at, updated_at
		FROM organizations
		WHERE org_id = $1`

	if err := r.db.GetContext(ctx, &org, query, orgID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get organization by org_id: %w", err)
	}
	return &org, nil
}

// Create creates a new organization
func (r *OrganizationRepository) Create(ctx context.Context, org *models.Organization) error {
	query := `
		INSERT INTO organizations (
			org_id, tenant_id, name, state, created_at, updated_at
		) VALUES (
			:org_id, :tenant_id, :name, :state, NOW(), NOW()
		) RETURNING created_at, updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close statement in organization repository: %v", err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, org).Scan(&org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}

	return nil
}

// GetByID retrieves an organization by UUID, scoped to a tenant for defense-in-depth.
func (r *OrganizationRepository) GetByID(ctx context.Context, orgID uuid.UUID, tenantID int64) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       can_have_base_stations, max_base_station_count, max_endpoint_count,
		       tags, created_at, updated_at
		FROM organizations
		WHERE org_id = $1 AND tenant_id = $2`

	err := r.db.GetContext(ctx, &org, query, orgID, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization %s not found", orgID)
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}

	return &org, nil
}

// GetByIDUnscoped retrieves an organization by UUID without tenant scoping.
func (r *OrganizationRepository) GetByIDUnscoped(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       can_have_base_stations, max_base_station_count, max_endpoint_count,
		       tags, created_at, updated_at
		FROM organizations
		WHERE org_id = $1`

	err := r.db.GetContext(ctx, &org, query, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get organization unscoped: %w", err)
	}

	return &org, nil
}

// Update updates an existing organization, scoped to a tenant for defense-in-depth.
func (r *OrganizationRepository) Update(ctx context.Context, orgID uuid.UUID, tenantID int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setClauses := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+2)
	args = append(args, orgID, tenantID)

	i := 3
	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, i))
		args = append(args, value)
		i++
	}

	// Always update updated_at
	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(
		"UPDATE organizations SET %s WHERE org_id = $1 AND tenant_id = $2",
		strings.Join(setClauses, ", "),
	)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization %s not found", orgID)
	}

	return nil
}

// Delete hard-deletes an organization and its members, scoped to a tenant for defense-in-depth.
// Also deletes the associated tenant if orphaned.
func (r *OrganizationRepository) Delete(ctx context.Context, orgID uuid.UUID, tenantID int64) error {
	// Start transaction for cascading delete
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Delete organization members first (FK constraint)
	_, err = tx.ExecContext(ctx, `DELETE FROM organization_members WHERE org_id = $1`, orgID)
	if err != nil {
		return fmt.Errorf("delete organization members: %w", err)
	}

	// api_keys.org_id is NO ACTION, so keys must be deleted before the org (same tx).
	_, err = tx.ExecContext(ctx, `DELETE FROM api_keys WHERE org_id = $1`, orgID)
	if err != nil {
		return fmt.Errorf("delete organization api keys: %w", err)
	}

	// Delete the organization (tenant-scoped)
	result, err := tx.ExecContext(ctx, `DELETE FROM organizations WHERE org_id = $1 AND tenant_id = $2`, orgID, tenantID)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization %s not found", orgID)
	}

	// Check if tenant is orphaned (no remaining orgs) and delete if so
	var remainingOrgs int
	err = tx.GetContext(ctx, &remainingOrgs, `SELECT COUNT(*) FROM organizations WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return fmt.Errorf("count remaining orgs: %w", err)
	}

	if remainingOrgs == 0 {
		// Tenant is orphaned, delete it
		_, err = tx.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
		if err != nil {
			return fmt.Errorf("delete orphaned tenant: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ListOrgMembers retrieves all members of an organization
func (r *OrganizationRepository) ListOrgMembers(ctx context.Context, orgID uuid.UUID, status string) ([]*models.OrganizationMember, error) {
	var members []*models.OrganizationMember

	var query string
	var args []interface{}

	if status != "" {
		query = `
			SELECT org_id, user_id, role, status, created_at, updated_at
			FROM organization_members
			WHERE org_id = $1 AND status = $2
			ORDER BY created_at ASC`
		args = []interface{}{orgID, status}
	} else {
		query = `
			SELECT org_id, user_id, role, status, created_at, updated_at
			FROM organization_members
			WHERE org_id = $1
			ORDER BY created_at ASC`
		args = []interface{}{orgID}
	}

	err := r.db.SelectContext(ctx, &members, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list organization members: %w", err)
	}

	return members, nil
}

// CheckUserMembership validates that a user belongs to an organization
func (r *OrganizationRepository) CheckUserMembership(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM organization_members
		WHERE org_id = $1 AND user_id = $2 AND status = $3`

	err := r.db.GetContext(ctx, &count, query, orgID, userID, models.OrganizationMemberStatusActive)
	if err != nil {
		return false, fmt.Errorf("check user membership: %w", err)
	}

	return count > 0, nil
}

// AddMember adds a user to an organization with specified role
func (r *OrganizationRepository) AddMember(ctx context.Context, member *models.OrganizationMember) error {
	query := `
		INSERT INTO organization_members (
			org_id, user_id, role, status, created_at, updated_at
		) VALUES (
			:org_id, :user_id, :role, :status, NOW(), NOW()
		) RETURNING created_at, updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close statement in organization repository: %v", err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, member).Scan(&member.CreatedAt, &member.UpdatedAt)
	if err != nil {
		return fmt.Errorf("add organization member: %w", err)
	}

	return nil
}

// RemoveMember soft-removes a user from an organization (sets status = 'removed')
func (r *OrganizationRepository) RemoveMember(ctx context.Context, orgID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE organization_members
		SET status = $3, updated_at = NOW()
		WHERE org_id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, orgID, userID, models.OrganizationMemberStatusRemoved)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("member not found in organization")
	}

	return nil
}

// UpdateMemberRole changes a member's role
func (r *OrganizationRepository) UpdateMemberRole(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, role string) error {
	query := `
		UPDATE organization_members
		SET role = $3, updated_at = NOW()
		WHERE org_id = $1 AND user_id = $2 AND status = $4`

	result, err := r.db.ExecContext(ctx, query, orgID, userID, role, models.OrganizationMemberStatusActive)
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("active member not found in organization")
	}

	return nil
}

// ListUserMemberships retrieves all organizations a user belongs to with org details.
// Used for profile endpoint to show user's org memberships.
func (r *OrganizationRepository) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*models.OrganizationMembershipWithOrg, error) {
	var memberships []*models.OrganizationMembershipWithOrg

	query := `
		SELECT
			o.org_id,
			o.name AS org_name,
			om.role,
			om.status,
			om.is_org_admin,
			om.is_base_station_admin,
			om.is_endpoint_admin,
			om.created_at
		FROM organization_members om
		JOIN organizations o ON o.org_id = om.org_id
		WHERE om.user_id = $1 AND om.status = $2
		ORDER BY om.created_at ASC`

	err := r.db.SelectContext(ctx, &memberships, query, userID, models.OrganizationMemberStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list user memberships: %w", err)
	}

	return memberships, nil
}

// ListUserMembershipsByTenant retrieves organizations a user belongs to within a specific tenant.
// Adds tenant_id filter to prevent cross-tenant data leak.
func (r *OrganizationRepository) ListUserMembershipsByTenant(ctx context.Context, userID uuid.UUID, tenantID int64) ([]*models.OrganizationMembershipWithOrg, error) {
	var memberships []*models.OrganizationMembershipWithOrg

	query := `
		SELECT
			o.org_id,
			o.name AS org_name,
			om.role,
			om.status,
			om.is_org_admin,
			om.is_base_station_admin,
			om.is_endpoint_admin,
			om.created_at
		FROM organization_members om
		JOIN organizations o ON o.org_id = om.org_id
		WHERE om.user_id = $1 AND om.status = $2 AND o.tenant_id = $3
		ORDER BY om.created_at ASC`

	err := r.db.SelectContext(ctx, &memberships, query, userID, models.OrganizationMemberStatusActive, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list user memberships by tenant: %w", err)
	}

	return memberships, nil
}

// GetOrgByExternalID retrieves an organization by external IdP identifier.
// Used for OIDC external_org_claim resolution.
func (r *OrganizationRepository) GetOrgByExternalID(ctx context.Context, externalID string) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       can_have_base_stations, max_base_station_count, max_endpoint_count,
		       tags, created_at, updated_at
		FROM organizations
		WHERE external_id = $1`

	err := r.db.GetContext(ctx, &org, query, externalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get organization by external_id: %w", err)
	}

	return &org, nil
}

// ListOrganizations retrieves organizations with optional tenant filter and pagination.
// Returns organizations slice and total count for pagination.
func (r *OrganizationRepository) ListOrganizations(ctx context.Context, tenantID *int64, limit, offset int) ([]*models.Organization, int64, error) {
	var orgs []*models.Organization
	var total int64

	// Build query with optional tenant filter
	baseQuery := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       can_have_base_stations, max_base_station_count, max_endpoint_count,
		       tags, created_at, updated_at
		FROM organizations`
	countQuery := `SELECT COUNT(*) FROM organizations`

	var args []interface{}
	whereClause := ""
	if tenantID != nil {
		whereClause = ` WHERE tenant_id = $1`
		args = append(args, *tenantID)
	}

	// Get total count
	err := r.db.GetContext(ctx, &total, countQuery+whereClause, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count organizations: %w", err)
	}

	// Get paginated results
	paginatedQuery := baseQuery + whereClause + ` ORDER BY created_at DESC`
	if tenantID != nil {
		paginatedQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", 2, 3)
		args = append(args, limit, offset)
	} else {
		paginatedQuery += " LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	}

	err = r.db.SelectContext(ctx, &orgs, paginatedQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list organizations: %w", err)
	}

	return orgs, total, nil
}

// ListOrgMembersWithEmail returns members with email from users table (JOIN query).
// Returns paginated results and total count matching the status filter.
// The status parameter uses three modes:
//   - "" (empty) — no status filter (all statuses)
//   - models.OrganizationMemberStatusFilterInactive ("inactive") — status != 'active'
//   - Any other value ("active", "invited", "removed") — exact match
func (r *OrganizationRepository) ListOrgMembersWithEmail(ctx context.Context, orgID uuid.UUID, status string, limit, offset int) ([]*models.OrganizationMemberWithEmail, int64, error) {
	baseWhere := `om.org_id = $1`
	args := []interface{}{orgID}
	argIdx := 1

	if status == models.OrganizationMemberStatusFilterInactive {
		argIdx++
		baseWhere += fmt.Sprintf(` AND om.status != $%d`, argIdx)
		args = append(args, models.OrganizationMemberStatusActive)
	} else if status != "" {
		argIdx++
		baseWhere += fmt.Sprintf(` AND om.status = $%d`, argIdx)
		args = append(args, status)
	}

	// Count query
	var total int64
	countQuery := `SELECT COUNT(*) FROM organization_members om WHERE ` + baseWhere
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count org members with email: %w", err)
	}

	// Data query with pagination
	query := `SELECT om.org_id, om.user_id, u.email, om.role, om.status,
		       om.is_org_admin, om.is_base_station_admin, om.is_endpoint_admin,
		       om.created_at, om.updated_at
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE ` + baseWhere + ` ORDER BY om.created_at ASC`

	argIdx++
	query += fmt.Sprintf(` LIMIT $%d`, argIdx)
	args = append(args, limit)
	argIdx++
	query += fmt.Sprintf(` OFFSET $%d`, argIdx)
	args = append(args, offset)

	var members []*models.OrganizationMemberWithEmail
	if err := r.db.SelectContext(ctx, &members, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list org members with email: %w", err)
	}

	return members, total, nil
}

// GetOrgMemberWithEmail returns a single member with email.
// Used for organization user detail view in admin UI.
func (r *OrganizationRepository) GetOrgMemberWithEmail(ctx context.Context, orgID, userID uuid.UUID) (*models.OrganizationMemberWithEmail, error) {
	var member models.OrganizationMemberWithEmail

	query := `
		SELECT om.org_id, om.user_id, u.email, om.role, om.status,
		       om.is_org_admin, om.is_base_station_admin, om.is_endpoint_admin,
		       om.created_at, om.updated_at
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.org_id = $1 AND om.user_id = $2`

	err := r.db.GetContext(ctx, &member, query, orgID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get org member with email: %w", err)
	}

	return &member, nil
}

// CountOrgMembers returns total count of org members matching status filter.
// Required for list response pagination.
func (r *OrganizationRepository) CountOrgMembers(ctx context.Context, orgID uuid.UUID, status string) (int64, error) {
	var count int64

	query := `SELECT COUNT(*) FROM organization_members WHERE org_id = $1`
	args := []interface{}{orgID}

	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("count org members: %w", err)
	}

	return count, nil
}

// CountOrgMembersByRole returns the count of active members with the given role.
func (r *OrganizationRepository) CountOrgMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM organization_members WHERE org_id = $1 AND role = $2 AND status = $3`
	err := r.db.GetContext(ctx, &count, query, orgID, role, models.OrganizationMemberStatusActive)
	if err != nil {
		return 0, fmt.Errorf("count org members by role: %w", err)
	}
	return count, nil
}

// CheckBaseStationQuota validates BS quota for an organization.
// Returns error if quota exceeded or organization cannot have base stations.
func (r *OrganizationRepository) CheckBaseStationQuota(ctx context.Context, orgID uuid.UUID) error {
	var org struct {
		CanHaveBaseStations bool `db:"can_have_base_stations"`
		MaxBaseStationCount *int `db:"max_base_station_count"`
	}

	// Get org quota settings
	query := `
		SELECT can_have_base_stations, max_base_station_count
		FROM organizations
		WHERE org_id = $1`

	err := r.db.GetContext(ctx, &org, query, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("get org quota settings: %w", err)
	}

	// Check if org can have base stations
	if !org.CanHaveBaseStations {
		return storage.ErrBaseStationsNotAllowed
	}

	// Check max base station count (NULL = unlimited)
	if org.MaxBaseStationCount != nil {
		var currentCount int
		countQuery := `SELECT COUNT(*) FROM basestations WHERE tenant_id = (SELECT tenant_id FROM organizations WHERE org_id = $1)`
		err = r.db.GetContext(ctx, &currentCount, countQuery, orgID)
		if err != nil {
			return fmt.Errorf("count basestations: %w", err)
		}

		if currentCount >= *org.MaxBaseStationCount {
			return storage.ErrBaseStationQuotaExceeded
		}
	}

	return nil
}

// CheckEndpointQuota validates endpoint quota for an organization.
// Returns error if quota exceeded.
func (r *OrganizationRepository) CheckEndpointQuota(ctx context.Context, orgID uuid.UUID) error {
	var maxEndpointCount *int

	// Get org quota settings
	query := `SELECT max_endpoint_count FROM organizations WHERE org_id = $1`

	err := r.db.GetContext(ctx, &maxEndpointCount, query, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("get org quota settings: %w", err)
	}

	// Check max endpoint count (NULL = unlimited)
	if maxEndpointCount != nil {
		var currentCount int
		countQuery := `SELECT COUNT(*) FROM endpoints WHERE tenant_id = (SELECT tenant_id FROM organizations WHERE org_id = $1)`
		err = r.db.GetContext(ctx, &currentCount, countQuery, orgID)
		if err != nil {
			return fmt.Errorf("count endpoints: %w", err)
		}

		if currentCount >= *maxEndpointCount {
			return storage.ErrEndpointQuotaExceeded
		}
	}

	return nil
}

// UpdateMemberPermissions updates member permission flags (explicit booleans).
// Permissions are independent of role - role is only for ownership semantics.
func (r *OrganizationRepository) UpdateMemberPermissions(ctx context.Context, orgID, userID uuid.UUID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin bool) error {
	query := `
		UPDATE organization_members
		SET is_org_admin = $3, is_base_station_admin = $4, is_endpoint_admin = $5, updated_at = NOW()
		WHERE org_id = $1 AND user_id = $2 AND status = $6`

	result, err := r.db.ExecContext(ctx, query, orgID, userID, isOrgAdmin, isBaseStationAdmin, isEndpointAdmin, models.OrganizationMemberStatusActive)
	if err != nil {
		return fmt.Errorf("update member permissions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("active member not found in organization")
	}

	return nil
}
