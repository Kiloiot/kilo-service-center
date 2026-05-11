package interfaces

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// UserRepository defines operations for user persistence.
type UserRepository interface {
	// Create inserts a new user.
	Create(ctx context.Context, user *models.User) error

	// GetByID retrieves a user by UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// GetByEmail retrieves a user by email address.
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// GetByExternalID retrieves a user by external provider ID (OIDC).
	GetByExternalID(ctx context.Context, externalID string) (*models.User, error)

	// Update modifies an existing user.
	Update(ctx context.Context, user *models.User) error

	// SetPasswordHash updates only the password hash field.
	SetPasswordHash(ctx context.Context, id uuid.UUID, hash string) error

	// Delete removes a user by ID.
	Delete(ctx context.Context, id uuid.UUID) error

	// List returns users with pagination.
	List(ctx context.Context, limit, offset int) ([]*models.User, error)

	// Count returns total user count.
	Count(ctx context.Context) (int64, error)
}
