package adapters

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RefreshTokenStoreAdapter adapts postgres.RefreshTokenRepository to provide
// refresh token operations for authentication.
type RefreshTokenStoreAdapter struct {
	repo *postgres.RefreshTokenRepository
}

// NewRefreshTokenStoreAdapter creates a new adapter with the given database connection
func NewRefreshTokenStoreAdapter(db *sqlx.DB) *RefreshTokenStoreAdapter {
	repo := postgres.NewRefreshTokenRepository(db)
	return &RefreshTokenStoreAdapter{
		repo: repo.(*postgres.RefreshTokenRepository),
	}
}

// Create stores a new refresh token (hash pre-computed by caller)
func (a *RefreshTokenStoreAdapter) Create(ctx context.Context, token *models.RefreshToken) error {
	if err := a.repo.Create(ctx, token); err != nil {
		return fmt.Errorf("refresh token adapter: create: %w", err)
	}
	return nil
}

// GetByHash retrieves a refresh token by its hash
func (a *RefreshTokenStoreAdapter) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token, err := a.repo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("refresh token adapter: get_by_hash: %w", err)
	}
	return token, nil
}

// RevokeByHash marks a token as revoked
func (a *RefreshTokenStoreAdapter) RevokeByHash(ctx context.Context, tokenHash string) error {
	if err := a.repo.RevokeByHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("refresh token adapter: revoke_by_hash: %w", err)
	}
	return nil
}

// RevokeByUserID revokes all tokens for a user (family revocation)
func (a *RefreshTokenStoreAdapter) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := a.repo.RevokeByUserID(ctx, userID); err != nil {
		return fmt.Errorf("refresh token adapter: revoke_by_user_id: %w", err)
	}
	return nil
}

// MarkReplaced links old token to new token during rotation
func (a *RefreshTokenStoreAdapter) MarkReplaced(ctx context.Context, oldTokenID, newTokenID uuid.UUID) error {
	if err := a.repo.MarkReplaced(ctx, oldTokenID, newTokenID); err != nil {
		return fmt.Errorf("refresh token adapter: mark_replaced: %w", err)
	}
	return nil
}
