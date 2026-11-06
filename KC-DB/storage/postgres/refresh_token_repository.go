package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
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

const insertRefreshTokenQuery = `
	INSERT INTO refresh_tokens (
		id, user_id, token_hash, issued_at, expires_at,
		revoked_at, replaced_by, created_at
	) VALUES (
		:id, :user_id, :token_hash, :issued_at, :expires_at,
		:revoked_at, :replaced_by, :created_at
	)`

// Create inserts a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	if err := r.insert(ctx, r.db, token); err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateRefreshToken, err)
	}
	return nil
}

// insert stamps the creation times of token and stores it through ext.
func (r *RefreshTokenRepository) insert(ctx context.Context, ext sqlx.ExtContext, token *models.RefreshToken) error {
	now := r.clock.Now().UTC()
	token.CreatedAt = now
	if token.IssuedAt.IsZero() {
		token.IssuedAt = now
	}
	_, err := sqlx.NamedExecContext(ctx, ext, insertRefreshTokenQuery, token)
	return err
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found returns nil, not error
		}
		return nil, fmt.Errorf("%s: %w", errWrapGetRefreshToken, err)
	}

	return &token, nil
}

// RevokeByUserID revokes all tokens for a user (family revocation)
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapBeginRefreshTokenRevocation, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	// Waits for in-flight rotations of this user so the revoke also sees the tokens they mint.
	var locked int
	err = tx.GetContext(ctx, &locked, `SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE`, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", errWrapLockRefreshTokenUser, err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapRevokeRefreshTokensForUser, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapCommitRefreshTokenRevocation, err)
	}
	return nil
}

// Rotate atomically stores newToken and marks oldTokenID as replaced by it.
// It fails with storage.ErrRefreshTokenUserInactive when the owner of
// newToken is missing or deactivated, and with
// storage.ErrRefreshTokenNotClaimable when the old token is already
// replaced, revoked or expired; nothing is stored in either case.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapBeginRefreshTokenRotation, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	if err = requireActiveUser(ctx, tx, newToken.UserID); err != nil {
		return err
	}
	if err = r.insert(ctx, tx, newToken); err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateRefreshToken, err)
	}
	if err = claimRefreshToken(ctx, tx, oldTokenID, newToken.ID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapCommitRefreshTokenRotation, err)
	}
	return nil
}

// requireActiveUser share-locks the user row for the rest of tx and refuses a
// missing or deactivated user.
func requireActiveUser(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) error {
	var isActive bool
	err := tx.GetContext(ctx, &isActive, `SELECT is_active FROM users WHERE id = $1 FOR SHARE`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return storage.ErrRefreshTokenUserInactive
	}
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapLockRefreshTokenUser, err)
	}
	if !isActive {
		return storage.ErrRefreshTokenUserInactive
	}
	return nil
}

// claimRefreshToken marks a still usable token as replaced by newTokenID.
func claimRefreshToken(ctx context.Context, tx *sqlx.Tx, oldTokenID, newTokenID uuid.UUID) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET replaced_by = $2
		WHERE id = $1 AND replaced_by IS NULL AND revoked_at IS NULL AND expires_at > NOW()`,
		oldTokenID, newTokenID)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapClaimRefreshToken, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetRowsAffected, err)
	}
	if rowsAffected == 0 {
		return storage.ErrRefreshTokenNotClaimable
	}
	return nil
}
