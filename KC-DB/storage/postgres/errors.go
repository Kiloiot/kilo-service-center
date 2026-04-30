package postgres

import (
	"errors"
	"fmt"

	kcerrors "github.com/kilocenter/KC-DB/common/errors"
	"github.com/lib/pq"
)

// Downlink dispatch errors
var (
	// ErrDownlinkAlreadyReserved indicates downlink was reserved by concurrent dispatcher
	// This occurs when FOR UPDATE SKIP LOCKED finds no available rows due to concurrent reservation
	ErrDownlinkAlreadyReserved = errors.New("downlink already reserved for dispatch")

	// ErrNotImplemented indicates method only available within transaction context
	// Returned by non-transactional *DB stubs for tx-only methods
	ErrNotImplemented = errors.New("method not implemented outside transaction context")
)

// Message repository errors
var (
	// ErrNoMatchingMessage indicates no message matched UPDATE criteria (dedup window edge case)
	// Caller should log with context - repo layer returns sentinel only (no logging)
	ErrNoMatchingMessage = errors.New("no matching message found")
)

// IsUniqueViolation checks if error is PostgreSQL unique constraint violation (SQLSTATE 23505)
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

// WrapDuplicateError wraps PostgreSQL unique violations as kcerrors.ErrDuplicate
// while preserving original error details for diagnostics
// **DOUBLE %w FORMAT**: ErrDuplicate FIRST so errors.Is() works, then pq.Error for errors.As()
func WrapDuplicateError(err error, resource string) error {
	if err == nil {
		return nil
	}

	if IsUniqueViolation(err) {
		// Preserve both the semantic error and original pq.Error
		var pqErr *pq.Error
		errors.As(err, &pqErr)

		// **CRITICAL**: ErrDuplicate must be FIRST %w for errors.Is(wrapped, ErrDuplicate) to succeed
		// Second %w preserves original pq.Error for errors.As(wrapped, &pqErr)
		return fmt.Errorf("%s already exists (constraint: %s, table: %s): %w: %w",
			resource,
			pqErr.Constraint,
			pqErr.Table,
			kcerrors.ErrDuplicate, // FIRST %w
			err)                   // SECOND %w (original pq.Error)
	}

	return err
}
