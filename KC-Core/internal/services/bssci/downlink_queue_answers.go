package bssciservices

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// DownlinkHolderWriter records the base station that acknowledged a
// downlink's dlDataQue as its holder and confirms the reserved row queued.
type DownlinkHolderWriter interface {
	UpdateDownlinkBaseStation(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64) error
	MarkReservedAsQueued(ctx context.Context, queID uint64, tenantID int64, bsEUI uint64, txTime int64, packetCnt *uint32, orgID *uuid.UUID) error
}

// ProcessQueueAck records the base station that answered a dlDataQue with
// dlDataQueRsp (BSSCI §3.12) as the holder of the downlink and repeats the
// idempotent reserved-to-queued confirmation, repairing a row whose
// dispatcher confirmation was lost. Both writes are attempted.
func (d *downlinkService) ProcessQueueAck(ctx context.Context, session *bssci.Session, ack bssci.QueueAcknowledgement) error {
	queueID, tenantID, err := queueOwner(ack.QueueID, ack.OwnerTenant)
	if err != nil {
		return err
	}
	var failures []error
	if err := d.holders.UpdateDownlinkBaseStation(ctx, queueID, tenantID, session.BaseStationEUI); err != nil {
		failures = append(failures, fmt.Errorf("%w: %w", errRecordDownlinkHolder, err))
	}
	if err := d.holders.MarkReservedAsQueued(ctx, queueID, tenantID, session.BaseStationEUI,
		d.clock.Now().UnixNano(), nil, ack.OrganizationID); err != nil {
		failures = append(failures, fmt.Errorf("%w: %w", errConfirmDownlinkQueued, err))
	}
	return errors.Join(failures...)
}

// ProcessQueueError fails a downlink whose dlDataQue the base station
// answered with error (BSSCI §3.17): it will never be transmitted, so its
// originators learn it was discarded as invalid (SCACI §3.12).
func (d *downlinkService) ProcessQueueError(ctx context.Context, session *bssci.Session, rejection bssci.QueueRejection) error {
	queueID, tenantID, err := queueOwner(rejection.QueueID, rejection.OwnerTenant)
	if err != nil {
		return err
	}
	reason := fmt.Sprintf(failureReasonStationErrorFmt, rejection.Code, rejection.Message)
	downlink, err := d.downlinks.FailQueuedDownlink(ctx, rejection.QueueID, tenantID, session.BaseStationEUI, reason)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailRejectedDownlink, err)
	}
	d.tenantResolver.UnregisterQueueTenant(rejection.QueueID)

	discarded := mioty.DLDataResult{EpEui: rejection.EndpointEUI, QueId: queueID, Result: mioty.ResultInvalid}
	d.results.ReportStationResult(ctx, downlink, discarded, session)
	return nil
}

// queueOwner validates the downlink a base station's dlDataQue answer names,
// returning its queue id as the wire carries it, and parses the tenant
// owning it.
func queueOwner(queueID int64, owner string) (uint64, int64, error) {
	wire, err := wireQueueID(queueID)
	if err != nil {
		return 0, 0, err
	}
	tenantID, err := strconv.ParseInt(owner, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", errInvalidOwnerTenant, err)
	}
	return wire, tenantID, nil
}

// wireQueueID is a stored queue id as the wire carries it; ids are allocated positive.
func wireQueueID(queueID int64) (uint64, error) {
	if queueID <= 0 {
		return 0, fmt.Errorf(errFmtInvalidQueueID, queueID)
	}
	return uint64(queueID), nil
}
