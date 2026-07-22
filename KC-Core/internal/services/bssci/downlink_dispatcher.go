package bssciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/google/uuid"
)

// downlinkDispatcher implements bssci.DownlinkDispatcher (BSSCI rev1 §5.12 /
// classic §3.12).
//
// The dispatcher is the single owner of the downlink queue status lifecycle
// pending → reserved → queued for both delivery paths: DispatchIfAvailable
// (auto-dispatch on dlOpen=true) and DispatchQueue (SCACI-initiated immediate
// dispatch). The reservation commits in a short transaction BEFORE any network
// I/O; an uncertain (ambiguous) send leaves the row reserved and is confirmed
// by the idempotent reserved→queued update, repeated from the dlDataQueRsp
// handler for crash recovery. A definite pre-write failure releases the row
// back to pending (documented at-least-once retry semantics).
type downlinkDispatcher struct {
	logger  logger.Logger
	storage interfaces.Storage // BeginTx() for reservation + regular repository for confirmation
	sendFn  SendDLQueueFunc    // Injected function matching Server.SendDLDataQueue signature
}

// SendDLQueueFunc matches signature of Server.SendDLDataQueue
// Enables dependency injection for testability without depending on full Server
type SendDLQueueFunc func(sessionID string, epEUI uint64, payloads [][]byte, queID int64,
	priority float32, cntDepend bool, packetCnt []int64, format uint8,
	responseExp, responsePrio, dlWindReq, expOnly bool, tenantID int64,
	dlRxStatQry bool) error

// Ensure interface compliance
var _ bssci.DownlinkDispatcher = (*downlinkDispatcher)(nil)

// NewDownlinkDispatcher creates a new downlink dispatcher service.
//
// Parameters:
//   - log: Logger for dispatch events
//   - storage: interfaces.Storage providing BeginTx() and the downlink repository
//   - sendFn: Function to send downlink (typically bssciServer.SendDLDataQueue)
func NewDownlinkDispatcher(
	log logger.Logger,
	storage interfaces.Storage,
	sendFn SendDLQueueFunc,
) bssci.DownlinkDispatcher {
	return &downlinkDispatcher{
		logger:  log,
		storage: storage,
		sendFn:  sendFn,
	}
}

// DispatchIfAvailable reserves the highest-priority pending downlink for the
// endpoint and dispatches it through the shared dispatch path.
//
// Lifecycle:
//  1. Short transaction: ReserveNextPendingDownlink (FOR UPDATE SKIP LOCKED +
//     UPDATE status='reserved'), then commit BEFORE any network I/O
//  2. Send via SendDLDataQueue (dlRxStatQry pairing included)
//  3. Idempotent reserved→queued confirmation on the regular repository
func (d *downlinkDispatcher) DispatchIfAvailable(
	ownerCtx context.Context,
	ownerTenantID int64,
	ownerOrgUUID uuid.UUID,
	session *bssci.Session,
	epEUI uint64,
	_ bool, // responseExp - reserved for future SCACI notification integration
	_ bool, // dlAck - reserved for future acknowledgment handling
) (bool, error) {
	// Guard: No tenant means we can't query the queue safely
	if ownerTenantID == 0 {
		d.logger.WarnContext(ownerCtx, bssci.LogDispatcherNoTenant, "epEui", epEUI)
		return false, nil
	}

	// Convert epEUI to bytea for DB query
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	// Short reservation transaction: reserve and commit before any wire write
	tx, err := d.storage.BeginTx(ownerCtx)
	if err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherTxBeginFailed, "error", err)
		return false, nil // Graceful degradation - don't fail uplink
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()

	// Atomic select + reserve with SKIP LOCKED
	// Pass nil for orgID since BSSCI dispatcher uses tenant-level isolation
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(ownerCtx, ownerTenantID, epEUIBytes, session.BaseStationEUI, nil)
	if err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherQueryFailed, "error", err, "epEui", epEUI)
		return false, nil
	}
	if dl == nil {
		d.logger.DebugContext(ownerCtx, bssci.LogDispatcherNoPending, "epEui", epEUI)
		return false, nil // No pending downlinks - normal case
	}

	if err := tx.Commit(); err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherTxCommitFailed, "error", err)
		return false, nil
	}
	rollback = false

	return d.dispatchReserved(ownerCtx, ownerTenantID, ownerOrgUUID, session, epEUI, dl)
}

// DispatchQueue reserves one exact pending queue row (by queue ID, tenant, and
// endpoint) and dispatches it through the shared dispatch path. Used for
// SCACI-initiated immediate delivery (SCACI §3.10.1). Returns dispatched=false
// with a nil error when no matching row is in 'pending' state (already
// dispatched, revoked, or foreign).
func (d *downlinkDispatcher) DispatchQueue(
	ownerCtx context.Context,
	ownerTenantID int64,
	ownerOrgUUID uuid.UUID,
	session *bssci.Session,
	queueID uint64,
	epEUI uint64,
) (bool, error) {
	if ownerTenantID == 0 {
		d.logger.WarnContext(ownerCtx, bssci.LogDispatcherNoTenant, "epEui", epEUI)
		return false, nil
	}

	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	// Single-statement exact reservation (pending → reserved), atomic without
	// an explicit transaction. Org filter stays nil: BSSCI dispatch uses
	// tenant-level isolation, matching DispatchIfAvailable.
	dl, err := d.storage.MIOTYDownlinks().ReservePendingDownlinkByQueueID(ownerCtx, ownerTenantID, nil, queueID, epEUIBytes, session.BaseStationEUI)
	if err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherQueryFailed, "error", err, "epEui", epEUI, "queId", queueID)
		return false, err
	}
	if dl == nil {
		d.logger.WarnContext(ownerCtx, bssci.LogDispatcherNoPending, "epEui", epEUI, "queId", queueID)
		return false, nil
	}

	return d.dispatchReserved(ownerCtx, ownerTenantID, ownerOrgUUID, session, epEUI, dl)
}

// dispatchReserved sends a reserved queue row and confirms it as queued. The
// row is already durably 'reserved' and no transaction is open.
//
// Failure semantics:
//   - Ambiguous wire write (bssci.ErrAmbiguousWrite): the row stays reserved -
//     never back to pending, because a duplicate send with a new operation ID
//     could corrupt the stream. The send layer closes the connection; resume
//     reissues the persisted operations with their original IDs and the
//     dlDataQueRsp handler repairs the reserved→queued status.
//   - Definite pre-write failure: the row is released back to pending
//     (documented at-least-once retry).
//   - Queued-confirmation failure after a successful send: the row stays
//     reserved and the send is reported dispatched; the idempotent
//     confirmation is repeated from the dlDataQueRsp handler.
func (d *downlinkDispatcher) dispatchReserved(
	ownerCtx context.Context,
	ownerTenantID int64,
	ownerOrgUUID uuid.UUID,
	session *bssci.Session,
	epEUI uint64,
	dl *storage.DownlinkMessage,
) (bool, error) {
	// Build payloads array (UserData takes precedence, fallback to single Payload)
	payloads := dl.UserData
	if len(payloads) == 0 && len(dl.Payload) > 0 {
		payloads = [][]byte{dl.Payload}
	}

	// Dispatch via SendDLDataQueue (three-way handshake; pairs the §5.16 /
	// §3.16 dlRxStatQry ahead of the queue frame when the row requests it)
	err := d.sendFn(
		session.ID,
		epEUI,
		payloads,
		dl.QueID,
		dl.Priority,
		dl.CntDepend,
		dl.PacketCntArray,
		dl.Format,
		dl.ResponseExp,
		dl.ResponsePrio,
		dl.DlWindReq,
		dl.ExpOnly,
		ownerTenantID,
		dl.DlRxStatQry,
	)
	if err != nil {
		if errors.Is(err, bssci.ErrAmbiguousWrite) {
			d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherSendFailed,
				"queId", dl.QueID, "epEui", epEUI, "error", err)
			return false, err
		}
		// Definite pre-write failure: release the reservation for retry
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherSendFailed,
			"queId", dl.QueID, "epEui", epEUI, "error", err)
		if relErr := d.storage.MIOTYDownlinks().UpdateDownlinkStatus(ownerCtx,
			strconv.FormatInt(dl.ID, 10), bssci.DLQueueStatusPending, nil); relErr != nil {
			d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherReleaseFailed,
				"queId", dl.QueID, "error", relErr)
		}
		return false, err
	}

	// Confirm reserved→queued with transmission metadata (idempotent single
	// statement on the regular repository - no transaction spans the send).
	// packetCnt = nil (unknown until the dlDataRes transmission result, BSSCI 5.14)
	txTime := time.Now().UnixNano()
	if err := d.storage.MIOTYDownlinks().MarkReservedAsQueued(
		ownerCtx,
		uint64(dl.QueID), //nolint:gosec // G115: QueID is always positive (DB-assigned sequence)
		ownerTenantID,
		session.BaseStationEUI,
		txTime,
		nil, // packetCnt - set when the dlDataRes transmission result arrives
		nil, // orgID - BSSCI uses tenant-level isolation
	); err != nil {
		// The send happened: report dispatched, leave the row reserved. The
		// dlDataQueRsp handler repeats the idempotent confirmation.
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherMarkSentFailed,
			"queId", dl.QueID, "error", err)
		return true, nil
	}

	d.logger.InfoContext(ownerCtx, bssci.LogDispatcherSuccess,
		"queId", dl.QueID,
		"epEui", epEUI,
		"bsEui", session.BaseStationEUI,
		"ownerTenantId", ownerTenantID,
		"ownerOrgUUID", ownerOrgUUID)

	return true, nil
}
