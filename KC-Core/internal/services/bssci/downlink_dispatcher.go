package bssciservices

import (
	"context"
	"encoding/binary"
	"errors"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
)

// DownlinkReserver reserves a pending queue row for dispatch. Reservation is
// transactional (FOR UPDATE SKIP LOCKED) and commits before the caller performs
// any network I/O; the transaction itself stays inside the storage adapter.
type DownlinkReserver interface {
	ReserveNextPending(ctx context.Context, tenantID int64, epEUI []byte,
		bsEUI uint64) (*storage.DownlinkMessage, error)
	ReserveByQueueID(ctx context.Context, tenantID int64, orgID uuid.UUID, queueID uint64,
		epEUI []byte, bsEUI uint64) (*storage.DownlinkMessage, error)
}

// DownlinkConfirmer records the outcome of a dispatch attempt on a row that is
// already durably reserved.
type DownlinkConfirmer interface {
	UpdateDownlinkStatus(ctx context.Context, id string, status mioty.DLQueueStatus, orgID *uuid.UUID) error
	MarkReservedAsQueued(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64,
		txTime int64, packetCnt *uint32, orgID *uuid.UUID) error
}

// DownlinkWindowClaimer hands a telegram's downlink window to one dispatch
// (radio §3.6.1: one downlink per window), whichever station reported it.
type DownlinkWindowClaimer interface {
	ClaimDownlinkWindow(ctx context.Context, ownerTenantID int64, messageID string) (bool, error)
	ReleaseDownlinkWindow(ctx context.Context, ownerTenantID int64, messageID string) error
}

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
//
// The delivery organization is always the reserved row's organization_id: the
// row records which organization enqueued the downlink, and enqueue rejects a
// row without one, so dispatch never substitutes a resolver-derived or
// session-derived organization.
type downlinkDispatcher struct {
	logger   logger.Logger
	reserver DownlinkReserver
	queue    DownlinkConfirmer
	windows  DownlinkWindowClaimer
	sendFn   SendDLQueueFunc // Injected function matching Server.SendDLDataQueue signature
	clock    clock.Clock
}

// SendDLQueueFunc matches signature of Server.SendDLDataQueue
// Enables dependency injection for testability without depending on full Server
type SendDLQueueFunc func(sessionID string, epEUI uint64, payloads [][]byte, queID int64,
	priority float32, cntDepend bool, packetCnt []int64, format uint8,
	responseExp, responsePrio, dlWindReq, expOnly bool, tenantID int64,
	orgID *uuid.UUID, dlRxStatQry bool) error

// Ensure interface compliance
var _ bssci.DownlinkDispatcher = (*downlinkDispatcher)(nil)

// NewDownlinkDispatcher creates a new downlink dispatcher service; every
// collaborator is mandatory, so a wiring fault surfaces at startup instead of
// on the first downlink window.
//
// Parameters:
//   - log: Logger for dispatch events
//   - reserver: Reserves a queue row for dispatch, owning its transaction
//   - queue: Confirms or releases an already reserved row
//   - windows: Hands each telegram's downlink window to one dispatch
//   - sendFn: Function to send downlink (typically bssciServer.SendDLDataQueue)
//   - clk: Clock stamping the queued transmission time
func NewDownlinkDispatcher(
	log logger.Logger,
	reserver DownlinkReserver,
	queue DownlinkConfirmer,
	windows DownlinkWindowClaimer,
	sendFn SendDLQueueFunc,
	clk clock.Clock,
) (bssci.DownlinkDispatcher, error) {
	switch {
	case log == nil:
		return nil, ErrNilDispatcherLogger
	case reserver == nil:
		return nil, ErrNilDownlinkReserver
	case queue == nil:
		return nil, ErrNilDownlinkConfirmer
	case windows == nil:
		return nil, ErrNilDownlinkWindowClaimer
	case sendFn == nil:
		return nil, ErrNilDownlinkSendFunc
	case clk == nil:
		return nil, ErrNilDispatcherClock
	}
	return &downlinkDispatcher{
		logger:   log,
		reserver: reserver,
		queue:    queue,
		windows:  windows,
		sendFn:   sendFn,
		clock:    clk,
	}, nil
}

// DispatchIfAvailable fills the downlink window of the telegram with the
// highest-priority pending downlink of the endpoint. Every station that
// received the telegram reports the window; the first bidirectional one
// claims it, and a dispatch that sends nothing gives the claim back, so at
// most one downlink goes out per window (radio §3.6.1). The reserved row's
// organization_id is authoritative for the delivery.
//
// Lifecycle:
//  1. Claim the telegram's window
//  2. Short transaction: ReserveNextPendingDownlink (FOR UPDATE SKIP LOCKED +
//     UPDATE status='reserved'), then commit BEFORE any network I/O
//  3. Send via SendDLDataQueue (dlRxStatQry pairing included)
//  4. Idempotent reserved→queued confirmation on the regular repository
func (d *downlinkDispatcher) DispatchIfAvailable(
	ctx context.Context,
	ownerTenantID int64,
	session *bssci.Session,
	epEUI uint64,
	messageID string,
	_ bool, // responseExp - reserved for future SCACI notification integration
) (bool, error) {
	// Guard: No tenant means we can't query the queue safely
	if ownerTenantID == 0 {
		d.logger.WarnContext(ctx, bssci.LogDispatcherNoTenant, logger.FieldEpEui, epEUI)
		return false, nil
	}
	if !session.Bidirectional {
		d.logger.DebugContext(ctx, bssci.LogDispatcherStationNotBidirectional,
			logger.FieldEpEui, epEUI, logger.FieldBsEui, session.BaseStationEUI)
		return false, nil
	}
	claimed, err := d.windows.ClaimDownlinkWindow(ctx, ownerTenantID, messageID)
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogDispatcherWindowClaimFailed, logger.FieldEpEui, epEUI, logger.FieldError, err)
		return false, nil
	}
	if !claimed {
		d.logger.DebugContext(ctx, bssci.LogDispatcherWindowAlreadyClaimed,
			logger.FieldEpEui, epEUI, logger.FieldBsEui, session.BaseStationEUI)
		return false, nil
	}

	dispatched, err := d.dispatchNextPending(ctx, ownerTenantID, session, epEUI)
	// A frame that may be on the wire keeps the window: a second downlink could land in it.
	if !dispatched && !errors.Is(err, bssci.ErrAmbiguousWrite) {
		if releaseErr := d.windows.ReleaseDownlinkWindow(ctx, ownerTenantID, messageID); releaseErr != nil {
			d.logger.ErrorContext(ctx, bssci.LogDispatcherWindowReleaseFailed,
				logger.FieldEpEui, epEUI, logger.FieldError, releaseErr)
		}
	}
	return dispatched, err
}

// dispatchNextPending reserves the endpoint's highest-priority pending
// downlink and dispatches it through the shared dispatch path.
func (d *downlinkDispatcher) dispatchNextPending(ctx context.Context, ownerTenantID int64, session *bssci.Session, epEUI uint64) (bool, error) {
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	// Reserve and commit before any wire write. The selected row carries the
	// organization it was enqueued under.
	dl, err := d.reserver.ReserveNextPending(ctx, ownerTenantID, epEUIBytes, session.BaseStationEUI)
	if errors.Is(err, storage.ErrNotFound) {
		d.logger.DebugContext(ctx, bssci.LogDispatcherNoPending, logger.FieldEpEui, epEUI)
		return false, nil
	}
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogDispatcherQueryFailed, logger.FieldError, err, logger.FieldEpEui, epEUI)
		return false, nil // Graceful degradation - don't fail uplink
	}

	return d.dispatchReserved(ctx, ownerTenantID, session, epEUI, dl)
}

// DispatchQueue reserves one exact pending queue row (by queue ID, tenant,
// endpoint, and the organization the downlink was enqueued under) and
// dispatches it through the shared dispatch path. Used for SCACI-initiated
// immediate delivery (SCACI §3.10.1). Returns dispatched=false with a nil
// error when no matching row is in 'pending' state (already dispatched,
// revoked, or foreign).
func (d *downlinkDispatcher) DispatchQueue(
	ctx context.Context,
	ownerTenantID int64,
	enqueueOrgUUID uuid.UUID,
	session *bssci.Session,
	queueID uint64,
	epEUI uint64,
) (bool, error) {
	if ownerTenantID == 0 {
		d.logger.WarnContext(ctx, bssci.LogDispatcherNoTenant, logger.FieldEpEui, epEUI)
		return false, nil
	}
	// Fail closed on a missing organization: the exact reservation is scoped to
	// the organization the row was enqueued under, so a request that cannot
	// name it must not reserve anything.
	if enqueueOrgUUID == uuid.Nil {
		d.logger.WarnContext(ctx, bssci.LogDispatcherNoOrganization, logger.FieldEpEui, epEUI, logger.FieldQueID, queueID)
		return false, nil
	}

	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	// Exact reservation (pending → reserved), scoped to the enqueuing organization.
	dl, err := d.reserver.ReserveByQueueID(ctx, ownerTenantID, enqueueOrgUUID, queueID, epEUIBytes, session.BaseStationEUI)
	if errors.Is(err, storage.ErrNotFound) {
		d.logger.WarnContext(ctx, bssci.LogDispatcherNoPending, logger.FieldEpEui, epEUI, logger.FieldQueID, queueID)
		return false, nil
	}
	if err != nil {
		d.logger.ErrorContext(ctx, bssci.LogDispatcherQueryFailed, logger.FieldError, err, logger.FieldEpEui, epEUI, logger.FieldQueID, queueID)
		return false, err
	}

	return d.dispatchReserved(ctx, ownerTenantID, session, epEUI, dl)
}

// dispatchReserved sends a reserved queue row and confirms it as queued. The
// row is already durably 'reserved' and no transaction is open.
//
// Failure semantics:
//   - Ambiguous wire write (bssci.ErrAmbiguousWrite): the row stays reserved -
//     never back to pending, because a duplicate send with a new operation ID
//     could corrupt the stream. The send layer closes the connection; resume
//     reissues the persisted operations with their original IDs and the
//     dlDataQueRsp handler repairs the reserved→queued status, while a
//     session that is not resumed reclaims the row (ReclaimReservations).
//   - Definite failure (the connection closed before the frame was written in
//     full, so it cannot have been delivered): the row is released back to
//     pending (documented at-least-once retry).
//   - Queued-confirmation failure after a successful send: the row stays
//     reserved and the send is reported dispatched; the idempotent
//     confirmation is repeated from the dlDataQueRsp handler.
func (d *downlinkDispatcher) dispatchReserved(
	ctx context.Context,
	ownerTenantID int64,
	session *bssci.Session,
	epEUI uint64,
	dl *storage.DownlinkMessage,
) (bool, error) {
	// Every queue row carries the organization it was enqueued under (enqueue
	// rejects a row without one and the column is NOT NULL). A reservation
	// that returns an ownerless row means that invariant was bypassed: this
	// flow must not send it, so log the violation and stop with zero writes,
	// leaving the row reserved for operator inspection.
	if dl.OrganizationID == nil || *dl.OrganizationID == uuid.Nil {
		d.logger.ErrorContext(ctx, bssci.LogDispatcherOrgMismatch,
			logger.FieldQueID, dl.QueID, logger.FieldEpEui, epEUI)
		return false, bssci.ErrDispatchOrgMismatch
	}

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
		dl.OrganizationID,
		dl.DlRxStatQry,
	)
	if err != nil {
		if errors.Is(err, bssci.ErrAmbiguousWrite) {
			d.logger.ErrorContext(ctx, bssci.LogDispatcherSendFailed,
				logger.FieldQueID, dl.QueID, logger.FieldEpEui, epEUI, logger.FieldError, err)
			return false, err
		}
		// Definite pre-write failure: release the reservation for retry
		d.logger.ErrorContext(ctx, bssci.LogDispatcherSendFailed,
			logger.FieldQueID, dl.QueID, logger.FieldEpEui, epEUI, logger.FieldError, err)
		if relErr := d.queue.UpdateDownlinkStatus(ctx,
			strconv.FormatInt(dl.ID, 10), bssci.DLQueueStatusPending, dl.OrganizationID); relErr != nil {
			d.logger.ErrorContext(ctx, bssci.LogDispatcherReleaseFailed,
				logger.FieldQueID, dl.QueID, logger.FieldError, relErr)
		}
		return false, err
	}

	// Confirm reserved→queued with transmission metadata (idempotent single
	// statement on the regular repository - no transaction spans the send).
	// packetCnt = nil (unknown until the dlDataRes transmission result, BSSCI 5.14)
	txTime := d.clock.Now().UnixNano()
	if err := d.queue.MarkReservedAsQueued(
		ctx,
		uint64(dl.QueID), //nolint:gosec // G115: QueID is always positive (DB-assigned sequence)
		ownerTenantID,
		session.BaseStationEUI,
		txTime,
		nil, // packetCnt - set when the dlDataRes transmission result arrives
		dl.OrganizationID,
	); err != nil {
		// The send happened: report dispatched, leave the row reserved. The
		// dlDataQueRsp handler repeats the idempotent confirmation.
		d.logger.ErrorContext(ctx, bssci.LogDispatcherMarkSentFailed,
			logger.FieldQueID, dl.QueID, logger.FieldError, err)
		return true, nil
	}

	d.logger.InfoContext(ctx, bssci.LogDispatcherSuccess,
		logger.FieldQueID, dl.QueID,
		logger.FieldEpEui, epEUI,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOwnerTenantID, ownerTenantID,
		logger.FieldOwnerOrgUUID, *dl.OrganizationID)

	return true, nil
}
