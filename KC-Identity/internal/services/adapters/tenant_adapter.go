// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"

	dbadapters "github.com/Kiloiot/kilo-service-center/KC-DB/storage/adapters"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	adminservice "github.com/Kiloiot/kilo-service-center/KC-Identity/internal/services/admin"
)

// TenantStoreAdapterWrapper wraps TenantStoreAdapter to implement admin.TenantStore.
// The KC-DB adapter uses *string for description, but admin.TenantStore uses string.
type TenantStoreAdapterWrapper struct {
	adapter *dbadapters.TenantStoreAdapter
}

// NewTenantStoreAdapterWrapper creates a new wrapper.
func NewTenantStoreAdapterWrapper(adapter *dbadapters.TenantStoreAdapter) *TenantStoreAdapterWrapper {
	return &TenantStoreAdapterWrapper{adapter: adapter}
}

// GetTenant retrieves a tenant by ID.
func (w *TenantStoreAdapterWrapper) GetTenant(ctx context.Context, id int64) (*models.Tenant, error) {
	return w.adapter.GetTenantByID(ctx, id)
}

// CreateTenant creates a new tenant (bridges string to *string).
func (w *TenantStoreAdapterWrapper) CreateTenant(ctx context.Context, name, description string) (*models.Tenant, error) {
	var descPtr *string
	if description != "" {
		descPtr = &description
	}
	return w.adapter.CreateTenant(ctx, name, descPtr)
}

// DeleteTenant deletes a tenant by ID.
func (w *TenantStoreAdapterWrapper) DeleteTenant(ctx context.Context, id int64) error {
	return w.adapter.DeleteTenant(ctx, id)
}

// Ensure TenantStoreAdapterWrapper implements admin.TenantStore
var _ adminservice.TenantStore = (*TenantStoreAdapterWrapper)(nil)
