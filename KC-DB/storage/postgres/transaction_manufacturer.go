package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// transactionalManufacturerRepository implements interfaces.ManufacturerRepository within a transaction
type transactionalManufacturerRepository struct {
	tx *sql.Tx
	db *DB
}

// Ensure interface compliance
var _ interfaces.ManufacturerRepository = (*transactionalManufacturerRepository)(nil)

// Create creates a new manufacturer within the transaction
func (r *transactionalManufacturerRepository) Create(ctx context.Context, params *models.ManufacturerCreateParams) (*models.Manufacturer, error) {
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
			$1, $2, $3, $4, $5, $6, NOW(), NOW()
		) RETURNING created_at, updated_at`

	err := r.tx.QueryRowContext(ctx, query,
		mfr.ID, mfr.TenantID, mfr.IsSystem, mfr.Name,
		mfr.Website, mfr.IsVerified,
	).Scan(&mfr.CreatedAt, &mfr.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, storage.ErrDuplicateKey
		}
		return nil, fmt.Errorf("create manufacturer: %w", err)
	}

	return mfr, nil
}

// GetByID retrieves a manufacturer by ID with tenant isolation within the transaction
func (r *transactionalManufacturerRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Manufacturer, error) {
	query := `
		SELECT id, tenant_id, name, website, is_verified, created_at, updated_at
		FROM manufacturers
		WHERE tenant_id = $1 AND id = $2`

	mfr := &models.Manufacturer{}
	var website sql.NullString

	err := r.tx.QueryRowContext(ctx, query, tenantID, id).Scan(
		&mfr.ID, &mfr.TenantID, &mfr.Name,
		&website,
		&mfr.IsVerified, &mfr.CreatedAt, &mfr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get manufacturer: %w", err)
	}

	if website.Valid {
		mfr.Website = &website.String
	}

	return mfr, nil
}

// List retrieves manufacturers for a tenant with pagination and optional search within the transaction
func (r *transactionalManufacturerRepository) List(ctx context.Context, params *models.ManufacturerListParams) ([]*models.Manufacturer, error) {
	query := `
		SELECT
			m.id, m.tenant_id, m.name, m.website, m.is_verified, m.created_at, m.updated_at,
			COALESCE(dm.model_count, 0) AS model_count
		FROM manufacturers m
		LEFT JOIN (
			SELECT manufacturer_id, COUNT(*) AS model_count
			FROM device_models
			WHERE tenant_id = $1
			GROUP BY manufacturer_id
		) dm ON dm.manufacturer_id = m.id
		WHERE m.tenant_id = $1`

	args := []interface{}{params.TenantID}
	argIndex := 2

	if params.SearchTerm != "" {
		query += fmt.Sprintf(" AND m.name ILIKE $%d", argIndex)
		args = append(args, "%"+params.SearchTerm+"%")
		argIndex++
	}

	query += " ORDER BY LOWER(name) ASC, name ASC, id ASC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, params.Limit)
		argIndex++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, params.Offset)
	}

	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list manufacturers: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "List")
		}
	}()

	var manufacturers []*models.Manufacturer
	for rows.Next() {
		mfr := &models.Manufacturer{}
		var website sql.NullString

		err := rows.Scan(
			&mfr.ID, &mfr.TenantID, &mfr.Name,
			&website,
			&mfr.IsVerified, &mfr.CreatedAt, &mfr.UpdatedAt,
			&mfr.ModelCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan manufacturer: %w", err)
		}

		if website.Valid {
			mfr.Website = &website.String
		}

		manufacturers = append(manufacturers, mfr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manufacturers: %w", err)
	}

	return manufacturers, nil
}

// Count returns the total count of manufacturers for a tenant within the transaction
func (r *transactionalManufacturerRepository) Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM manufacturers WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)`

	err := r.tx.QueryRowContext(ctx, query, tenantID, isSystem).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count manufacturers: %w", err)
	}

	return count, nil
}

// Update updates an existing manufacturer within the transaction
func (r *transactionalManufacturerRepository) Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.ManufacturerUpdateParams) error {
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

	result, err := r.tx.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return storage.ErrDuplicateKey
		}
		return fmt.Errorf("update manufacturer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// Delete deletes a manufacturer by ID within the isSystem-selected scope within the transaction
func (r *transactionalManufacturerRepository) Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error {
	query := `DELETE FROM manufacturers WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`

	result, err := r.tx.ExecContext(ctx, query, tenantID, isSystem, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("delete manufacturer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// SetVerified updates the is_verified flag for a manufacturer within the transaction
func (r *transactionalManufacturerRepository) SetVerified(ctx context.Context, tenantID int64, id uuid.UUID, verified bool) error {
	query := `UPDATE manufacturers SET is_verified = $3, updated_at = NOW() WHERE tenant_id = $1 AND id = $2`

	result, err := r.tx.ExecContext(ctx, query, tenantID, id, verified)
	if err != nil {
		return fmt.Errorf("set manufacturer verified: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}
