package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/jmoiron/sqlx"
)

// UpdateWithEUI atomically cascades an EUI change across all dependent tables and updates endpoint fields
func (r *EndPointRepository) UpdateWithEUI(ctx context.Context, tenantID int64, oldEui []byte, endpoint *models.EndPoint) (_ *models.EndPoint, err error) {
	if len(oldEui) != 8 || len(endpoint.EUI) != 8 {
		return nil, errTextInvalidEUILengthExpected8Bytes
	}
	newEui := endpoint.EUI[:]

	tx, commit, rollback, err := r.beginCascade(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapBeginTransaction, err)
	}
	defer rollback(&err)

	// Verify tenant ownership
	var epID int64
	err = sqlx.GetContext(ctx, tx, &epID, "SELECT id FROM endpoints WHERE tenant_id = $1 AND ep_eui = $2", tenantID, oldEui)
	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapVerifyEndpointOwnership, err)
	}

	if err := r.updateEndpointFields(ctx, tx, epID, tenantID, newEui, endpoint); err != nil {
		if classified := classifyEndpointPQError(err); classified != nil {
			return nil, classified
		}
		return nil, fmt.Errorf("%s: %w", errWrapUpdateEndpoints, err)
	}

	// Cascade EUI to dependent tables

	// BYTEA columns: downlink_queue.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE downlink_queue SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDownlinkQueueEpEui, err)
	}

	// BYTEA columns: dl_rx_status.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDlRxStatusEpEui, err)
	}

	// BYTEA columns: dl_rx_status_queries.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE dl_rx_status_queries SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDlRxStatusQueriesEpEui, err)
	}

	// BYTEA columns: messages.ep_eui (8-byte big-endian per migration 000135)
	_, err = tx.ExecContext(ctx, "UPDATE messages SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMessagesEpEui, err)
	}

	// BYTEA columns: messages_archive.ep_eui (rebuilt LIKE messages by migration 000139)
	_, err = tx.ExecContext(ctx, "UPDATE messages_archive SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMessagesArchiveEpEui, err)
	}

	// Preserved legacy archive (pre-000139) participates in identity
	// maintenance so its rows never carry a stale EUI
	if err := updateLegacyArchiveEUI(ctx, tx, legacyArchiveEpEUI, newEui, oldEui); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMessagesArchivePre000139EpEui, err)
	}

	// BYTEA columns: mioty_message_deduplication.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE mioty_message_deduplication SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateMiotyMessageDeduplicationEpEui, err)
	}

	// BYTEA columns: roaming_events.ep_eui
	_, err = tx.ExecContext(ctx, "UPDATE roaming_events SET ep_eui = $1 WHERE ep_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateRoamingEventsEpEui, err)
	}

	// BYTEA columns: bssci_pending_operations.endpoint_eui (nullable)
	_, err = tx.ExecContext(ctx, "UPDATE bssci_pending_operations SET endpoint_eui = $1 WHERE endpoint_eui = $2", newEui, oldEui)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateBssciPendingOperationsEndpointEui, err)
	}

	err = commit()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapCommitTransaction, err)
	}

	endpoint.ID = epID
	return endpoint, nil
}

// beginCascade opens the transaction for a multi-table cascade, or reuses the
// enclosing transaction when the repository already runs inside one. The
// returned rollback is safe to defer unconditionally with the caller's named
// error result: after commit it does nothing, and a failed rollback joins err.
func (r *EndPointRepository) beginCascade(ctx context.Context) (sqlx.ExtContext, func() error, func(err *error), error) {
	if r.conn == nil {
		return r.db, func() error { return nil }, func(*error) {}, nil
	}
	tx, err := r.conn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	rollback := func(err *error) { sqlcleanup.RollbackUncommitted(tx, errWrapRollbackTransaction, err) }
	return tx, tx.Commit, rollback, nil
}
