package postgres

import (
	"errors"
	"fmt"

	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/lib/pq"
)

// Downlink dispatch errors
var (
	// ErrDownlinkAlreadyReserved indicates downlink was reserved by concurrent dispatcher
	// This occurs when FOR UPDATE SKIP LOCKED finds no available rows due to concurrent reservation
	ErrDownlinkAlreadyReserved = errors.New("downlink already reserved for dispatch")
)

// Message repository errors
var ()

const (
	pqCodeUniqueViolation     = "23505"
	pqCodeForeignKeyViolation = "23503"
	pqCodeCheckViolation      = "23514"
	pqCodeInvalidTextRep      = "22P02"
)

// constraintDownlinkQueueID is the installation-wide UNIQUE(que_id) constraint
// of downlink_queue (migration 028).
const constraintDownlinkQueueID = "unique_queue_id"

// IsUniqueViolation checks if error is PostgreSQL unique constraint violation (SQLSTATE 23505)
// PostgreSQL error codes matched when translating driver errors.
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == pqCodeUniqueViolation
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
		return fmt.Errorf(errFmtAlreadyExistsConstraintTable,
			resource,
			pqErr.Constraint,
			pqErr.Table,
			kcerrors.ErrDuplicate, // FIRST %w
			err)                   // SECOND %w (original pq.Error)
	}

	return err
}
