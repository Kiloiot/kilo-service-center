package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RefreshTokenRepository implements the RefreshTokenRepository interface for PostgreSQL
// Supports refresh token rotation with reuse detection
type RefreshTokenRepository struct {
	clock clock.Clock
	db    *sqlx.DB
}

// NewRefreshTokenRepository creates a new PostgreSQL RefreshToken repository
func NewRefreshTokenRepository(db *sqlx.DB, clk clock.Clock) *RefreshTokenRepository {
	return &RefreshTokenRepository{clock: clk, db: db}
}

// Create inserts a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	now := r.clock.Now().UTC()
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
		return fmt.Errorf("%s: %w", errWrapCreateRefreshToken, err)
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
		return nil, fmt.Errorf("%s: %w", errWrapGetRefreshToken, err)
	}

	return &token, nil
}

// RevokeByUserID revokes all tokens for a user (family revocation)
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapRevokeRefreshTokensForUser, err)
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
		return fmt.Errorf("%s: %w", errWrapMarkRefreshTokenReplaced, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(errFmtRefreshTokenNotFound, oldTokenID)
	}

	return nil
}
