// Package interfaces defines the storage interfaces for KiloCenter
package interfaces

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/storage/models"
)

// BlueprintRepository defines the interface for blueprint storage operations.
// Blueprints define payload decoding specifications per MIOTY Application Layer.
// Each blueprint belongs to a device model and contains a JSON specification.
// Migration 000104 creates the blueprints table.
type BlueprintRepository interface {
	// Create creates a new blueprint
	Create(ctx context.Context, params *models.BlueprintCreateParams) (*models.Blueprint, error)

	// GetByID retrieves a blueprint by ID with tenant isolation
	GetByID(ctx context.Context, tenantID int64, id uuid.UUID) (*models.Blueprint, error)

	// GetByVersion retrieves a blueprint by device model ID and version
	GetByVersion(ctx context.Context, tenantID int64, deviceModelID uuid.UUID, version string) (*models.Blueprint, error)

	// GetByTypeEUI retrieves a blueprint by Type EUI with tenant isolation
	// Type EUI is an 8-byte MIOTY identifier unique per tenant
	GetByTypeEUI(ctx context.Context, tenantID int64, typeEUI []byte) (*models.Blueprint, error)

	// GetDefaultForModel retrieves the default blueprint for a device model
	// Returns nil if no default is set
	GetDefaultForModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (*models.Blueprint, error)

	// ListByDeviceModel retrieves blueprints for a device model with pagination
	ListByDeviceModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID, limit, offset int) ([]*models.Blueprint, error)

	// List retrieves blueprints for a tenant with pagination and optional filters
	List(ctx context.Context, params *models.BlueprintListParams) ([]*models.Blueprint, error)

	// ListWithModel retrieves blueprints with joined device model and manufacturer data
	ListWithModel(ctx context.Context, params *models.BlueprintListParams) ([]*models.BlueprintWithModel, error)

	// Count returns the total count of blueprints for a tenant
	Count(ctx context.Context, tenantID int64) (int64, error)

	// CountByDeviceModel returns the count of blueprints for a device model
	CountByDeviceModel(ctx context.Context, tenantID int64, deviceModelID uuid.UUID) (int64, error)

	// Update updates an existing blueprint
	Update(ctx context.Context, tenantID int64, id uuid.UUID, params *models.BlueprintUpdateParams) error

	// SetDefault sets a blueprint as the default for its device model
	// Clears any existing default for the same model
	SetDefault(ctx context.Context, tenantID int64, id uuid.UUID) error

	// ClearDefault clears the default flag for a blueprint
	ClearDefault(ctx context.Context, tenantID int64, id uuid.UUID) error

	// UpdateRegistryInfo updates the GitHub registry metadata for a blueprint
	UpdateRegistryInfo(ctx context.Context, tenantID int64, id uuid.UUID, repo, commitSHA, prURL string, verified bool) error

	// Delete deletes a blueprint by ID with tenant isolation
	// Returns ErrRecordNotFound if blueprint doesn't exist
	Delete(ctx context.Context, tenantID int64, id uuid.UUID) error
}
