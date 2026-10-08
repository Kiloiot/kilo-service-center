package grpc

import (
	"context"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
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

// EndpointStatsStore aggregates the message statistics of an endpoint's
// uplinks received at or after since.
type EndpointStatsStore interface {
	GetMessageStatsByEndpointSince(ctx context.Context, epEui uint64, tenantID int64, since time.Time) (*mioty.MessageStats, error)
}

// OperationStatusAdapter defines the interface for endpoint operation history queries.
type OperationStatusAdapter interface {
	// GetEndpointOperations retrieves recent operations for an endpoint
	// Uses default categories and lookback window from bssci constants
	GetEndpointOperations(ctx context.Context, endpointID, tenantID int64, limit, offset int) ([]models.SystemEvent, error)
}
