package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// SQL fragment constants for query building
const orderByCreatedAtDesc = " ORDER BY created_at DESC"

// BlueprintRepository implements the BlueprintRepository interface for PostgreSQL
type BlueprintRepository struct {
	db sqlx.ExtContext
}

// NewBlueprintRepository creates a new PostgreSQL Blueprint repository
func NewBlueprintRepository(db sqlx.ExtContext) *BlueprintRepository {
	return &BlueprintRepository{db: db}
}

// Create creates a new blueprint
func (r *BlueprintRepository) Create(ctx context.Context, params *models.BlueprintCreateParams) (*models.Blueprint, error) {
	bp := &models.Blueprint{
		ID:            uuid.New(),
		DeviceModelID: params.DeviceModelID,
		IsSystem:      params.IsSystem,
		Version:       params.Version,
		TypeEUI:       params.TypeEUI,
		SpecJSON:      params.SpecJSON,
		IsDefault:     params.IsDefault,
	}
	if !params.IsSystem {
		bp.TenantID = &params.TenantID
	}

	// Demote the current default within the same ownership.
	if params.IsDefault {
		var err error
		if params.IsSystem {
			_, err = r.db.ExecContext(ctx,
				`UPDATE blueprints SET is_default = false, updated_at = NOW()
				 WHERE is_system AND device_model_id = $1 AND is_default = true`,
				params.DeviceModelID)
		} else {
			_, err = r.db.ExecContext(ctx,
				`UPDATE blueprints SET is_default = false, updated_at = NOW()
				 WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true`,
				params.TenantID, params.DeviceModelID)
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapClearExistingDefault, err)
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
			id, device_model_id, tenant_id, is_system, version, type_eui, spec_json, is_default,
			registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12,
			NOW(), NOW()
		) RETURNING created_at, updated_at`

	err := r.db.QueryRowxContext(
		ctx, query,
		bp.ID, bp.DeviceModelID, bp.TenantID, bp.IsSystem, bp.Version, bp.TypeEUI,
		specJSONParam, bp.IsDefault,
		bp.RegistryRepo, bp.RegistryCommit, bp.RegistryVerified, bp.RegistryPRURL,
	).Scan(&bp.CreatedAt, &bp.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeUniqueViolation {
			return nil, storage.ErrDuplicateKey
		}
		return nil, fmt.Errorf("%s: %w", errWrapCreateBlueprint, err)
	}

	return bp, nil
}

// GetByID retrieves a blueprint by ID with tenant isolation
func (r *BlueprintRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Blueprint, error) {
	var bp models.Blueprint
	query := `
		SELECT id, device_model_id, tenant_id, is_system, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE (tenant_id = $1 OR is_system) AND id = $2`

	err := sqlx.GetContext(ctx, r.db, &bp, query, tenantID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrRecordNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetBlueprint, err)
	}

	return &bp, nil
}

// GetByTypeEUI retrieves a blueprint by Type EUI with tenant isolation
func (r *BlueprintRepository) GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.Blueprint, error) {
	var bp models.Blueprint
	query := `
		SELECT id, device_model_id, tenant_id, is_system, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE (tenant_id = $1 OR is_system) AND type_eui = $2
		ORDER BY (tenant_id IS NOT DISTINCT FROM $1) DESC, is_default DESC, created_at DESC
		LIMIT 1`

	err := sqlx.GetContext(ctx, r.db, &bp, query, tenantID, typeEUI)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrRecordNotFound
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetBlueprintByTypeEUI, err)
	}

	return &bp, nil
}

// GetDefaultForModel retrieves the default blueprint for a device model
func (r *BlueprintRepository) GetDefaultForModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (*models.Blueprint, error) {
	var bp models.Blueprint
	query := `
		SELECT id, device_model_id, tenant_id, is_system, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE (tenant_id = $1 OR is_system) AND device_model_id = $2 AND is_default = true`

	err := sqlx.GetContext(ctx, r.db, &bp, query, tenantID, deviceModelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No default is not an error
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetDefaultBlueprint, err)
	}

	return &bp, nil
}

// List retrieves blueprints for a tenant with pagination and optional filters
func (r *BlueprintRepository) List(ctx context.Context, params *models.BlueprintListParams) ([]*models.Blueprint, error) {
	var blueprints []*models.Blueprint

	query := `
		SELECT id, device_model_id, tenant_id, is_system, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)`

	args := []interface{}{params.TenantID, params.IsSystem}
	argIndex := 3

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
		query += fmt.Sprintf(" OFFSET $%d", argIndex) //nolint:gosec // G202: appends a parameter placeholder, values are bound
		args = append(args, params.Offset)
	}

	err := sqlx.SelectContext(ctx, r.db, &blueprints, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListBlueprints, err)
	}

	return blueprints, nil
}

// Count returns the total count of blueprints for a tenant
func (r *BlueprintRepository) Count(ctx context.Context, tenantID int64, isSystem bool) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM blueprints WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END)`

	err := sqlx.GetContext(ctx, r.db, &count, query, tenantID, isSystem)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountBlueprints, err)
	}

	return count, nil
}

// CountByDeviceModel returns the count of blueprints for a device model
func (r *BlueprintRepository) CountByDeviceModel(ctx context.Context, tenantID int64, isSystem bool, deviceModelID uuid.UUID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM blueprints WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND device_model_id = $3`

	err := sqlx.GetContext(ctx, r.db, &count, query, tenantID, isSystem, deviceModelID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapCountBlueprintsByDeviceModel, err)
	}

	return count, nil
}

// Update updates an existing blueprint
func (r *BlueprintRepository) Update(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, params *models.BlueprintUpdateParams) error {
	setClauses := make([]string, 0)
	args := []interface{}{tenantID, isSystem, id}
	argIndex := 4

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
		// If setting as default, clear other defaults for same model within the same ownership first
		if *params.IsDefault {
			var deviceModelID uuid.UUID
			err := sqlx.GetContext(ctx, r.db, &deviceModelID,
				`SELECT device_model_id FROM blueprints WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`,
				tenantID, isSystem, id)
			if err != nil {
				return fmt.Errorf("%s: %w", errWrapGetDeviceModelID, err)
			}

			_, err = r.db.ExecContext(ctx,
				`UPDATE blueprints SET is_default = false, updated_at = NOW()
				 WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND device_model_id = $3 AND is_default = true AND id != $4`,
				tenantID, isSystem, deviceModelID, id)
			if err != nil {
				return fmt.Errorf("%s: %w", errWrapClearExistingDefault, err)
			}
		}
		setClauses = append(setClauses, fmt.Sprintf("is_default = $%d", argIndex))
		args = append(args, *params.IsDefault)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	// #nosec G201 -- setClauses are constructed from safe column names with parameterized values
	query := fmt.Sprintf(
		"UPDATE blueprints SET %s WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3",
		strings.Join(setClauses, ", "),
	)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqCodeUniqueViolation {
			return storage.ErrDuplicateKey
		}
		return fmt.Errorf("%s: %w", errWrapUpdateBlueprint, err)
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

// SetDefault sets a blueprint as the default for its device model
func (r *BlueprintRepository) SetDefault(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error {
	// Get the device_model_id first
	var deviceModelID uuid.UUID
	err := sqlx.GetContext(ctx, r.db, &deviceModelID,
		`SELECT device_model_id FROM blueprints WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`,
		tenantID, isSystem, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrRecordNotFound
		}
		return fmt.Errorf("%s: %w", errWrapGetDeviceModelID, err)
	}

	// Clear any existing default for the model within the same ownership
	_, err = r.db.ExecContext(ctx,
		`UPDATE blueprints SET is_default = false, updated_at = NOW()
		 WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND device_model_id = $3 AND is_default = true`,
		tenantID, isSystem, deviceModelID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapClearExistingDefault, err)
	}

	// Set the new default
	result, err := r.db.ExecContext(ctx,
		`UPDATE blueprints SET is_default = true, updated_at = NOW()
		 WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`,
		tenantID, isSystem, id)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapSetDefault, err)
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

// UpdateRegistryInfo updates the GitHub registry metadata for a blueprint
func (r *BlueprintRepository) UpdateRegistryInfo(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID, repo, commitSHA, prURL string, verified bool) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE blueprints
		 SET registry_repo = $4, registry_commit_sha = $5, registry_pr_url = $6,
		     registry_verified = $7, updated_at = NOW()
		 WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`,
		tenantID, isSystem, id, repo, commitSHA, prURL, verified)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapUpdateRegistryInfo, err)
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

// Delete deletes a blueprint by ID within the isSystem-selected ownership scope
func (r *BlueprintRepository) Delete(ctx context.Context, tenantID int64, isSystem bool, id uuid.UUID) error {
	query := `DELETE FROM blueprints WHERE (CASE WHEN $2 THEN is_system ELSE tenant_id = $1 END) AND id = $3`

	result, err := r.db.ExecContext(ctx, query, tenantID, isSystem, id)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapDeleteBlueprint, err)
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
