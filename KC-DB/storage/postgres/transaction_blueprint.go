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

// transactionalBlueprintRepository implements interfaces.BlueprintRepository within a transaction
type transactionalBlueprintRepository struct {
	tx *sql.Tx
	db *DB
}

// Ensure interface compliance
var _ interfaces.BlueprintRepository = (*transactionalBlueprintRepository)(nil)

// Create creates a new blueprint within the transaction
func (r *transactionalBlueprintRepository) Create(ctx context.Context, params *models.BlueprintCreateParams) (*models.Blueprint, error) {
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
		_, err := r.tx.ExecContext(ctx,
			`UPDATE blueprints SET is_default = false, updated_at = NOW()
			 WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true`,
			params.TenantID, params.DeviceModelID)
		if err != nil {
			return nil, fmt.Errorf("clear existing default: %w", err)
		}
	}

	query := `
		INSERT INTO blueprints (
			id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
			registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
		) RETURNING created_at, updated_at`

	err := r.tx.QueryRowContext(ctx, query,
		bp.ID, bp.DeviceModelID, bp.TenantID, bp.Version, bp.TypeEUI, bp.SpecJSON, bp.IsDefault,
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

// GetByID retrieves a blueprint by ID with tenant isolation within the transaction
func (r *transactionalBlueprintRepository) GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Blueprint, error) {
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND id = $2`

	bp := &models.Blueprint{}
	var typeEUI, specJSON []byte
	var registryRepo, registryCommitSHA, registryPRURL sql.NullString
	var registryVerified sql.NullBool

	err := r.tx.QueryRowContext(ctx, query, tenantID, id).Scan(
		&bp.ID, &bp.DeviceModelID, &bp.TenantID, &bp.Version, &typeEUI, &specJSON, &bp.IsDefault,
		&registryRepo, &registryCommitSHA, &registryVerified, &registryPRURL,
		&bp.CreatedAt, &bp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get blueprint: %w", err)
	}

	bp.TypeEUI = typeEUI
	bp.SpecJSON = specJSON
	if registryRepo.Valid {
		bp.RegistryRepo = &registryRepo.String
	}
	if registryCommitSHA.Valid {
		bp.RegistryCommit = &registryCommitSHA.String
	}
	if registryPRURL.Valid {
		bp.RegistryPRURL = &registryPRURL.String
	}
	if registryVerified.Valid {
		bp.RegistryVerified = registryVerified.Bool
	}

	return bp, nil
}

// GetByVersion retrieves a blueprint by device model ID and version within the transaction
func (r *transactionalBlueprintRepository) GetByVersion(ctx context.Context, tenantID int64, deviceModelID uuid.UUID, version string) (*models.Blueprint, error) {
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND device_model_id = $2 AND version = $3`

	bp := &models.Blueprint{}
	var typeEUI, specJSON []byte
	var registryRepo, registryCommitSHA, registryPRURL sql.NullString
	var registryVerified sql.NullBool

	err := r.tx.QueryRowContext(ctx, query, tenantID, deviceModelID, version).Scan(
		&bp.ID, &bp.DeviceModelID, &bp.TenantID, &bp.Version, &typeEUI, &specJSON, &bp.IsDefault,
		&registryRepo, &registryCommitSHA, &registryVerified, &registryPRURL,
		&bp.CreatedAt, &bp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get blueprint by version: %w", err)
	}

	bp.TypeEUI = typeEUI
	bp.SpecJSON = specJSON
	if registryRepo.Valid {
		bp.RegistryRepo = &registryRepo.String
	}
	if registryCommitSHA.Valid {
		bp.RegistryCommit = &registryCommitSHA.String
	}
	if registryPRURL.Valid {
		bp.RegistryPRURL = &registryPRURL.String
	}
	if registryVerified.Valid {
		bp.RegistryVerified = registryVerified.Bool
	}

	return bp, nil
}

// GetByTypeEUI retrieves a blueprint by Type EUI with tenant isolation within the transaction
func (r *transactionalBlueprintRepository) GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.Blueprint, error) {
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND type_eui = $2`

	bp := &models.Blueprint{}
	var typeEUIBytes, specJSON []byte
	var registryRepo, registryCommitSHA, registryPRURL sql.NullString
	var registryVerified sql.NullBool

	err := r.tx.QueryRowContext(ctx, query, tenantID, typeEUI).Scan(
		&bp.ID, &bp.DeviceModelID, &bp.TenantID, &bp.Version, &typeEUIBytes, &specJSON, &bp.IsDefault,
		&registryRepo, &registryCommitSHA, &registryVerified, &registryPRURL,
		&bp.CreatedAt, &bp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("get blueprint by type EUI: %w", err)
	}

	bp.TypeEUI = typeEUIBytes
	bp.SpecJSON = specJSON
	if registryRepo.Valid {
		bp.RegistryRepo = &registryRepo.String
	}
	if registryCommitSHA.Valid {
		bp.RegistryCommit = &registryCommitSHA.String
	}
	if registryPRURL.Valid {
		bp.RegistryPRURL = &registryPRURL.String
	}
	if registryVerified.Valid {
		bp.RegistryVerified = registryVerified.Bool
	}

	return bp, nil
}

// GetDefaultForModel retrieves the default blueprint for a device model within the transaction
func (r *transactionalBlueprintRepository) GetDefaultForModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (*models.Blueprint, error) {
	query := `
		SELECT id, device_model_id, tenant_id, version, type_eui, spec_json, is_default,
		       registry_repo, registry_commit_sha, registry_verified, registry_pr_url,
		       created_at, updated_at
		FROM blueprints
		WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true`

	bp := &models.Blueprint{}
	var typeEUI, specJSON []byte
	var registryRepo, registryCommitSHA, registryPRURL sql.NullString
	var registryVerified sql.NullBool

	err := r.tx.QueryRowContext(ctx, query, tenantID, deviceModelID).Scan(
		&bp.ID, &bp.DeviceModelID, &bp.TenantID, &bp.Version, &typeEUI, &specJSON, &bp.IsDefault,
		&registryRepo, &registryCommitSHA, &registryVerified, &registryPRURL,
		&bp.CreatedAt, &bp.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No default is not an error
		}
		return nil, fmt.Errorf("get default blueprint: %w", err)
	}

	bp.TypeEUI = typeEUI
	bp.SpecJSON = specJSON
	if registryRepo.Valid {
		bp.RegistryRepo = &registryRepo.String
	}
	if registryCommitSHA.Valid {
		bp.RegistryCommit = &registryCommitSHA.String
	}
	if registryPRURL.Valid {
		bp.RegistryPRURL = &registryPRURL.String
	}
	if registryVerified.Valid {
		bp.RegistryVerified = registryVerified.Bool
	}

	return bp, nil
}

// ListByDeviceModel retrieves blueprints for a device model with pagination within the transaction
func (r *transactionalBlueprintRepository) ListByDeviceModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID, limit, offset int) ([]*models.Blueprint, error) {
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

	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list blueprints by device model: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "ListByDeviceModel")
		}
	}()

	return r.scanBlueprints(rows)
}

// List retrieves blueprints for a tenant with pagination and optional filters within the transaction
func (r *transactionalBlueprintRepository) List(ctx context.Context, params *models.BlueprintListParams) ([]*models.Blueprint, error) {
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

	query += " ORDER BY created_at DESC"

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
		return nil, fmt.Errorf("list blueprints: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "List")
		}
	}()

	return r.scanBlueprints(rows)
}

// ListWithModel retrieves blueprints with joined device model and manufacturer data within the transaction
func (r *transactionalBlueprintRepository) ListWithModel(ctx context.Context, params *models.BlueprintListParams) ([]*models.BlueprintWithModel, error) {
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

	rows, err := r.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list blueprints with model: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "ListWithModel")
		}
	}()

	var blueprints []*models.BlueprintWithModel
	for rows.Next() {
		bp := &models.BlueprintWithModel{}
		var typeEUI, specJSON []byte
		var registryRepo, registryCommitSHA, registryPRURL sql.NullString
		var registryVerified sql.NullBool

		err := rows.Scan(
			&bp.ID, &bp.DeviceModelID, &bp.TenantID, &bp.Version, &typeEUI, &specJSON, &bp.IsDefault,
			&registryRepo, &registryCommitSHA, &registryVerified, &registryPRURL,
			&bp.CreatedAt, &bp.UpdatedAt,
			&bp.DeviceModelName, &bp.DeviceModelCode,
			&bp.ManufacturerID, &bp.ManufacturerName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan blueprint with model: %w", err)
		}

		bp.TypeEUI = typeEUI
		bp.SpecJSON = specJSON
		if registryRepo.Valid {
			bp.RegistryRepo = &registryRepo.String
		}
		if registryCommitSHA.Valid {
			bp.RegistryCommit = &registryCommitSHA.String
		}
		if registryPRURL.Valid {
			bp.RegistryPRURL = &registryPRURL.String
		}
		if registryVerified.Valid {
			bp.RegistryVerified = registryVerified.Bool
		}

		blueprints = append(blueprints, bp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blueprints: %w", err)
	}

	return blueprints, nil
}

// Count returns the total count of blueprints for a tenant within the transaction
func (r *transactionalBlueprintRepository) Count(ctx context.Context, tenantID int64) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM blueprints WHERE tenant_id = $1`

	err := r.tx.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count blueprints: %w", err)
	}

	return count, nil
}

// CountByDeviceModel returns the count of blueprints for a device model within the transaction
func (r *transactionalBlueprintRepository) CountByDeviceModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM blueprints WHERE tenant_id = $1 AND device_model_id = $2`

	err := r.tx.QueryRowContext(ctx, query, tenantID, deviceModelID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count blueprints by device model: %w", err)
	}

	return count, nil
}

// Update updates an existing blueprint within the transaction
func (r *transactionalBlueprintRepository) Update(ctx context.Context, tenantID int64, id uuid.UUID, params *models.BlueprintUpdateParams) error {
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
		args = append(args, params.SpecJSON)
		argIndex++
	}

	if params.IsDefault != nil {
		// If setting as default, clear other defaults for same model first
		if *params.IsDefault {
			// Get the device_model_id first
			var deviceModelID uuid.UUID
			err := r.tx.QueryRowContext(ctx,
				`SELECT device_model_id FROM blueprints WHERE tenant_id = $1 AND id = $2`,
				tenantID, id).Scan(&deviceModelID)
			if err != nil {
				return fmt.Errorf("get device model id: %w", err)
			}

			_, err = r.tx.ExecContext(ctx,
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

	// #nosec G201 -- setClauses are constructed from safe column names with parameterized values
	query := fmt.Sprintf(
		"UPDATE blueprints SET %s WHERE tenant_id = $1 AND id = $2",
		strings.Join(setClauses, ", "),
	)

	result, err := r.tx.ExecContext(ctx, query, args...)
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

// SetDefault sets a blueprint as the default for its device model within the transaction
func (r *transactionalBlueprintRepository) SetDefault(ctx context.Context, tenantID int64, id uuid.UUID) error {
	// Get the device_model_id first
	var deviceModelID uuid.UUID
	err := r.tx.QueryRowContext(ctx,
		`SELECT device_model_id FROM blueprints WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan(&deviceModelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrNotFound
		}
		return fmt.Errorf("get device model id: %w", err)
	}

	// Clear any existing default for the model
	_, err = r.tx.ExecContext(ctx,
		`UPDATE blueprints SET is_default = false, updated_at = NOW()
		 WHERE tenant_id = $1 AND device_model_id = $2 AND is_default = true`,
		tenantID, deviceModelID)
	if err != nil {
		return fmt.Errorf("clear existing default: %w", err)
	}

	// Set the new default
	result, err := r.tx.ExecContext(ctx,
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

// ClearDefault clears the default flag for a blueprint within the transaction
func (r *transactionalBlueprintRepository) ClearDefault(ctx context.Context, tenantID int64, id uuid.UUID) error {
	result, err := r.tx.ExecContext(ctx,
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

// UpdateRegistryInfo updates the GitHub registry metadata for a blueprint within the transaction
func (r *transactionalBlueprintRepository) UpdateRegistryInfo(ctx context.Context, tenantID int64, id uuid.UUID, repo, commitSHA, prURL string, verified bool) error {
	result, err := r.tx.ExecContext(ctx,
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

// Delete deletes a blueprint by ID with tenant isolation within the transaction
func (r *transactionalBlueprintRepository) Delete(ctx context.Context, tenantID int64, id uuid.UUID) error {
	query := `DELETE FROM blueprints WHERE tenant_id = $1 AND id = $2`

	result, err := r.tx.ExecContext(ctx, query, tenantID, id)
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

// scanBlueprints is a helper to scan rows into blueprint structs
func (r *transactionalBlueprintRepository) scanBlueprints(rows *sql.Rows) ([]*models.Blueprint, error) {
	var blueprints []*models.Blueprint
	for rows.Next() {
		bp := &models.Blueprint{}
		var typeEUI, specJSON []byte
		var registryRepo, registryCommitSHA, registryPRURL sql.NullString
		var registryVerified sql.NullBool

		err := rows.Scan(
			&bp.ID, &bp.DeviceModelID, &bp.TenantID, &bp.Version, &typeEUI, &specJSON, &bp.IsDefault,
			&registryRepo, &registryCommitSHA, &registryVerified, &registryPRURL,
			&bp.CreatedAt, &bp.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan blueprint: %w", err)
		}

		bp.TypeEUI = typeEUI
		bp.SpecJSON = specJSON
		if registryRepo.Valid {
			bp.RegistryRepo = &registryRepo.String
		}
		if registryCommitSHA.Valid {
			bp.RegistryCommit = &registryCommitSHA.String
		}
		if registryPRURL.Valid {
			bp.RegistryPRURL = &registryPRURL.String
		}
		if registryVerified.Valid {
			bp.RegistryVerified = registryVerified.Bool
		}

		blueprints = append(blueprints, bp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blueprints: %w", err)
	}

	return blueprints, nil
}
