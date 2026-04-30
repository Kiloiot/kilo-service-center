// Package interfaces defines the storage interfaces for KiloCenter
package interfaces

import (
	"context"
	"time"

	"github.com/kilocenter/KC-DB/storage/models"
)

// OperationStatusRepository defines the interface for operation status queries
// Used by KC-Core adapters to fetch endpoint operation history
type OperationStatusRepository interface {
	// GetEndpointOperationsByID retrieves recent operations for an endpoint
	// Searches across event data, title, and description fields
	// Parameters:
	//   - endpointID: The endpoint database ID
	//   - tenantID: Tenant scope for security isolation
	//   - categories: Event categories to include
	//   - since: Lookback window
	//   - limit: Maximum number of results
	//   - offset: Pagination offset
	GetEndpointOperationsByID(
		ctx context.Context,
		endpointID int64,
		tenantID int64,
		categories []string,
		since time.Duration,
		limit int,
		offset int,
	) ([]models.SystemEvent, error)
}
