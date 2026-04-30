// Package interfaces defines the storage interfaces for KiloCenter
package interfaces

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/storage/models"
)

// DeviceModelRepository defines the interface for device model storage operations.
// Device models belong to manufacturers and own blueprints in the device catalog hierarchy.
// Migration 000103 creates the device_models table.
type DeviceModelRepository interface {
	// Create creates a new device model
	Create(ctx context.Context, params *models.DeviceModelCreateParams) (*models.DeviceModel, error)

	// GetByID retrieves a device model by ID with tenant isolation
	GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.DeviceModel, error)

	// GetByCode retrieves a device model by manufacturer ID and code
	GetByCode(ctx context.Context, tenantID int64, manufacturerID uuid.UUID, code string) (*models.DeviceModel, error)

	// GetByTypeEUI retrieves a device model by Type EUI with tenant isolation
	// Type EUI is an 8-byte MIOTY identifier
	GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.DeviceModel, error)

	// ListByManufacturer retrieves device models for a manufacturer with pagination
	ListByManufacturer(ctx context.Context, tenantID int64, manufacturerID uuid.UUID, limit, offset int) ([]*models.DeviceModel, error)

	// List retrieves device models for a tenant with pagination and optional filters
	List(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModel, error)

	// ListWithManufacturer retrieves device models with joined manufacturer data
	ListWithManufacturer(ctx context.Context, params *models.DeviceModelListParams) ([]*models.DeviceModelWithManufacturer, error)

	// Count returns the total count of device models for a tenant
	Count(ctx context.Context, tenantID int64) (int64, error)

	// CountByManufacturer returns the count of device models for a manufacturer
	CountByManufacturer(ctx context.Context, tenantID int64, manufacturerID uuid.UUID) (int64, error)

	// Update updates an existing device model
	Update(ctx context.Context, tenantID int64, id uuid.UUID, params *models.DeviceModelUpdateParams) error

	// Delete deletes a device model by ID with tenant isolation
	// Returns ErrRecordNotFound if device model doesn't exist
	// Returns ErrForeignKeyViolation if device model has associated blueprints
	Delete(ctx context.Context, tenantID int64, id uuid.UUID) error
}
