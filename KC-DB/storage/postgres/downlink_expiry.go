package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// DownlinkExpirySweep expires the downlinks whose lifetime elapsed.
type DownlinkExpirySweep struct {
	db    sqlx.ExtContext
	clock clock.Clock
	log   logger.Logger
}

// downlinkNotExpired keeps a downlink whose lifetime has elapsed by the time
// bound to the numbered parameter out of every reservation; the expiry sweep
// reports it instead.
func downlinkNotExpired(nowParam int) string {
	return fmt.Sprintf("(latest_at IS NULL OR latest_at > $%d)", nowParam)
}

// ExpireOverdueDownlinks marks up to limit in-flight downlinks whose lifetime
// has elapsed as expired, whichever tenant owns them, and returns them. A
// downlink a base station held (reserved or queued) names that station in
// BsEui, so it can be revoked there (BSSCI §3.13); BsEui is zero otherwise.
func (r *DownlinkExpirySweep) ExpireOverdueDownlinks(ctx context.Context, limit int) ([]*storage.DownlinkMessage, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s: %w", errWrapExpireOverdueDownlinks, storage.ErrInvalidInput)
	}
	rows, err := r.db.QueryxContext(ctx, sqlExpireOverdueDownlinks,
		mioty.DLQueueStatusExpired, mioty.ResultExpired, limit, r.clock.Now(), statusArray(heldStatuses))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapExpireOverdueDownlinks, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Warn(logMsgRowsClose, logger.FieldError, err, logger.FieldOperation, opExpireOverdueDownlinks)
		}
	}()
	var expired []*storage.DownlinkMessage
	for rows.Next() {
		var holder []byte
		dl, err := scanDownlinkOutcome(rows, &holder)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errWrapExpireOverdueDownlinks, err)
		}
		if len(holder) > 0 {
			dl.BsEui = mioty.EUI64FromBytes(holder)
		}
		dl.Status = mioty.DLQueueStatusExpired
		dl.Result = mioty.ResultExpired
		expired = append(expired, dl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapExpireOverdueDownlinks, err)
	}
	return expired, nil
}

// sqlExpireOverdueDownlinks expires the oldest overdue in-flight rows,
// skipping rows a dispatcher holds locked, and returns the station of each
// row that was held; a pending row may still carry the station of a released
// reservation, which no longer holds it.
const sqlExpireOverdueDownlinks = `
	UPDATE downlink_queue AS d
	SET status = $1, result = $2::text, transmission_result = $2::text, updated_at = $4
	FROM (
		SELECT id, status FROM downlink_queue
		WHERE ` + sqlDownlinkInFlight + ` AND latest_at <= $4
		ORDER BY latest_at
		LIMIT $3
		FOR UPDATE SKIP LOCKED
	) AS due
	WHERE d.id = due.id
	RETURNING d.id, d.que_id, d.ac_que_id, d.ep_eui, d.tenant_id, d.organization_id, d.ac_eui, d.ref,
		CASE WHEN due.status = ANY($5::text[]) THEN d.bs_eui END`

// heldStatuses are the queue states in which a base station holds a downlink.
var heldStatuses = []mioty.DLQueueStatus{mioty.DLQueueStatusReserved, mioty.DLQueueStatusQueued}
