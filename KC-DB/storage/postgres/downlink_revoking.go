package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/internal/sqlcleanup"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// DownlinkRevoking ends the downlinks a base station was asked to drop when
// their lifetime ended while it held them: each stays revoking, with its
// holder, until the station answers or starts a session that discarded it.
type DownlinkRevoking struct {
	db    sqlx.ExtContext
	clock clock.Clock
}

// ExpireRevokedDownlink ends expired the revoking downlink the answering
// station holds, the one a revocation names, once that station confirmed the
// revoke or answered it does not hold it (BSSCI §3.13, §3.17), and returns it
// for its originators; false when no such downlink is revoking, such as one a
// dlDataRes ended first or another station's.
func (r *DownlinkRevoking) ExpireRevokedDownlink(ctx context.Context, revocation storage.DownlinkRevocation) (*storage.DownlinkMessage, bool, error) {
	if revocation.Station == nil {
		return nil, false, fmt.Errorf("%s: %w", errWrapExpireRevokedDownlink, storage.ErrInvalidInput)
	}
	scope := newSQLScope(mioty.DLQueueStatusExpired, mioty.ResultExpired, r.clock.Now())
	scopeRevocation(scope, revocation)
	scope.equals(colStatus, mioty.DLQueueStatusRevoking)
	scope.equals(colBsEUI, mioty.EUI64Bytes(*revocation.Station))
	downlink, err := scanDownlinkOutcome(r.db.QueryRowxContext(ctx, sqlExpireRevoking+scope.where()+sqlReturningRevoked, scope.args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", errWrapExpireRevokedDownlink, err)
	}
	downlink.BsEui = *revocation.Station
	return expiredOutcome(downlink), true, nil
}

// ExpireStationRevocations ends expired every downlink the base station was
// asked to drop, of every tenant, and returns them for their originators: its
// new session is not resumed, so it discarded them with the previous one
// (BSSCI §1) and will never transmit them.
func (r *DownlinkRevoking) ExpireStationRevocations(ctx context.Context, bsEUI uint64) (expired []*storage.DownlinkMessage, err error) {
	scope := newSQLScope(mioty.DLQueueStatusExpired, mioty.ResultExpired, r.clock.Now())
	scope.equals(colStatus, mioty.DLQueueStatusRevoking)
	scope.equals(colBsEUI, mioty.EUI64Bytes(bsEUI))
	rows, err := r.db.QueryxContext(ctx, sqlExpireRevoking+scope.where()+sqlReturningRevoked, scope.args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapExpireStationRevocations, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapExpireStationRevocations, &err)
	return scanHeldOutcomes(rows, bsEUI, errWrapExpireStationRevocations, expiredOutcome)
}

// ListStationRevocations lists the downlinks the base station is asked to
// drop, of every tenant, so a session that resumed is asked again.
func (r *DownlinkRevoking) ListStationRevocations(ctx context.Context, bsEUI uint64) (revoking []*storage.DownlinkMessage, err error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT `+downlinkOutcomeColumns+` FROM downlink_queue WHERE status = $1 AND bs_eui = $2 ORDER BY id`,
		mioty.DLQueueStatusRevoking, mioty.EUI64Bytes(bsEUI))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapListStationRevocations, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapListStationRevocations, &err)
	return scanHeldOutcomes(rows, bsEUI, errWrapListStationRevocations, revokingOutcome)
}

// ClaimUnansweredRevocations returns up to limit downlinks of any tenant that
// stay revoking at one of the connected stations although that station was
// last asked to drop them at least unansweredFor ago, oldest ask first, and
// records that it is asked again now. Rows another sweep holds locked are
// skipped.
func (r *DownlinkRevoking) ClaimUnansweredRevocations(ctx context.Context, connected []uint64, unansweredFor time.Duration, limit int) (claimed []*storage.DownlinkMessage, err error) {
	if limit <= 0 || unansweredFor <= 0 {
		return nil, fmt.Errorf("%s: %w", errWrapClaimUnansweredRevocations, storage.ErrInvalidInput)
	}
	stations := make([][]byte, len(connected))
	for i, bsEUI := range connected {
		stations[i] = mioty.EUI64Bytes(bsEUI)
	}
	now := r.clock.Now()
	rows, err := r.db.QueryxContext(ctx, sqlClaimUnansweredRevocations,
		mioty.DLQueueStatusRevoking, pq.ByteaArray(stations), now.Add(-unansweredFor), limit, now)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapClaimUnansweredRevocations, err)
	}
	defer sqlcleanup.CloseRows(rows, errWrapClaimUnansweredRevocations, &err)
	for rows.Next() {
		var holder []byte
		downlink, scanErr := scanDownlinkOutcome(rows, &holder)
		if scanErr != nil {
			return nil, fmt.Errorf("%s: %w", errWrapClaimUnansweredRevocations, scanErr)
		}
		downlink.BsEui = mioty.EUI64FromBytes(holder)
		claimed = append(claimed, revokingOutcome(downlink))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapClaimUnansweredRevocations, err)
	}
	return claimed, nil
}

// sqlClaimUnansweredRevocations moves the ask time ($5) of the oldest revoking
// rows held by a station in $2 that were last asked no later than $3.
const sqlClaimUnansweredRevocations = `
	UPDATE downlink_queue AS d
	SET revoke_asked_at = $5, updated_at = $5
	FROM (
		SELECT id FROM downlink_queue
		WHERE status = $1 AND bs_eui = ANY($2::bytea[]) AND (revoke_asked_at IS NULL OR revoke_asked_at <= $3)
		ORDER BY revoke_asked_at NULLS FIRST
		LIMIT $4
		FOR UPDATE SKIP LOCKED
	) AS unanswered
	WHERE d.id = unanswered.id` + sqlReturningHeldOutcome

// SQL of the revoke completions: the expiry they record and the row they return.
const (
	sqlExpireRevoking   = `UPDATE downlink_queue SET status = $1, result = $2::text, transmission_result = $2::text, updated_at = $3 WHERE `
	sqlReturningRevoked = ` RETURNING ` + downlinkOutcomeColumns
)

// scanHeldOutcomes reads the outcome rows of the downlinks bsEUI holds, each
// marked by mark.
func scanHeldOutcomes(rows *sqlx.Rows, bsEUI uint64, wrap string, mark func(*storage.DownlinkMessage) *storage.DownlinkMessage) ([]*storage.DownlinkMessage, error) {
	var downlinks []*storage.DownlinkMessage
	for rows.Next() {
		downlink, err := scanDownlinkOutcome(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", wrap, err)
		}
		downlink.BsEui = bsEUI
		downlinks = append(downlinks, mark(downlink))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", wrap, err)
	}
	return downlinks, nil
}

// expiredOutcome marks a downlink ended expired.
func expiredOutcome(downlink *storage.DownlinkMessage) *storage.DownlinkMessage {
	downlink.Status = mioty.DLQueueStatusExpired
	downlink.Result = mioty.ResultExpired
	return downlink
}

// revokingOutcome marks a downlink its holder is asked to drop.
func revokingOutcome(downlink *storage.DownlinkMessage) *storage.DownlinkMessage {
	downlink.Status = mioty.DLQueueStatusRevoking
	return downlink
}
