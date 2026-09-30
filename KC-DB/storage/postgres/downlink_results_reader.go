package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// DownlinkResultsReader lists the downlinks that reached a final state.
type DownlinkResultsReader struct {
	db  sqlx.ExtContext
	log logger.Logger
}

// downlinkResultColumns follow downlinkListingColumns in the results
// listing: the transmission a base station reported wins over the queue's
// own record of it.
const downlinkResultColumns = `COALESCE(transmission_result, result),
	COALESCE(transmission_time, tx_time), transmission_packet_cnt, endpoint_acked_at`

// GetDownlinkResults lists one page of a tenant's terminal downlink rows,
// newest transmission first, narrowed by the filter.
func (r *DownlinkResultsReader) GetDownlinkResults(ctx context.Context, tenantID int64, orgID *uuid.UUID, filter storage.DownlinkResultFilter, limit, offset int) ([]*storage.DownlinkMessage, int, error) {
	scope, err := downlinkResultScope(tenantID, orgID, filter)
	if err != nil {
		return nil, 0, err
	}
	var totalCount int
	if err := r.db.QueryRowxContext(ctx, "SELECT COUNT(*) FROM downlink_queue WHERE "+scope.where(), scope.args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountDownlinkResults, err)
	}
	query := "SELECT " + downlinkListingColumns + ", " + downlinkResultColumns +
		" FROM downlink_queue WHERE " + scope.where() +
		" ORDER BY transmitted_at DESC NULLS LAST, created_at DESC" + scope.page(limit, offset)
	rows, err := r.db.QueryContext(ctx, query, scope.args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapQueryDownlinkResults, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgRowsClose, logger.FieldError, err, logger.FieldOperation, opGetDownlinkResults)
		}
	}()
	var messages []*storage.DownlinkMessage
	for rows.Next() {
		msg, err := scanDownlinkResultRow(rows)
		if err != nil {
			return nil, 0, err
		}
		messages = append(messages, msg)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapErrorIteratingDownlinkResults, err)
	}
	return messages, totalCount, nil
}

// downlinkResultScope renders the predicate shared by the results listing
// and its count; the status must be terminal because in-flight rows belong
// to the queue view.
func downlinkResultScope(tenantID int64, orgID *uuid.UUID, filter storage.DownlinkResultFilter) (*sqlScope, error) {
	scope := newSQLScope()
	scope.equals(colTenantID, tenantID)
	switch status := mioty.DLQueueStatus(filter.Status); {
	case filter.Status == "":
		scope.anyOf(colStatus, statusArray(mioty.TerminalStatuses()))
	case status.Terminal():
		scope.equals(colStatus, filter.Status)
	default:
		return nil, fmt.Errorf(errFmtInvalidStatusFilter, filter.Status)
	}
	if len(filter.EpEUI) > 0 {
		scope.equals(colEpEUI, filter.EpEUI)
	}
	if len(filter.BsEUI) > 0 {
		scope.equals(colBsEUI, filter.BsEUI)
	}
	if filter.QueID != nil {
		scope.equals(colQueID, *filter.QueID)
	}
	scope.organization(orgID)
	if filter.From != nil {
		scope.atLeast(colTransmittedAt, *filter.From)
	}
	if filter.To != nil {
		scope.atMost(colTransmittedAt, *filter.To)
	}
	if filter.QueuedFrom != nil {
		scope.atLeast(colCreatedAt, *filter.QueuedFrom)
	}
	return scope, nil
}

// scanDownlinkResultRow reads one results row, including the transmission
// columns the queue listing does not carry.
func scanDownlinkResultRow(rows *sql.Rows) (*storage.DownlinkMessage, error) {
	var result sql.NullString
	var txTime, transmissionPacketCnt sql.NullInt64
	var endpointAckedAt sql.NullTime
	msg, err := scanDownlinkListing(rows, &result, &txTime, &transmissionPacketCnt, &endpointAckedAt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapScanDownlinkResult, err)
	}
	msg.Result = result.String
	msg.TxTime = txTime.Int64
	msg.TransmissionPacketCnt = transmissionPacketCnt.Int64
	msg.EndpointAckedAt = nullTimePtr(endpointAckedAt)
	return msg, nil
}
