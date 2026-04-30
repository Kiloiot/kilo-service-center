// Package interfaces defines the storage interfaces for KiloCenter
package interfaces

import (
	"context"

	"github.com/kilocenter/KC-DB/storage/models"
)

// IntegrationRepository defines the interface for integration storage operations (CRUD for API parity)
type IntegrationRepository interface {
	// Create creates a new integration
	Create(ctx context.Context, integration *models.Integration) error

	// GetByID retrieves an integration by ID with tenant isolation
	GetByID(ctx context.Context, id int64, tenantID int64) (*models.Integration, error)

	// ListByTenant retrieves paginated integrations for a tenant
	// Returns integrations, total count, and error
	ListByTenant(ctx context.Context, tenantID int64, limit, offset int) ([]*models.Integration, int64, error)

	// Update updates an existing integration
	Update(ctx context.Context, integration *models.Integration) error

	// Delete deletes an integration by ID with tenant isolation
	Delete(ctx context.Context, id int64, tenantID int64) error
}
