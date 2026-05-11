package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// DLRXStatusRepository manages downlink reception status from endpoints (BSSCI §5.15)
// This interface mirrors the concrete implementation in storage/postgres/dl_rx_status_repository.go
type DLRXStatusRepository interface {
	// CreateDLRXStatus persists a new DL RX status report
	CreateDLRXStatus(ctx context.Context, status *mioty.DLRXStatus) error

	// GetDLRXStatusByEndpoint retrieves DL RX status records for a specific endpoint
	// Returns status records and total count for pagination
	GetDLRXStatusByEndpoint(ctx context.Context, tenantID int64, epEui []byte,
		limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatus, int, error)

	// GetLatestDLRXStatus retrieves the most recent DL RX status for an endpoint
	GetLatestDLRXStatus(ctx context.Context, tenantID int64, epEui []byte) (*mioty.DLRXStatus, error)

	// GetLatestDLRXStatusPerEndpoint retrieves the latest status for all endpoints (aggregate)
	GetLatestDLRXStatusPerEndpoint(ctx context.Context, tenantID int64) ([]*mioty.DLRXStatus, error)

	// GetDLRXStatusByTimeRange retrieves DL RX status records within a time window
	GetDLRXStatusByTimeRange(ctx context.Context, tenantID int64,
		startTime, endTime time.Time) ([]*mioty.DLRXStatus, error)

	// GetAverageDLRXMetrics calculates average SNR and RSSI for an endpoint over a time period
	GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte,
		startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error)

	// DeleteOldDLRXStatus removes DL RX status records older than the retention period
	DeleteOldDLRXStatus(ctx context.Context, tenantID int64, retentionDays int) (int64, error)

	// CreateDLRXStatusQuery tracks a dlRxStatQry request for correlation (BSSCI §5.15 audit trail)
	// orgUUID is nullable for backward compatibility (nil if not available from context)
	CreateDLRXStatusQuery(ctx context.Context, tenantID int64, orgUUID *uuid.UUID, epEui, bsEui []byte, opId int64) error

	// MarkDLRXStatusReceived marks the OLDEST pending query for tenant+endpoint as received
	// The bsEui and bsOpID are stored for audit only - correlation uses tenant+endpoint, NOT opId
	// Returns true if a pending query was found and updated, false otherwise
	MarkDLRXStatusReceived(ctx context.Context, tenantID int64, epEui []byte, bsEui []byte, bsOpID int64) (found bool, err error)

	// ExpireDLRXStatusQuery marks queries older than cutoff as 'timeout'
	// Returns count of expired queries (used by cleanup job)
	ExpireDLRXStatusQuery(ctx context.Context, cutoff time.Time) (int64, error)

	// GetDLRXStatusQueryHistory retrieves query tracking records for an endpoint (BSSCI §5.15 telemetry)
	// Returns query records and total count for pagination
	GetDLRXStatusQueryHistory(ctx context.Context, tenantID int64, epEui []byte,
		limit, offset int, startTime, endTime *time.Time) ([]*mioty.DLRXStatusQuery, int, error)

	// GetDLRXStatusQueryStats calculates query status distribution for an endpoint (BSSCI §5.15 telemetry)
	// Returns counts of pending, received, and timeout queries
	GetDLRXStatusQueryStats(ctx context.Context, tenantID int64, epEui []byte,
		startTime, endTime *time.Time) (pending, received, timeout int64, err error)

	// GetLatestDLRXStatusByBaseStations returns latest DL RX status for each bs_eui in a single query (SCACI §3.8.1)
	// Used for batch hydration of dlRxSnr/dlRxRssi in multi-BS UL data without N+1 queries
	// Uses DISTINCT ON (Postgres-specific) to get one row per base station
	// Signature uses []byte for consistency with other DL RX methods (caller does uint64→bytes conversion)
	GetLatestDLRXStatusByBaseStations(ctx context.Context, tenantID int64, epEui []byte, bsEuis [][]byte) ([]*mioty.DLRXStatus, error)
}
