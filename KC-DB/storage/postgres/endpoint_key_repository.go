package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// endpointKeyRepository implements interfaces.EndPointKeyRepository
type endpointKeyRepository struct {
	db *DB
}

// NewEndPointKeyRepository creates a new endpoint key repository
func NewEndPointKeyRepository(db *DB) interfaces.EndPointKeyRepository {
	return &endpointKeyRepository{db: db}
}

// Create creates a new endpoint key
func (r *endpointKeyRepository) Create(ctx context.Context, key *models.EndPointKey) error {
	query := `
		INSERT INTO endpoint_keys (
			endpoint_id, tenant_id, key_type, key_version,
			key_value, key_id, algorithm,
			valid_from, valid_until, is_active,
			rotated_from, rotation_reason,
			metadata, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at`

	// Default metadata if nil
	if key.Metadata == nil {
		key.Metadata = json.RawMessage("{}")
	}

	err := r.db.QueryRow(ctx, query,
		key.EndPointID,
		key.TenantID,
		key.KeyType,
		key.KeyVersion,
		key.KeyValue,
		key.KeyID,
		key.Algorithm,
		key.ValidFrom,
		key.ValidUntil,
		key.IsActive,
		key.RotatedFrom,
		key.RotationReason,
		key.Metadata,
		key.CreatedBy,
	).Scan(&key.ID, &key.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create endpoint key: %w", err)
	}

	return nil
}

// GetActive retrieves the active key of a specific type for an endpoint
func (r *endpointKeyRepository) GetActive(ctx context.Context, endpointID string, keyType string) (*models.EndPointKey, error) {
	query := `
		SELECT 
			id, endpoint_id, tenant_id, key_type, key_version,
			key_value, key_id, algorithm,
			valid_from, valid_until, is_active,
			usage_count, last_used_at,
			rotated_from, rotation_reason,
			metadata, created_at, created_by
		FROM endpoint_keys
		WHERE endpoint_id = $1 AND key_type = $2 AND is_active = true
		AND (valid_until IS NULL OR valid_until > NOW())`

	key := &models.EndPointKey{}
	err := r.db.QueryRow(ctx, query, endpointID, keyType).Scan(
		&key.ID,
		&key.EndPointID,
		&key.TenantID,
		&key.KeyType,
		&key.KeyVersion,
		&key.KeyValue,
		&key.KeyID,
		&key.Algorithm,
		&key.ValidFrom,
		&key.ValidUntil,
		&key.IsActive,
		&key.UsageCount,
		&key.LastUsedAt,
		&key.RotatedFrom,
		&key.RotationReason,
		&key.Metadata,
		&key.CreatedAt,
		&key.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active key: %w", err)
	}

	return key, nil
}

// GetByID retrieves a key by ID
func (r *endpointKeyRepository) GetByID(ctx context.Context, id string) (*models.EndPointKey, error) {
	query := `
		SELECT 
			id, endpoint_id, tenant_id, key_type, key_version,
			key_value, key_id, algorithm,
			valid_from, valid_until, is_active,
			usage_count, last_used_at,
			rotated_from, rotation_reason,
			metadata, created_at, created_by
		FROM endpoint_keys
		WHERE id = $1`

	key := &models.EndPointKey{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&key.ID,
		&key.EndPointID,
		&key.TenantID,
		&key.KeyType,
		&key.KeyVersion,
		&key.KeyValue,
		&key.KeyID,
		&key.Algorithm,
		&key.ValidFrom,
		&key.ValidUntil,
		&key.IsActive,
		&key.UsageCount,
		&key.LastUsedAt,
		&key.RotatedFrom,
		&key.RotationReason,
		&key.Metadata,
		&key.CreatedAt,
		&key.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key by ID: %w", err)
	}

	return key, nil
}

// GetByEndPoint retrieves all keys for an endpoint
func (r *endpointKeyRepository) GetByEndPoint(ctx context.Context, endpointID string) ([]*models.EndPointKey, error) {
	query := `
		SELECT 
			id, endpoint_id, tenant_id, key_type, key_version,
			key_value, key_id, algorithm,
			valid_from, valid_until, is_active,
			usage_count, last_used_at,
			rotated_from, rotation_reason,
			metadata, created_at, created_by
		FROM endpoint_keys
		WHERE endpoint_id = $1
		ORDER BY key_type, key_version DESC`

	rows, err := r.db.Query(ctx, query, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to query endpoint keys: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// TODO: Repository lacks logger field - add for proper error tracking
			log.Printf("failed to close rows in endpoint keys query: %v", err)
		}
	}()

	var keys []*models.EndPointKey
	for rows.Next() {
		key := &models.EndPointKey{}
		err := rows.Scan(
			&key.ID,
			&key.EndPointID,
			&key.TenantID,
			&key.KeyType,
			&key.KeyVersion,
			&key.KeyValue,
			&key.KeyID,
			&key.Algorithm,
			&key.ValidFrom,
			&key.ValidUntil,
			&key.IsActive,
			&key.UsageCount,
			&key.LastUsedAt,
			&key.RotatedFrom,
			&key.RotationReason,
			&key.Metadata,
			&key.CreatedAt,
			&key.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan endpoint key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoint keys: %w", err)
	}

	return keys, nil
}

// Deactivate deactivates a key
func (r *endpointKeyRepository) Deactivate(ctx context.Context, id string) error {
	query := `
		UPDATE endpoint_keys SET
			is_active = false,
			valid_until = CASE 
				WHEN valid_until IS NULL OR valid_until > NOW() 
				THEN NOW() 
				ELSE valid_until 
			END
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// IncrementUsage increments the usage count and updates last used timestamp
func (r *endpointKeyRepository) IncrementUsage(ctx context.Context, id string) error {
	query := `
		UPDATE endpoint_keys SET
			usage_count = usage_count + 1,
			last_used_at = NOW()
		WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// RotateKey creates a new key version and deactivates the old one
func (r *endpointKeyRepository) RotateKey(ctx context.Context, oldKeyID string, newKey *models.EndPointKey, reason string) error {
	// This should be done in a transaction, but we'll implement it simply for now
	// In production, this would use the transaction support

	// First, deactivate the old key
	err := r.Deactivate(ctx, oldKeyID)
	if err != nil {
		return fmt.Errorf("failed to deactivate old key: %w", err)
	}

	// Set rotation metadata on new key
	newKey.RotatedFrom = &oldKeyID
	newKey.RotationReason = &reason

	// Create the new key
	err = r.Create(ctx, newKey)
	if err != nil {
		return fmt.Errorf("failed to create new key: %w", err)
	}

	return nil
}

// GetStats retrieves key statistics for an endpoint
func (r *endpointKeyRepository) GetStats(ctx context.Context, endpointID string) (*models.EndPointKeyStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_keys,
			COUNT(*) FILTER (WHERE is_active = true) as active_keys,
			COUNT(*) FILTER (WHERE key_type = 'network') as network_keys,
			COUNT(*) FILTER (WHERE key_type = 'application') as application_keys,
			COUNT(*) FILTER (WHERE key_type = 'join') as join_keys,
			COALESCE(SUM(usage_count), 0) as total_usage_count,
			COUNT(*) FILTER (WHERE rotated_from IS NOT NULL) as keys_rotated_count
		FROM endpoint_keys
		WHERE endpoint_id = $1`

	stats := &models.EndPointKeyStats{
		EndPointID: endpointID,
	}

	err := r.db.QueryRow(ctx, query, endpointID).Scan(
		&stats.TotalKeys,
		&stats.ActiveKeys,
		&stats.NetworkKeys,
		&stats.ApplicationKeys,
		&stats.JoinKeys,
		&stats.TotalUsageCount,
		&stats.KeysRotatedCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get key stats: %w", err)
	}

	return stats, nil
}

// ExpireKeys marks keys as expired based on their validity period
func (r *endpointKeyRepository) ExpireKeys(ctx context.Context) (int64, error) {
	query := `
		UPDATE endpoint_keys SET
			is_active = false
		WHERE is_active = true 
		AND valid_until IS NOT NULL 
		AND valid_until <= NOW()`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to expire keys: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
