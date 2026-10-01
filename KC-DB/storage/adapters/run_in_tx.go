package adapters

import (
	"context"
	"errors"
	"fmt"
)

// committable is how a transaction handle ends.
type committable interface {
	Commit() error
	Rollback() error
}

// runInTx opens a transaction with begin and runs fn with its handle. The
// transaction commits when fn returns nil and rolls back otherwise, including
// when fn panics - the panic continues to unwind after the rollback.
func runInTx[H committable](ctx context.Context, begin func(context.Context) (H, error), fn func(H) error) (err error) {
	tx, beginErr := begin(ctx)
	if beginErr != nil {
		return fmt.Errorf("%w: %w", ErrTxBegin, beginErr)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rbErr := tx.Rollback(); rbErr != nil && err != nil {
			err = errors.Join(err, fmt.Errorf("%s: %w", errWrapRollback, rbErr))
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		err = fmt.Errorf("%w: %w", ErrTxCommit, commitErr)
		return err
	}
	committed = true

	return nil
}
