package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// IntegrationRepository implements the IntegrationRepository interface for PostgreSQL (CRUD for API parity)
type IntegrationRepository struct {
	db *sqlx.DB
}

// NewIntegrationRepository creates a new PostgreSQL integration repository
func NewIntegrationRepository(db *sqlx.DB) interfaces.IntegrationRepository {
	return &IntegrationRepository{db: db}
}

// Create inserts a new integration
func (r *IntegrationRepository) Create(ctx context.Context, integration *models.Integration) error {
	now := time.Now().UTC()
	integration.CreatedAt = now
	integration.UpdatedAt = now

	query := `
		INSERT INTO integrations (
			org_id, tenant_id, name, description, type, config,
			event_filter, delivery_format, status, created_at, updated_at,
			created_by, updated_by
		) VALUES (
			:org_id, :tenant_id, :name, :description, :type, :config,
			:event_filter, :delivery_format, :status, :created_at, :updated_at,
			:created_by, :updated_by
		) RETURNING id`

	rows, err := r.db.NamedQueryContext(ctx, query, integration)
	if err != nil {
		return fmt.Errorf("create integration: %w", err)
	}
	defer rows.Close() //nolint:errcheck // Error from Close() in defer is not actionable

	if rows.Next() {
		if err := rows.Scan(&integration.ID); err != nil {
			return fmt.Errorf("scan integration id: %w", err)
		}
	}

	return nil
}

// GetByID retrieves an integration by ID with tenant isolation
func (r *IntegrationRepository) GetByID(ctx context.Context, id int64, tenantID int64) (*models.Integration, error) {
	var integration models.Integration
	query := `
		SELECT id, org_id, tenant_id, name, description, type, config,
			   event_filter, delivery_format, status, created_at, updated_at,
			   created_by, updated_by
		FROM integrations
		WHERE id = $1 AND tenant_id = $2`

	err := r.db.GetContext(ctx, &integration, query, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("integration %d: %w", id, interfaces.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("get integration: %w", err)
	}

	return &integration, nil
}

// ListByTenant retrieves paginated integrations for a tenant
func (r *IntegrationRepository) ListByTenant(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error) {
	var integrations []*models.Integration
	var count int64

	// Get total count
	countQuery := `SELECT COUNT(*) FROM integrations WHERE tenant_id = $1`
	err := r.db.GetContext(ctx, &count, countQuery, tenantID)
	if err != nil {
		return nil, 0, fmt.Errorf("count integrations: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, org_id, tenant_id, name, description, type, config,
			   event_filter, delivery_format, status, created_at, updated_at,
			   created_by, updated_by
		FROM integrations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	err = r.db.SelectContext(ctx, &integrations, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list integrations: %w", err)
	}

	return integrations, count, nil
}

// Update updates an existing integration
func (r *IntegrationRepository) Update(ctx context.Context, integration *models.Integration) error {
	integration.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE integrations SET
			name = :name,
			description = :description,
			config = :config,
			event_filter = :event_filter,
			status = :status,
			updated_at = :updated_at,
			updated_by = :updated_by
		WHERE id = :id AND tenant_id = :tenant_id`

	result, err := r.db.NamedExecContext(ctx, query, integration)
	if err != nil {
		return fmt.Errorf("update integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("integration %d: %w", integration.ID, interfaces.ErrRecordNotFound)
	}

	return nil
}

// Delete deletes an integration by ID with tenant isolation
func (r *IntegrationRepository) Delete(ctx context.Context, id int64, tenantID int64) error {
	query := `DELETE FROM integrations WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("integration %d: %w", id, interfaces.ErrRecordNotFound)
	}

	return nil
}
