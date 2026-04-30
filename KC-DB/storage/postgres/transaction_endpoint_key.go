package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kilocenter/KC-DB/storage/models"
)

// transactionalEndPointKeyRepository implements interfaces.EndPointKeyRepository within a transaction
type transactionalEndPointKeyRepository struct {
	tx *sql.Tx
	db *DB
}

// Create creates a new endpoint key within the transaction
func (r *transactionalEndPointKeyRepository) Create(ctx context.Context, key *models.EndPointKey) error {
	query := `
		INSERT INTO endpoint_keys (
			endpoint_id, key_type, key_value, key_version,
			valid_from, valid_until, is_active, usage_count,
			last_used_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	err := r.tx.QueryRowContext(ctx, query,
		key.EndPointID,
		key.KeyType,
		key.KeyValue,
		key.KeyVersion,
		key.ValidFrom,
		key.ValidUntil,
		key.IsActive,
		key.UsageCount,
		key.LastUsedAt,
		key.Metadata,
	).Scan(&key.ID, &key.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create endpoint key: %w", err)
	}

	return nil
}

// GetActive retrieves the active key of a specific type for an endpoint within the transaction
func (r *transactionalEndPointKeyRepository) GetActive(ctx context.Context, endpointID string, keyType string) (*models.EndPointKey, error) {
	query := `
		SELECT 
			id, endpoint_id, key_type, key_value, key_version,
			valid_from, valid_until, is_active, usage_count,
			last_used_at, metadata, created_at
		FROM endpoint_keys
		WHERE endpoint_id = $1 
		AND key_type = $2 
		AND is_active = true
		AND (valid_from IS NULL OR valid_from <= NOW())
		AND (valid_until IS NULL OR valid_until > NOW())
		ORDER BY key_version DESC
		LIMIT 1`

	endpointKey := &models.EndPointKey{}
	err := r.tx.QueryRowContext(ctx, query, endpointID, keyType).Scan(
		&endpointKey.ID,
		&endpointKey.EndPointID,
		&endpointKey.KeyType,
		&endpointKey.KeyValue,
		&endpointKey.KeyVersion,
		&endpointKey.ValidFrom,
		&endpointKey.ValidUntil,
		&endpointKey.IsActive,
		&endpointKey.UsageCount,
		&endpointKey.LastUsedAt,
		&endpointKey.Metadata,
		&endpointKey.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint key: %w", err)
	}

	return endpointKey, nil
}

// GetByID retrieves a key by ID within the transaction
func (r *transactionalEndPointKeyRepository) GetByID(ctx context.Context, id string) (*models.EndPointKey, error) {
	query := `
		SELECT 
			id, endpoint_euid, key_type, key_value, key_version,
			valid_from, valid_until, is_active, usage_count,
			last_used_at, metadata, created_at
		FROM endpoint_keys
		WHERE id = $1`

	endpointKey := &models.EndPointKey{}
	err := r.tx.QueryRowContext(ctx, query, id).Scan(
		&endpointKey.ID,
		&endpointKey.EndPointID,
		&endpointKey.KeyType,
		&endpointKey.KeyValue,
		&endpointKey.KeyVersion,
		&endpointKey.ValidFrom,
		&endpointKey.ValidUntil,
		&endpointKey.IsActive,
		&endpointKey.UsageCount,
		&endpointKey.LastUsedAt,
		&endpointKey.Metadata,
		&endpointKey.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	return endpointKey, nil
}

// GetByEndPoint retrieves all keys for an endpoint within the transaction
func (r *transactionalEndPointKeyRepository) GetByEndPoint(ctx context.Context, endpointID string) ([]*models.EndPointKey, error) {
	query := `
		SELECT 
			id, endpoint_id, key_type, key_value, key_version,
			valid_from, valid_until, is_active, usage_count,
			last_used_at, metadata, created_at
		FROM endpoint_keys
		WHERE endpoint_id = $1
		ORDER BY key_type, key_version DESC`

	rows, err := r.tx.QueryContext(ctx, query, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "GetByEndPoint")
		}
	}()

	var endpointKeys []*models.EndPointKey
	for rows.Next() {
		endpointKey := &models.EndPointKey{}
		err := rows.Scan(
			&endpointKey.ID,
			&endpointKey.EndPointID,
			&endpointKey.KeyType,
			&endpointKey.KeyValue,
			&endpointKey.KeyVersion,
			&endpointKey.ValidFrom,
			&endpointKey.ValidUntil,
			&endpointKey.IsActive,
			&endpointKey.UsageCount,
			&endpointKey.LastUsedAt,
			&endpointKey.Metadata,
			&endpointKey.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		endpointKeys = append(endpointKeys, endpointKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating keys: %w", err)
	}

	return endpointKeys, nil
}

// Deactivate deactivates a key within the transaction
func (r *transactionalEndPointKeyRepository) Deactivate(ctx context.Context, id string) error {
	query := `
		UPDATE endpoint_keys SET
			is_active = false,
			valid_until = NOW()
		WHERE id = $1`

	result, err := r.tx.ExecContext(ctx, query, id)
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

// IncrementUsage increments the usage count and updates last used timestamp within the transaction
func (r *transactionalEndPointKeyRepository) IncrementUsage(ctx context.Context, id string) error {
	query := `
		UPDATE endpoint_keys SET
			usage_count = usage_count + 1,
			last_used_at = NOW()
		WHERE id = $1`

	result, err := r.tx.ExecContext(ctx, query, id)
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

// RotateKey creates a new key version and deactivates the old one within the transaction
func (r *transactionalEndPointKeyRepository) RotateKey(ctx context.Context, oldKeyID string, newKey *models.EndPointKey, reason string) error {
	// Start by deactivating the old key
	if err := r.Deactivate(ctx, oldKeyID); err != nil {
		return fmt.Errorf("failed to deactivate old key: %w", err)
	}

	// Update the old key's metadata with rotation reason
	if reason != "" {
		updateQuery := `
			UPDATE endpoint_keys SET
				metadata = jsonb_set(
					COALESCE(metadata, '{}'::jsonb),
					'{rotation_reason}',
					to_jsonb($2::text)
				)
			WHERE id = $1`

		if _, err := r.tx.ExecContext(ctx, updateQuery, oldKeyID, reason); err != nil {
			return fmt.Errorf("failed to update rotation reason: %w", err)
		}
	}

	// Create the new key
	if err := r.Create(ctx, newKey); err != nil {
		return fmt.Errorf("failed to create new key: %w", err)
	}

	return nil
}

// GetStats retrieves key statistics for an endpoint within the transaction
func (r *transactionalEndPointKeyRepository) GetStats(ctx context.Context, endpointID string) (*models.EndPointKeyStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_keys,
			COUNT(CASE WHEN is_active THEN 1 END) as active_keys,
			COUNT(CASE WHEN NOT is_active THEN 1 END) as inactive_keys,
			COUNT(CASE WHEN valid_until < NOW() THEN 1 END) as expired_keys,
			SUM(usage_count) as total_usage
		FROM endpoint_keys
		WHERE endpoint_id = $1`

	stats := &models.EndPointKeyStats{
		EndPointID: endpointID,
	}

	err := r.tx.QueryRowContext(ctx, query, endpointID).Scan(
		&stats.TotalKeys,
		&stats.ActiveKeys,
		&stats.InactiveKeys,
		&stats.ExpiredKeys,
		&stats.TotalUsage,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get key stats: %w", err)
	}

	// Get key type breakdown
	typeQuery := `
		SELECT key_type, COUNT(*) as count
		FROM endpoint_keys
		WHERE endpoint_id = $1
		GROUP BY key_type`

	rows, err := r.tx.QueryContext(ctx, typeQuery, endpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to get key type breakdown: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.db.log.Warn("rows close failed", "error", err, "operation", "GetStats")
		}
	}()

	var keyTypeCounts = make(map[string]int64)
	for rows.Next() {
		var keyType string
		var count int64
		if err := rows.Scan(&keyType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan key type: %w", err)
		}
		keyTypeCounts[keyType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating key types: %w", err)
	}

	stats.KeyTypeBreakdown = make(map[string]int64)
	for keyType, count := range keyTypeCounts {
		stats.KeyTypeBreakdown[keyType] = count
	}

	return stats, nil
}

// ExpireKeys marks keys as expired based on their validity period within the transaction
func (r *transactionalEndPointKeyRepository) ExpireKeys(ctx context.Context) (int64, error) {
	query := `
		UPDATE endpoint_keys SET
			is_active = false,
			metadata = jsonb_set(
				COALESCE(metadata, '{}'::jsonb),
				'{expired_reason}',
				'"validity_period_exceeded"'
			)
		WHERE is_active = true 
		AND valid_until IS NOT NULL 
		AND valid_until < NOW()`

	result, err := r.tx.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to expire keys: %w", err)
	}

	return result.RowsAffected()
}
