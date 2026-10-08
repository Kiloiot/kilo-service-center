package postgres

import (
	"fmt"
	"strconv"
)

// The Application Center's queue id is an unsigned 64-bit numeric (SCACI
// §3.10.1), stored as itself in the NUMERIC(20,0) ac_que_id column. The
// database/sql driver cannot carry a uint64 at or above 2^63, so the id
// crosses the driver as its decimal text in both directions.

// applicationQueueIDParam renders the id for the ac_que_id column; nil is NULL.
func applicationQueueIDParam(id *uint64) interface{} {
	if id == nil {
		return nil
	}
	return strconv.FormatUint(*id, 10)
}

// applicationQueueIDScanner reads the ac_que_id column into the id; NULL is nil.
type applicationQueueIDScanner struct {
	dst **uint64
}

// Scan implements sql.Scanner.
func (s applicationQueueIDScanner) Scan(src interface{}) error {
	var text string
	switch v := src.(type) {
	case nil:
		*s.dst = nil
		return nil
	case []byte:
		text = string(v)
	case string:
		text = v
	default:
		return fmt.Errorf(errFmtScanApplicationQueueID, src)
	}
	id, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapParseApplicationQueueID, err)
	}
	*s.dst = &id
	return nil
}
