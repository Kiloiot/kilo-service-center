package interfaces

import (
	"context"
	"time"

	"github.com/kilocenter/KC-DB/storage/models"
)

// BaseStationRepository defines the interface for Base Station data operations
type BaseStationRepository interface {
	// Create creates a new Base Station
	Create(ctx context.Context, baseStation *models.BaseStation) error

	// GetByID retrieves a Base Station by ID
	GetByID(ctx context.Context, tenantID, id int64) (*models.BaseStation, error)

	// GetByEUI retrieves a Base Station by EUI within a tenant
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error)

	// GetByEUIGlobal retrieves a Base Station by EUI across all tenants
	// Used during BSSCI connect handshake when tenant is not yet resolved
	GetByEUIGlobal(ctx context.Context, eui []byte) (*models.BaseStation, error)

	// Update updates an existing Base Station
	Update(ctx context.Context, tenantID, id int64, updates map[string]interface{}) error

	// Delete deletes a Base Station
	Delete(ctx context.Context, tenantID, id int64) error

	// List retrieves Base Stations based on filter criteria
	List(ctx context.Context, filter *models.BaseStationFilter) ([]*models.BaseStation, int64, error)

	// UpdateConnectionStatus updates the connection status of a Base Station
	UpdateConnectionStatus(ctx context.Context, tenantID, id int64, isOnline bool, lastError *string) error

	// UpdateSessionInfo updates the session information of a Base Station
	UpdateSessionInfo(ctx context.Context, tenantID int64, eui []byte, sessionUUID string) error

	// GetStatistics retrieves statistics for Base Stations
	GetStatistics(ctx context.Context, tenantID int64) (*BaseStationStatistics, error)

	// UpdateEUI updates the Base Station EUI with transactional cascade to all dependent tables
	UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error)

	// ListAllLocations retrieves all base stations with coordinates across all tenants
	ListAllLocations(ctx context.Context) ([]*models.BaseStation, error)

	// Propagation state management (single row per base station, no tenant scoping)
	GetPropagationState(ctx context.Context, baseStationID int64) (*models.BaseStationPropagationState, error)
	UpsertPropagationState(ctx context.Context, state *models.BaseStationPropagationState) error
	UpdatePropagationStatus(ctx context.Context, baseStationID int64, status string, lastError *string) error
	IncrementRetryCount(ctx context.Context, baseStationID int64, nextRetryAt time.Time) error
}

// BaseStationStatistics represents aggregated statistics for Base Stations
type BaseStationStatistics struct {
	TotalCount   int64 `db:"total_count" json:"total_count"`
	OnlineCount  int64 `db:"online_count" json:"online_count"`
	OfflineCount int64 `db:"offline_count" json:"offline_count"`
	BSSCICount   int64 `db:"bssci_count" json:"bssci_count"`
	MQTTCount    int64 `db:"mqtt_count" json:"mqtt_count"`
}
