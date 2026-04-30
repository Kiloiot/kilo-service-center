package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// RegistrationRepository handles atomic self-service account registration.
type RegistrationRepository struct {
	db *sqlx.DB
}

// NewRegistrationRepository creates a new registration repository.
func NewRegistrationRepository(db *sqlx.DB) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

// RegisterAccount atomically creates a user, tenant, organization, and membership.
// All operations run within a single transaction — partial state is impossible.
func (r *RegistrationRepository) RegisterAccount(ctx context.Context, params *interfaces.RegistrationParams) (*interfaces.RegistrationResult, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin registration transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC()
	params.User.CreatedAt = now
	params.User.UpdatedAt = now

	// Insert user
	userQuery := `
		INSERT INTO users (
			id, external_id, email, email_verified, password_hash,
			is_admin, is_active, is_tenant_manager, is_base_station_manager,
			is_endpoint_manager, note, first_name, last_name, company_name,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		) RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(ctx, userQuery,
		params.User.ID, params.User.ExternalID, params.User.Email, params.User.EmailVerified, params.User.PasswordHash,
		params.User.IsAdmin, params.User.IsActive, params.User.IsTenantManager, params.User.IsBaseStationManager,
		params.User.IsEndpointManager, params.User.Note, params.User.FirstName, params.User.LastName, params.User.CompanyName,
		params.User.CreatedAt, params.User.UpdatedAt,
	).Scan(&params.User.ID, &params.User.CreatedAt, &params.User.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	// Insert tenant
	var tenantID int64
	tenantQuery := `INSERT INTO tenants (name, status, created_at, updated_at) VALUES ($1, $2, $3, $3) RETURNING id`
	err = tx.QueryRowContext(ctx, tenantQuery, params.CompanyName, models.TenantStatusActive, now).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}

	// Insert organization
	orgID := uuid.New()
	org := &models.Organization{
		OrgID:    orgID,
		TenantID: tenantID,
		Name:     params.CompanyName,
		State:    models.OrganizationStateActive,
	}

	orgQuery := `
		INSERT INTO organizations (org_id, tenant_id, name, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING created_at, updated_at`
	err = tx.QueryRowContext(ctx, orgQuery, org.OrgID, org.TenantID, org.Name, org.State, now).Scan(&org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert organization: %w", err)
	}

	// Insert organization membership (owner with full admin permissions)
	memberQuery := `
		INSERT INTO organization_members (
			org_id, user_id, role, status,
			is_org_admin, is_base_station_admin, is_endpoint_admin,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`
	_, err = tx.ExecContext(ctx, memberQuery,
		orgID, params.User.ID, models.OrganizationRoleOwner, models.OrganizationMemberStatusActive,
		true, true, true, now)
	if err != nil {
		return nil, fmt.Errorf("insert organization member: %w", err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("commit registration transaction: %w", err)
	}

	return &interfaces.RegistrationResult{
		User:         params.User,
		Organization: org,
		TenantID:     tenantID,
	}, nil
}

// RegisterCEAccount atomically creates a user and adds them to the existing CE default org.
// No new tenant or organization is created.
func (r *RegistrationRepository) RegisterCEAccount(ctx context.Context, params *interfaces.CERegistrationParams) (*interfaces.RegistrationResult, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin CE registration transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC()
	params.User.CreatedAt = now
	params.User.UpdatedAt = now

	// Insert user
	userQuery := `
		INSERT INTO users (
			id, external_id, email, email_verified, password_hash,
			is_admin, is_active, is_tenant_manager, is_base_station_manager,
			is_endpoint_manager, note, first_name, last_name, company_name,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		) RETURNING id, created_at, updated_at`

	err = tx.QueryRowContext(ctx, userQuery,
		params.User.ID, params.User.ExternalID, params.User.Email, params.User.EmailVerified, params.User.PasswordHash,
		params.User.IsAdmin, params.User.IsActive, params.User.IsTenantManager, params.User.IsBaseStationManager,
		params.User.IsEndpointManager, params.User.Note, params.User.FirstName, params.User.LastName, params.User.CompanyName,
		params.User.CreatedAt, params.User.UpdatedAt,
	).Scan(&params.User.ID, &params.User.CreatedAt, &params.User.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	// Lock the organization row to serialize concurrent first-user registration
	var lockedOrgID string
	err = tx.QueryRowContext(ctx,
		`SELECT org_id FROM organizations WHERE org_id = $1 FOR UPDATE`,
		params.OrgID,
	).Scan(&lockedOrgID)
	if err != nil {
		return nil, fmt.Errorf("lock CE org for member count: %w", err)
	}

	// Count active members to determine if this is the first user
	var memberCount int
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM organization_members WHERE org_id = $1 AND status = $2`,
		params.OrgID, models.OrganizationMemberStatusActive,
	).Scan(&memberCount)
	if err != nil {
		return nil, fmt.Errorf("count CE org members: %w", err)
	}

	// First user becomes owner with full admin rights; subsequent users are members
	role := models.OrganizationRoleMember
	isOrgAdmin := false
	isBSAdmin := params.User.IsAdmin
	isEPAdmin := params.User.IsAdmin
	if memberCount == 0 {
		role = models.OrganizationRoleOwner
		isOrgAdmin = true
		isBSAdmin = true
		isEPAdmin = true
	}

	memberQuery := `
		INSERT INTO organization_members (
			org_id, user_id, role, status,
			is_org_admin, is_base_station_admin, is_endpoint_admin,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`
	_, err = tx.ExecContext(ctx, memberQuery,
		params.OrgID, params.User.ID, role, models.OrganizationMemberStatusActive,
		isOrgAdmin, isBSAdmin, isEPAdmin, now)
	if err != nil {
		return nil, fmt.Errorf("insert CE organization member: %w", err)
	}

	// Look up org details for the result
	var org models.Organization
	orgQuery := `SELECT org_id, tenant_id, name, state, created_at, updated_at FROM organizations WHERE org_id = $1`
	err = tx.QueryRowContext(ctx, orgQuery, params.OrgID).Scan(
		&org.OrgID, &org.TenantID, &org.Name, &org.State, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("lookup CE default org: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("commit CE registration transaction: %w", err)
	}

	return &interfaces.RegistrationResult{
		User:         params.User,
		Organization: &org,
		TenantID:     params.TenantID,
	}, nil
}
