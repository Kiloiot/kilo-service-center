package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// APIKeyRepository defines operations for API key persistence.
type APIKeyRepository interface {
	// Create inserts a new API key.
	Create(ctx context.Context, key *models.APIKey) error

	// GetByID retrieves an API key by UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)

	// GetByHash retrieves an API key by its SHA-256 hash.
	// Used for authentication lookup during request validation.
	GetByHash(ctx context.Context, hash string) (*models.APIKey, error)

	// Delete removes an API key by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// List returns API keys for a tenant and organization with optional user filter.
	// If userID is nil, returns all keys for the tenant+org (admin view).
	// If userID is provided, returns keys owned by that user plus service account keys.
	List(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, error)

	// Count returns total API key count for a tenant and organization with optional user filter.
	Count(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID) (int64, error)

	// UpdateLastUsed updates the last_used_at timestamp.
	// Called on successful authentication to track key usage.
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// Deactivate marks a key as inactive without deleting it.
	// Preserves audit trail while preventing further use.
	Deactivate(ctx context.Context, id uuid.UUID) error

	// GetByIDAndOrg retrieves an API key by UUID with organization ownership check.
	// Returns ErrRecordNotFound if key doesn't exist or belongs to different org.
	GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error)

	// DeleteByIDAndOrg removes an API key by ID with organization ownership check.
	// Returns ErrRecordNotFound if key doesn't exist or belongs to different org.
	DeleteByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) error
}
