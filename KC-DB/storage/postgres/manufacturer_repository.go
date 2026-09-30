package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ManufacturerRepository implements the ManufacturerRepository interface for PostgreSQL
type ManufacturerRepository struct {
	log logger.Logger
	db  *sqlx.DB
}

// NewManufacturerRepository creates a new PostgreSQL Manufacturer repository
func NewManufacturerRepository(db *sqlx.DB, log logger.Logger) *ManufacturerRepository {
	return &ManufacturerRepository{
		log: log, db: db}
}

// Create creates a new manufacturer
func (r *ManufacturerRepository) Create(ctx context.Context, params *models.ManufacturerCreateParams) (*models.Manufacturer, error) {
	mfr := &models.Manufacturer{
		ID:         uuid.New(),
		IsSystem:   params.IsSystem,
		Name:       params.Name,
		Website:    params.Website,
		IsVerified: false,
	}
	if !params.IsSystem {
		mfr.TenantID = &params.TenantID
	}

	query := `
		INSERT INTO manufacturers (
			id, tenant_id, is_system, name, website, is_verified, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :is_system, :name, :website, :is_verified, NOW(), NOW()
		) RETURNING created_at, updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapPrepareStatement, err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			r.log.Warn(logMsgCloseStmtManufacturers, logger.FieldError, err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, mfr).Scan(&mfr.CreatedAt, &mfr.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeUniqueViolation {
			return nil, storage.ErrDuplicateKey
		}
		return nil, fmt.Errorf("%s: %w", errWrapCreateManufacturer, err)
	}

	return mfr, nil
}

// GetByID retrieves a manufacturer by ID with tenant isolation
func (r *ManufacturerRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Manufacturer, error) {
	var mfr models.Manufacturer
	// Resolve by id across both ownership scopes (tenant Custom + globally-visible System).
	query := `
		SELECT id, tenant_id, is_system, name, website, is_verified, created_at, updated_at
		FROM manufacturers
		WHERE (tenant_id = $1 OR is_system) AND id = $2`

	err := r.db.GetContext(ctx, &mfr, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrRecordNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetManufacturer, err)
	}

	return &mfr, nil
}

// List retrieves manufacturers for a tenant with pagination and optional search
func (r *ManufacturerRepository) List(ctx context.Context, params *models.ManufacturerListParams) ([]*models.Manufacturer, error) {
	var manufacturers []*models.Manufacturer

	// Scope by is_system via a CASE so $1/$2 stay bound regardless of world — no arg renumbering.
	query := `
		SELECT
			m.id, m.tenant_id, m.is_system, m.name, m.website, m.is_verified, m.created_at, m.updated_at,
			COALESCE(dm.model_count, 0) AS model_count
		FROM manufacturers m
		LEFT JOIN (
			SELECT manufacturer_id, COUNT(*) AS model_count
			FROM device_models
			WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)
			GROUP BY manufacturer_id
		) dm ON dm.manufacturer_id = m.id
		WHERE (CASE WHEN $2 THEN m.is_system ELSE m.tenant_id = $1 END)`

	args := []interface{}{params.TenantID, params.IsSystem}
	argIndex := 3

	if params.SearchTerm != "" {
		query += fmt.Sprintf(" AND m.name ILIKE $%d", argIndex)
		args = append(args, "%"+params.SearchTerm+"%")
		argIndex++
	}

	query += orderByNameAsc

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, params.Limit)
		argIndex++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex) //nolint:gosec // G202: appends a parameter placeholder, values are bound
		args = append(args, params.Offset)
	}

	err := r.db.SelectContext(ctx, &manufacturers, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListManufacturers, err)
	}

	return manufacturers, nil
}

// Count returns the total count of manufacturers for a tenant
func (r *ManufacturerRepository) Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM manufacturers WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)`

	err := r.db.GetContext(ctx, &count, query, tenantID, isSystem)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountManufacturers, err)
	}

	return count, nil
}

// Update updates an existing manufacturer
func (r *ManufacturerRepository) Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.ManufacturerUpdateParams) error {
	setClauses := make([]string, 0)
	args := []interface{}{tenantID, isSystem, id}
	argIndex := 4

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *params.Name)
		argIndex++
	}

	if params.Website != nil {
		setClauses = append(setClauses, fmt.Sprintf("website = $%d", argIndex))
		args = append(args, *params.Website)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	// #nosec G201 -- setClauses are constructed from safe column names with parameterized values
	query := fmt.Sprintf(
		"UPDATE manufacturers SET %s WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3",
		strings.Join(setClauses, ", "),
	)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeUniqueViolation {
			return storage.ErrDuplicateKey
		}
		return fmt.Errorf("%s: %w", errWrapUpdateManufacturer, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return storage.ErrRecordNotFound
	}

	return nil
}

// Delete deletes a manufacturer by ID within the isSystem-selected ownership scope
func (r *ManufacturerRepository) Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error {
	query := `DELETE FROM manufacturers WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`

	result, err := r.db.ExecContext(ctx, query, tenantID, isSystem, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeForeignKeyViolation {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: %w", errWrapDeleteManufacturer, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return storage.ErrRecordNotFound
	}

	return nil
}
