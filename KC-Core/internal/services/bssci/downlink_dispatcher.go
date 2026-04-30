package bssciservices

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// downlinkDispatcher implements bssci.DownlinkDispatcher (BSSCI §5.10.2).
//
// Handles automatic downlink dispatch when endpoint signals dlOpen=true.
// Uses transactional reserve→send→mark-queued pattern with FOR UPDATE SKIP LOCKED
// to prevent concurrent dispatchers from reserving the same downlink.
type downlinkDispatcher struct {
	logger  logger.Logger
	storage interfaces.Storage // interfaces.Storage for BeginTx()
	sendFn  SendDLQueueFunc    // Injected function matching Server.SendDLDataQueue signature
}

// SendDLQueueFunc matches signature of Server.SendDLDataQueue
// Enables dependency injection for testability without depending on full Server
type SendDLQueueFunc func(sessionID string, epEUI uint64, payloads [][]byte, queID int64,
	priority float32, cntDepend bool, packetCnt []int64, format uint8,
	responseExp, responsePrio, dlWindReq, expOnly bool, tenantID int64) error

// Ensure interface compliance
var _ bssci.DownlinkDispatcher = (*downlinkDispatcher)(nil)

// NewDownlinkDispatcher creates a new downlink dispatcher service.
//
// Parameters:
//   - log: Logger for dispatch events
//   - storage: interfaces.Storage providing BeginTx() for transactions
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

// DispatchIfAvailable performs transactional reserve→send→mark-queued for pending downlinks.
//
// Lifecycle:
//  1. Begin transaction
//  2. ReserveNextPendingDownlink (SELECT FOR UPDATE SKIP LOCKED + UPDATE status='reserved')
//  3. Build payloads from DownlinkMessage
//  4. Send via SendDLDataQueue (three-way handshake)
//  5. MarkReservedAsQueued (UPDATE status='queued', transmission_time, tx_bs_eui)
//  6. Commit transaction
//
// If step 4 fails, transaction rollback returns status to 'pending' automatically.
// If step 5 fails, transaction rollback returns status to 'reserved' then 'pending'.
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

	// Begin transaction via interfaces.Storage
	tx, err := d.storage.BeginTx(ownerCtx)
	if err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherTxBeginFailed, "error", err)
		return false, nil // Graceful degradation - don't fail uplink
	}
	defer func() {
		// Rollback is no-op if already committed
		_ = tx.Rollback()
	}()

	// 1. Atomic select + reserve with SKIP LOCKED
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

	// 2. Build payloads array (UserData takes precedence, fallback to single Payload)
	payloads := dl.UserData
	if len(payloads) == 0 && len(dl.Payload) > 0 {
		payloads = [][]byte{dl.Payload}
	}

	// 3. Dispatch via SendDLDataQueue (uses existing three-way handshake)
	err = d.sendFn(
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
	)
	if err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherSendFailed,
			"queId", dl.QueID, "epEui", epEUI, "error", err)
		// tx.Rollback() called by defer - reserved->pending automatically
		return false, nil
	}

	// 4. Mark sent with transmission metadata
	// packetCnt = nil (unknown until dlDataQueCmp returns actual value)
	// Pass nil for orgID since BSSCI dispatcher uses tenant-level isolation
	txTime := time.Now().UnixNano()
	if err := tx.MIOTYDownlinks().MarkReservedAsQueued(
		ownerCtx,
		uint64(dl.QueID), //nolint:gosec // G115: QueID is always positive (DB-assigned sequence)
		ownerTenantID,
		session.BaseStationEUI,
		txTime,
		nil, // packetCnt - dlDataQueCmp will update
		nil, // orgID - BSSCI uses tenant-level isolation
	); err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherMarkSentFailed,
			"queId", dl.QueID, "error", err)
		return false, nil
	}

	// 5. Commit transaction
	if err := tx.Commit(); err != nil {
		d.logger.ErrorContext(ownerCtx, bssci.LogDispatcherTxCommitFailed, "error", err)
		return false, nil
	}

	d.logger.InfoContext(ownerCtx, bssci.LogDispatcherSuccess,
		"queId", dl.QueID,
		"epEui", epEUI,
		"bsEui", session.BaseStationEUI,
		"ownerTenantId", ownerTenantID,
		"ownerOrgUUID", ownerOrgUUID)

	return true, nil
}
