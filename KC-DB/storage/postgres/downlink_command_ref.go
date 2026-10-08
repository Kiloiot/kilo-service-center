package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// inCommandTransaction runs enqueue in a transaction that, for a command with
// a ref, first takes the lock of that ref, so a reception of the same command
// waits until the first one committed or rolled back and then sees its row.
func (r *DownlinkQueueWriter) inCommandTransaction(ctx context.Context, command storage.DownlinkCommandRef, enqueue func(*sqlx.Tx) error) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapEnqueueDownlink, err)
	}
	defer sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, &err)
	if command.Ref != "" {
		if _, err = tx.ExecContext(ctx, sqlLockCommandRef, command.TenantID, command.OrganizationID, mioty.EUI64Bytes(command.EpEUI), command.Ref); err != nil {
			return fmt.Errorf("%s: %w", errWrapEnqueueDownlink, err)
		}
	}
	if err = enqueue(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", errWrapEnqueueDownlink, err)
	}
	return nil
}

// admitDeadline refuses a downlink whose command deadline, at the stored
// precision, is not after enqueuedAt, unless the command's ref names a
// downlink already queued, which reports the repeat instead of a refusal of
// the first command.
func admitDeadline(ctx context.Context, tx sqlx.QueryerContext, command storage.DownlinkCommandRef, expiresAt *time.Time, enqueuedAt time.Time) error {
	if expiresAt == nil || expiresAt.Truncate(storedTimePrecision).After(enqueuedAt) {
		return nil
	}
	if command.Ref == "" {
		return storage.ErrDownlinkDeadlineElapsed
	}
	queued, err := commandQueued(ctx, tx, command)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %w", errWrapEnqueueDownlink, err)
	case queued:
		return storage.ErrDownlinkRefTaken
	default:
		return storage.ErrDownlinkDeadlineElapsed
	}
}

// CommandQueued reports whether the organization already queued a downlink
// for the endpoint under the MQTT command's ref, in flight or finished.
func (r *DownlinkQueueWriter) CommandQueued(ctx context.Context, command storage.DownlinkCommandRef) (bool, error) {
	queued, err := commandQueued(ctx, r.db, command)
	if err != nil {
		return false, fmt.Errorf("%s: %w", errWrapCommandQueued, err)
	}
	return queued, nil
}

// commandQueued runs sqlCommandRefQueued for the command.
func commandQueued(ctx context.Context, db sqlx.QueryerContext, command storage.DownlinkCommandRef) (bool, error) {
	var queued bool
	err := db.QueryRowxContext(ctx, sqlCommandRefQueued,
		command.TenantID, command.OrganizationID, mioty.EUI64Bytes(command.EpEUI), command.Ref).Scan(&queued)
	return queued, err
}

// sqlLockCommandRef serializes the receptions of one MQTT command, named by
// its tenant, organization, endpoint and ref, for the rest of the transaction.
const sqlLockCommandRef = `SELECT pg_advisory_xact_lock(hashtextextended(concat_ws(chr(31), $1::text, $2::text, encode($3::bytea, 'hex'), $4::text), 0))`

// sqlCommandRefQueued reports whether the organization queued a downlink for
// the endpoint under the ref.
const sqlCommandRefQueued = `
	SELECT EXISTS (SELECT 1 FROM downlink_queue
	               WHERE tenant_id = $1 AND organization_id = $2 AND ep_eui = $3 AND ref = $4)`
