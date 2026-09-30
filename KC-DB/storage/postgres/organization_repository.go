package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// OrganizationRepository implements the OrganizationRepository interface for PostgreSQL
type OrganizationRepository struct {
	log logger.Logger
	db  *sqlx.DB
}

const orgUpdateFixedArgs = 2

// NewOrganizationRepository creates a new PostgreSQL Organization repository
// orgUpdateFixedArgs counts the leading fixed parameters (org id, tenant) of
// the dynamic update statement.
func NewOrganizationRepository(db *sqlx.DB, log logger.Logger) *OrganizationRepository {
	return &OrganizationRepository{
		log: log, db: db}
}

// GetTenantByOrgID resolves organization UUID to numeric tenant ID
// Hot path for org.Resolver.LookupTenant - optimized with unique index
func (r *OrganizationRepository) GetTenantByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var tenantID int64
	query := `SELECT tenant_id FROM organizations WHERE org_id = $1`

	err := r.db.GetContext(ctx, &tenantID, query, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf(errFmtOrganizationNotFound, orgID, storage.ErrNotFound)
		}
		return 0, fmt.Errorf("%s: %w", errWrapResolveOrganizationTenant, err)
	}

	return tenantID, nil
}

// GetOrgByTenantID returns the default organization for a tenant
// Used for fallback when no org header provided (community mode)
func (r *OrganizationRepository) GetOrgByTenantID(ctx context.Context, tenantID int64) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       tags, created_at, updated_at
		FROM organizations
		WHERE tenant_id = $1
		ORDER BY created_at ASC
		LIMIT 1`

	err := r.db.GetContext(ctx, &org, query, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(errFmtNoOrganizationFoundForTenant, tenantID, storage.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetOrganizationByTenant, err)
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
		return fmt.Errorf("%s: %w", errWrapPrepareStatement, err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.log.Warn(logMsgCloseStmtOrganizations, logger.FieldError, err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, org).Scan(&org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateOrganization, err)
	}

	return nil
}

// GetByID retrieves an organization by UUID, scoped to a tenant for defense-in-depth.
func (r *OrganizationRepository) GetByID(ctx context.Context, orgID uuid.UUID, tenantID int64) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       tags, created_at, updated_at
		FROM organizations
		WHERE org_id = $1 AND tenant_id = $2`

	err := r.db.GetContext(ctx, &org, query, orgID, tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf(errFmtOrganizationNotFound, orgID, storage.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetOrganization, err)
	}

	return &org, nil
}

// GetByIDUnscoped retrieves an organization by UUID without tenant scoping.
func (r *OrganizationRepository) GetByIDUnscoped(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       tags, created_at, updated_at
		FROM organizations
		WHERE org_id = $1`

	err := r.db.GetContext(ctx, &org, query, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetOrganizationUnscoped, err)
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
	args := make([]interface{}, 0, len(updates)+orgUpdateFixedArgs)
	args = append(args, orgID, tenantID)

	argIndex := len(args) + 1
	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Always update updated_at
	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(
		"UPDATE organizations SET %s WHERE org_id = $1 AND tenant_id = $2",
		strings.Join(setClauses, ", "),
	)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateOrganization, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(errFmtOrganizationNotFound, orgID, storage.ErrNotFound)
	}

	return nil
}

// Delete hard-deletes an organization and its members, scoped to a tenant for defense-in-depth.
// Also deletes the associated tenant if orphaned.
func (r *OrganizationRepository) Delete(ctx context.Context, orgID uuid.UUID, tenantID int64) (err error) {
	// Start transaction for cascading delete; rolling back after a successful
	// commit is a no-op.
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapBeginTransaction, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	// Delete organization members first (FK constraint)
	_, err = tx.ExecContext(ctx, `DELETE FROM organization_members WHERE org_id = $1`, orgID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapDeleteOrganizationMembers, err)
	}

	// api_keys.org_id is NO ACTION, so keys must be deleted before the org (same tx).
	_, err = tx.ExecContext(ctx, `DELETE FROM api_keys WHERE org_id = $1`, orgID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapDeleteOrganizationAPIKeys, err)
	}

	// Delete the organization (tenant-scoped)
	result, err := tx.ExecContext(ctx, `DELETE FROM organizations WHERE org_id = $1 AND tenant_id = $2`, orgID, tenantID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapDeleteOrganization, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(errFmtOrganizationNotFound, orgID, storage.ErrNotFound)
	}

	// Check if tenant is orphaned (no remaining orgs) and delete if so
	var remainingOrgs int
	err = tx.GetContext(ctx, &remainingOrgs, `SELECT COUNT(*) FROM organizations WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCountRemainingOrgs, err)
	}

	if remainingOrgs == 0 {
		// Tenant is orphaned, delete it
		_, err = tx.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID)
		if err != nil {
			return fmt.Errorf("%s: %w", errWrapDeleteOrphanedTenant, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapCommitTransaction, err)
	}

	return nil
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
		return fmt.Errorf("%s: %w", errWrapPrepareStatement, err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.log.Warn(logMsgCloseStmtOrganizations, logger.FieldError, err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, member).Scan(&member.CreatedAt, &member.UpdatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapAddOrganizationMember, err)
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
		return fmt.Errorf("%s: %w", errWrapRemoveMember, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", errWrapMemberOfOrganization, storage.ErrNotFound)
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
		return fmt.Errorf("%s: %w", errWrapUpdateMemberRole, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", errWrapActiveMemberOfOrganization, storage.ErrNotFound)
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
		return nil, fmt.Errorf("%s: %w", errWrapListUserMemberships, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapListUserMembershipsByTenant, err)
	}

	return memberships, nil
}

// GetOrgByExternalID retrieves an organization by external IdP identifier.
// Used for OIDC external_org_claim resolution.
func (r *OrganizationRepository) GetOrgByExternalID(ctx context.Context, externalID string) (*models.Organization, error) {
	var org models.Organization
	query := `
		SELECT org_id, tenant_id, name, state, external_id, description,
		       tags, created_at, updated_at
		FROM organizations
		WHERE external_id = $1`

	err := r.db.GetContext(ctx, &org, query, externalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetOrganizationByExternalID, err)
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
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountOrganizations, err)
	}

	// Get paginated results
	paginatedQuery := baseQuery + whereClause + ` ORDER BY created_at DESC`
	if tenantID != nil {
		paginatedQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", orgUpdateFixedArgs, orgUpdateFixedArgs+1)
		args = append(args, limit, offset)
	} else {
		paginatedQuery += " LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	}

	err = r.db.SelectContext(ctx, &orgs, paginatedQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapListOrganizations, err)
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
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountOrgMembersWithEmail, err)
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
		return nil, 0, fmt.Errorf("%s: %w", errWrapListOrgMembersWithEmail, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapGetOrgMemberWithEmail, err)
	}

	return &member, nil
}

// CountOrgMembersByRole returns the count of active members with the given role.
func (r *OrganizationRepository) CountOrgMembersByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM organization_members WHERE org_id = $1 AND role = $2 AND status = $3`
	err := r.db.GetContext(ctx, &count, query, orgID, role, models.OrganizationMemberStatusActive)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountOrgMembersByRole, err)
	}
	return count, nil
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
		return fmt.Errorf("%s: %w", errWrapUpdateMemberPermissions, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", errWrapActiveMemberOfOrganization, storage.ErrNotFound)
	}

	return nil
}
