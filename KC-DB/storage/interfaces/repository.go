// Package interfaces defines the storage interfaces for KiloCenter
package interfaces

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// EndPointSessionRepository defines operations for endpoint session management
type EndPointSessionRepository interface {
	// Create creates a new endpoint session
	Create(ctx context.Context, session *models.EndPointSession) error

	// GetActive retrieves the active session for an endpoint
	GetActive(ctx context.Context, endpointID string) (*models.EndPointSession, error)

	// GetByID retrieves a session by ID
	GetByID(ctx context.Context, id string) (*models.EndPointSession, error)

	// Update updates an endpoint session
	Update(ctx context.Context, session *models.EndPointSession) error
}
