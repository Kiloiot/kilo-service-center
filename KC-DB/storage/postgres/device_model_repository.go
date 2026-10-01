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

// DeviceModelRepository implements the DeviceModelRepository interface for PostgreSQL
type DeviceModelRepository struct {
	log logger.Logger
	db  sqlx.ExtContext
}

// NewDeviceModelRepository creates a new PostgreSQL DeviceModel repository
func NewDeviceModelRepository(db sqlx.ExtContext, log logger.Logger) *DeviceModelRepository {
	return &DeviceModelRepository{
		log: log, db: db}
}

// Create creates a new device model
func (r *DeviceModelRepository) Create(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error) {
	model := &models.DeviceModel{
		ID:             uuid.New(),
		ManufacturerID: params.ManufacturerID,
		IsSystem:       params.IsSystem,
		Name:           params.Name,
		Code:           params.Code,
		TypeEUI:        params.TypeEUI,
		Description:    params.Description,
		DatasheetURL:   params.DatasheetURL,
	}
	if !params.IsSystem {
		model.TenantID = &params.TenantID
	}

	query := `
		INSERT INTO device_models (
			id, manufacturer_id, tenant_id, is_system, name, code, type_eui, description, datasheet_url, created_at, updated_at
		) VALUES (
			:id, :manufacturer_id, :tenant_id, :is_system, :name, :code, :type_eui, :description, :datasheet_url, NOW(), NOW()
		) RETURNING created_at, updated_at`

	rows, err := sqlx.NamedQueryContext(ctx, r.db, query, model)
	if err == nil {
		defer func() {
			if closeErr := rows.Close(); closeErr != nil {
				r.log.Warn(logMsgCloseStmtDeviceModels, logger.FieldError, closeErr)
			}
		}()
		if !rows.Next() {
			err = sql.ErrNoRows
		} else {
			err = rows.Scan(&model.CreatedAt, &model.UpdatedAt)
		}
	}
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeUniqueViolation {
			return nil, storage.ErrDuplicateKey
		}
		return nil, fmt.Errorf("%s: %w", errWrapCreateDeviceModel, err)
	}

	return model, nil
}

// GetByID retrieves a device model by ID with tenant isolation
func (r *DeviceModelRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error) {
	var model models.DeviceModel
	// Resolve by id across both ownership scopes (tenant Custom + globally-visible System).
	query := `
		SELECT id, manufacturer_id, tenant_id, is_system, name, code, type_eui, description, datasheet_url, created_at, updated_at
		FROM device_models
		WHERE (tenant_id = $1 OR is_system) AND id = $2`

	err := sqlx.GetContext(ctx, r.db, &model, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrRecordNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetDeviceModel, err)
	}

	return &model, nil
}

// GetByTypeEUI retrieves a device model by Type EUI with tenant isolation
func (r *DeviceModelRepository) GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.DeviceModel, error) {
	var model models.DeviceModel
	query := `
		SELECT id, manufacturer_id, tenant_id, name, code, type_eui, description, datasheet_url, created_at, updated_at
		FROM device_models
		WHERE tenant_id = $1 AND type_eui = $2`

	err := sqlx.GetContext(ctx, r.db, &model, query, tenantID, typeEUI)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrRecordNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetDeviceModelByTypeEUI, err)
	}

	return &model, nil
}

// List retrieves device models for a tenant with pagination and optional filters
func (r *DeviceModelRepository) List(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModel, error) {
	var deviceModels []*models.DeviceModel

	// Scope by is_system via a CASE so $1/$2 stay bound regardless of world — no arg renumbering.
	query := `
		SELECT
			dm.id, dm.manufacturer_id, dm.tenant_id, dm.is_system, dm.name, dm.code, dm.type_eui, dm.description, dm.datasheet_url, dm.created_at, dm.updated_at,
			COALESCE(bp.blueprint_count, 0) AS blueprint_count
		FROM device_models dm
		LEFT JOIN (
			SELECT device_model_id, COUNT(*) AS blueprint_count
			FROM blueprints
			WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)
			GROUP BY device_model_id
		) bp ON bp.device_model_id = dm.id
		WHERE (CASE WHEN $2 THEN dm.is_system ELSE dm.tenant_id = $1 END)`

	args := []interface{}{params.TenantID, params.IsSystem}
	argIndex := 3

	if params.ManufacturerID != nil {
		query += fmt.Sprintf(" AND dm.manufacturer_id = $%d", argIndex)
		args = append(args, *params.ManufacturerID)
		argIndex++
	}

	if params.SearchTerm != "" {
		query += fmt.Sprintf(" AND (dm.name ILIKE $%d OR dm.code ILIKE $%d)", argIndex, argIndex)
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

	err := sqlx.SelectContext(ctx, r.db, &deviceModels, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListDeviceModels, err)
	}

	return deviceModels, nil
}

// Count returns the total count of device models for a tenant
func (r *DeviceModelRepository) Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM device_models WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)`

	err := sqlx.GetContext(ctx, r.db, &count, query, tenantID, isSystem)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountDeviceModels, err)
	}

	return count, nil
}

// CountByManufacturer returns the count of device models for a manufacturer
func (r *DeviceModelRepository) CountByManufacturer(ctx context.Context, tenantID int64, isSystem bool, manufacturerID uuid.UUID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM device_models WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND manufacturer_id = $3`

	err := sqlx.GetContext(ctx, r.db, &count, query, tenantID, isSystem, manufacturerID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountDeviceModelsByManufacturer, err)
	}

	return count, nil
}

// Update updates an existing device model
func (r *DeviceModelRepository) Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.DeviceModelUpdateParams) error {
	setClauses := make([]string, 0)
	args := []interface{}{tenantID, isSystem, id}
	argIndex := 4

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *params.Name)
		argIndex++
	}

	if params.Code != nil {
		setClauses = append(setClauses, fmt.Sprintf("code = $%d", argIndex))
		args = append(args, *params.Code)
		argIndex++
	}

	// TypeEUI can be set to nil to clear it
	if params.TypeEUI != nil {
		setClauses = append(setClauses, fmt.Sprintf("type_eui = $%d", argIndex))
		args = append(args, params.TypeEUI)
		argIndex++
	}

	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *params.Description)
		argIndex++
	}

	if params.DatasheetURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("datasheet_url = $%d", argIndex))
		args = append(args, *params.DatasheetURL)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	// #nosec G201 -- setClauses are constructed from safe column names with parameterized values
	query := fmt.Sprintf(
		"UPDATE device_models SET %s WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3",
		strings.Join(setClauses, ", "),
	)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeUniqueViolation {
			return storage.ErrDuplicateKey
		}
		return fmt.Errorf("%s: %w", errWrapUpdateDeviceModel, err)
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

// Delete deletes a device model by ID within the isSystem-selected ownership scope
func (r *DeviceModelRepository) Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error {
	query := `DELETE FROM device_models WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`

	result, err := r.db.ExecContext(ctx, query, tenantID, isSystem, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeForeignKeyViolation {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("%s: %w", errWrapDeleteDeviceModel, err)
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
