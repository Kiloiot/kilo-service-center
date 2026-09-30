package bssciservices

import (
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// StatusTransitioner records an endpoint's attachment status and the detach
// state a detachment clears.
type StatusTransitioner interface {
	RestateEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error)
	TransitionEndpointStatus(ctx context.Context, tenantID, endpointID int64, status string) (bool, error)
	EndpointDetachStateUpdate(ctx context.Context, tenantID, endpointID int64, p models.EndpointDetachStateParams) error
}

// EndpointStatusNotice is an attachment decision as the endpoint owner's
// subscribers are told it.
type EndpointStatusNotice struct {
	TenantID int64
	Status   *bssci.EPStatusData
	// HeardBy is the base station that heard an over-the-air attach or
	// detach; nil for a decision taken in the service center.
	HeardBy *uint64
}

// EndpointStatusNotifier tells the endpoint owner's subscribers about an
// attachment decision.
type EndpointStatusNotifier interface {
	NotifyEndpointStatus(ctx context.Context, notice EndpointStatusNotice)
}

type endpointAttachmentDecider struct {
	status   StatusTransitioner
	notifier EndpointStatusNotifier
}

// NewEndpointAttachmentDecider builds the one place an endpoint's attachment
// is decided and announced.
func NewEndpointAttachmentDecider(status StatusTransitioner, notifier EndpointStatusNotifier) (bssci.AttachmentDecider, error) {
	if status == nil {
		return nil, errNilStatusTransitioner
	}
	if notifier == nil {
		return nil, errNilEndpointStatusNotifier
	}
	return &endpointAttachmentDecider{status: status, notifier: notifier}, nil
}

// Decide records the decision and announces it when it changed the
// endpoint's status. An over-the-air attach is announced whenever it is
// heard: it starts a new attachment with its own nonce and signature.
func (d *endpointAttachmentDecider) Decide(ctx context.Context, decision bssci.AttachmentDecision) (bool, error) {
	changed, err := d.record(ctx, decision)
	if err != nil {
		return false, err
	}
	if changed || startsOverTheAirAttachment(decision) {
		d.notifier.NotifyEndpointStatus(ctx, noticeOf(decision))
	}
	return changed, nil
}

// record stores the decision. An attach restates the keys and parameters the
// base stations hold even when the endpoint was attached already, so its time
// is always recorded (BSSCI §3.8); a detach carries nothing and records its
// time only when it changes the status.
func (d *endpointAttachmentDecider) record(ctx context.Context, decision bssci.AttachmentDecision) (bool, error) {
	switch decision.Status {
	case bssci.EndpointStatusAttached:
		attached, err := d.status.RestateEndpointStatus(ctx, decision.TenantID, decision.EndpointID, bssci.EndpointStatusAttached)
		if err != nil {
			return false, fmt.Errorf("%w: %w", errRecordAttachment, err)
		}
		return attached, nil
	case endpoint.EndpointStatusDetached:
		detached, err := endpoint.DetachEndpoint(ctx, d.status, decision.TenantID, decision.EndpointID, detachTelemetryOf(decision))
		if err != nil {
			return false, fmt.Errorf("%w: %w", errRecordDetachment, err)
		}
		return detached, nil
	default:
		return false, fmt.Errorf(errFmtUnknownAttachmentDecision, decision.Status)
	}
}

func startsOverTheAirAttachment(decision bssci.AttachmentDecision) bool {
	return decision.OverTheAir != nil && decision.Status == bssci.EndpointStatusAttached
}

func detachTelemetryOf(decision bssci.AttachmentDecision) *endpoint.DetachTelemetry {
	if decision.OverTheAir == nil {
		return nil
	}
	return decision.OverTheAir.Telemetry
}

// noticeOf is the decision as the owner's subscribers are told it: an
// over-the-air decision carries the fields the base station reported and the
// station that heard it (SCACI §3.13.1), one taken in the service center only
// the endpoint and its status.
func noticeOf(decision bssci.AttachmentDecision) EndpointStatusNotice {
	notice := EndpointStatusNotice{
		TenantID: decision.TenantID,
		Status:   &bssci.EPStatusData{EpEui: decision.EpEUI, EpStatus: decision.Status},
	}
	if report := decision.OverTheAir; report != nil {
		heardBy := report.BaseStationEUI
		notice.HeardBy = &heardBy
		if report.Status != nil {
			reported := *report.Status
			reported.EpEui, reported.EpStatus = decision.EpEUI, decision.Status
			notice.Status = &reported
		}
	}
	return notice
}
