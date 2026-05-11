package interfaces

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/google/uuid"
)

// MIOTYDownlinkRepository provides MIOTY-specific downlink queue operations
// Organization filters support SCACI §3.10 tenant isolation
type MIOTYDownlinkRepository interface {
	// Queue Management (4 read methods)
	GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error)
	GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error)
	GetDownlinkByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) (*storage.DownlinkMessage, error)
	// GetDownlinkResults retrieves downlink results with optional organization filter.
	// orgID filters by organization; nil = no org filter (backward compatible)
	GetDownlinkResults(ctx context.Context, deviceEUI string, tenantID string, orgID *uuid.UUID, statusFilter string, timeFrom, timeTo *time.Time, limit, offset int) ([]*storage.DownlinkMessage, int, error)

	// Queue Mutations (5 write methods)
	EnqueueDownlink(ctx context.Context, downlink *storage.DownlinkMessage) (*storage.DownlinkMessage, error)
	// UpdateDownlinkStatus updates downlink status with optional organization filter.
	// orgID filters by organization; nil = no org filter (backward compatible)
	UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error
	// UpdateDownlinkResult updates downlink result with optional organization filter.
	// orgID filters by organization; nil = no org filter (backward compatible)
	UpdateDownlinkResult(ctx context.Context, queId int64, result string, txTime *int64, packetCnt *uint32, bsEUI []byte, epEUI []byte, tenantID string, orgID *uuid.UUID) error
	UpdateDownlinkBaseStation(ctx context.Context, queId uint64, tenantID string, bsEUI uint64) error
	RevokeDownlink(ctx context.Context, queId int64, tenantID string) error

	// Transactional Dispatch Methods (BSSCI §5.10.2)
	// These methods are ONLY valid within a transaction context via Transaction.MIOTYDownlinks()
	// Non-transactional *DB implementations return ErrNotImplemented

	// ReserveNextPendingDownlink selects+reserves highest-priority pending downlink for dispatch.
	// Uses FOR UPDATE SKIP LOCKED to avoid blocking concurrent dispatchers.
	// Returns nil, nil if no pending downlinks available (not an error).
	// tenantID is int64 (owner tenant, not session tenant for roaming safety).
	// orgID filters by organization; nil = no org filter (backward compatible)
	ReserveNextPendingDownlink(ctx context.Context, tenantID int64, epEUI []byte, bsEUI uint64, orgID *uuid.UUID) (*storage.DownlinkMessage, error)

	// MarkReservedAsQueued transitions reserved → queued with transmission metadata.
	// Sets status='queued', transmission_time, tx_bs_eui. transmission_result stays NULL.
	// packetCnt is nullable - pass nil if unknown (dlDataQueCmp will update later).
	// orgID filters by organization; nil = no org filter (backward compatible)
	MarkReservedAsQueued(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64, txTime int64, packetCnt *uint32, orgID *uuid.UUID) error
}
