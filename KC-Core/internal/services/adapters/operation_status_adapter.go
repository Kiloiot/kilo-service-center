// Package adapters provides storage adapters bridging KC-DB to gRPC services.
package adapters

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// DefaultOperationCategories provides event categories for endpoint operations
var DefaultOperationCategories = []string{
	models.EventCategoryEndpoint,
	models.EventCategoryProtocol,
	models.EventCategoryBSSCI,
	models.EventCategorySCACI,
}

// OperationStatusAdapter adapts interfaces.OperationStatusRepository for gRPC handlers.
// Provides simplified access with default categories and lookback window.
type OperationStatusAdapter struct {
	repo interfaces.OperationStatusRepository
}

// NewOperationStatusAdapter creates a new adapter for operation status queries.
func NewOperationStatusAdapter(repo interfaces.OperationStatusRepository) *OperationStatusAdapter {
	return &OperationStatusAdapter{repo: repo}
}

// GetEndpointOperations retrieves recent operations for an endpoint.
// Uses default categories and lookback window from bssci constants.
func (a *OperationStatusAdapter) GetEndpointOperations(
	ctx context.Context,
	endpointID, tenantID int64,
	limit, offset int,
) ([]models.SystemEvent, error) {
	return a.repo.GetEndpointOperationsByID(
		ctx,
		endpointID,
		tenantID,
		DefaultOperationCategories,
		time.Duration(bssci.OperationLookbackHours)*time.Hour,
		limit,
		offset,
	)
}
