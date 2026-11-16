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

// CreateForPassword opens a session for an active user who still has passwordHash.
func (a *RefreshTokenStoreAdapter) CreateForPassword(ctx context.Context, token *models.RefreshToken, passwordHash string) error {
	if err := a.repo.CreateForPassword(ctx, token, passwordHash); err != nil {
		return fmt.Errorf("%s: %w", errWrapRefreshTokenAdapterCreateForPassword, err)
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

// Rotate atomically stores newToken and marks oldTokenID as replaced by it.
func (a *RefreshTokenStoreAdapter) Rotate(ctx context.Context, oldTokenID uuid.UUID, newToken *models.RefreshToken) error {
	if err := a.repo.Rotate(ctx, oldTokenID, newToken); err != nil {
		return fmt.Errorf("%s: %w", errWrapRefreshTokenAdapterRotate, err)
	}
	return nil
}
