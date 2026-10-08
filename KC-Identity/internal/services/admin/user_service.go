package admin

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/auth"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/google/uuid"
)

// UserAdminService implements grpcservices.AdminUserService.
type UserAdminService struct {
	store    UserAdminStore
	sessions SessionRevoker
	logger   logger.Logger
}

// NewUserAdminService creates a new user admin service.
func NewUserAdminService(store UserAdminStore, sessions SessionRevoker, log logger.Logger) *UserAdminService {
	return &UserAdminService{
		store:    store,
		sessions: sessions,
		logger:   log,
	}
}

// Create creates a new user.
func (s *UserAdminService) Create(ctx context.Context, req *grpcservices.UserCreateRequest) (*models.User, error) {
	req.Email = auth.NormalizeEmail(req.Email)

	// Validate password strength
	if err := auth.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check for duplicate email
	_, err := s.store.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrUserEmailExists
	}
	if !errors.Is(err, storage.ErrRecordNotFound) {
		s.logger.ErrorContext(ctx, LogUserEmailCheckFailed, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpCheckEmail, err)
	}

	// Generate salt and hash password
	salt := make([]byte, config.AuthPBKDF2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		s.logger.ErrorContext(ctx, LogUserSaltGenerationFailed, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGenerateSalt, err)
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
		s.logger.ErrorContext(ctx, LogUserCreateFailed, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpCreateUser, err)
	}

	s.logger.InfoContext(ctx, LogUserCreated, logger.FieldUserIDCamel, user.ID)
	return user, nil
}

// GetByID retrieves a user by ID.
func (s *UserAdminService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, LogUserGetFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetUser, err)
	}
	return user, nil
}

// Update modifies an existing user.
func (s *UserAdminService) Update(ctx context.Context, id uuid.UUID, req *grpcservices.UserUpdateRequest) (*models.User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, LogUserGetForUpdateFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetUser, err)
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
		if err != nil && !errors.Is(err, storage.ErrRecordNotFound) {
			s.logger.ErrorContext(ctx, LogUserEmailCheckFailed, logger.FieldError, err)
			return nil, fmt.Errorf("%s: %w", errOpCheckEmail, err)
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
		s.logger.ErrorContext(ctx, LogUserUpdateFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpUpdateUser, err)
	}

	s.logger.InfoContext(ctx, LogUserUpdated, logger.FieldUserIDCamel, id)
	return user, nil
}

// Delete removes a user.
func (s *UserAdminService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, LogUserGetForDeleteFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpGetUser, err)
	}

	if err := s.store.Delete(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, LogUserDeleteFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return ErrUserDeleteFailed
	}

	s.logger.InfoContext(ctx, LogUserDeleted, logger.FieldUserIDCamel, id)
	return nil
}

// List returns paginated users.
func (s *UserAdminService) List(ctx context.Context, limit, offset int) ([]*models.User, int64, error) {
	users, err := s.store.List(ctx, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogUserListFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpListUsers, err)
	}

	count, err := s.store.Count(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, LogUserCountFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpCountUsers, err)
	}

	return users, count, nil
}

// UpdatePassword changes a user's password.
func (s *UserAdminService) UpdatePassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	_, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, LogUserGetForPasswordChangeFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpGetUser, err)
	}

	if err := auth.ValidatePassword(newPassword); err != nil {
		return err
	}

	salt := make([]byte, config.AuthPBKDF2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		s.logger.ErrorContext(ctx, LogUserSaltGenerationFailed, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpGenerateSalt, err)
	}
	passwordHash := auth.HashPassword(newPassword, salt, config.AuthPBKDF2Iterations)

	if err := s.store.SetPasswordHash(ctx, id, passwordHash); err != nil {
		s.logger.ErrorContext(ctx, LogUserSetPasswordHashFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpSetPassword, err)
	}

	// Revoked after the new hash is stored, so no login with the old password can slip in between.
	if err := s.sessions.RevokeByUserID(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, LogUserSessionsRevokeFailed, logger.FieldUserIDCamel, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpRevokeSessions, err)
	}

	s.logger.InfoContext(ctx, LogUserPasswordChanged, logger.FieldUserIDCamel, id)
	return nil
}

// GetByEmail retrieves a user by email address.
func (s *UserAdminService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	email = auth.NormalizeEmail(email)
	user, err := s.store.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetUserByEmail, err)
	}
	return user, nil
}

// Ensure UserAdminService implements grpcservices.AdminUserService
var _ grpcservices.AdminUserService = (*UserAdminService)(nil)
