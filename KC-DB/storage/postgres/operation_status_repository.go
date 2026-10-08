package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/queries"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// OperationStatusRepository measures lookback windows from the clock that stamps events, never NOW().
type OperationStatusRepository struct {
	db    *sqlx.DB
	clock clock.Clock
}

// Operation types accepted by the status queries.
const (
	operationTypeAttach = "attach"
	operationTypeDetach = "detach"
)

// NewOperationStatusRepository creates a new operation status repository.
func NewOperationStatusRepository(db *sqlx.DB, clk clock.Clock) *OperationStatusRepository {
	return &OperationStatusRepository{db: db, clock: clk}
}

// cutoff is the earliest event time inside a lookback window.
func (r *OperationStatusRepository) cutoff(lookback time.Duration) time.Time {
	return r.clock.Now().Add(-lookback)
}

// GetOperationEventsByID retrieves events for a specific operation ID
// Parameters:
//   - operationID: The operation identifier to search for
//   - categories: Event categories to include (e.g., ["bssci", "scaci"])
//   - since: Lookback window (e.g., 24 hours)
//   - operationType: Optional filter - "attach", "detach", or "" for all
//   - limit: Maximum number of results
func (r *OperationStatusRepository) GetOperationEventsByID(
	ctx context.Context,
	operationID string,
	categories []string,
	since time.Duration,
	operationType string,
	limit int,
) ([]models.SystemEvent, error) {
	// Select appropriate event type filter
	var eventFilter string
	switch operationType {
	case operationTypeAttach:
		eventFilter = queries.EventFilterAttachOps
	case operationTypeDetach:
		eventFilter = queries.EventFilterDetachOps
	case "":
		eventFilter = queries.EventFilterAllOps
	default:
		return nil, fmt.Errorf(errFmtInvalidOperationType, errors.ErrInvalidInput, operationType)
	}

	// Build query with event type filter
	query := fmt.Sprintf(queries.SQLOperationEventsByID, eventFilter)

	// Execute query
	var events []models.SystemEvent
	err := r.db.SelectContext(
		ctx,
		&events,
		query,
		pq.Array(categories),
		r.cutoff(since),
		operationID,
		limit,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return []models.SystemEvent{}, nil // Return empty slice, not error
		}
		return nil, fmt.Errorf("%w: %w", errors.ErrDatabase, err)
	}

	return events, nil
}

// GetEndpointOperationsByID retrieves recent operations for an endpoint
// Searches across event data, title, and description fields
// Parameters:
//   - endpointID: The endpoint database ID
//   - tenantID: Tenant scope for security isolation
//   - categories: Event categories to include
//   - since: Lookback window
//   - limit: Maximum number of results
//   - offset: Pagination offset
func (r *OperationStatusRepository) GetEndpointOperationsByID(
	ctx context.Context,
	endpointID int64,
	tenantID int64,
	categories []string,
	since time.Duration,
	limit int,
	offset int,
) ([]models.SystemEvent, error) {
	// Resolve endpoint ID to EUI with tenant scoping
	var euiBytes []byte
	err := r.db.GetContext(ctx, &euiBytes,
		`SELECT ep_eui FROM endpoints WHERE id = $1 AND tenant_id = $2`,
		endpointID, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(errFmtEndpointNotFoundForTenant, errors.ErrNotFound)
		}
		return nil, fmt.Errorf("%w: %w", errors.ErrDatabase, err)
	}

	eui := mioty.OptionalEUI64FromBytes(euiBytes)
	if eui == nil {
		return nil, fmt.Errorf("%s: %w", errWrapPostgresOperationStatusGetEndpointOperationsInvalidEUI, errors.ErrNotFound)
	}
	euiHex := mioty.FormatEUI64(*eui)

	// Search events by hex EUI (matches original handler behavior)
	var events []models.SystemEvent
	err = r.db.SelectContext(
		ctx,
		&events,
		queries.SQLEndpointOperationEvents,
		pq.Array(categories),
		r.cutoff(since),
		euiHex,
		limit,
		offset,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return []models.SystemEvent{}, nil // Return empty slice, not error
		}
		return nil, fmt.Errorf("%w: %w", errors.ErrDatabase, err)
	}

	return events, nil
}

// BaseStationStatus represents BSSCI connection status with event summary and session metadata
// BSSCI §5-5.3: Session fields support offline station troubleshooting via most recent session LEFT JOIN
type BaseStationStatus struct {
	BsEUI           []byte          `db:"bs_eui"`
	Name            string          `db:"name"`
	IsOnline        bool            `db:"is_online"`
	LastSeenAt      *time.Time      `db:"last_seen_at"`
	Bidi            sql.NullBool    `db:"bidi"`
	Latitude        sql.NullFloat64 `db:"latitude"`
	Longitude       sql.NullFloat64 `db:"longitude"`
	Altitude        sql.NullFloat64 `db:"altitude"`
	RecentEvents    int             `db:"recent_events"`
	SnScUUID        sql.NullString  `db:"sn_sc_uuid"`       // Session SC UUID (NULL if no recent session)
	SnBsUUID        sql.NullString  `db:"sn_bs_uuid"`       // Session BS UUID (NULL if no recent session)
	SnBsOpId        sql.NullInt64   `db:"sn_bs_op_id"`      // Last BS operation ID (NULL if no recent session)
	SnScOpId        sql.NullInt64   `db:"sn_sc_op_id"`      // Last SC operation ID (NULL if no recent session)
	CanResume       sql.NullBool    `db:"can_resume"`       // Session resumable flag (NULL if no recent session)
	Encoding        sql.NullString  `db:"encoding"`         // Message encoding (NULL if no recent session)
	ProtocolVersion sql.NullString  `db:"protocol_version"` // BSSCI protocol version (NULL if no recent session)
	ConnectInfo     sql.NullString  `db:"connect_info"`     // Connect info JSON blob (NULL if no recent session)
	LastPingAt      sql.NullTime    `db:"last_ping_at"`     // Last ping timestamp (NULL if no ping received, BSSCI §5.4)
}

// GetBSSCIStatusSummary retrieves base station status with recent event counts
// Parameters:
//   - tenantID: Tenant ID for filtering
//   - categories: Event categories to include
//   - eventWindow: Event count lookback (e.g., 24 hours)
func (r *OperationStatusRepository) GetBSSCIStatusSummary(
	ctx context.Context,
	tenantID int64,
	categories []string,
	eventWindow time.Duration,
) ([]BaseStationStatus, error) {
	var statuses []BaseStationStatus
	err := r.db.SelectContext(
		ctx,
		&statuses,
		queries.SQLBSSCIStatusSummary,
		tenantID,
		pq.Array(categories),
		r.cutoff(eventWindow),
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return []BaseStationStatus{}, nil // Return empty slice, not error
		}
		return nil, fmt.Errorf("%w: %w", errors.ErrDatabase, err)
	}

	return statuses, nil
}

// GetEventSummary retrieves event type counts within time window
// Parameters:
//   - categories: Event categories to include
//   - since: Lookback window (e.g., 5 minutes)
func (r *OperationStatusRepository) GetEventSummary(
	ctx context.Context,
	categories []string,
	since time.Duration,
) (summary map[string]int, err error) {
	rows, err := r.db.QueryContext(ctx, queries.SQLEventSummary,
		pq.Array(categories), r.cutoff(since))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrDatabase, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapIterateEventSummary, &err)

	summary = make(map[string]int)
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("%w: %w", errors.ErrDatabase, err)
		}
		summary[eventType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrDatabase, err)
	}

	return summary, nil
}
