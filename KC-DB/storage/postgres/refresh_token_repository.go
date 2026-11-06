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

// CreateForPassword opens a session: it stores token only while its owner is
// active and still has passwordHash, the hash the caller verified or set. It
// fails with storage.ErrUserInactive or storage.ErrUserCredentialsChanged and
// stores nothing otherwise.
func (r *RefreshTokenRepository) CreateForPassword(ctx context.Context, token *models.RefreshToken, passwordHash string) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapBeginRefreshTokenCreation, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	owner, err := lockActiveOwner(ctx, tx, token.UserID)
	if err != nil {
		return err
	}
	if owner.PasswordHash == nil || *owner.PasswordHash != passwordHash {
		return storage.ErrUserCredentialsChanged
	}
	if err = r.insert(ctx, tx, token); err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateRefreshToken, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapCommitRefreshTokenCreation, err)
	}
	return nil
}

// insert stamps the creation times of token and stores it within tx.
func (r *RefreshTokenRepository) insert(ctx context.Context, tx *sqlx.Tx, token *models.RefreshToken) error {
	now := r.clock.Now().UTC()
	token.CreatedAt = now
	if token.IssuedAt.IsZero() {
		token.IssuedAt = now
	}
	_, err := tx.NamedExecContext(ctx, insertRefreshTokenQuery, token)
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
// It fails with storage.ErrUserInactive when the owner of
// newToken is missing or deactivated, and with
// storage.ErrRefreshTokenNotClaimable when the old token belongs to another
// user or is already replaced, revoked or expired; nothing is stored in either
// case.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapBeginRefreshTokenRotation, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)

	if _, err = lockActiveOwner(ctx, tx, newToken.UserID); err != nil {
		return err
	}
	if err = r.insert(ctx, tx, newToken); err != nil {
		return fmt.Errorf("%s: %w", errWrapCreateRefreshToken, err)
	}
	if err = claimRefreshToken(ctx, tx, oldTokenID, newToken); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapCommitRefreshTokenRotation, err)
	}
	return nil
}

// tokenOwner is the state of a user that decides whether a session may be
// opened or renewed for them.
type tokenOwner struct {
	IsActive     bool    `db:"is_active"`
	PasswordHash *string `db:"password_hash"`
}

// lockActiveOwner share-locks the user row for the rest of tx, so a
// deactivation, password change or revocation of the user waits for tx, and
// refuses a missing or deactivated user.
func lockActiveOwner(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) (*tokenOwner, error) {
	var owner tokenOwner
	err := tx.GetContext(ctx, &owner, `SELECT is_active, password_hash FROM users WHERE id = $1 FOR SHARE`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrUserInactive
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapLockRefreshTokenUser, err)
	}
	if !owner.IsActive {
		return nil, storage.ErrUserInactive
	}
	return &owner, nil
}

// claimRefreshToken marks a still usable token of the successor's owner as
// replaced by the successor.
func claimRefreshToken(ctx context.Context, tx *sqlx.Tx, oldTokenID uuid.UUID, successor *models.RefreshToken) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET replaced_by = $2
		WHERE id = $1 AND user_id = $3 AND replaced_by IS NULL AND revoked_at IS NULL AND expires_at > NOW()`,
		oldTokenID, successor.ID, successor.UserID)
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
