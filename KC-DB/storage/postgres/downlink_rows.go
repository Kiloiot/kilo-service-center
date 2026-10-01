package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// rowScanner is the single-row scan surface shared by *sql.Row and *sqlx.Row.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// sqlExecQuerier is the subset of *sqlx.Tx / *sqlx.DB the shared statement
// helpers run on, inside a transaction or outside one.
type sqlExecQuerier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
}

// downlinkQueueColumns is the shared column list scanned by
// scanDownlinkQueueRow for dispatch reads and RETURNING clauses.
const downlinkQueueColumns = `id, que_id, ep_eui, tenant_id, organization_id, payload, priority, status,
	       cnt_depend, packet_cnt, format, response_exp, response_prio,
	       dl_wind_req, exp_only, dl_rx_stat_qry, user_data, created_at`

// scanDownlinkQueueRow scans one downlink_queue row (downlinkQueueColumns
// order) into a storage.DownlinkMessage. Returns sql.ErrNoRows unwrapped so
// callers can map "no matching row" to their contract.
func scanDownlinkQueueRow(row rowScanner) (*storage.DownlinkMessage, error) {
	var dl storage.DownlinkMessage
	var epEuiBytes []byte
	var rowTenantID int64
	var packetCntArray pq.Int64Array
	var userDataJSON []byte
	var orgID *uuid.UUID

	err := row.Scan(
		&dl.ID, &dl.QueID, &epEuiBytes, &rowTenantID, &orgID, &dl.Payload, &dl.Priority,
		&dl.Status, &dl.CntDepend, &packetCntArray, &dl.Format,
		&dl.ResponseExp, &dl.ResponsePrio, &dl.DlWindReq, &dl.ExpOnly, &dl.DlRxStatQry,
		&userDataJSON, &dl.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert bytea to string for storage struct
	dl.EPEUI = mioty.FormatEUIBytes(epEuiBytes)
	dl.TenantID = fmt.Sprintf("%d", rowTenantID)
	dl.OrganizationID = orgID

	// Convert packet counter array
	if packetCntArray != nil {
		dl.PacketCntArray = []int64(packetCntArray)
	}

	if err := applyDownlinkUserData(&dl, userDataJSON); err != nil {
		return nil, err
	}
	return &dl, nil
}
