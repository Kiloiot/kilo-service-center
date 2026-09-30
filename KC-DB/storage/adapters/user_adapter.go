package adapters

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/google/uuid"
)

// UserStoreAdapter adapts postgres.UserRepository to provide
// user storage operations for authentication.
type UserStoreAdapter struct {
	repo *postgres.UserRepository
}

// NewUserStoreAdapter creates a new adapter with the given database connection
func NewUserStoreAdapter(repo *postgres.UserRepository) *UserStoreAdapter {
	return &UserStoreAdapter{repo: repo}
}

// GetByEmail retrieves a user by email address
func (a *UserStoreAdapter) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := a.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUserAdapterGetByEmail, err)
	}
	return user, nil
}

// GetByID retrieves a user by UUID
func (a *UserStoreAdapter) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUserAdapterGetByID, err)
	}
	return user, nil
}

// GetByExternalID retrieves a user by external provider ID (OIDC)
func (a *UserStoreAdapter) GetByExternalID(ctx context.Context, externalID string) (*models.User, error) {
	user, err := a.repo.GetByExternalID(ctx, externalID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUserAdapterGetByExternalID, err)
	}
	return user, nil
}

// Create creates a new user
func (a *UserStoreAdapter) Create(ctx context.Context, user *models.User) error {
	if err := a.repo.Create(ctx, user); err != nil {
		return fmt.Errorf("%s: %w", errWrapUserAdapterCreate, err)
	}
	return nil
}

// Update modifies an existing user
func (a *UserStoreAdapter) Update(ctx context.Context, user *models.User) error {
	if err := a.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("%s: %w", errWrapUserAdapterUpdate, err)
	}
	return nil
}

// SetPasswordHash updates only the password hash field
func (a *UserStoreAdapter) SetPasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	if err := a.repo.SetPasswordHash(ctx, id, hash); err != nil {
		return fmt.Errorf("%s: %w", errWrapUserAdapterSetPasswordHash, err)
	}
	return nil
}

// List returns users with pagination
func (a *UserStoreAdapter) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	users, err := a.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUserAdapterList, err)
	}
	return users, nil
}

// Count returns total user count
func (a *UserStoreAdapter) Count(ctx context.Context) (int64, error) {
	count, err := a.repo.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", errWrapUserAdapterCount, err)
	}
	return count, nil
}

// Delete removes a user by ID
func (a *UserStoreAdapter) Delete(ctx context.Context, id uuid.UUID) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("%s: %w", errWrapUserAdapterDelete, err)
	}
	return nil
}
