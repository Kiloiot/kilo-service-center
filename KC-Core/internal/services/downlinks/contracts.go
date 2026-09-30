package downlinks

import (
	"context"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Queuer queues a downlink through the SCACI handler core, the path a
// socket dlDataQue takes (SCACI §3.10).
type Queuer interface {
	QueueDownlinkInternal(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue) (*scaci.DLDataQueueResult, error)
}

// PendingEditor rewrites a downlink no base station holds yet;
// storage.ErrDownlinkNotFound and storage.ErrDownlinkNotPending tell why none was.
type PendingEditor interface {
	UpdatePendingDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, epEUI []byte, queID int64, patch storage.DownlinkPatch) (*storage.DownlinkMessage, error)
}

// Revoker revokes a downlink through the single revoke path SCACI uses.
type Revoker interface {
	RevokeDownlink(ctx context.Context, ref scheduler.DownlinkRef) (bsEui uint64, err error)
}

// EndpointLookup finds a tenant's endpoint; storage.ErrNotFound when it has none.
type EndpointLookup interface {
	GetByEUI(ctx context.Context, eui []byte, tenantID int64) (*models.EndPoint, error)
}

// UpdateRecorder records a pending downlink rewritten in the queue, so every
// open view of the endpoint's queue shows the new content.
type UpdateRecorder interface {
	RecordPendingUpdated(ctx context.Context, downlink *storage.DownlinkMessage) error
}

// AuditRecorder records an operator action.
type AuditRecorder interface {
	Record(ctx context.Context, ev audit.Event)
}

// Owner is the organization of a tenant a downlink is queued for, and the endpoint.
type Owner struct {
	TenantID       int64
	OrganizationID uuid.UUID
	EpEUI          uint64
}

// Target names one of an owner's downlinks by its service center queue id.
type Target struct {
	Owner
	QueID int64
}
