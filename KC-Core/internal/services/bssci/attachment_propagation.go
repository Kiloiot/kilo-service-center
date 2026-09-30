package bssciservices

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// AttachPropagateSender sends an attachment decision to the connected base
// stations.
type AttachPropagateSender interface {
	SendAttachPropagateToAll(endpointEUI uint64, nwkSnKey []byte, shortAddr uint16,
		bidirectional bool, lastPacketCnt uint32, dualChannel bool,
		repetition uint8, wideCarrOff bool, longBlkDist bool) []error
	SendDetachPropagateToAll(endpointEUI uint64) []error
	// GetConnectedSessionEUIs names the connected base stations for the operation events.
	GetConnectedSessionEUIs() []string
}

// AttachmentPropagation sends an attach or detach decided in the service
// center to the connected base stations in the background and records the
// operation's start and failure events.
type AttachmentPropagation struct {
	stations    AttachPropagateSender
	sessionKeys bssci.NetworkSessionKeySource
	events      SystemEventRecorder
	runner      BackgroundRunner
	clock       clock.Clock
	logger      logger.Logger
}

// NewAttachmentPropagation creates the propagation of attachment decisions.
func NewAttachmentPropagation(
	stations AttachPropagateSender,
	sessionKeys bssci.NetworkSessionKeySource,
	events SystemEventRecorder,
	runner BackgroundRunner,
	clk clock.Clock,
	log logger.Logger,
) (*AttachmentPropagation, error) {
	switch {
	case stations == nil:
		return nil, errNilAttachPropagateSender
	case sessionKeys == nil:
		return nil, errNilAttachmentSessionKeys
	case events == nil:
		return nil, errNilAttachmentEvents
	case runner == nil:
		return nil, errNilBackgroundRunner
	case clk == nil:
		return nil, errNilAttachmentClock
	case log == nil:
		return nil, errNilAttachmentLogger
	}
	return &AttachmentPropagation{stations: stations, sessionKeys: sessionKeys, events: events, runner: runner, clock: clk, logger: log}, nil
}

// PropagateAttach sends the endpoint's attPrp to every connected base
// station in the background.
func (p *AttachmentPropagation) PropagateAttach(ctx context.Context, tenantID int64, ep *models.EndPoint) *grpcservices.EndpointOperationResult {
	op := p.start(ctx, bssci.OperationIDPrefixAttach, bssci.OperationTypeAttach, tenantID, ep)
	p.runInBackground(ctx, func(ctx context.Context) {
		if err := p.sendAttach(ctx, ep); err != nil {
			p.recordFailure(ctx, op, err, bssci.EventTypeAttachPropagateFailed)
		}
	})
	return op.result()
}

// PropagateDetach sends the endpoint's detPrp to every connected base
// station in the background.
func (p *AttachmentPropagation) PropagateDetach(ctx context.Context, tenantID int64, ep *models.EndPoint) *grpcservices.EndpointOperationResult {
	op := p.start(ctx, bssci.OperationIDPrefixDetach, bssci.OperationTypeDetach, tenantID, ep)
	p.runInBackground(ctx, func(ctx context.Context) {
		if errs := p.stations.SendDetachPropagateToAll(op.epEUI); len(errs) > 0 {
			p.recordFailure(ctx, op, errs[0], bssci.EventTypeDetachPropagateFailed)
		}
	})
	return op.result()
}

// runInBackground outlives the request that started it; its cancellation
// must not drop the failure event.
func (p *AttachmentPropagation) runInBackground(ctx context.Context, fn func(context.Context)) {
	p.runner.Go(context.WithoutCancel(ctx), func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, propagateAsyncTimeout)
		defer cancel()
		fn(ctx)
	})
}

func (p *AttachmentPropagation) sendAttach(ctx context.Context, ep *models.EndPoint) error {
	nwkSnKey, err := p.sessionKeys.NetworkSessionKey(ctx, ep)
	if err != nil {
		return err
	}
	if len(nwkSnKey) != endpointKeyLength {
		p.logger.ErrorContext(ctx, LogInvalidNetworkKeyBypassedValidation,
			logger.FieldLength, len(nwkSnKey),
			logger.FieldEpEui, mioty.FormatEUI64(ep.EUI.ToUint64()))
		return fmt.Errorf(errFmtNetworkKeyLength, len(nwkSnKey))
	}
	var shortAddr uint16
	if ep.ShAddr != nil {
		shortAddr = *ep.ShAddr
	}
	var repetition uint8
	if ep.Repetition {
		repetition = 1
	}
	errs := p.stations.SendAttachPropagateToAll(ep.EUI.ToUint64(), nwkSnKey, shortAddr, ep.Bidi,
		ep.LastPacketCnt, ep.DualChan, repetition, ep.WideCarrOff, ep.LongBlkDist)
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// propagationOperation is one propagation as its events describe it.
type propagationOperation struct {
	id         string
	tenantID   int64
	endpointID int64
	epEUI      uint64
	stations   []string
}

func (o propagationOperation) result() *grpcservices.EndpointOperationResult {
	return &grpcservices.EndpointOperationResult{OperationID: o.id, Status: bssci.OperationStatusInitiated}
}

func (p *AttachmentPropagation) start(ctx context.Context, idPrefix, operationType string, tenantID int64, ep *models.EndPoint) propagationOperation {
	op := propagationOperation{
		id:         fmt.Sprintf(bssci.OperationIDFormat, idPrefix, ep.ID, p.clock.Now().UnixNano()),
		tenantID:   tenantID,
		endpointID: ep.ID,
		epEUI:      ep.EUI.ToUint64(),
		stations:   p.stations.GetConnectedSessionEUIs(),
	}
	p.recordStart(ctx, op, operationType)
	return op
}

func (p *AttachmentPropagation) eventDetails(op propagationOperation) map[string]interface{} {
	details := map[string]interface{}{
		bssci.EventKeyOperationID:    op.id,
		bssci.EventKeyEndpointID:     op.endpointID,
		bssci.EventKeyEpEui:          mioty.FormatEUI64(op.epEUI),
		bssci.EventKeyTargetBS:       describeTargetBaseStations(op.stations),
		bssci.EventKeyTargetBSList:   op.stations,
		bssci.EventKeyTargetBSCount:  len(op.stations),
		bssci.EventKeyBsEui:          bssci.BaseStationNone,
		bssci.EventKeyBaseStationEui: bssci.BaseStationNone,
	}
	if len(op.stations) > 0 {
		details[bssci.EventKeyBsEui] = op.stations[0]
		details[bssci.EventKeyBaseStationEui] = op.stations[0]
	}
	return details
}

func (p *AttachmentPropagation) recordFailure(ctx context.Context, op propagationOperation, cause error, eventType string) {
	details := p.eventDetails(op)
	details[bssci.EventKeyError] = cause.Error()
	if err := p.storeEvent(ctx, op, eventType, models.EventSeverityError, bssci.TitleOperationFailed,
		fmt.Sprintf(bssci.DescriptionOperationFailedFormat, describeTargetBaseStations(op.stations), cause), details); err != nil {
		p.logger.ErrorContext(ctx, LogFailedToLogFailureEvent, logger.FieldError, err)
	}
}

func (p *AttachmentPropagation) recordStart(ctx context.Context, op propagationOperation, operationType string) {
	details := p.eventDetails(op)
	details[bssci.EventKeyOperationType] = operationType
	eventType, title, description := bssci.EventTypeAttachPropagateInitiated, bssci.TitleAttachPropagateInitiated, bssci.DescriptionAttachPropagateEndpoint
	if operationType == bssci.OperationTypeDetach {
		eventType, title, description = bssci.EventTypeDetachPropagateInitiated, bssci.TitleDetachPropagateInitiated, bssci.DescriptionDetachPropagateEndpoint
	}
	if err := p.storeEvent(ctx, op, eventType, models.EventSeverityInfo, title, description, details); err != nil {
		p.logger.WarnContext(ctx, LogFailedToLogStartEvent, logger.FieldError, err)
	}
}

// storeEvent records one propagation event for the endpoint's tenant.
func (p *AttachmentPropagation) storeEvent(ctx context.Context, op propagationOperation, eventType, severity, title, description string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf(errFmtEncodeEventDetails, err)
	}
	now := p.clock.Now()
	endpointID := op.endpointID
	return p.events.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", op.tenantID),
		EventType:   eventType,
		Category:    bssci.EventCategoryEndpoint,
		Severity:    severity,
		Title:       title,
		Description: description,
		Details:     detailsJSON,
		Status:      bssci.EventStatusNew,
		SourceType:  bssci.EventSourceTypeEndpoint,
		SourceName:  mioty.FormatEUI64(op.epEUI),
		EndpointID:  &endpointID,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

// describeTargetBaseStations generates human-readable description
func describeTargetBaseStations(list []string) string {
	switch len(list) {
	case 0:
		return descNoBaseStations
	case 1:
		return fmt.Sprintf(descFmtSingleBaseStation, list[0])
	default:
		return fmt.Sprintf(descFmtBaseStationCount, len(list))
	}
}
