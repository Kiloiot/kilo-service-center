package adapters

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-DB/storage/postgres"
)

// TenantStoreAdapter adapts postgres.TenantRepository to provide
// tenant storage operations.
type TenantStoreAdapter struct {
	repo *postgres.TenantRepository
}

// NewTenantStoreAdapter creates a new adapter with the given database connection
func NewTenantStoreAdapter(db *sqlx.DB) *TenantStoreAdapter {
	return &TenantStoreAdapter{
		repo: postgres.NewTenantRepository(db),
	}
}

// ListTenants retrieves all tenants with optional status filtering
// statusFilter: "active", "inactive", or empty string for all
func (a *TenantStoreAdapter) ListTenants(ctx context.Context, statusFilter string) ([]models.Tenant, error) {
	tenants, err := a.repo.ListTenants(ctx, statusFilter)
	if err != nil {
		return nil, fmt.Errorf("tenant adapter: list: %w", err)
	}
	return tenants, nil
}

// CreateTenant creates a new tenant
func (a *TenantStoreAdapter) CreateTenant(ctx context.Context, name string, description *string) (*models.Tenant, error) {
	tenant, err := a.repo.CreateTenant(ctx, name, description)
	if err != nil {
		return nil, fmt.Errorf("tenant adapter: create: %w", err)
	}
	return tenant, nil
}

// GetTenantByID retrieves a tenant by ID
func (a *TenantStoreAdapter) GetTenantByID(ctx context.Context, id int64) (*models.Tenant, error) {
	tenant, err := a.repo.GetTenantByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("tenant adapter: get: %w", err)
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
		return nil, fmt.Errorf("tenant adapter: update: no fields to update")
	}

	tenant, err := a.repo.UpdateTenant(ctx, id, updateMap)
	if err != nil {
		return nil, fmt.Errorf("tenant adapter: update: %w", err)
	}
	return tenant, nil
}

// SetTenantStatus updates tenant status (active/inactive)
func (a *TenantStoreAdapter) SetTenantStatus(ctx context.Context, id int64, active bool) error {
	if err := a.repo.SetStatus(ctx, id, active); err != nil {
		return fmt.Errorf("tenant adapter: set_status: %w", err)
	}
	return nil
}

// DeleteTenant removes a tenant - used for org creation rollback
func (a *TenantStoreAdapter) DeleteTenant(ctx context.Context, id int64) error {
	if err := a.repo.DeleteTenant(ctx, id); err != nil {
		return fmt.Errorf("tenant adapter: delete: %w", err)
	}
	return nil
}
