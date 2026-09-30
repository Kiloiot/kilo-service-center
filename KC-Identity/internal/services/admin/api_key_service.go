// Package admin provides admin-facing service implementations.
package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/google/uuid"
)

// APIKeyStore persists API keys for tenants and organizations.
type APIKeyStore interface {
	Create(ctx context.Context, key *models.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, error)
	Count(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID) (int64, error)
	GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error)
	DeleteByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) error
}

// APIKeyAdminService implements grpcservices.APIKeyService.
type APIKeyAdminService struct {
	store  APIKeyStore
	logger logger.Logger
}

// NewAPIKeyAdminService creates a new API key admin service.
func NewAPIKeyAdminService(store APIKeyStore, log logger.Logger) *APIKeyAdminService {
	return &APIKeyAdminService{
		store:  store,
		logger: log,
	}
}

// Create generates a new API key.
func (s *APIKeyAdminService) Create(ctx context.Context, req *grpcservices.APIKeyCreateRequest) (*grpcservices.APIKeyCreateResponse, error) {
	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		s.logger.ErrorContext(ctx, LogAPIKeyGenerateFailed, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGenerateKey, err)
	}
	key := base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(key))
	keyHash := hex.EncodeToString(hash[:])

	// Extract prefix for identification
	keyPrefix := key[:8]

	apiKey := &models.APIKey{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		OrgID:     req.OrgID,
		UserID:    req.UserID,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeyType:   req.KeyType,
		ExpiresAt: req.ExpiresAt,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.store.Create(ctx, apiKey); err != nil {
		s.logger.ErrorContext(ctx, LogAPIKeyCreateFailed, logger.FieldName, req.Name, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpCreateAPIKey, err)
	}

	s.logger.InfoContext(ctx, LogAPIKeyCreated, logger.FieldKeyID, apiKey.ID, logger.FieldName, req.Name, logger.FieldType, req.KeyType)

	return &grpcservices.APIKeyCreateResponse{
		Key:    key, // Only returned once on creation
		APIKey: apiKey,
	}, nil
}

// GetByID retrieves an API key by ID.
func (s *APIKeyAdminService) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	key, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		s.logger.ErrorContext(ctx, LogAPIKeyGetFailed, logger.FieldKeyID, id, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetAPIKey, err)
	}
	return key, nil
}

// Delete removes an API key.
func (s *APIKeyAdminService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetAPIKey, err)
	}

	if err := s.store.Delete(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, LogAPIKeyDeleteFailed, logger.FieldKeyID, id, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpDeleteAPIKey, err)
	}

	s.logger.InfoContext(ctx, LogAPIKeyDeleted, logger.FieldKeyID, id)
	return nil
}

// List returns API keys with pagination scoped to a tenant and organization.
func (s *APIKeyAdminService) List(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, int64, error) {
	keys, err := s.store.List(ctx, tenantID, orgID, userID, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAPIKeyListFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpListAPIKeys, err)
	}

	count, err := s.store.Count(ctx, tenantID, orgID, userID)
	if err != nil {
		s.logger.ErrorContext(ctx, LogAPIKeyCountFailed, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errOpCountAPIKeys, err)
	}

	return keys, count, nil
}

// GetByIDAndOrg retrieves an API key by ID with organization ownership check.
func (s *APIKeyAdminService) GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error) {
	key, err := s.store.GetByIDAndOrg(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		s.logger.ErrorContext(ctx, LogAPIKeyGetByOrgFailed, logger.FieldKeyID, id, logger.FieldOrgIDCamel, orgID, logger.FieldError, err)
		return nil, fmt.Errorf("%s: %w", errOpGetAPIKey, err)
	}
	return key, nil
}

// DeleteByIDAndOrg removes an API key with organization ownership check.
func (s *APIKeyAdminService) DeleteByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) error {
	// Verify ownership before deletion
	_, err := s.store.GetByIDAndOrg(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("%s: %w", errOpGetAPIKey, err)
	}

	if err := s.store.DeleteByIDAndOrg(ctx, id, orgID); err != nil {
		s.logger.ErrorContext(ctx, LogAPIKeyDeleteFailed, logger.FieldKeyID, id, logger.FieldOrgIDCamel, orgID, logger.FieldError, err)
		return fmt.Errorf("%s: %w", errOpDeleteAPIKey, err)
	}

	s.logger.InfoContext(ctx, LogAPIKeyDeleted, logger.FieldKeyID, id, logger.FieldOrgIDCamel, orgID)
	return nil
}

// Ensure APIKeyAdminService implements grpcservices.APIKeyService
var _ grpcservices.APIKeyService = (*APIKeyAdminService)(nil)
