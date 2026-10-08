package postgres

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// downlinkListingColumns are the columns the queue and results listings both
// show; each listing selects its own columns after them.
const downlinkListingColumns = `id, ep_eui, tenant_id, organization_id, payload,
	priority, status, attempts, max_attempts,
	que_id, cnt_depend, packet_cnt, format,
	response_exp, response_prio, dl_wind_req, exp_only, dl_rx_stat_qry,
	bs_eui, created_at, transmitted_at, acknowledged_at, user_data`

// downlinkListingRow receives one listing row: downlinkListingColumns into
// the message and the listing's own columns into extra.
type downlinkListingRow struct {
	msg                   storage.DownlinkMessage
	tenantID              int64
	orgID                 *uuid.UUID
	epEUI, bsEUI          []byte
	packetCnt             pq.Int64Array
	transmittedAt, accept sql.NullTime
	userData              []byte
}

// scanDownlinkListing reads downlinkListingColumns followed by the listing's
// own columns into extra.
func scanDownlinkListing(row rowScanner, extra ...interface{}) (*storage.DownlinkMessage, error) {
	var r downlinkListingRow
	dest := append([]interface{}{
		&r.msg.ID, &r.epEUI, &r.tenantID, &r.orgID, &r.msg.Payload,
		&r.msg.Priority, &r.msg.Status, &r.msg.Attempts, &r.msg.MaxAttempts,
		&r.msg.QueID, &r.msg.CntDepend, &r.packetCnt, &r.msg.Format,
		&r.msg.ResponseExp, &r.msg.ResponsePrio, &r.msg.DlWindReq, &r.msg.ExpOnly, &r.msg.DlRxStatQry,
		&r.bsEUI, &r.msg.CreatedAt, &r.transmittedAt, &r.accept, &r.userData,
	}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	msg := r.msg
	msg.EPEUI = mioty.FormatEUIBytes(r.epEUI)
	msg.TenantID = strconv.FormatInt(r.tenantID, 10)
	msg.OrganizationID = r.orgID
	msg.BsEui = mioty.EUI64FromBytes(r.bsEUI)
	if r.packetCnt != nil {
		msg.PacketCntArray = []int64(r.packetCnt)
	}
	msg.SentAt = nullTimePtr(r.transmittedAt)
	msg.AcceptedAt = nullTimePtr(r.accept)
	if err := applyDownlinkUserData(&msg, r.userData); err != nil {
		return nil, err
	}
	return &msg, nil
}

// nullTimePtr is the time a nullable column holds, nil when it holds none.
func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
