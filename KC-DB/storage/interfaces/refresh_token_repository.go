package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// RefreshTokenRepository defines operations for refresh token persistence.
// Supports refresh token rotation with reuse detection.
type RefreshTokenRepository interface {
	// Create inserts a new refresh token.
	// tokenHash must be pre-computed by the caller (SHA256 hex).
	Create(ctx context.Context, token *models.RefreshToken) error

	// GetByHash retrieves a refresh token by its hash.
	// Returns nil if not found.
	GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)

	// RevokeByHash marks a token as revoked by its hash.
	// Sets revoked_at to current time.
	RevokeByHash(ctx context.Context, tokenHash string) error

	// RevokeByUserID revokes all tokens for a user (family revocation).
	// Used when reuse is detected to invalidate entire token family.
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error

	// MarkReplaced sets the replaced_by field to link old token to new token.
	// Used during token rotation.
	MarkReplaced(ctx context.Context, oldTokenID, newTokenID uuid.UUID) error

	// DeleteExpired removes expired tokens older than the specified duration.
	// Used for periodic cleanup.
	DeleteExpired(ctx context.Context) (int64, error)
}
