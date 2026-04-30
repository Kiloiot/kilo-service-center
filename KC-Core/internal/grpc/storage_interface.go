package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/models"
	"github.com/kilocenter/KC-DB/storage/postgres"
)

// DLRXStatusQueryStorage defines the minimal storage interface for DL RX status query telemetry.
// Exposes only the methods used by the gRPC service layer, enabling lightweight test fakes.
//
// BSSCI §5.15: query tracking and telemetry exposure.
type DLRXStatusQueryStorage interface {
	// GetDLRXStatusQueryHistory retrieves query tracking records for an endpoint (BSSCI §5.15 telemetry)
	// Returns query records and total count for pagination
	GetDLRXStatusQueryHistory(ctx context.Context, tenantID int64, epEui []byte,
		limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatusQuery, int, error)

	// GetDLRXStatusQueryStats calculates query status distribution for an endpoint (BSSCI §5.15 telemetry)
	// Returns counts of pending, received, and timeout queries
	GetDLRXStatusQueryStats(ctx context.Context, tenantID int64, epEui []byte,
		startTime, endTime *time.Time) (pending, received, timeout int64, err error)
}

// Compile-time verification that postgres.DB satisfies DLRXStatusQueryStorage interface
var _ DLRXStatusQueryStorage = (*postgres.DB)(nil)

// MessageStore defines the minimal storage interface for base station message statistics.
// Exposes only the methods needed for retrieving aggregated statistics per base station.
type MessageStore interface {
	// GetBaseStationMessageStats retrieves aggregated message statistics for a base station
	// Returns counts and averages for various time periods
	GetBaseStationMessageStats(ctx context.Context, tenantID int64, bsEui []byte,
		startTime, endTime *time.Time) (*mioty.BaseStationMessageStats, error)

	// GetBaseStationEndpointCounts retrieves per-endpoint message counts for a base station
	// Returns a map of endpoint EUI (hex string) to message count
	GetBaseStationEndpointCounts(ctx context.Context, tenantID int64, bsEui []byte,
		startTime, endTime *time.Time) (map[string]int64, error)

	// GetBaseStationLastSeen retrieves the last seen timestamp for a base station
	GetBaseStationLastSeen(ctx context.Context, tenantID int64, bsEui []byte) (*time.Time, error)
}

// Compile-time verification that postgres.DB satisfies MessageStore interface
var _ MessageStore = (*postgres.DB)(nil)

// EndpointStatsStore defines the minimal storage interface for endpoint message statistics.
// Note: This interface is satisfied by adapters/wrappers, not directly by postgres.DB
type EndpointStatsStore interface {
	// GetMessageStatsByEndpoint retrieves aggregated message statistics for an endpoint
	// Returns TotalCount, UniqueEndpoints, AvgRSSI, AvgSNR, FirstSeen, LastSeen, ActiveDays
	GetMessageStatsByEndpoint(ctx context.Context, epEui uint64, tenantID int64) (*mioty.MessageStats, error)
}

// OperationStatusAdapter defines the interface for endpoint operation history queries.
type OperationStatusAdapter interface {
	// GetEndpointOperations retrieves recent operations for an endpoint
	// Uses default categories and lookback window from bssci constants
	GetEndpointOperations(ctx context.Context, endpointID, tenantID int64, limit, offset int) ([]models.SystemEvent, error)
}

// DownlinkStore defines the minimal storage interface for downlink operations.
type DownlinkStore interface {
	// EnqueueDownlink persists a downlink message to the database
	EnqueueDownlink(ctx context.Context, msg *storage.DownlinkMessage) (*storage.DownlinkMessage, error)

	// UpdateDownlinkStatus updates the status of a downlink message
	// orgID is passed for audit tracking (can be nil for BSSCI-only paths)
	UpdateDownlinkStatus(ctx context.Context, id string, status string, orgID *uuid.UUID) error
}

// Compile-time verification that postgres.DB satisfies DownlinkStore interface
var _ DownlinkStore = (*postgres.DB)(nil)

// SCACIDownlinkQueuer defines the minimal interface for SCACI downlink queueing.
// Enables the gRPC service to delegate downlink operations to the SCACI handler core,
// ensuring single-source processing for both socket and gRPC paths.
//
// SCACI §3.10: dlDataQue handler integration.
type SCACIDownlinkQueuer interface {
	// QueueDownlinkInternal queues a downlink message via the SCACI handler core.
	// This method shares the same processing path as socket-based dlDataQue operations.
	//
	// Parameters:
	//   - ctx: Request context for timeout/cancellation
	//   - tenantID: Numeric tenant ID for isolation
	//   - orgID: Organization UUID from fail-closed interceptor (mandatory in strict mode)
	//   - req: MIOTY downlink queue request
	//
	// Returns:
	//   - *DLDataQueueResult: Queue ID, BS EUI, op ID, and status on success
	//   - error: SCACI error token wrapped in error if processing fails
	QueueDownlinkInternal(ctx context.Context, tenantID int64, orgID *uuid.UUID, req *mioty.DLDataQueue) (*scaci.DLDataQueueResult, error)
}

// Compile-time verification that scaci.Server satisfies SCACIDownlinkQueuer interface
var _ SCACIDownlinkQueuer = (*scaci.Server)(nil)
