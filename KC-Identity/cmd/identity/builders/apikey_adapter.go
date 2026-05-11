package builders

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	identitygrpc "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/grpc"
	"github.com/google/uuid"
)

// apiKeyLookupAdapter bridges interfaces.APIKeyRepository to identitygrpc.APIKeyLookup.
type apiKeyLookupAdapter struct {
	repo interfaces.APIKeyRepository
}

// newAPIKeyLookupAdapter creates a new adapter.
func newAPIKeyLookupAdapter(repo interfaces.APIKeyRepository) identitygrpc.APIKeyLookup {
	return &apiKeyLookupAdapter{repo: repo}
}

// GetByHash looks up an API key by hash and maps to APIKeyInfo.
func (a *apiKeyLookupAdapter) GetByHash(ctx context.Context, hash string) (identitygrpc.APIKeyInfo, error) {
	key, err := a.repo.GetByHash(ctx, hash)
	if err != nil {
		return identitygrpc.APIKeyInfo{}, err
	}

	info := identitygrpc.APIKeyInfo{
		ID:             key.ID,
		TenantID:       key.TenantID,
		OrganizationID: key.OrgID,
		IsActive:       key.IsActive,
		IsExpired:      key.IsExpired(),
	}
	if key.UserID != nil {
		info.UserID = *key.UserID
	}
	return info, nil
}

// UpdateLastUsed updates the last-used timestamp for an API key.
func (a *apiKeyLookupAdapter) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return a.repo.UpdateLastUsed(ctx, id)
}
