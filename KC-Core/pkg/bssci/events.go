package bssci

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// protocolEventRecorder writes the failure events the protocol handshakes
// raise against the owning tenant.
type protocolEventRecorder struct {
	store EventStore
	clock clock.Clock
	log   logger.Logger
}

func newProtocolEventRecorder(store EventStore, clk clock.Clock, log logger.Logger) *protocolEventRecorder {
	return &protocolEventRecorder{store: store, clock: clk, log: log}
}

// propagateFailure identifies one propagate a base station rejected.
type propagateFailure struct {
	epEUI, bsEUI uint64
	opID         int64
	result       int
}

// recordPropagateFailure emits the endpoint and base-station failure events
// of a rejected propagate, both carrying the same details.
func (r *protocolEventRecorder) recordPropagateFailure(ownerCtx context.Context, ownerTenantID int64, cfg propagateResponseConfig, f propagateFailure) {
	details, err := json.Marshal(map[string]interface{}{
		models.EventDetailKeyEpEui:       mioty.FormatEUI64(f.epEUI),
		models.EventDetailKeyBsEui:       mioty.FormatEUI64(f.bsEUI),
		models.EventDetailKeyOperation:   cfg.operationType,
		models.EventDetailKeyOperationID: fmt.Sprintf("%d", f.opID),
		models.EventDetailKeyFailureCode: f.result,
		models.EventDetailKeyReason:      fmt.Sprintf(PropagateFailureReasonFormat, f.result),
	})
	if err != nil {
		r.log.WarnContext(ownerCtx, LogBSSCIFailedToCreateFailureEvent, logger.FieldError, err)
		return
	}
	r.recordEndpointFailure(ownerCtx, ownerTenantID, cfg, f.epEUI, f.bsEUI, f.result, details)
	r.recordBaseStationFailure(ownerCtx, ownerTenantID, cfg, f.epEUI, f.bsEUI, f.result, details)
}

// recordEndpointFailure emits the endpoint-side SystemEvent for a propagate
// failure with source fields populated for UI filtering.
func (r *protocolEventRecorder) recordEndpointFailure(
	ownerCtx context.Context,
	ownerTenantID int64,
	cfg propagateResponseConfig,
	epEUI, bsEUI uint64,
	result int,
	details json.RawMessage,
) {
	epEUIStr := mioty.FormatEUI64(epEUI)
	bsEUIStr := mioty.FormatEUI64(bsEUI)
	if err := r.store.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", ownerTenantID),
		EventType:   cfg.eventType,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityError,
		Title:       fmt.Sprintf(cfg.titleFormat, epEUIStr, bsEUIStr),
		Description: fmt.Sprintf(PropagateFailureDescriptionFormat, result),
		Details:     details,
		Status:      EventStatusNew,
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  epEUIStr,
		CreatedAt:   r.clock.Now(),
	}); err != nil {
		r.log.WarnContext(ownerCtx, LogBSSCIFailedToCreateFailureEvent, logger.FieldError, err)
	}
}

// recordBaseStationFailure emits the symmetric base-station-side
// SystemEvent for a propagate failure (BSSCI §5.8.3 / §3.9 roaming).
func (r *protocolEventRecorder) recordBaseStationFailure(
	ownerCtx context.Context,
	ownerTenantID int64,
	cfg propagateResponseConfig,
	epEUI, bsEUI uint64,
	result int,
	details json.RawMessage,
) {
	epEUIStr := mioty.FormatEUI64(epEUI)
	bsEUIStr := mioty.FormatEUI64(bsEUI)
	if err := r.store.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", ownerTenantID),
		EventType:   cfg.eventType,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityError,
		Title:       fmt.Sprintf(TitleRejectedOperationForEndpoint, cfg.operationType, epEUIStr),
		Description: fmt.Sprintf(PropagateFailureDescriptionFormat, result),
		Details:     details,
		Status:      EventStatusNew,
		SourceType:  mioty.SourceTypeBaseStation,
		SourceName:  bsEUIStr,
		CreatedAt:   r.clock.Now(),
	}); err != nil {
		r.log.WarnContext(ownerCtx, LogBSSCIFailedToCreateBaseStationFailureEvent, logger.FieldError, err)
	}
}

// recordResumeRefused records, under the station's tenant, that a base
// station refused the session resume the service center offered on it.
func (r *protocolEventRecorder) recordResumeRefused(ctx context.Context, ownerTenantID int64, session *Session, code int, message string) {
	bsEUIStr := mioty.FormatEUI64(session.BaseStationEUI)
	details, err := json.Marshal(map[string]interface{}{
		models.EventDetailKeyBsEui:       bsEUIStr,
		models.EventDetailKeyFailureCode: code,
		models.EventDetailKeyMessage:     message,
	})
	if err != nil {
		r.log.WarnContext(ctx, LogBSSCIFailedToRecordRefusedResume, logger.FieldError, err)
		return
	}
	if err := r.store.CreateEvent(ctx, &models.SystemEvent{
		TenantID:      fmt.Sprintf("%d", ownerTenantID),
		EventType:     models.EventTypeSessionResumeRefused,
		Category:      models.EventCategoryBSSCI,
		Severity:      SeverityWarning,
		Title:         fmt.Sprintf(models.EventTitleSessionResumeRefused, bsEUIStr),
		Description:   eventDescResumeRefused,
		Details:       details,
		Status:        EventStatusNew,
		SourceType:    mioty.SourceTypeBaseStation,
		SourceName:    bsEUIStr,
		BasestationID: registeredStationID(session),
		CreatedAt:     r.clock.Now(),
	}); err != nil {
		r.log.WarnContext(ctx, LogBSSCIFailedToRecordRefusedResume, logger.FieldError, err)
	}
}

// registeredStationID is the registration the connect was admitted under,
// for base-station scoped event feeds.
func registeredStationID(session *Session) *int64 {
	if session.pendingBaseStation == nil {
		return nil
	}
	return &session.pendingBaseStation.ID
}
