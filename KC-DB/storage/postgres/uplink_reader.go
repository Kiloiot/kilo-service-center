package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// ulDataColumns are the columns of a stored ulData row, in scanULDataRow order.
const ulDataColumns = `id, tenant_id, org_uuid, command_type, op_id, ep_eui, bs_eui,
	rx_time, packet_cnt, snr, rssi, user_data,
	dl_open, response_exp, dl_ack,
	rx_duration, eq_snr, profile, mode, format, subpackets,
	base_stations, duplicate, packet_cnt_reused,
	received_at, processed_at,
	decoded_payload, blueprint_type_eui, blueprint_version_id, decode_status, decode_error_code,
	created_at`

// Predicates of the ulData listing.
const (
	// sqlReceivedByStationFmt matches an uplink the station received as the
	// primary receiver or as one of the SCACI §3.8.1 receptions.
	sqlReceivedByStationFmt = "(bs_eui = %s OR base_stations @> %s::jsonb)"
	// sqlUplinkSearchFmt matches the search pattern in the hex of the
	// endpoint, the base station or the payload, ignoring case.
	sqlUplinkSearchFmt = "(encode(ep_eui, 'hex') ILIKE %[1]s OR encode(bs_eui, 'hex') ILIKE %[1]s OR encode(user_data, 'hex') ILIKE %[1]s)"
	// sqlContainsPatternFmt is the LIKE pattern matching the term anywhere.
	sqlContainsPatternFmt = "%%%s%%"
	// sqlULDataOrder lists newest first; uplinks sharing a reception time
	// follow their id, so every read pages them in the same order.
	sqlULDataOrder = " ORDER BY rx_time DESC, id DESC"
	// sqlULDataStoredOrder lists the newest stored first, ties by id.
	sqlULDataStoredOrder = " ORDER BY created_at DESC, id DESC"
)

// ulDataDocuments are the JSONB columns of a ulData row, decoded apart from
// the scan so a listing can skip a malformed document without failing.
type ulDataDocuments struct {
	subpackets, baseStations []byte
}

// apply decodes the documents into the message; each document that does not
// decode is left unset and reported.
func (d ulDataDocuments) apply(msg *mioty.ULDataMessage) (subpacketsErr, baseStationsErr error) {
	if d.subpackets != nil {
		var subpackets mioty.Subpackets
		if subpacketsErr = json.Unmarshal(d.subpackets, &subpackets); subpacketsErr == nil {
			msg.Subpackets = &subpackets
		}
	}
	if d.baseStations != nil {
		var receptions []mioty.BaseStationReception
		if baseStationsErr = json.Unmarshal(d.baseStations, &receptions); baseStationsErr == nil {
			msg.BaseStations = receptions
		}
	}
	return subpacketsErr, baseStationsErr
}

// scanULDataRow reads one row of ulDataColumns.
func scanULDataRow(row rowScanner) (*mioty.ULDataMessage, ulDataDocuments, error) {
	var msg mioty.ULDataMessage
	var docs ulDataDocuments
	var epEUI, bsEUI, decodedPayload []byte
	var duplicate bool
	var decodeStatus, decodeErrorCode sql.NullString
	err := row.Scan(
		&msg.ID, &msg.TenantID, &msg.OrgUUID, &msg.CommandType, &msg.OpId, &epEUI, &bsEUI,
		&msg.RxTime, &msg.PacketCnt, &msg.SNR, &msg.RSSI, &msg.UserData,
		&msg.DlOpen, &msg.ResponseExp, &msg.DlAck,
		&msg.RxDuration, &msg.EqSnr, &msg.Profile, &msg.Mode, &msg.Format, &docs.subpackets,
		&docs.baseStations, &duplicate, &msg.PacketCntReused,
		&msg.ReceivedAt, &msg.ProcessedAt,
		&decodedPayload, &msg.BlueprintTypeEUI, &msg.BlueprintVersionID, &decodeStatus, &decodeErrorCode,
		&msg.StoredAt,
	)
	if err != nil {
		return nil, docs, err
	}
	msg.EpEui = mioty.EUI64FromBytes(epEUI)
	msg.BsEui = mioty.EUI64FromBytes(bsEUI)
	msg.DecodedPayload = decodedPayload
	msg.DecodeStatus = decodeStatus.String
	msg.DecodeErrorCode = decodeErrorCode.String
	msg.Duplicate = &duplicate
	return &msg, docs, nil
}

// GetULDataMessage retrieves a single UL Data message by ID
func (r *MessageRepository) GetULDataMessage(ctx context.Context, id string, tenantID int64) (*mioty.ULDataMessage, error) {
	msg, docs, err := scanULDataRow(r.db.QueryRowContext(ctx,
		`SELECT `+ulDataColumns+` FROM messages WHERE id = $1 AND tenant_id = $2`, id, tenantID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %w", errTextMessageNotFound, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetULDataMessage, err)
	}
	subpacketsErr, baseStationsErr := docs.apply(msg)
	if subpacketsErr != nil {
		return nil, fmt.Errorf("%s: %w", errWrapDeserializeSubpackets, subpacketsErr)
	}
	if baseStationsErr != nil {
		return nil, fmt.Errorf("%s: %w", errWrapDeserializeBaseStations, baseStationsErr)
	}
	return msg, nil
}

// ListULDataMessages lists one page of stored ulData, newest first; the
// propagate messages sharing the table are not uplinks and never listed.
func (r *MessageRepository) ListULDataMessages(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, int64, error) {
	scope := ulDataWhere(filter)
	var total int64
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM messages WHERE "+scope.where(), scope.args...); err != nil {
		return nil, 0, fmt.Errorf("%s: %w", errWrapCountMessages, err)
	}
	messages, err := r.queryULData(ctx, scope, sqlULDataOrder, filter)
	if err != nil {
		return nil, 0, err
	}
	return messages, total, nil
}

// ListStoredULData lists one page of stored ulData in the order the database
// stored it, newest first, without a total; the live stream reads it.
func (r *MessageRepository) ListStoredULData(ctx context.Context, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, error) {
	return r.queryULData(ctx, ulDataWhere(filter), sqlULDataStoredOrder, filter)
}

// queryULData reads one page of the scope in the order given.
func (r *MessageRepository) queryULData(ctx context.Context, scope *sqlScope, order string, filter mioty.ULDataMessageFilter) ([]*mioty.ULDataMessage, error) {
	query := "SELECT " + ulDataColumns + " FROM messages WHERE " + scope.where() + order + scope.page(filter.Limit, filter.Offset)
	rows, err := r.db.QueryContext(ctx, query, scope.args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListMessages, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsMessages, logger.FieldError, err)
		}
	}()
	messages := make([]*mioty.ULDataMessage, 0)
	for rows.Next() {
		msg, err := r.scanListedULData(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateMessages, err)
	}
	return messages, nil
}

// scanListedULData reads one listed row; a document that does not decode is
// logged and left out rather than failing the listing.
func (r *MessageRepository) scanListedULData(rows *sql.Rows) (*mioty.ULDataMessage, error) {
	msg, docs, err := scanULDataRow(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapScanMessage, err)
	}
	subpacketsErr, baseStationsErr := docs.apply(msg)
	if baseStationsErr != nil {
		r.log.Warn(logMsgDeserializeBaseStations, logger.FieldError, baseStationsErr)
	}
	if subpacketsErr != nil {
		r.log.Warn(logMsgDeserializeSubpackets, logger.FieldError, subpacketsErr)
	}
	return msg, nil
}

// ulDataWhere renders the predicate shared by the ulData listing and its
// count. A base station filter matches the primary receiver and every
// station in the SCACI §3.8.1 baseStations receptions.
func ulDataWhere(filter mioty.ULDataMessageFilter) *sqlScope {
	scope := newSQLScope()
	scope.equals(colTenantID, filter.TenantID)
	scope.equals(colCommandType, mioty.CmdULData)
	if filter.EpEui != nil {
		scope.equals(colEpEUI, mioty.EUI64Bytes(*filter.EpEui))
	}
	if filter.BsEui != nil {
		scope.and(fmt.Sprintf(sqlReceivedByStationFmt, scope.bind(mioty.EUI64Bytes(*filter.BsEui)),
			scope.bind(fmt.Sprintf(bsContainmentFmt, *filter.BsEui))))
	}
	if filter.StartTime != nil {
		scope.atLeast(colRxTime, filter.StartTime.UnixNano())
	}
	if filter.EndTime != nil {
		scope.atMost(colRxTime, filter.EndTime.UnixNano())
	}
	if filter.StoredSince != nil {
		scope.atLeast(colCreatedAt, *filter.StoredSince)
	}
	if filter.Duplicate != nil {
		scope.equals(colDuplicate, *filter.Duplicate)
	}
	if filter.DlOpen != nil {
		scope.equals(colDlOpen, *filter.DlOpen)
	}
	if filter.Profile != nil && *filter.Profile != "" {
		scope.equals(colProfile, *filter.Profile)
	}
	if filter.Mode != nil && *filter.Mode != "" {
		scope.equals(colMode, *filter.Mode)
	}
	if filter.SearchTerm != nil && *filter.SearchTerm != "" {
		scope.and(fmt.Sprintf(sqlUplinkSearchFmt, scope.bind(fmt.Sprintf(sqlContainsPatternFmt, *filter.SearchTerm))))
	}
	return scope
}
