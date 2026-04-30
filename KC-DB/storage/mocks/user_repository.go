package mocks

import (
	"context"

	"github.com/google/uuid"

	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
)

// MockUserRepository implements interfaces.UserRepository for testing.
// Nil function pointers return ErrMockNotConfigured to fail fast.
type MockUserRepository struct {
	CreateFn          func(ctx context.Context, user *models.User) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmailFn      func(ctx context.Context, email string) (*models.User, error)
	GetByExternalIDFn func(ctx context.Context, externalID string) (*models.User, error)
	UpdateFn          func(ctx context.Context, user *models.User) error
	SetPasswordHashFn func(ctx context.Context, id uuid.UUID, hash string) error
	DeleteFn          func(ctx context.Context, id uuid.UUID) error
	ListFn            func(ctx context.Context, limit, offset int) ([]*models.User, error)
	CountFn           func(ctx context.Context) (int64, error)
}

// Ensure MockUserRepository implements interfaces.UserRepository.
var _ interfaces.UserRepository = (*MockUserRepository)(nil)

// Create inserts a new user.
func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	if m.CreateFn == nil {
		return ErrMockNotConfigured
	}
	return m.CreateFn(ctx, user)
}

// GetByID retrieves a user by UUID.
func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.GetByIDFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.GetByIDFn(ctx, id)
}

// GetByEmail retrieves a user by email address.
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.GetByEmailFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.GetByEmailFn(ctx, email)
}

// GetByExternalID retrieves a user by external provider ID (OIDC).
func (m *MockUserRepository) GetByExternalID(ctx context.Context, externalID string) (*models.User, error) {
	if m.GetByExternalIDFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.GetByExternalIDFn(ctx, externalID)
}

// Update modifies an existing user.
func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	if m.UpdateFn == nil {
		return ErrMockNotConfigured
	}
	return m.UpdateFn(ctx, user)
}

// SetPasswordHash updates only the password hash field.
func (m *MockUserRepository) SetPasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	if m.SetPasswordHashFn == nil {
		return ErrMockNotConfigured
	}
	return m.SetPasswordHashFn(ctx, id, hash)
}

// Delete removes a user by ID.
func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn == nil {
		return ErrMockNotConfigured
	}
	return m.DeleteFn(ctx, id)
}

// List returns users with pagination.
func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	if m.ListFn == nil {
		return nil, ErrMockNotConfigured
	}
	return m.ListFn(ctx, limit, offset)
}

// Count returns total user count.
func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	if m.CountFn == nil {
		return 0, ErrMockNotConfigured
	}
	return m.CountFn(ctx)
}
