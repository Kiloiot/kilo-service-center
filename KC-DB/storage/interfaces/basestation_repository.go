package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
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

	// List retrieves Base Stations based on filter criteria
	List(ctx context.Context, filter *models.BaseStationFilter) ([]*models.BaseStation, int64, error)

	// UpdateConnectionStatus updates the connection status of a Base Station
	UpdateConnectionStatus(ctx context.Context, tenantID, id int64, isOnline bool, lastError *string) error

	// UpdateTLSFingerprintIfBlank persists the certificate fingerprint only
	// while the stored tls_cert_fingerprint is still NULL or empty. Returns
	// whether a row was updated; false signals a concurrent writer already
	// set it (callers reload and compare).
	UpdateTLSFingerprintIfBlank(ctx context.Context, tenantID, id int64, fingerprint string) (bool, error)

	// UpdateTLSCertExpiryIfBlank persists the station certificate's expiry
	// only while the stored tls_cert_expires_at is still NULL; an existing
	// expiry is never overwritten. Returns whether a row was updated.
	UpdateTLSCertExpiryIfBlank(ctx context.Context, tenantID, id int64, expiresAt time.Time) (bool, error)

	// GetStatistics retrieves statistics for Base Stations
	GetStatistics(ctx context.Context, tenantID int64) (*models.BaseStationStatistics, error)

	// UpdateEUI updates the Base Station EUI with transactional cascade to all dependent tables
	UpdateEUI(ctx context.Context, tenantID int64, oldEui, newEui []byte) (*models.BaseStation, error)

	// ListAllLocations retrieves all base stations with coordinates across all tenants
	ListAllLocations(ctx context.Context) ([]*models.BaseStation, error)
}
