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
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// SQL fragment constants for query building
const orderByCreatedAtDesc = " ORDER BY created_at DESC"

// BlueprintRepository implements the BlueprintRepository interface for PostgreSQL
type BlueprintRepository struct {
	db *sqlx.DB
}

// NewBlueprintRepository creates a new PostgreSQL Blueprint repository
func NewBlueprintRepository(db *sqlx.DB) interfaces.BlueprintRepository {
	return &BlueprintRepository{db: db}
}

// Create creates a new blueprint
func (r *BlueprintRepository) Create(ctx context.Context, params *models.BlueprintCreateParams) (*models.Blueprint, error) {
	bp := &models.Blueprint{
		ID:            uuid.New(),
		DeviceModelID: params.DeviceModelID,
		TenantID:      params.TenantID,
		Version:       params.Version,
		TypeEUI:       params.TypeEUI,
		SpecJSON:      params.SpecJSON,
		IsDefault:     params.IsDefault,
	}

	// If this is the default, clear any existing defaults for the same model first
	if params.IsDefault {
		_, err := r.db.ExecContext(ctx,
			`UPDATE blueprints SET is_default = false, updated_at = NOW()
			 WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true`,
			params.TenantID, params.DeviceModelID)
		if err != nil {
			return nil, fmt.Errorf("clear existing default: %w", err)
		}
	}

	// Convert SpecJSON []byte to *string so lib/pq sends it as text format.
	// lib/pq sends []byte as PostgreSQL binary format which JSONB columns reject.
	var specJSONParam *string
	if len(bp.SpecJSON) > 0 {
		s := string(bp.SpecJSON)
		specJSONParam = &s
	}

	query := `
		INSERT INTO blueprints (
			id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
			registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			NOW(), NOW()
		) RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		bp.ID, bp.DeviceModelID, bp.TenantID, bp.Version, bp.TypeEUI,
		specJSONParam, bp.IsDefault,
		bp.RegistryRepo, bp.RegistryCommit, bp.RegistryVerified, bp.RegistryPRURL,
	).Scan(&bp.CreatedAt, &bp.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, storage.ErrDuplicateKey
		}
		return nil, fmt.Errorf("create blueprint: %w", err)
	}

	return bp, nil
}

// GetByID retrieves a blueprint by ID with tenant isolation
func (r *BlueprintRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Blueprint, error) {
	var bp models.Blueprint
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND id = $2`

	err := r.db.GetContext(ctx, &bp, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get blueprint: %w", err)
	}

	return &bp, nil
}

// GetByVersion retrieves a blueprint by device model ID and version
func (r *BlueprintRepository) GetByVersion(ctx context.Context, tenantID int64, deviceModelID uuid.UUID, version string) (*models.Blueprint, error) {
	var bp models.Blueprint
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND device_model_id = $2 AND version = $3`

	err := r.db.GetContext(ctx, &bp, query, tenantID, deviceModelID, version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get blueprint by version: %w", err)
	}

	return &bp, nil
}

// GetByTypeEUI retrieves a blueprint by Type EUI with tenant isolation
func (r *BlueprintRepository) GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.Blueprint, error) {
	var bp models.Blueprint
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND type_eui = $2`

	err := r.db.GetContext(ctx, &bp, query, tenantID, typeEUI)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get blueprint by type EUI: %w", err)
	}

	return &bp, nil
}

// GetDefaultForModel retrieves the default blueprint for a device model
func (r *BlueprintRepository) GetDefaultForModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (*models.Blueprint, error) {
	var bp models.Blueprint
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true`

	err := r.db.GetContext(ctx, &bp, query, tenantID, deviceModelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No default is not an error
		}
		return nil, fmt.Errorf("get default blueprint: %w", err)
	}

	return &bp, nil
}

// ListByDeviceModel retrieves blueprints for a device model with pagination
func (r *BlueprintRepository) ListByDeviceModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID, limit, offset int) ([]*models.Blueprint, error) {
	var blueprints []*models.Blueprint

	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND device_model_id = $2
		ORDER BY is_default DESC, version DESC`

	args := []interface{}{tenantID, deviceModelID}
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

	err := r.db.SelectContext(ctx, &blueprints, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list blueprints by device model: %w", err)
	}

	return blueprints, nil
}

// List retrieves blueprints for a tenant with pagination and optional filters
func (r *BlueprintRepository) List(ctx context.Context, params *models.BlueprintListParams) ([]*models.Blueprint, error) {
	var blueprints []*models.Blueprint

	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1`

	args := []interface{}{params.TenantID}
	argIndex := 2

	if params.DeviceModelID != nil {
		query += fmt.Sprintf(" AND device_model_id = $%d", argIndex)
		args = append(args, *params.DeviceModelID)
		argIndex++
	}

	query += orderByCreatedAtDesc

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, params.Limit)
		argIndex++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, params.Offset)
	}

	err := r.db.SelectContext(ctx, &blueprints, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list blueprints: %w", err)
	}

	return blueprints, nil
}

// ListWithModel retrieves blueprints with joined device model and manufacturer data
func (r *BlueprintRepository) ListWithModel(ctx context.Context, params *models.BlueprintListParams) ([]*models.BlueprintWithModel, error) {
	var blueprints []*models.BlueprintWithModel

	query := `
		SELECT bp.id, bp.device_model_id, bp.tenant_id, bp.version, bp.type_eui, bp.spec_json, bp.is_default,
		       bp.registry_repo, bp.registry_commit_sha, bp.registry_verified, bp.registry_pr_url,
		       bp.created_at, bp.updated_at,
		       dm.name AS device_model_name, dm.code AS device_model_code,
		       m.id AS manufacturer_id, m.name AS manufacturer_name
		FROM blueprints bp
		JOIN device_models dm ON bp.device_model_id = dm.id
		JOIN manufacturers m ON dm.manufacturer_id = m.id
		WHERE bp.tenant_id = $1`

	args := []interface{}{params.TenantID}
	argIndex := 2

	if params.DeviceModelID != nil {
		query += fmt.Sprintf(" AND bp.device_model_id = $%d", argIndex)
		args = append(args, *params.DeviceModelID)
		argIndex++
	}

	query += " ORDER BY bp.created_at DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, params.Limit)
		argIndex++
	}

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, params.Offset)
	}

	err := r.db.SelectContext(ctx, &blueprints, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list blueprints with model: %w", err)
	}

	return blueprints, nil
}

// Count returns the total count of blueprints for a tenant
func (r *BlueprintRepository) Count(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM blueprints WHERE tenant_id = $1`

	err := r.db.GetContext(ctx, &count, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("count blueprints: %w", err)
	}

	return count, nil
}

// CountByDeviceModel returns the count of blueprints for a device model
func (r *BlueprintRepository) CountByDeviceModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM blueprints WHERE tenant_id = $1 AND device_model_id = $2`

	err := r.db.GetContext(ctx, &count, query, tenantID, deviceModelID)
	if err != nil {
		return 0, fmt.Errorf("count blueprints by device model: %w", err)
	}

	return count, nil
}

// Update updates an existing blueprint
func (r *BlueprintRepository) Update(ctx context.Context, tenantID int64, id uuid.UUID, params *models.BlueprintUpdateParams) error {
	setClauses := make([]string, 0)
	args := []interface{}{tenantID, id}
	argIndex := 3

	if params.Version != nil {
		setClauses = append(setClauses, fmt.Sprintf("version = $%d", argIndex))
		args = append(args, *params.Version)
		argIndex++
	}

	if params.TypeEUI != nil {
		setClauses = append(setClauses, fmt.Sprintf("type_eui = $%d", argIndex))
		args = append(args, params.TypeEUI)
		argIndex++
	}

	if params.SpecJSON != nil {
		setClauses = append(setClauses, fmt.Sprintf("spec_json = $%d", argIndex))
		s := string(params.SpecJSON)
		args = append(args, &s)
		argIndex++
	}

	if params.IsDefault != nil {
		// If setting as default, clear other defaults for same model first
		if *params.IsDefault {
			// Get the device_model_id first
			var deviceModelID uuid.UUID
			err := r.db.GetContext(ctx,
				&deviceModelID,
				`SELECT device_model_id FROM blueprints WHERE tenant_id = $1 AND id = $2`,
				tenantID, id)
			if err != nil {
				return fmt.Errorf("get device model id: %w", err)
			}

			_, err = r.db.ExecContext(ctx,
				`UPDATE blueprints SET is_default = false, updated_at = NOW()
				 WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true AND id != $3`,
				tenantID, deviceModelID, id)
			if err != nil {
				return fmt.Errorf("clear existing default: %w", err)
			}
		}
		setClauses = append(setClauses, fmt.Sprintf("is_default = $%d", argIndex))
		args = append(args, *params.IsDefault)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(
		"UPDATE blueprints SET %s WHERE tenant_id = $1 AND id = $2",
		strings.Join(setClauses, ", "),
	)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return storage.ErrDuplicateKey
		}
		return fmt.Errorf("update blueprint: %w", err)
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

// SetDefault sets a blueprint as the default for its device model
func (r *BlueprintRepository) SetDefault(ctx context.Context, tenantID int64, id uuid.UUID) error {
	// Get the device_model_id first
	var deviceModelID uuid.UUID
	err := r.db.GetContext(ctx,
		&deviceModelID,
		`SELECT device_model_id FROM blueprints WHERE tenant_id = $1 AND id = $2`,
		tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("get device model id: %w", err)
	}

	// Clear any existing default for the model
	_, err = r.db.ExecContext(ctx,
		`UPDATE blueprints SET is_default = false, updated_at = NOW()
		 WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true`,
		tenantID, deviceModelID)
	if err != nil {
		return fmt.Errorf("clear existing default: %w", err)
	}

	// Set the new default
	result, err := r.db.ExecContext(ctx,
		`UPDATE blueprints SET is_default = true, updated_at = NOW()
		 WHERE tenant_id = $1 AND id = $2`,
		tenantID, id)
	if err != nil {
		return fmt.Errorf("set default: %w", err)
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

// ClearDefault clears the default flag for a blueprint
func (r *BlueprintRepository) ClearDefault(ctx context.Context, tenantID int64, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE blueprints SET is_default = false, updated_at = NOW()
		 WHERE tenant_id = $1 AND id = $2`,
		tenantID, id)
	if err != nil {
		return fmt.Errorf("clear default: %w", err)
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

// UpdateRegistryInfo updates the GitHub registry metadata for a blueprint
func (r *BlueprintRepository) UpdateRegistryInfo(ctx context.Context, tenantID int64, id uuid.UUID, repo, commitSHA, prURL string, verified bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE blueprints
		 SET registry_repo = $3, registry_commit_sha = $4, registry_pr_url = $5,
		     registry_verified = $6, updated_at = NOW()
		 WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, repo, commitSHA, prURL, verified)
	if err != nil {
		return fmt.Errorf("update registry info: %w", err)
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

// Delete deletes a blueprint by ID with tenant isolation
func (r *BlueprintRepository) Delete(ctx context.Context, tenantID int64, id uuid.UUID) error {
	query := `DELETE FROM blueprints WHERE tenant_id = $1 AND id = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete blueprint: %w", err)
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
