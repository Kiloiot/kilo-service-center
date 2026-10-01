package postgres

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// DownlinkQueueLookup reads queue rows by endpoint, queue id and packet counter.
type DownlinkQueueLookup struct {
	db          sqlx.ExtContext
	queueReader *DownlinkQueueReader
	log         logger.Logger
}

// GetDownlinkQueue retrieves pending downlink messages for a device
// Delegates to DownlinkQueueReader for DRY implementation
func (r *DownlinkQueueLookup) GetDownlinkQueue(ctx context.Context, deviceEUI string, tenantID string) ([]*storage.DownlinkMessage, error) {
	// Convert tenant ID
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(errFmtInvalidTenantID, tenantID)
	}

	// Prepare optional endpoint filter
	var epFilter *[8]byte
	if deviceEUI != "" {
		epEuiBytes, err := hex.DecodeString(deviceEUI)
		if err != nil || len(epEuiBytes) != 8 {
			return nil, fmt.Errorf(errFmtInvalidEPEUI, deviceEUI)
		}
		var eui [8]byte
		copy(eui[:], epEuiBytes)
		epFilter = &eui
	}

	return r.ListInFlightDownlinks(ctx, tid, storage.DownlinkQueueFilter{EpEUI: epFilter})
}

// ListInFlightDownlinks lists the tenant's in-flight downlinks the filter
// narrows, bounded like the tenant queue listing.
func (r *DownlinkQueueLookup) ListInFlightDownlinks(ctx context.Context, tenantID int64, filter storage.DownlinkQueueFilter) ([]*storage.DownlinkMessage, error) {
	return r.queueReader.ListTenantQueue(ctx, tenantID, filter, tenantQueueReadLimit, 0)
}

// GetDownlinkByQueueID retrieves the tenant's downlink by its queue ID, with
// the user data of each packet counter it was queued for.
func (r *DownlinkQueueLookup) GetDownlinkByQueueID(ctx context.Context, queId uint64, tenantID string) (*storage.DownlinkMessage, error) {
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(errFmtInvalidTenantID, tenantID)
	}
	scope := newSQLScope()
	scope.equals(colQueID, queId)
	scope.equals(colTenantID, tid)
	return r.downlinkIn(ctx, scope)
}

// GetDownlinkByRevocation retrieves the downlink a revocation names, of the
// organization and endpoint when it names them; sql.ErrNoRows when no row of
// that owner carries the queue id.
func (r *DownlinkQueueLookup) GetDownlinkByRevocation(ctx context.Context, revocation storage.DownlinkRevocation) (*storage.DownlinkMessage, error) {
	scope := newSQLScope()
	scopeRevocation(scope, revocation)
	return r.downlinkIn(ctx, scope)
}

// downlinkIn reads the one downlink row the scope names.
func (r *DownlinkQueueLookup) downlinkIn(ctx context.Context, scope *sqlScope) (*storage.DownlinkMessage, error) {
	var msg storage.DownlinkMessage
	var epEuiBytes, bsEuiBytes, acEuiBytes, userDataJSON []byte
	var msgTenantID int64
	var packetCnts pq.Int64Array
	err := r.db.QueryRowxContext(ctx, `
		SELECT id, ep_eui, bs_eui, que_id, ac_que_id, ac_eui, tenant_id, organization_id, status, dl_rx_stat_qry, created_at,
		       payload, cnt_depend, packet_cnt, user_data
		FROM downlink_queue
		WHERE `+scope.where(), scope.args...).Scan(
		&msg.ID, &epEuiBytes, &bsEuiBytes, &msg.QueID, applicationQueueIDScanner{&msg.ACQueID}, &acEuiBytes, &msgTenantID, &msg.OrganizationID, &msg.Status, &msg.DlRxStatQry, &msg.CreatedAt,
		&msg.Payload, &msg.CntDepend, &packetCnts, &userDataJSON,
	)
	if err != nil {
		return nil, err
	}
	msg.EPEUI = mioty.FormatEUIBytes(epEuiBytes)
	msg.ACEUI = mioty.OptionalEUI64FromBytes(acEuiBytes)
	msg.TenantID = strconv.FormatInt(msgTenantID, 10)
	msg.PacketCntArray = []int64(packetCnts)
	msg.BsEui = mioty.EUI64FromBytes(bsEuiBytes)
	if err := applyDownlinkUserData(&msg, userDataJSON); err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetDownlinksByPacketCnt lists the tenant's in-flight counter-dependent
// downlinks of the endpoint scheduled for the packet counter (SCACI §3.11.1),
// newest first; empty when none is.
func (r *DownlinkQueueLookup) GetDownlinksByPacketCnt(ctx context.Context, tenantID string, epEui string, packetCnt uint32) ([]*storage.DownlinkMessage, error) {
	tid, err := strconv.ParseInt(tenantID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf(errFmtInvalidTenantID, tenantID)
	}
	epEuiBytes, err := hex.DecodeString(epEui)
	if err != nil || len(epEuiBytes) != 8 {
		return nil, fmt.Errorf(errFmtInvalidEndpointEUI, epEui)
	}
	return r.ListPacketCounterDownlinks(ctx, storage.PacketCounterDownlinks{
		TenantID:  tid,
		EpEUI:     mioty.EUI64FromBytes(epEuiBytes),
		PacketCnt: packetCnt,
	})
}

// ListPacketCounterDownlinks lists the in-flight counter-dependent downlinks
// the query names, newest first; empty when none is.
func (r *DownlinkQueueLookup) ListPacketCounterDownlinks(ctx context.Context, query storage.PacketCounterDownlinks) ([]*storage.DownlinkMessage, error) {
	scope := newSQLScope()
	scope.equals(colEpEUI, mioty.EUI64Bytes(query.EpEUI))
	scope.equals(colTenantID, query.TenantID)
	scope.organization(query.OrganizationID)
	scope.and(sqlDownlinkInFlight)
	scope.and(sqlCounterDependent)
	scope.and(fmt.Sprintf(sqlAnyOfFmt, scope.bind(int64(query.PacketCnt)), colPacketCnt))

	rows, err := r.db.QueryxContext(ctx, sqlDownlinksByPacketCounter+scope.where()+sqlNewestFirst, scope.args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryDownlinkByPacketCounter, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgRowsClose, logger.FieldError, err, logger.FieldOperation, opGetDownlinksByPacketCnt)
		}
	}()
	var scheduled []*storage.DownlinkMessage
	for rows.Next() {
		msg, err := scanPacketCounterRow(rows)
		if err != nil {
			return nil, err
		}
		scheduled = append(scheduled, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryDownlinkByPacketCounter, err)
	}
	return scheduled, nil
}

// scanPacketCounterRow reads one row of the packet counter lookup.
func scanPacketCounterRow(rows *sqlx.Rows) (*storage.DownlinkMessage, error) {
	var msg storage.DownlinkMessage
	var epEuiBytes, bsEuiBytes, acEuiBytes []byte
	var msgTenantID int64
	var packetCntArray pq.Int64Array
	if err := rows.Scan(&msg.ID, &epEuiBytes, &bsEuiBytes, &msg.QueID, applicationQueueIDScanner{&msg.ACQueID}, &acEuiBytes, &msgTenantID,
		&msg.OrganizationID, &msg.Status, &msg.CntDepend, &packetCntArray, &msg.CreatedAt); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapQueryDownlinkByPacketCounter, err)
	}
	msg.EPEUI = mioty.FormatEUIBytes(epEuiBytes)
	msg.ACEUI = mioty.OptionalEUI64FromBytes(acEuiBytes)
	msg.TenantID = strconv.FormatInt(msgTenantID, 10)
	msg.BsEui = mioty.EUI64FromBytes(bsEuiBytes)
	msg.PacketCntArray = []int64(packetCntArray)
	return &msg, nil
}

// SQL of the packet counter lookup: the selected columns, the counter
// dependence it requires and its order.
const (
	sqlDownlinksByPacketCounter = `SELECT id, ep_eui, bs_eui, que_id, ac_que_id, ac_eui, tenant_id, organization_id, status, cnt_depend, packet_cnt, created_at
	FROM downlink_queue WHERE `
	sqlCounterDependent = "cnt_depend = true"
	sqlNewestFirst      = " ORDER BY created_at DESC"
)
