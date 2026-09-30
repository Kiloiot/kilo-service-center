package bssciservices

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ApplicationCenterResults delivers a downlink result to queuer, the
// Application Center that queued the downlink (SCACI §3.12), under acQueID,
// the queue id it assigned (SCACI §3.12.1).
type ApplicationCenterResults interface {
	BroadcastDLDataResult(ctx context.Context, queuer scaci.ApplicationCenter, acQueID uint64, result *mioty.DLDataResult) error
}

// DownlinkResultPublisher publishes the result a downlink ended with on the
// MQTT topic of the organization that queued it, with ref, the correlation
// ref of the MQTT command that queued it (empty for none).
type DownlinkResultPublisher interface {
	PublishDownlinkResult(ctx context.Context, orgUUID, ref string, result *mioty.DLDataResult) error
}

// DownlinkResultEvents records a downlink result in the owner tenant's events.
type DownlinkResultEvents interface {
	RecordDLResult(ctx context.Context, tenant string, session *bssci.Session, result *mioty.DLDataResult) error
	RecordQueueExpiry(ctx context.Context, downlink *storage.DownlinkMessage) error
}

// BackgroundRunner runs work detached from the request that started it.
type BackgroundRunner interface {
	Go(ctx context.Context, fn func(context.Context))
}

// DownlinkResultReporter is the one place the result a downlink ended with
// reaches its originators: the Application Center that queued it, the MQTT
// downlink_result topic of its organization, and the owner tenant's events.
// Delivery to the Application Center and MQTT runs in the background, so a
// base station exchange never waits for either. The endpoint's later
// acknowledgement of a transmitted downlink is not a result it ended with:
// EndpointAckRecorder records it and the delivery outbox publishes it.
type DownlinkResultReporter struct {
	applicationCenters ApplicationCenterResults
	mqtt               DownlinkResultPublisher
	events             DownlinkResultEvents
	work               BackgroundRunner
	logger             logger.Logger
}

// NewDownlinkResultReporter builds the reporter; every collaborator is mandatory.
func NewDownlinkResultReporter(
	applicationCenters ApplicationCenterResults,
	mqtt DownlinkResultPublisher,
	events DownlinkResultEvents,
	work BackgroundRunner,
	log logger.Logger,
) (*DownlinkResultReporter, error) {
	switch {
	case applicationCenters == nil:
		return nil, ErrNilApplicationCenterResults
	case mqtt == nil:
		return nil, ErrNilDownlinkResultPublisher
	case events == nil:
		return nil, ErrNilDownlinkResultEvents
	case work == nil:
		return nil, ErrNilReporterRunner
	case log == nil:
		return nil, ErrNilReporterLogger
	}
	return &DownlinkResultReporter{
		applicationCenters: applicationCenters,
		mqtt:               mqtt,
		events:             events,
		work:               work,
		logger:             log,
	}, nil
}

// ReportStationResult reports the result a base station gave for the
// downlink (BSSCI §3.14, or invalid for a queue it refused, §3.17); bsEui
// names the station only for a sent result (SCACI §3.12.1).
func (r *DownlinkResultReporter) ReportStationResult(ctx context.Context, downlink *storage.DownlinkMessage,
	result mioty.DLDataResult, station *bssci.Session,
) {
	reported := bssci.ResultWithTransmitter(result, station.BaseStationEUI)
	r.deliver(ctx, downlink, reported)
	if err := r.events.RecordDLResult(ctx, downlink.TenantID, station, &reported); err != nil {
		r.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToRecordDLDataResultEvent, logger.FieldError, err)
	}
}

// ReportExpiredInQueue reports a downlink the service center expired before
// any base station transmitted it.
func (r *DownlinkResultReporter) ReportExpiredInQueue(ctx context.Context, downlink *storage.DownlinkMessage) error {
	epEUI, err := validation.ParseEUI(downlink.EPEUI)
	if err != nil {
		return fmt.Errorf("%w: %w", errUnidentifiedDownlink, err)
	}
	queID, ok := downlink.WireQueueID()
	if !ok {
		return fmt.Errorf(errFmtInvalidQueueID, downlink.QueID)
	}
	r.deliver(ctx, downlink, mioty.DLDataResult{EpEui: epEUI, QueId: queID, Result: mioty.ResultExpired})
	if err := r.events.RecordQueueExpiry(ctx, downlink); err != nil {
		r.logger.ErrorContext(ctx, bssci.LogBSSCIFailedToRecordDLDataResultEvent, logger.FieldError, err)
	}
	return nil
}

// deliver hands the result to the Application Center and to MQTT in the
// background, detached from the caller's cancellation.
func (r *DownlinkResultReporter) deliver(ctx context.Context, downlink *storage.DownlinkMessage, result mioty.DLDataResult) {
	tenantID, err := strconv.ParseInt(downlink.TenantID, 10, 64)
	if err != nil {
		r.logger.ErrorContext(ctx, LogDownlinkResultOwnerUnparsable,
			logger.FieldQueID, downlink.QueID, logger.FieldError, err)
		return
	}
	r.deliverToQueuer(ctx, context.WithoutCancel(ctx), tenantID, downlink, result)
	r.publishToOrganization(ctx, downlink, result)
}

// publishToOrganization publishes the result in the background, detached
// from the caller's cancellation, for the organization that queued the
// downlink; a downlink whose organization is unknown is published to none.
func (r *DownlinkResultReporter) publishToOrganization(ctx context.Context, downlink *storage.DownlinkMessage, result mioty.DLDataResult) {
	if downlink.OrganizationID == nil {
		r.logger.WarnContext(ctx, bssci.LogBSSCIMQTTPublishSkippedOrgUnresolved,
			logger.FieldQueID, downlink.QueID, logger.FieldEvent, bssci.MQTTEventKeyDownlinkResult)
		return
	}
	orgUUID := downlink.OrganizationID.String()
	r.work.Go(context.WithoutCancel(ctx), func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, downlinkResultDeliveryTimeout)
		defer cancel()
		if err := r.mqtt.PublishDownlinkResult(ctx, orgUUID, downlink.Ref, &result); err != nil {
			r.logger.WarnContext(ctx, bssci.LogBSSCIFailedToPublishDLResultToMQTT,
				logger.FieldQueID, downlink.QueID, logger.FieldError, err)
		}
	})
}

// DownlinkResultsWithoutMQTT is the result publisher of a process that runs
// without MQTT: results reach the Application Centers and the events only.
type DownlinkResultsWithoutMQTT struct{}

// PublishDownlinkResult publishes nothing.
func (DownlinkResultsWithoutMQTT) PublishDownlinkResult(context.Context, string, string, *mioty.DLDataResult) error {
	return nil
}

// deliverToQueuer reports the result to the Application Center that queued
// the downlink, of the downlink's tenant and organization, under the queue id
// it assigned, in the background. A downlink queued without an Application
// Center queue id (gRPC, MQTT) has no Application Center to tell, and one
// whose row names none (queued before the row stored it, and not resolved on
// upgrade) is told to none rather than to a guessed one.
func (r *DownlinkResultReporter) deliverToQueuer(ctx, detached context.Context, tenantID int64, downlink *storage.DownlinkMessage, result mioty.DLDataResult) {
	switch {
	case downlink.ACQueID == nil:
		r.logger.DebugContext(ctx, scaci.LogSCACIDLResultNotQueuedByApplicationCenter,
			logger.FieldQueID, result.QueId, logger.FieldTenantIDCamel, tenantID)
		return
	case downlink.ACEUI == nil:
		r.logger.WarnContext(ctx, scaci.LogSCACIDLResultQueuerUnknown,
			logger.FieldQueID, result.QueId, logger.FieldTenantIDCamel, tenantID)
		return
	}
	queuer := scaci.ApplicationCenterOf(tenantID, downlink.OrganizationID, *downlink.ACEUI)
	applicationQueueID := *downlink.ACQueID
	r.work.Go(detached, func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, downlinkResultDeliveryTimeout)
		defer cancel()
		if err := r.applicationCenters.BroadcastDLDataResult(ctx, queuer, applicationQueueID, &result); err != nil {
			r.logger.WarnContext(ctx, bssci.LogBSSCIFailedToBroadcastDLResultToSCACI,
				logger.FieldQueID, result.QueId, logger.FieldError, err)
		}
	})
}
