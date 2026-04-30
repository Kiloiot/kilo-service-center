package admin

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-Identity/internal/services/auth"
	"github.com/kilocenter/KC-Identity/internal/services/grpcservices"
)

// UserAdminService implements grpcservices.AdminUserService.
type UserAdminService struct {
	store  UserAdminStore
	logger logger.Logger
}

// NewUserAdminService creates a new user admin service.
func NewUserAdminService(store UserAdminStore, log logger.Logger) *UserAdminService {
	return &UserAdminService{
		store:  store,
		logger: log,
	}
}

// Create creates a new user.
func (s *UserAdminService) Create(ctx context.Context, req *grpcservices.UserCreateRequest) (*models.User, error) {
	req.Email = auth.NormalizeEmail(req.Email)

	// Validate password strength
	if err := auth.ValidatePassword(req.Password); err != nil {
		return nil, ErrUserPasswordWeak
	}

	// Check for duplicate email
	_, err := s.store.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrUserEmailExists
	}
	if !errors.Is(err, interfaces.ErrRecordNotFound) {
		s.logger.ErrorContext(ctx, "failed to check email existence", "error", err)
		return nil, fmt.Errorf("check email: %w", err)
	}

	// Generate salt and hash password
	salt := make([]byte, config.AuthPBKDF2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		s.logger.ErrorContext(ctx, "failed to generate salt", "error", err)
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	passwordHash := auth.HashPassword(req.Password, salt, config.AuthPBKDF2Iterations)

	// Create user model
	user := &models.User{
		ID:                   uuid.New(),
		Email:                req.Email,
		EmailVerified:        req.EmailVerified,
		PasswordHash:         &passwordHash,
		IsAdmin:              req.IsAdmin,
		IsActive:             req.IsActive,
		IsTenantManager:      req.IsTenantManager,
		IsBaseStationManager: req.IsBaseStationManager,
		IsEndpointManager:    req.IsEndpointManager,
		Note:                 &req.Note,
	}

	// Set profile fields if provided
	if req.FirstName != "" {
		user.FirstName = &req.FirstName
	}
	if req.LastName != "" {
		user.LastName = &req.LastName
	}
	if req.CompanyName != "" {
		user.CompanyName = &req.CompanyName
	}

	// Admin users automatically get all manager permissions
	if user.IsAdmin {
		user.IsTenantManager = true
		user.IsBaseStationManager = true
		user.IsEndpointManager = true
	}

	if err := s.store.Create(ctx, user); err != nil {
		s.logger.ErrorContext(ctx, "failed to create user", "error", err)
		return nil, fmt.Errorf("create user: %w", err)
	}

	s.logger.InfoContext(ctx, "user created", "userId", user.ID)
	return user, nil
}

// GetByID retrieves a user by ID.
func (s *UserAdminService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get user", "userId", id, "error", err)
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// Update modifies an existing user.
func (s *UserAdminService) Update(ctx context.Context, id uuid.UUID, req *grpcservices.UserUpdateRequest) (*models.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get user for update", "userId", id, "error", err)
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Check for duplicate email if email is being changed
	if req.Email != nil {
		normalized := auth.NormalizeEmail(*req.Email)
		req.Email = &normalized
	}
	if req.Email != nil && *req.Email != auth.NormalizeEmail(user.Email) {
		existing, err := s.store.GetByEmail(ctx, *req.Email)
		if err == nil && existing != nil && existing.ID != id {
			return nil, ErrUserEmailExists
		}
		if err != nil && !errors.Is(err, interfaces.ErrRecordNotFound) {
			s.logger.ErrorContext(ctx, "failed to check email existence", "error", err)
			return nil, fmt.Errorf("check email: %w", err)
		}
		user.Email = *req.Email
	}

	// Apply partial updates
	if req.Note != nil {
		user.Note = req.Note
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.EmailVerified != nil {
		user.EmailVerified = *req.EmailVerified
	}
	if req.IsTenantManager != nil {
		user.IsTenantManager = *req.IsTenantManager
	}
	if req.IsBaseStationManager != nil {
		user.IsBaseStationManager = *req.IsBaseStationManager
	}
	if req.IsEndpointManager != nil {
		user.IsEndpointManager = *req.IsEndpointManager
	}

	// Admin users automatically get all manager permissions
	if req.IsAdmin != nil && *req.IsAdmin {
		user.IsTenantManager = true
		user.IsBaseStationManager = true
		user.IsEndpointManager = true
	}

	if err := s.store.Update(ctx, user); err != nil {
		s.logger.ErrorContext(ctx, "failed to update user", "userId", id, "error", err)
		return nil, fmt.Errorf("update user: %w", err)
	}

	s.logger.InfoContext(ctx, "user updated", "userId", id)
	return user, nil
}

// Delete removes a user.
func (s *UserAdminService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get user for delete", "userId", id, "error", err)
		return fmt.Errorf("get user: %w", err)
	}

	if err := s.store.Delete(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete user", "userId", id, "error", err)
		return ErrUserDeleteFailed
	}

	s.logger.InfoContext(ctx, "user deleted", "userId", id)
	return nil
}

// List returns paginated users.
func (s *UserAdminService) List(ctx context.Context, limit, offset int) ([]*models.User, int64, error) {
	users, err := s.store.List(ctx, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list users", "error", err)
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	count, err := s.store.Count(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to count users", "error", err)
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	return users, count, nil
}

// UpdatePassword changes a user's password.
func (s *UserAdminService) UpdatePassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	_, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get user for password change", "userId", id, "error", err)
		return fmt.Errorf("get user: %w", err)
	}

	if err := auth.ValidatePassword(newPassword); err != nil {
		return ErrUserPasswordWeak
	}

	salt := make([]byte, config.AuthPBKDF2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		s.logger.ErrorContext(ctx, "failed to generate salt", "error", err)
		return fmt.Errorf("generate salt: %w", err)
	}
	passwordHash := auth.HashPassword(newPassword, salt, config.AuthPBKDF2Iterations)

	if err := s.store.SetPasswordHash(ctx, id, passwordHash); err != nil {
		s.logger.ErrorContext(ctx, "failed to set password hash", "userId", id, "error", err)
		return fmt.Errorf("set password: %w", err)
	}

	s.logger.InfoContext(ctx, "user password changed", "userId", id)
	return nil
}

// GetByEmail retrieves a user by email address.
func (s *UserAdminService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	email = auth.NormalizeEmail(email)
	user, err := s.store.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

// Ensure UserAdminService implements grpcservices.AdminUserService
var _ grpcservices.AdminUserService = (*UserAdminService)(nil)
