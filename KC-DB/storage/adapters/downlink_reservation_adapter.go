package adapters

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// DownlinkReservationAdapter exposes the transactional half of the downlink
// dispatch lifecycle as single atomic operations.
//
// ReserveNextPendingDownlink uses FOR UPDATE SKIP LOCKED and is only valid
// inside a transaction, so its callers previously had to open, commit and roll
// back one themselves. Keeping that here means no transaction handle crosses
// into the service layer and the reservation cannot be left open by a caller
// that returns early.
type DownlinkReservationAdapter struct {
	begin beginDownlinkTx
}

// NewDownlinkReservationAdapter creates a reservation adapter over the store.
func NewDownlinkReservationAdapter(db *postgres.DB) *DownlinkReservationAdapter {
	return &DownlinkReservationAdapter{begin: downlinkBeginner(db)}
}

// ReserveNextPending reserves the highest-priority pending downlink for an
// endpoint and commits before returning, so the caller may perform network I/O
// without holding a transaction open. Nothing pending is storage.ErrNotFound.
// The reserved row's organization_id is authoritative for the dispatch.
func (a *DownlinkReservationAdapter) ReserveNextPending(ctx context.Context, tenantID int64,
	epEUI []byte, bsEUI uint64,
) (*storage.DownlinkMessage, error) {
	return a.reserve(ctx, func(tx downlinkTxHandle) (*storage.DownlinkMessage, error) {
		return tx.MIOTYDownlinks().ReserveNextPendingDownlink(ctx, tenantID, epEUI, bsEUI)
	})
}

// ReserveByQueueID reserves one exact pending queue row scoped to the
// organization the downlink was enqueued under. The underlying statement is
// atomic on its own; it runs in the same transaction shape as
// ReserveNextPending so both reservation paths behave identically on failure.
func (a *DownlinkReservationAdapter) ReserveByQueueID(ctx context.Context, tenantID int64,
	orgID uuid.UUID, queueID uint64, epEUI []byte, bsEUI uint64,
) (*storage.DownlinkMessage, error) {
	return a.reserve(ctx, func(tx downlinkTxHandle) (*storage.DownlinkMessage, error) {
		return tx.MIOTYDownlinks().ReservePendingDownlinkByQueueID(ctx, tenantID, orgID, queueID, epEUI, bsEUI)
	})
}

// reserve runs fn in a transaction, committing only when a row was reserved.
// The transaction is marked closed before Commit is attempted, so the deferred
// rollback never runs against a finished transaction; a panic rolls back
// before it continues to unwind, so a transaction is never abandoned mid-flight.
func (a *DownlinkReservationAdapter) reserve(ctx context.Context,
	fn func(t downlinkTxHandle) (*storage.DownlinkMessage, error),
) (dl *storage.DownlinkMessage, err error) {
	tx, err := a.begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTxBegin, err)
	}

	open := true
	defer func() {
		if !open {
			return
		}
		// Surface a rollback failure even when the reservation itself
		// succeeded: a rollback that cannot release the transaction leaves the
		// row locked, and swallowing it would hide that from the caller.
		if rbErr := tx.Rollback(); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("%w: %w", ErrTxRollback, rbErr))
		}
	}()

	dl, err = fn(tx)
	if err != nil {
		// storage.ErrNotFound (nothing pending) rolls back like any other failure.
		return nil, fmt.Errorf("%s: %w", errWrapDownlinkReservationReserve, err)
	}

	// Commit finishes the transaction whether it succeeds or fails; mark it
	// closed first so the deferred rollback does not fire on a finished
	// transaction and join a spurious sql.ErrTxDone.
	open = false
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTxCommit, err)
	}

	return dl, nil
}
