package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// DownlinkStationOutcomes records what a base station holding a downlink
// answered: its holder, its result, its refusal, and the endpoint's
// acknowledgement of a transmitted one, which it also reads back for delivery.
type DownlinkStationOutcomes struct {
	db    sqlx.ExtContext
	clock clock.Clock
}

// UpdateDownlinkResult records the result a base station reported for the
// tenant's downlink it holds, reserved, queued or asked to drop (BSSCI
// §3.14): only a result proves what became of a downlink being revoked, so a
// "sent" it reports first is its outcome. It returns the row for its
// originators. The endpoint, the tenant and the holding station are part of
// the match, so a station cannot finish another endpoint's, another tenant's
// or another station's downlink, nor one back in the queue:
// storage.ErrDownlinkNotFound for each, as for an unknown queue id. A
// downlink that already finished, expired or revoked included, keeps its
// outcome: storage.ErrDownlinkFinished, or storage.ErrDownlinkSentAfterExpiry
// for the first "sent" its holder reports after it ended expired.
func (r *DownlinkStationOutcomes) UpdateDownlinkResult(ctx context.Context, tenantID int64, bsEUI uint64, result *mioty.DLDataResult) (*storage.DownlinkMessage, error) {
	var transmissionPacketCnt *int64
	if result.PacketCnt != nil {
		counter := int64(*result.PacketCnt)
		transmissionPacketCnt = &counter
	}
	now := r.clock.Now()
	status, transmittedAt := resultTransition(result.Result, now)
	// Explicit ::text / ::bigint casts let lib/pq lock down parameter types
	// when $1 is referenced twice and $2, $9 and $10 can be NULL.
	row := r.db.QueryRowxContext(ctx, `
		UPDATE downlink_queue
		SET result = $1::text,
		    tx_time = $2::bigint,
		    transmission_result = $1::text,
		    transmission_time = $2::bigint,
		    transmission_packet_cnt = $4::bigint,
		    status = COALESCE($9::text, status),
		    transmitted_at = COALESCE($10::timestamptz, transmitted_at),
		    updated_at = $8
		WHERE que_id = $5
		  AND ep_eui = $6
		  AND tenant_id = $7
		  AND bs_eui = $3
		  AND status = ANY($11::text[])
		RETURNING `+downlinkOutcomeColumns,
		result.Result, result.TxTime, mioty.EUI64Bytes(bsEUI), transmissionPacketCnt, result.QueId, mioty.EUI64Bytes(result.EpEui), tenantID, now, status, transmittedAt,
		statusArray(heldStatuses))
	downlink, err := scanDownlinkOutcome(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.finishedOrMissing(ctx, tenantID, bsEUI, result)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapUpdateDownlinkResult, err)
	}
	return downlink, nil
}

// resultTransition is the queue status a reported result ends the downlink
// in and, for a sent one, when it was transmitted; nil keeps the column.
func resultTransition(result string, now time.Time) (status, transmittedAt interface{}) {
	queueStatus, ok := mioty.QueueStatusForResult(result)
	if !ok {
		return nil, nil
	}
	if queueStatus == mioty.DLQueueStatusTransmitted {
		transmittedAt = now
	}
	return string(queueStatus), transmittedAt
}

// finishedOrMissing tells a result for a downlink that already finished from
// one for a downlink the reporting station does not hold: one the tenant's
// endpoint never had, one another station holds, or one back in the queue. A
// "sent" from the station that held a downlink which then ended expired
// contradicts the expiry its originators were told; only the first such
// report is told apart, so a station repeating it cannot multiply it.
func (r *DownlinkStationOutcomes) finishedOrMissing(ctx context.Context, tenantID int64, bsEUI uint64, result *mioty.DLDataResult) error {
	var status mioty.DLQueueStatus
	var sentAfterExpiry bool
	err := r.db.QueryRowxContext(ctx, sqlFinishedDownlink,
		result.QueId, mioty.EUI64Bytes(result.EpEui), tenantID, statusArray(mioty.TerminalStatuses()),
		mioty.EUI64Bytes(bsEUI), r.clock.Now(), mioty.DLQueueStatusExpired, result.Result == mioty.ResultSent).Scan(&status, &sentAfterExpiry)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return storage.ErrDownlinkNotFound
	case err != nil:
		return fmt.Errorf("%s: %w", errWrapUpdateDownlinkResult, err)
	case sentAfterExpiry:
		return storage.ErrDownlinkSentAfterExpiry
	default:
		return storage.ErrDownlinkFinished
	}
}

// sqlFinishedDownlink reads the newest finished row of the queue id and, for
// a "sent" ($8) from the station ($5) that held it when it ended expired
// ($7), records at $6 that this station reported it sent unless already
// recorded, telling whether this report recorded it.
const sqlFinishedDownlink = `
	WITH finished AS (
		SELECT id, status, bs_eui FROM downlink_queue
		WHERE que_id = $1 AND ep_eui = $2 AND tenant_id = $3 AND status = ANY($4::text[])
		ORDER BY id DESC
		LIMIT 1
	), reported AS (
		UPDATE downlink_queue AS d SET sent_after_expiry_at = $6
		FROM finished
		WHERE d.id = finished.id AND $8::boolean AND finished.status = $7 AND finished.bs_eui = $5
		  AND d.sent_after_expiry_at IS NULL
		RETURNING d.id
	)
	SELECT status, EXISTS (SELECT 1 FROM reported) FROM finished`

// UpdateDownlinkBaseStation records the base station's acceptance of the
// tenant's downlink (dlDataQueRsp, BSSCI §3.12) while that station still holds
// it, reserved, queued or asked to drop; a late answer from a station that no longer holds
// the row, like one arriving after a reconnect released it, matches nothing.
// storage.ErrDownlinkNotFound covers that, an unknown queue id and another
// tenant's downlink alike.
func (r *DownlinkStationOutcomes) UpdateDownlinkBaseStation(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE downlink_queue
		   SET acknowledged_at = $4,
		       updated_at = $4
		 WHERE que_id = $2
		   AND tenant_id = $3
		   AND bs_eui = $1
		   AND status = ANY($5::text[])`,
		mioty.EUI64Bytes(bsEUI), queID, tenantID, r.clock.Now(), statusArray(heldStatuses))
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapSetDownlinkOwner, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", errWrapGetAffectedRows, err)
	}
	if rowsAffected == 0 {
		return storage.ErrDownlinkNotFound
	}
	return nil
}

// FailQueuedDownlink fails the tenant's downlink the base station holds
// (reserved, queued or asked to drop) and answered with error, recording the station's
// error as the failure reason, and returns the row for its originators;
// storage.ErrDownlinkNotFound when no such row matched, another station's
// downlink included.
func (r *DownlinkStationOutcomes) FailQueuedDownlink(ctx context.Context, queID int64, tenantID int64, bsEUI uint64, reason string) (*storage.DownlinkMessage, error) {
	row := r.db.QueryRowxContext(ctx, `
		UPDATE downlink_queue
		SET status = $1, failure_reason = $2, updated_at = $5
		WHERE que_id = $3 AND tenant_id = $4 AND bs_eui = $6 AND status = ANY($7::text[])
		RETURNING `+downlinkOutcomeColumns,
		mioty.DLQueueStatusFailed, reason, queID, tenantID, r.clock.Now(), mioty.EUI64Bytes(bsEUI), statusArray(heldStatuses))
	downlink, err := scanDownlinkOutcome(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrDownlinkNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapFailQueuedDownlink, err)
	}
	return downlink, nil
}

// MarkEndpointAcknowledged records that the endpoint acknowledged the
// downlink the tenant transmitted in the window of ack.WindowPacketCnt (BSSCI
// §3.10.1 dlAck) and, in the same statement, queues the acknowledgement on
// ack.Channels. It returns the row it marked for the downlink's originators;
// false when no unacknowledged one matched, and then nothing is queued.
func (r *DownlinkStationOutcomes) MarkEndpointAcknowledged(ctx context.Context, ack models.EndpointAckRequest) (*storage.DownlinkMessage, bool, error) {
	messageID, err := uuid.Parse(ack.MessageID)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w: %w", errWrapMarkEndpointAcknowledged, storage.ErrInvalidInput, err)
	}
	row := r.db.QueryRowxContext(ctx, sqlMarkEndpointAcknowledged,
		ack.TenantID, mioty.EUI64Bytes(ack.EpEUI), ack.WindowPacketCnt, mioty.DLQueueStatusTransmitted, r.clock.Now(),
		messageID, deliveryChannelArray(ack.Channels))
	downlink, err := scanDownlinkOutcome(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", errWrapMarkEndpointAcknowledged, err)
	}
	return downlink, true, nil
}

// sqlMarkEndpointAcknowledged acknowledges the newest transmitted downlink of
// the window and queues its acknowledgement, so a mark is never left without
// its delivery; a repeated acknowledgement finds it already marked and queues
// nothing. The outer endpoint_acked_at test makes the second of two
// concurrent receptions of one uplink, which picked the same row, mark
// nothing once the first committed.
const sqlMarkEndpointAcknowledged = `
	WITH acknowledged AS (
		UPDATE downlink_queue
		SET endpoint_acked_at = $5, updated_at = $5
		WHERE id = (
			SELECT id FROM downlink_queue
			WHERE tenant_id = $1 AND ep_eui = $2 AND transmission_packet_cnt = $3
			  AND status = $4 AND endpoint_acked_at IS NULL
			ORDER BY transmitted_at DESC NULLS LAST, id DESC
			LIMIT 1
		) AND endpoint_acked_at IS NULL
		RETURNING ` + downlinkOutcomeColumns + `
	), queued AS (
		INSERT INTO message_delivery_outbox (message_id, channel, owner_tenant_id, acknowledged_downlink_id, next_attempt_at, created_at)
		SELECT $6, channel, $1, acknowledged.id, $5, $5
		FROM acknowledged CROSS JOIN unnest($7::message_delivery_channel[]) AS channel
	)
	SELECT ` + downlinkOutcomeColumns + ` FROM acknowledged`

// GetAcknowledgedDownlink reads the tenant's downlink its endpoint
// acknowledged, with the window it was transmitted in, for the organization
// its acknowledgement is published to; storage.ErrNotFound when the tenant
// has no such acknowledged downlink.
func (r *DownlinkStationOutcomes) GetAcknowledgedDownlink(ctx context.Context, tenantID, downlinkID int64) (*storage.DownlinkMessage, error) {
	var window int64
	row := r.db.QueryRowxContext(ctx, `
		SELECT `+downlinkOutcomeColumns+`, transmission_packet_cnt
		FROM downlink_queue
		WHERE id = $1 AND tenant_id = $2 AND endpoint_acked_at IS NOT NULL`, downlinkID, tenantID)
	downlink, err := scanDownlinkOutcome(row, &window)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%s: %w", errWrapGetAcknowledgedDownlink, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapGetAcknowledgedDownlink, err)
	}
	downlink.TransmissionPacketCnt = window
	return downlink, nil
}
