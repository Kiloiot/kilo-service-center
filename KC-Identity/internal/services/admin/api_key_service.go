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
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/grpcservices"
	"github.com/google/uuid"
)

// APIKeyAdminService implements grpcservices.APIKeyService.
type APIKeyAdminService struct {
	store  interfaces.APIKeyRepository
	logger logger.Logger
}

// NewAPIKeyAdminService creates a new API key admin service.
func NewAPIKeyAdminService(store interfaces.APIKeyRepository, log logger.Logger) *APIKeyAdminService {
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
		s.logger.ErrorContext(ctx, "failed to generate key bytes", "error", err)
		return nil, fmt.Errorf("generate key: %w", err)
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
		s.logger.ErrorContext(ctx, "failed to create api key", "name", req.Name, "error", err)
		return nil, fmt.Errorf("create api key: %w", err)
	}

	s.logger.InfoContext(ctx, "api key created", "keyId", apiKey.ID, "name", req.Name, "type", req.KeyType)

	return &grpcservices.APIKeyCreateResponse{
		Key:    key, // Only returned once on creation
		APIKey: apiKey,
	}, nil
}

// GetByID retrieves an API key by ID.
func (s *APIKeyAdminService) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	key, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get api key", "keyId", id, "error", err)
		return nil, fmt.Errorf("get api key: %w", err)
	}
	return key, nil
}

// Delete removes an API key.
func (s *APIKeyAdminService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("get api key: %w", err)
	}

	if err := s.store.Delete(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete api key", "keyId", id, "error", err)
		return fmt.Errorf("delete api key: %w", err)
	}

	s.logger.InfoContext(ctx, "api key deleted", "keyId", id)
	return nil
}

// List returns API keys with pagination scoped to a tenant and organization.
func (s *APIKeyAdminService) List(ctx context.Context, tenantID int64, orgID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]*models.APIKey, int64, error) {
	keys, err := s.store.List(ctx, tenantID, orgID, userID, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list api keys", "error", err)
		return nil, 0, fmt.Errorf("list api keys: %w", err)
	}

	count, err := s.store.Count(ctx, tenantID, orgID, userID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to count api keys", "error", err)
		return nil, 0, fmt.Errorf("count api keys: %w", err)
	}

	return keys, count, nil
}

// GetByIDAndOrg retrieves an API key by ID with organization ownership check.
func (s *APIKeyAdminService) GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*models.APIKey, error) {
	key, err := s.store.GetByIDAndOrg(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get api key by org", "keyId", id, "orgId", orgID, "error", err)
		return nil, fmt.Errorf("get api key: %w", err)
	}
	return key, nil
}

// DeleteByIDAndOrg removes an API key with organization ownership check.
func (s *APIKeyAdminService) DeleteByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) error {
	// Verify ownership before deletion
	_, err := s.store.GetByIDAndOrg(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, interfaces.ErrRecordNotFound) {
			return ErrAPIKeyNotFound
		}
		return fmt.Errorf("get api key: %w", err)
	}

	if err := s.store.DeleteByIDAndOrg(ctx, id, orgID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete api key", "keyId", id, "orgId", orgID, "error", err)
		return fmt.Errorf("delete api key: %w", err)
	}

	s.logger.InfoContext(ctx, "api key deleted", "keyId", id, "orgId", orgID)
	return nil
}

// Ensure APIKeyAdminService implements grpcservices.APIKeyService
var _ grpcservices.APIKeyService = (*APIKeyAdminService)(nil)
