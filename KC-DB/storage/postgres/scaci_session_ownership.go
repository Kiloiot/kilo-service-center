package postgres

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// DisconnectAbandonedSessions hands the sessions a previous process of the
// service center scEUI left live back to the resumable disconnected state,
// as the loss of their connections would have (SCACI §1), and returns their
// ids. A row without an owner predates ownership tracking and is claimed for
// scEUI; rows of other service centers are not touched. The last write to a
// row is the closest known end time.
func (r *SCACISessionRepository) DisconnectAbandonedSessions(ctx context.Context, scEUI models.EUI) ([]int64, error) {
	// sc_eui IS NULL adopts the rows written before migration 000180 once:
	// the service center EUI is configuration the migration cannot know.
	query := `
		UPDATE scaci_sessions
		SET status = $1, sc_eui = $2, disconnected_at = COALESCE(disconnected_at, updated_at)
		WHERE status IN ($3, $4)
		  AND (sc_eui = $2 OR sc_eui IS NULL)
		RETURNING id`

	rows, err := r.db.QueryContext(ctx, query, models.SCACISessionStatusDisconnected, scEUI[:],
		models.SCACISessionStatusActive, models.SCACISessionStatusResumed)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapDisconnectAbandonedSessions, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgCloseRowsAbandonedSessions, logger.FieldError, err)
		}
	}()
	return sessionIDsOf(rows)
}
