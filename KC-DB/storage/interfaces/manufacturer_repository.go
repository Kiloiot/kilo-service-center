// Package interfaces defines the storage interfaces for KiloCenter
package interfaces

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/storage/models"
)

// ManufacturerRepository defines the interface for manufacturer storage operations.
// Manufacturers are tenant-isolated and own device models in the device catalog hierarchy.
// Migration 000102 creates the manufacturers table.
// Migration 000107 simplifies to name + website only.
type ManufacturerRepository interface {
	// Create creates a new manufacturer
	Create(ctx context.Context, params *models.ManufacturerCreateParams) (*models.Manufacturer, error)

	// GetByID retrieves a manufacturer by ID with tenant isolation
	GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Manufacturer, error)

	// List retrieves manufacturers for a tenant with pagination and optional search
	List(ctx context.Context, params *models.ManufacturerListParams) ([]*models.Manufacturer, error)

	// Count returns the total count of manufacturers for a tenant
	Count(ctx context.Context, tenantID int64) (int64, error)

	// Update updates an existing manufacturer
	Update(ctx context.Context, tenantID int64, id uuid.UUID, params *models.ManufacturerUpdateParams) error

	// Delete deletes a manufacturer by ID with tenant isolation
	// Returns ErrRecordNotFound if manufacturer doesn't exist
	// Returns ErrForeignKeyViolation if manufacturer has associated device models
	Delete(ctx context.Context, tenantID int64, id uuid.UUID) error

	// SetVerified updates the is_verified flag for a manufacturer
	SetVerified(ctx context.Context, tenantID int64, id uuid.UUID, verified bool) error
}
