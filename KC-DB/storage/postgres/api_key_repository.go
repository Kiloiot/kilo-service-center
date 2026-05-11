package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// APIKeyRepository implements the APIKeyRepository interface for PostgreSQL.
type APIKeyRepository struct {
	db *sqlx.DB
}

// NewAPIKeyRepository creates a new PostgreSQL API key repository
func NewAPIKeyRepository(db *sqlx.DB) interfaces.APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create inserts a new API key
func (r *APIKeyRepository) Create(ctx context.Context, key *models.APIKey) error {
	now := time.Now().UTC()
	key.CreatedAt = now

	query := `
		INSERT INTO api_keys (
			id, tenant_id, org_id, user_id, name, key_hash, key_prefix,
			key_type, expires_at, last_used_at, is_active, created_at
		) VALUES (
			:id, :tenant_id, :org_id, :user_id, :name, :key_hash, :key_prefix,
			:key_type, :expires_at, :last_used_at, :is_active, :created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}

	return nil
}

// GetByID retrieves an API key by UUID
func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	var key models.APIKey
	query := `
		SELECT id, tenant_id, org_id, user_id, name, key_hash, key_prefix,
			   key_type, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE id = $1`

	err := r.db.GetContext(ctx, &key, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("api key %s: %w", id, interfaces.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("get api key: %w", err)
	}

	return &key, nil
}

// GetByHash retrieves an API key by its SHA-256 hash
func (r *APIKeyRepository) GetByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	var key models.APIKey
	query := `
		SELECT id, tenant_id, org_id, user_id, name, key_hash, key_prefix,
			   key_type, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE key_hash = $1`

	err := r.db.GetContext(ctx, &key, query, hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("api key with hash: %w", interfaces.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}

	return &key, nil
}

// Delete removes an API key by ID
func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("api key %s: %w", id, interfaces.ErrRecordNotFound)
	}

	return nil
}

// List returns API keys for a tenant and organization with optional user filter
func (r *APIKeyRepository) List(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, error) {
	var keys []*models.APIKey

	if userID != nil {
		// User-scoped: return user's keys plus service account keys within the organization
		query := `
			SELECT id, tenant_id, org_id, user_id, name, key_hash, key_prefix,
				   key_type, expires_at, last_used_at, is_active, created_at
			FROM api_keys
			WHERE tenant_id = $1 AND org_id = $2 AND (user_id = $3 OR user_id IS NULL)
			ORDER BY created_at DESC
			LIMIT $4 OFFSET $5`

		err := r.db.SelectContext(ctx, &keys, query, tenantID, orgID, *userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("list api keys: %w", err)
		}
	} else {
		// Admin view: return all keys for tenant within the organization
		query := `
			SELECT id, tenant_id, org_id, user_id, name, key_hash, key_prefix,
				   key_type, expires_at, last_used_at, is_active, created_at
			FROM api_keys
			WHERE tenant_id = $1 AND org_id = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4`

		err := r.db.SelectContext(ctx, &keys, query, tenantID, orgID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("list api keys: %w", err)
		}
	}

	return keys, nil
}

// Count returns total API key count for a tenant and organization with optional user filter
func (r *APIKeyRepository) Count(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID) (int64, error) {
	var count int64

	if userID != nil {
		query := `SELECT COUNT(*) FROM api_keys WHERE tenant_id = $1 AND org_id = $2 AND (user_id = $3 OR user_id IS NULL)`
		err := r.db.GetContext(ctx, &count, query, tenantID, orgID, *userID)
		if err != nil {
			return 0, fmt.Errorf("count api keys: %w", err)
		}
	} else {
		query := `SELECT COUNT(*) FROM api_keys WHERE tenant_id = $1 AND org_id = $2`
		err := r.db.GetContext(ctx, &count, query, tenantID, orgID)
		if err != nil {
			return 0, fmt.Errorf("count api keys: %w", err)
		}
	}

	return count, nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("update last used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("api key %s: %w", id, interfaces.ErrRecordNotFound)
	}

	return nil
}

// Deactivate marks a key as inactive without deleting it
func (r *APIKeyRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET is_active = false WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deactivate api key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("api key %s: %w", id, interfaces.ErrRecordNotFound)
	}

	return nil
}

// GetByIDAndOrg retrieves an API key by UUID with organization ownership check.
func (r *APIKeyRepository) GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error) {
	var key models.APIKey
	query := `
		SELECT id, tenant_id, org_id, user_id, name, key_hash, key_prefix,
			   key_type, expires_at, last_used_at, is_active, created_at
		FROM api_keys
		WHERE id = $1 AND org_id = $2`

	err := r.db.GetContext(ctx, &key, query, id, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("api key %s: %w", id, interfaces.ErrRecordNotFound)
		}
		return nil, fmt.Errorf("get api key: %w", err)
	}

	return &key, nil
}

// DeleteByIDAndOrg removes an API key by ID with organization ownership check.
func (r *APIKeyRepository) DeleteByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1 AND org_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, orgID)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("api key %s: %w", id, interfaces.ErrRecordNotFound)
	}

	return nil
}
