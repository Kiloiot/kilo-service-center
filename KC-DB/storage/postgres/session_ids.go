package postgres

import (
	"database/sql"
	"fmt"
)

// sessionIDsOf collects the session ids a statement returned; the caller
// closes rows.
func sessionIDsOf(rows *sql.Rows) ([]int64, error) {
	var sessionIDs []int64
	for rows.Next() {
		var sessionID int64
		if err := rows.Scan(&sessionID); err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapScanRetiredSessionID, err)
		}
		sessionIDs = append(sessionIDs, sessionID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapIterateRetiredSessionIDs, err)
	}
	return sessionIDs, nil
}
