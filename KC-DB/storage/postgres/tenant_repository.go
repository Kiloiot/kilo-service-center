package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/models"
)

// TenantRepository provides internal postgres-specific tenant operations.
// This is an internal implementation not exposed outside the postgres package.
type TenantRepository struct {
	db *sqlx.DB
}

// NewTenantRepository creates a new internal tenant repository
func NewTenantRepository(db *sqlx.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// ListTenants retrieves all tenants with optional status filtering
func (r *TenantRepository) ListTenants(ctx context.Context, statusFilter string) ([]models.Tenant, error) {
	var tenants []models.Tenant

	query := `
		SELECT id, name, description, status, created_at, updated_at
		FROM tenants
	`

	args := []interface{}{}
	if statusFilter != "" {
		query += ` WHERE status = $1`
		args = append(args, statusFilter)
	}

	query += ` ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &tenants, query, args...); err != nil {
		return nil, fmt.Errorf("postgres: tenant: list: %w", err)
	}

	return tenants, nil
}

// CreateTenant creates a new tenant with the provided details
func (r *TenantRepository) CreateTenant(ctx context.Context, name string, description *string) (*models.Tenant, error) {
	query := `
		INSERT INTO tenants (name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, name, description, status, created_at, updated_at
	`

	var tenant models.Tenant
	if err := r.db.GetContext(ctx, &tenant, query, name, description, models.TenantStatusActive); err != nil {
		return nil, fmt.Errorf("postgres: tenant: create: %w", err)
	}

	return &tenant, nil
}

// GetTenantByID retrieves a tenant by ID
func (r *TenantRepository) GetTenantByID(ctx context.Context, id int64) (*models.Tenant, error) {
	query := `
		SELECT id, name, description, status, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	var tenant models.Tenant
	if err := r.db.GetContext(ctx, &tenant, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("postgres: tenant: get: tenant not found (id=%d): %w", id, err)
		}
		return nil, fmt.Errorf("postgres: tenant: get: %w", err)
	}

	return &tenant, nil
}

// UpdateTenant updates tenant fields dynamically using a map of field updates
func (r *TenantRepository) UpdateTenant(ctx context.Context, id int64, updates map[string]interface{}) (*models.Tenant, error) {
	if len(updates) == 0 {
		return nil, fmt.Errorf("postgres: tenant: update: no fields to update")
	}

	// Build SET clause dynamically
	setClauses := []string{}
	args := []interface{}{}
	argNum := 1

	// Always update updated_at
	updates["updated_at"] = "NOW()"

	for field, value := range updates {
		if field == "updated_at" {
			setClauses = append(setClauses, "updated_at = NOW()")
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field, argNum))
		args = append(args, value)
		argNum++
	}

	// Add tenant ID as final parameter
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE tenants
		SET %s
		WHERE id = $%d
		RETURNING id, name, description, status, created_at, updated_at
	`, strings.Join(setClauses, ", "), argNum)

	var tenant models.Tenant
	if err := r.db.GetContext(ctx, &tenant, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("postgres: tenant: update: tenant not found (id=%d): %w", id, err)
		}
		return nil, fmt.Errorf("postgres: tenant: update: %w", err)
	}

	return &tenant, nil
}

// SetStatus updates the status field (for soft delete/activation)
func (r *TenantRepository) SetStatus(ctx context.Context, id int64, active bool) error {
	status := models.TenantStatusActive
	if !active {
		status = models.TenantStatusInactive
	}

	query := `
		UPDATE tenants
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("postgres: tenant: set_status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: tenant: set_status: check rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("postgres: tenant: set_status: tenant not found (id=%d)", id)
	}

	return nil
}

// DeleteTenant removes a tenant by ID - used for org creation rollback
func (r *TenantRepository) DeleteTenant(ctx context.Context, id int64) error {
	query := `DELETE FROM tenants WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("postgres: tenant: delete: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: tenant: delete: check rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("postgres: tenant: delete: tenant not found (id=%d)", id)
	}

	return nil
}
