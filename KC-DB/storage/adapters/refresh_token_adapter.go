package adapters

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/google/uuid"
)

// RefreshTokenStoreAdapter adapts postgres.RefreshTokenRepository to provide
// refresh token operations for authentication.
type RefreshTokenStoreAdapter struct {
	repo *postgres.RefreshTokenRepository
}

// NewRefreshTokenStoreAdapter creates a new adapter with the given database connection
func NewRefreshTokenStoreAdapter(repo *postgres.RefreshTokenRepository) *RefreshTokenStoreAdapter {
	return &RefreshTokenStoreAdapter{repo: repo}
}

// Create stores a new refresh token (hash pre-computed by caller)
func (a *RefreshTokenStoreAdapter) Create(ctx context.Context, token *models.RefreshToken) error {
	if err := a.repo.Create(ctx, token); err != nil {
		return fmt.Errorf("%s: %w", errWrapRefreshTokenAdapterCreate, err)
	}
	return nil
}

// GetByHash retrieves a refresh token by its hash
func (a *RefreshTokenStoreAdapter) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token, err := a.repo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapRefreshTokenAdapterGetByHash, err)
	}
	return token, nil
}

// RevokeByUserID revokes all tokens for a user (family revocation)
func (a *RefreshTokenStoreAdapter) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := a.repo.RevokeByUserID(ctx, userID); err != nil {
		return fmt.Errorf("%s: %w", errWrapRefreshTokenAdapterRevokeByUserID, err)
	}
	return nil
}

// MarkReplaced links old token to new token during rotation
func (a *RefreshTokenStoreAdapter) MarkReplaced(ctx context.Context, oldTokenID, newTokenID uuid.UUID) error {
	if err := a.repo.MarkReplaced(ctx, oldTokenID, newTokenID); err != nil {
		return fmt.Errorf("%s: %w", errWrapRefreshTokenAdapterMarkReplaced, err)
	}
	return nil
}
