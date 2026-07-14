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

// transactionalDeviceModelRepository implements interfaces.DeviceModelRepository within a transaction
type transactionalDeviceModelRepository struct {
	tx *sql.Tx
	db *DB
}

// Ensure interface compliance
var _ interfaces.DeviceModelRepository = (*transactionalDeviceModelRepository)(nil)

// Create creates a new device model within the transaction
func (r *transactionalDeviceModelRepository) Create(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error) {
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
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		) RETURNING created_at, updated_at`

	err := r.tx.QueryRowContext(ctx, query,
		model.ID, model.ManufacturerID, model.TenantID, model.IsSystem, model.Name, model.Code,
		model.TypeEUI, model.Description, model.DatasheetURL,
	).Scan(&model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, storage.ErrDuplicateKey
		}
		return nil, fmt.Errorf("create device model: %w", err)
	}

	return model, nil
}

// GetByID retrieves a device model by ID with tenant isolation within the transaction
func (r *transactionalDeviceModelRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error) {
	query := `
		SELECT id, manufacturer_id, tenant_id, name, code, type_eui, description, datasheet_url, created_at, updated_at
		FROM device_models
		WHERE tenant_id = $1 AND id = $2`

	model := &models.DeviceModel{}
	var typeEUI []byte
	var description, datasheetURL sql.NullString

	err := r.tx.QueryRowContext(ctx, query, tenantID, id).Scan(
		&model.ID, &model.ManufacturerID, &model.TenantID, &model.Name, &model.Code,
		&typeEUI, &description, &datasheetURL,
		&model.CreatedAt, &model.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get device model: %w", err)
	}

	model.TypeEUI = typeEUI
	if description.Valid {
		model.Description = &description.String
	}
	if datasheetURL.Valid {
		model.DatasheetURL = &datasheetURL.String
	}

	return model, nil
}

// GetByCode retrieves a device model by manufacturer ID and code within the transaction
func (r *transactionalDeviceModelRepository) GetByCode(ctx context.Context, tenantID int64, manufacturerID uuid.UUID, code string) (*models.DeviceModel, error) {
	query := `
		SELECT id, manufacturer_id, tenant_id, name, code, type_eui, description, datasheet_url, created_at, updated_at
		FROM device_models
		WHERE tenant_id = $1 AND manufacturer_id = $2 AND code = $3`

	model := &models.DeviceModel{}
	var typeEUI []byte
	var description, datasheetURL sql.NullString

	err := r.tx.QueryRowContext(ctx, query, tenantID, manufacturerID, code).Scan(
		&model.ID, &model.ManufacturerID, &model.TenantID, &model.Name, &model.Code,
		&typeEUI, &description, &datasheetURL,
		&model.CreatedAt, &model.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get device model by code: %w", err)
	}

	model.TypeEUI = typeEUI
	if description.Valid {
		model.Description = &description.String
	}
	if datasheetURL.Valid {
		model.DatasheetURL = &datasheetURL.String
	}

	return model, nil
}

// GetByTypeEUI retrieves a device model by Type EUI with tenant isolation within the transaction
func (r *transactionalDeviceModelRepository) GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.DeviceModel, error) {
	query := `
		SELECT id, manufacturer_id, tenant_id, name, code, type_eui, description, datasheet_url, created_at, updated_at
		FROM device_models
		WHERE tenant_id = $1 AND type_eui = $2`

	model := &models.DeviceModel{}
	var typeEUIBytes []byte
	var description, datasheetURL sql.NullString

	err := r.tx.QueryRowContext(ctx, query, tenantID, typeEUI).Scan(
		&model.ID, &model.ManufacturerID, &model.TenantID, &model.Name, &model.Code,
		&typeEUIBytes, &description, &datasheetURL,
		&model.CreatedAt, &model.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get device model by type EUI: %w", err)
	}

	model.TypeEUI = typeEUIBytes
	if description.Valid {
		model.Description = &description.String
	}
	if datasheetURL.Valid {
		model.DatasheetURL = &datasheetURL.String
	}

	return model, nil
}

// ListByManufacturer retrieves device models for a manufacturer with pagination within the transaction
func (r *transactionalDeviceModelRepository) ListByManufacturer(ctx context.Context, tenantID int64, manufacturerID uuid.UUID, limit, offset int) ([]*models.DeviceModel, error) {
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
		WHERE dm.tenant_id = $1 AND dm.manufacturer_id = $2
		ORDER BY LOWER(name) ASC, name ASC, id ASC`

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

	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list device models by manufacturer: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "ListByManufacturer")
		}
	}()

	return r.scanDeviceModels(rows)
}

// List retrieves device models for a tenant with pagination and optional filters within the transaction
func (r *transactionalDeviceModelRepository) List(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModel, error) {
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
		return nil, fmt.Errorf("list device models: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "List")
		}
	}()

	return r.scanDeviceModels(rows)
}

// ListWithManufacturer retrieves device models with joined manufacturer data within the transaction
func (r *transactionalDeviceModelRepository) ListWithManufacturer(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModelWithManufacturer, error) {
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

	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list device models with manufacturer: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "ListWithManufacturer")
		}
	}()

	var deviceModels []*models.DeviceModelWithManufacturer
	for rows.Next() {
		model := &models.DeviceModelWithManufacturer{}
		var typeEUI []byte
		var description, datasheetURL sql.NullString

		err := rows.Scan(
			&model.ID, &model.ManufacturerID, &model.TenantID, &model.Name, &model.Code,
			&typeEUI, &description, &datasheetURL,
			&model.CreatedAt, &model.UpdatedAt,
			&model.ManufacturerName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan device model with manufacturer: %w", err)
		}

		model.TypeEUI = typeEUI
		if description.Valid {
			model.Description = &description.String
		}
		if datasheetURL.Valid {
			model.DatasheetURL = &datasheetURL.String
		}

		deviceModels = append(deviceModels, model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device models: %w", err)
	}

	return deviceModels, nil
}

// Count returns the total count of device models for a tenant within the transaction
func (r *transactionalDeviceModelRepository) Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM device_models WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)`

	err := r.tx.QueryRowContext(ctx, query, tenantID, isSystem).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count device models: %w", err)
	}

	return count, nil
}

// CountByManufacturer returns the count of device models for a manufacturer within the transaction
func (r *transactionalDeviceModelRepository) CountByManufacturer(ctx context.Context, tenantID int64, isSystem bool, manufacturerID uuid.UUID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM device_models WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND manufacturer_id = $3`

	err := r.tx.QueryRowContext(ctx, query, tenantID, isSystem, manufacturerID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count device models by manufacturer: %w", err)
	}

	return count, nil
}

// Update updates an existing device model within the transaction
func (r *transactionalDeviceModelRepository) Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.DeviceModelUpdateParams) error {
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

	result, err := r.tx.ExecContext(ctx, query, args...)
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

// Delete deletes a device model by ID within the isSystem-selected scope within the transaction
func (r *transactionalDeviceModelRepository) Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error {
	query := `DELETE FROM device_models WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`

	result, err := r.tx.ExecContext(ctx, query, tenantID, isSystem, id)
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

// scanDeviceModels is a helper to scan rows into device model structs
func (r *transactionalDeviceModelRepository) scanDeviceModels(rows *sql.Rows) ([]*models.DeviceModel, error) {
	var deviceModels []*models.DeviceModel
	for rows.Next() {
		model := &models.DeviceModel{}
		var typeEUI []byte
		var description, datasheetURL sql.NullString

		err := rows.Scan(
			&model.ID, &model.ManufacturerID, &model.TenantID, &model.Name, &model.Code,
			&typeEUI, &description, &datasheetURL,
			&model.CreatedAt, &model.UpdatedAt,
			&model.BlueprintCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan device model: %w", err)
		}

		model.TypeEUI = typeEUI
		if description.Valid {
			model.Description = &description.String
		}
		if datasheetURL.Valid {
			model.DatasheetURL = &datasheetURL.String
		}

		deviceModels = append(deviceModels, model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device models: %w", err)
	}

	return deviceModels, nil
}
