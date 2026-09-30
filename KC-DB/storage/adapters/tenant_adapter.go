package adapters

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// TenantStoreAdapter adapts postgres.TenantRepository to provide
// tenant storage operations.
type TenantStoreAdapter struct {
	repo *postgres.TenantRepository
}

// NewTenantStoreAdapter creates a new adapter with the given database connection
func NewTenantStoreAdapter(repo *postgres.TenantRepository) *TenantStoreAdapter {
	return &TenantStoreAdapter{
		repo: repo,
	}
}

// ListTenants retrieves all tenants with optional status filtering
// statusFilter: "active", "inactive", or empty string for all
func (a *TenantStoreAdapter) ListTenants(ctx context.Context, statusFilter string) ([]models.Tenant, error) {
	tenants, err := a.repo.ListTenants(ctx, statusFilter)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapTenantAdapterList, err)
	}
	return tenants, nil
}

// CreateTenant creates a new tenant
func (a *TenantStoreAdapter) CreateTenant(ctx context.Context, name string, description *string) (*models.Tenant, error) {
	tenant, err := a.repo.CreateTenant(ctx, name, description)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapTenantAdapterCreate, err)
	}
	return tenant, nil
}

// GetTenantByID retrieves a tenant by ID
func (a *TenantStoreAdapter) GetTenantByID(ctx context.Context, id int64) (*models.Tenant, error) {
	tenant, err := a.repo.GetTenantByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapTenantAdapterGet, err)
	}
	return tenant, nil
}

// UpdateTenant updates tenant fields dynamically
// Pass nil for fields that should not be updated
func (a *TenantStoreAdapter) UpdateTenant(ctx context.Context, id int64, name *string, description *string) (*models.Tenant, error) {
	// Convert parameters to map for postgres layer
	updateMap := make(map[string]interface{})
	if name != nil {
		updateMap["name"] = *name
	}
	if description != nil {
		updateMap["description"] = *description
	}

	if len(updateMap) == 0 {
		return nil, errTextTenantAdapterUpdateNoFieldsUpdate
	}

	tenant, err := a.repo.UpdateTenant(ctx, id, updateMap)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapTenantAdapterUpdate, err)
	}
	return tenant, nil
}

// DeleteTenant removes a tenant - used for org creation rollback
func (a *TenantStoreAdapter) DeleteTenant(ctx context.Context, id int64) error {
	if err := a.repo.DeleteTenant(ctx, id); err != nil {
		return fmt.Errorf("%s: %w", errWrapTenantAdapterDelete, err)
	}
	return nil
}
