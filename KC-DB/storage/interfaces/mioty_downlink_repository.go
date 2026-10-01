package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// MIOTYDownlinkRepository provides MIOTY-specific downlink queue operations
// Organization filters support SCACI §3.10 tenant isolation
type MIOTYDownlinkRepository interface {
	// Queue Management (4 read methods)
	GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error)
	GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error)
	GetDownlinksByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) ([]*storage.DownlinkMessage, error)
	// GetDownlinkResults retrieves downlink results with optional organization filter.
	// orgID filters by organization; nil = no org filter (backward compatible)
	GetDownlinkResults(ctx context.Context, tenantID int64, orgID *uuid.UUID, filter storage.DownlinkResultFilter, limit, offset int) ([]*storage.DownlinkMessage, int, error)

	// UpdatePendingDownlink rewrites a downlink that is still pending; ErrDownlinkNotFound
	// when no row matches the tenant and organization, ErrDownlinkNotPending otherwise.
	UpdatePendingDownlink(ctx context.Context, tenantID int64, orgID *uuid.UUID, epEUI []byte, queID int64, patch storage.DownlinkPatch) (*storage.DownlinkMessage, error)

	// Queue Mutations (5 write methods)
	EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage, lifetime time.Duration) (*storage.DownlinkMessage, error)
	// UpdateDownlinkStatus updates downlink status with optional organization filter.
	// orgID filters by organization; nil = no org filter (backward compatible)
	UpdateDownlinkStatus(ctx context.Context, id string, status mioty.DLQueueStatus, orgID *uuid.UUID) error
	// UpdateDownlinkResult records the result a base station reported for the
	// tenant's downlink that station holds and returns the row for its originators.
	UpdateDownlinkResult(ctx context.Context, tenantID int64, bsEUI uint64, result *mioty.DLDataResult) (*storage.DownlinkMessage, error)
	// UpdateDownlinkBaseStation records the acceptance of a downlink the station still holds.
	UpdateDownlinkBaseStation(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64) error
	RevokeDownlink(ctx context.Context, revocation storage.DownlinkRevocation) (bool, error)

	// MarkReservedAsQueued transitions reserved → queued with transmission metadata.
	// Sets status='queued', transmission_time, tx_bs_eui. transmission_result stays NULL.
	// Idempotent: a row already in 'queued' state succeeds without modification;
	// any other state returns an error. Available on both the regular repository
	// and the transactional wrapper.
	// packetCnt is nullable - pass nil if unknown (set from the dlDataRes transmission result, BSSCI 5.14).
	// orgID scopes the update to the row's exact organization; nil = no org
	// filter (internal repair paths only). A NULL organization row never
	// satisfies an org-scoped update.
	MarkReservedAsQueued(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64, txTime int64, packetCnt *uint32, orgID *uuid.UUID) error
}

// MIOTYDownlinkTxRepository is the downlink reservation inside a transaction:
// the next pending row is reserved under the row lock, and an exact row by
// its queue id, organization and endpoint.
type MIOTYDownlinkTxRepository interface {
	ReserveNextPendingDownlink(ctx context.Context, tenantID int64, epEUI []byte, bsEUI uint64) (*storage.DownlinkMessage, error)
	// ReservePendingDownlinkByQueueID atomically reserves one exact pending
	// queue row for dispatch. The row transitions pending → reserved only when
	// que_id, tenant_id, ep_eui, and organization_id all match the request (a
	// NULL organization row never satisfies the request). organizationID is the
	// organization the downlink was enqueued under. Returns storage.ErrNotFound
	// when no row in 'pending' state matches (already dispatched, revoked, or
	// foreign).
	ReservePendingDownlinkByQueueID(ctx context.Context, tenantID int64, organizationID uuid.UUID, queueID uint64, epEUI []byte, bsEUI uint64) (*storage.DownlinkMessage, error)
}
