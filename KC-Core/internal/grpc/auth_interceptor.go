// Package grpc provides gRPC server implementation and authentication interceptors for KiloCenter.
package grpc

import (
	"context"

	"github.com/google/uuid"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/grpc/interceptors"
	"github.com/kilocenter/KC-DB/storage/models"
)

// AuthInterceptor is a type alias for the extracted interceptors.AuthInterceptor.
type AuthInterceptor = interceptors.AuthInterceptor

// AuthConfig is a type alias for the extracted interceptors.AuthConfig.
type AuthConfig = interceptors.AuthConfig

// OrganizationResolver is a type alias for the extracted interceptors.OrganizationResolver.
type OrganizationResolver = interceptors.OrganizationResolver

// TenantResolver is a type alias for the extracted interceptors.TenantResolver.
type TenantResolver = interceptors.TenantResolver

// NewAuthInterceptor delegates to the extracted constructor.
var NewAuthInterceptor = interceptors.NewAuthInterceptor

// APIKeyAuthenticator is the KC-Core-internal contract returning *models.APIKey.
// Satisfied by infra.Storage.APIKeys() — the builder wraps this with CoreAPIKeyAdapter
// before passing to the extracted interceptor.
type APIKeyAuthenticator interface {
	GetByHash(ctx context.Context, hash string) (*models.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// CoreAPIKeyAdapter bridges APIKeyAuthenticator (*models.APIKey) → interceptors.APIKeyAuthenticator (*APIKeyRecord).
type CoreAPIKeyAdapter struct {
	repo APIKeyAuthenticator
}

// NewCoreAPIKeyAdapter wraps a KC-Core APIKeyAuthenticator for use with the extracted interceptor.
func NewCoreAPIKeyAdapter(repo APIKeyAuthenticator) interceptors.APIKeyAuthenticator {
	return &CoreAPIKeyAdapter{repo: repo}
}

// LookupByHash retrieves and converts an API key by its SHA-256 hash.
func (a *CoreAPIKeyAdapter) LookupByHash(ctx context.Context, hash string) (*interceptors.APIKeyRecord, error) {
	key, err := a.repo.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return &interceptors.APIKeyRecord{
		ID:             key.ID,
		TenantID:       key.TenantID,
		OrganizationID: key.OrgID,
		UserID:         key.UserID,
		IsActive:       key.IsActive,
		IsExpired:      key.IsExpired(),
	}, nil
}

// UpdateLastUsed delegates to the underlying repository.
func (a *CoreAPIKeyAdapter) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return a.repo.UpdateLastUsed(ctx, id)
}

// GetTenantFromContext extracts tenant ID from gRPC context.
// Delegates to KC-Core/pkg/grpc for cross-service use.
func GetTenantFromContext(ctx context.Context) (int64, error) {
	return grpcerrors.GetTenantFromContext(ctx)
}

// GetUserFromContext extracts user ID from gRPC context as UUID.
// Delegates to KC-Core/pkg/grpc for cross-service use.
func GetUserFromContext(ctx context.Context) (uuid.UUID, error) {
	return grpcerrors.GetUserFromContext(ctx)
}
