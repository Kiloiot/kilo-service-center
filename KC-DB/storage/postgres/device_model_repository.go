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
	"github.com/lib/pq"
)

// DeviceModelRepository implements the DeviceModelRepository interface for PostgreSQL
type DeviceModelRepository struct {
	db *sqlx.DB
}

// NewDeviceModelRepository creates a new PostgreSQL DeviceModel repository
func NewDeviceModelRepository(db *sqlx.DB) interfaces.DeviceModelRepository {
	return &DeviceModelRepository{db: db}
}

// Create creates a new device model
func (r *DeviceModelRepository) Create(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error) {
	model := &models.DeviceModel{
		ID:             uuid.New(),
		ManufacturerID: params.ManufacturerID,
		TenantID:       params.TenantID,
		Name:           params.Name,
		Code:           params.Code,
		TypeEUI:        params.TypeEUI,
		Description:    params.Description,
		DatasheetURL:   params.DatasheetURL,
	}

	query := `
		INSERT INTO device_models (
			id, manufacturer_id, tenant_id, name, code, type_eui, description, datasheet_url, created_at, updated_at
		) VALUES (
			:id, :manufacturer_id, :tenant_id, :name, :code, :type_eui, :description, :datasheet_url, NOW(), NOW()
		) RETURNING created_at, updated_at`

	stmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Printf("failed to close statement in device_model repository: %v", err)
		}
	}()

	err = stmt.QueryRowxContext(ctx, model).Scan(&model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, storage.ErrDuplicateKey
		}
		return nil, fmt.Errorf("create device model: %w", err)
	}

	return model, nil
}

// GetByID retrieves a device model by ID with tenant isolation
func (r *DeviceModelRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error) {
	var model models.DeviceModel
	query := `
		SELECT id, manufacturer_id, tenant_id, name, code, type_eui, description, datasheet_url, created_at, updated_at
		FROM device_models
		WHERE tenant_id = $1 AND id = $2`

	err := r.db.GetContext(ctx, &model, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get device model: %w", err)
	}

	return &model, nil
}

// GetByCode retrieves a device model by manufacturer ID and code
func (r *DeviceModelRepository) GetByCode(ctx context.Context, tenantID int64, manufacturerID uuid.UUID, code string) (*models.DeviceModel, error) {
	var model models.DeviceModel
	query := `
		SELECT id, manufacturer_id, tenant_id, name, code, type_eui, description, datasheet_url, created_at, updated_at
		FROM device_models
		WHERE tenant_id = $1 AND manufacturer_id = $2 AND code = $3`

	err := r.db.GetContext(ctx, &model, query, tenantID, manufacturerID, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get device model by code: %w", err)
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

	err := r.db.GetContext(ctx, &model, query, tenantID, typeEUI)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get device model by type EUI: %w", err)
	}

	return &model, nil
}

// ListByManufacturer retrieves device models for a manufacturer with pagination
func (r *DeviceModelRepository) ListByManufacturer(ctx context.Context, tenantID int64, manufacturerID uuid.UUID, limit, offset int) ([]*models.DeviceModel, error) {
	var models []*models.DeviceModel

	query := `
		SELECT
			dm.id, dm.manufacturer_id, dm.tenant_id, dm.name, dm.code, dm.type_eui, dm.description, dm.datasheet_url, dm.created_at, dm.updated_at,
			COALESCE(bp.blueprint_count, 0) AS blueprint_count
		FROM device_models dm
		LEFT JOIN (
			SELECT device_model_id, COUNT(*) AS blueprint_count
			FROM blueprints
			WHERE tenant_id = $1
			GROUP BY device_model_id
		) bp ON bp.device_model_id = dm.id
		WHERE dm.tenant_id = $1 AND dm.manufacturer_id = $2` + orderByNameAsc

	args := []interface{}{tenantID, manufacturerID}
	argIndex := 3

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	err := r.db.SelectContext(ctx, &models, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list device models by manufacturer: %w", err)
	}

	return models, nil
}

// List retrieves device models for a tenant with pagination and optional filters
func (r *DeviceModelRepository) List(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModel, error) {
	var deviceModels []*models.DeviceModel

	query := `
		SELECT
			dm.id, dm.manufacturer_id, dm.tenant_id, dm.name, dm.code, dm.type_eui, dm.description, dm.datasheet_url, dm.created_at, dm.updated_at,
			COALESCE(bp.blueprint_count, 0) AS blueprint_count
		FROM device_models dm
		LEFT JOIN (
			SELECT device_model_id, COUNT(*) AS blueprint_count
			FROM blueprints
			WHERE tenant_id = $1
			GROUP BY device_model_id
		) bp ON bp.device_model_id = dm.id
		WHERE dm.tenant_id = $1`

	args := []interface{}{params.TenantID}
	argIndex := 2

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
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, params.Offset)
	}

	err := r.db.SelectContext(ctx, &deviceModels, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list device models: %w", err)
	}

	return deviceModels, nil
}

// ListWithManufacturer retrieves device models with joined manufacturer data
func (r *DeviceModelRepository) ListWithManufacturer(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModelWithManufacturer, error) {
	var deviceModels []*models.DeviceModelWithManufacturer

	query := `
		SELECT dm.id, dm.manufacturer_id, dm.tenant_id, dm.name, dm.code, dm.type_eui,
		       dm.description, dm.datasheet_url, dm.created_at, dm.updated_at,
		       m.name AS manufacturer_name
		FROM device_models dm
		JOIN manufacturers m ON dm.manufacturer_id = m.id
		WHERE dm.tenant_id = $1`

	args := []interface{}{params.TenantID}
	argIndex := 2

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

	query += " ORDER BY LOWER(dm.name) ASC, dm.name ASC, dm.id ASC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, params.Limit)
		argIndex++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, params.Offset)
	}

	err := r.db.SelectContext(ctx, &deviceModels, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list device models with manufacturer: %w", err)
	}

	return deviceModels, nil
}

// Count returns the total count of device models for a tenant
func (r *DeviceModelRepository) Count(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM device_models WHERE tenant_id = $1`

	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("count device models: %w", err)
	}

	return count, nil
}

// CountByManufacturer returns the count of device models for a manufacturer
func (r *DeviceModelRepository) CountByManufacturer(ctx context.Context, tenantID int64, manufacturerID uuid.UUID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM device_models WHERE tenant_id = $1 AND manufacturer_id = $2`

	err := r.db.GetContext(ctx, &count, query, tenantID, manufacturerID)
	if err != nil {
		return 0, fmt.Errorf("count device models by manufacturer: %w", err)
	}

	return count, nil
}

// Update updates an existing device model
func (r *DeviceModelRepository) Update(ctx context.Context, tenantID int64, id uuid.UUID, params *models.DeviceModelUpdateParams) error {
	setClauses := make([]string, 0)
	args := []interface{}{tenantID, id}
	argIndex := 3

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

	query := fmt.Sprintf(
		"UPDATE device_models SET %s WHERE tenant_id = $1 AND id = $2",
		strings.Join(setClauses, ", "),
	)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return storage.ErrDuplicateKey
		}
		return fmt.Errorf("update device model: %w", err)
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

// Delete deletes a device model by ID with tenant isolation
func (r *DeviceModelRepository) Delete(ctx context.Context, tenantID int64, id uuid.UUID) error {
	query := `DELETE FROM device_models WHERE tenant_id = $1 AND id = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return storage.ErrForeignKeyViolation
		}
		return fmt.Errorf("delete device model: %w", err)
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
