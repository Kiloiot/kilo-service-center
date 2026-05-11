package interfaces

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// DownlinkQueueReader defines the interface for read-only downlink queue operations
// Used by gRPC handlers to expose queue status without write access
type DownlinkQueueReader interface {
	// ListTenantQueue retrieves pending downlink messages for a tenant
	// epFilter optionally filters by endpoint EUI
	// Returns messages ordered by priority DESC, created_at ASC
	ListTenantQueue(ctx context.Context, tenantID int64, epFilter *[8]byte, limit, offset int) ([]*storage.DownlinkMessage, error)

	// CountTenantQueue returns the count of pending downlink messages for a tenant
	// epFilter optionally filters by endpoint EUI (must match ListTenantQueue filter)
	CountTenantQueue(ctx context.Context, tenantID int64, epFilter *[8]byte) (int64, error)
}

// DownlinkQueueStore provides tenant ownership lookup for downlink queues
// Used by BSSCI TenantResolver and DownlinkService for queue-to-tenant mapping
// during result processing (§5.13, §5.14)
type DownlinkQueueStore interface {
	// GetTenantIDByQueueID resolves the tenant that owns a specific downlink queue
	// Returns sql.ErrNoRows if queue does not exist
	GetTenantIDByQueueID(ctx context.Context, queueID uint64) (tenantID int64, err error)
}
