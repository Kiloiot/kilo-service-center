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

	// GetAverageDLRXMetrics calculates average SNR and RSSI for an endpoint over a time period
	GetAverageDLRXMetrics(ctx context.Context, tenantID int64, epEui []byte,
		startTime, endTime *time.Time) (avgSnr, avgRssi float64, count int, err error)

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

	// GetDLRXStatusSinceLastHeard returns, per base station, the latest DL RX status reported
	// after the endpoint was last heard: the dlRxSnr/dlRxRssi of its next uplink (SCACI §3.8.1)
	GetDLRXStatusSinceLastHeard(ctx context.Context, tenantID int64, epEui []byte, bsEuis [][]byte) ([]*mioty.DLRXStatus, error)
}
