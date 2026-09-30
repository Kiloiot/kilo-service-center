package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// EndpointReader reads endpoint records by identity and by tenant.
type EndpointReader interface {
	Get(ctx context.Context, eui models.EUI) (*models.EndPoint, error)
	GetByEUI(ctx context.Context, tenantID int64, eui []byte) (*models.EndPoint, error)
	GetByTenant(ctx context.Context, tenantID int64) ([]*models.EndPoint, error)
	CountByTenant(ctx context.Context, tenantID int64) (int64, error)
	ListByTenantPaginated(ctx context.Context, tenantID int64, limit, offset int) ([]*models.EndPoint, error)
	GetByID(ctx context.Context, id int64, tenantID int64) (*models.EndPoint, error)
	// GetByAttachmentChangedSince returns the tenant's endpoints whose status was changed to,
	// or restated as, status after since, or every recorded one when since is nil.
	GetByAttachmentChangedSince(ctx context.Context, tenantID int64, status string, since *time.Time) ([]*models.EndPoint, error)
}

// EndpointWriter creates, updates and deletes endpoint records.
type EndpointWriter interface {
	Create(ctx context.Context, endpoint *models.EndPoint) error
	// CreateWithStatus creates the endpoint and records its attachment status in one transaction.
	CreateWithStatus(ctx context.Context, endpoint *models.EndPoint, status string) error
	Update(ctx context.Context, endpoint *models.EndPoint) error
	UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (*models.EndPoint, error)
	CheckEUIUnique(ctx context.Context, eui []byte) error
	DeleteByTenant(ctx context.Context, tenantID int64, eui []byte) (int64, error)
}

// EndpointLifecycleRepository records the protocol-driven state an endpoint accumulates over its life:
// registration, attach/detach sessions, radio metrics and last-seen bookkeeping.
type EndpointLifecycleRepository interface {
	UpdateLastSeen(ctx context.Context, tenantID int64, eui models.EUI, frameCount uint32) error
	UpdateRadioMetricsSelective(ctx context.Context, tenantID int64, eui models.EUI, update models.RadioMetricsUpdate) error
	EndpointRegistrationUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointRegistrationParams) error
	EndpointAttachmentStateUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointAttachmentStateParams) error
	EndpointAttachSessionUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointAttachSessionParams) error
	EndpointDetachStateUpdate(ctx context.Context, tenantID int64, endpointID int64, p models.EndpointDetachStateParams) error
	TransitionEndpointStatus(ctx context.Context, tenantID int64, endpointID int64, status string) (bool, error)
	// RestateEndpointStatus sets ep_status and records the time even when the status stays.
	RestateEndpointStatus(ctx context.Context, tenantID int64, endpointID int64, status string) (bool, error)
	GetPreferredBsEui(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error)
	// RestartPacketCounter zeroes the counters an over-the-air attach restarts (radio protocol §3.6.5.3).
	RestartPacketCounter(ctx context.Context, tenantID int64, endpointID int64) error
	// LockAttachCounter reads the attach counter and holds the endpoint row until the
	// transaction ends, so attaches of one endpoint run one at a time.
	LockAttachCounter(ctx context.Context, tenantID int64, endpointID int64) (*uint32, error)
}

// EndpointRepository is the full endpoint persistence surface; consumers depend on
// the narrower interfaces above and only the storage layer implements the union.
type EndpointRepository interface {
	EndpointReader
	EndpointWriter
	EndpointLifecycleRepository
}
