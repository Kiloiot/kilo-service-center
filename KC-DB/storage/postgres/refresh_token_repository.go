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

// RefreshTokenRepository implements the RefreshTokenRepository interface for PostgreSQL
// Supports refresh token rotation with reuse detection
type RefreshTokenRepository struct {
	db *sqlx.DB
}

// NewRefreshTokenRepository creates a new PostgreSQL RefreshToken repository
func NewRefreshTokenRepository(db *sqlx.DB) interfaces.RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create inserts a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	now := time.Now().UTC()
	token.CreatedAt = now
	if token.IssuedAt.IsZero() {
		token.IssuedAt = now
	}

	query := `
		INSERT INTO refresh_tokens (
			id, user_id, token_hash, issued_at, expires_at,
			revoked_at, replaced_by, created_at
		) VALUES (
			:id, :user_id, :token_hash, :issued_at, :expires_at,
			:revoked_at, :replaced_by, :created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

// GetByHash retrieves a refresh token by its hash
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `
		SELECT id, user_id, token_hash, issued_at, expires_at,
			   revoked_at, replaced_by, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	err := r.db.GetContext(ctx, &token, query, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found returns nil, not error
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	return &token, nil
}

// RevokeByHash marks a token as revoked by its hash
func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found or already revoked")
	}

	return nil
}

// RevokeByUserID revokes all tokens for a user (family revocation)
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("revoke refresh tokens for user: %w", err)
	}

	return nil
}

// MarkReplaced sets the replaced_by field to link old token to new token
func (r *RefreshTokenRepository) MarkReplaced(ctx context.Context, oldTokenID, newTokenID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET replaced_by = $2
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, oldTokenID, newTokenID)
	if err != nil {
		return fmt.Errorf("mark refresh token replaced: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token %s not found", oldTokenID)
	}

	return nil
}

// DeleteExpired removes expired tokens older than their expiry time
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired refresh tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rowsAffected, nil
}
