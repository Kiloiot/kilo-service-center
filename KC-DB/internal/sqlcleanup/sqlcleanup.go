// Package sqlcleanup reports the failures of deferred result-set closes and
// transaction rollbacks through the caller's named error result, so no
// cleanup error is discarded.
package sqlcleanup

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
)

// rollbacker is the part of a transaction a deferred rollback needs.
type rollbacker interface {
	Rollback() error
}

// CloseRows closes a result set from a deferred call and reports a close
// failure through the caller's named error result, wrapped with wrap, unless
// the caller already returns an error.
func CloseRows(rows io.Closer, wrap string, err *error) {
	if closeErr := rows.Close(); closeErr != nil && *err == nil {
		*err = fmt.Errorf("%s: %w", wrap, closeErr)
	}
}

// RollbackUncommitted rolls back a transaction from a deferred call. A
// committed transaction reports sql.ErrTxDone, which is expected; any other
// rollback failure, wrapped with wrap, joins the caller's named error result.
func RollbackUncommitted(tx rollbacker, wrap string, err *error) {
	rollbackErr := tx.Rollback()
	if rollbackErr == nil || errors.Is(rollbackErr, sql.ErrTxDone) {
		return
	}
	*err = errors.Join(*err, fmt.Errorf("%s: %w", wrap, rollbackErr))
}
