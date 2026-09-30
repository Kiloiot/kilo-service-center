package bssci

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// recordNonBidiAttachFailure records that a bidirectional endpoint cannot be
// propagated to a base station without downlink capability.
func (s *Server) recordNonBidiAttachFailure(ctx context.Context, session *Session, endpointEUI uint64) {
	details, err := json.Marshal(map[string]interface{}{
		models.EventDetailKeyEpEui:  mioty.FormatEUI64(endpointEUI),
		models.EventDetailKeyBsEui:  mioty.FormatEUI64(session.BaseStationEUI),
		models.EventDetailKeyReason: eventReasonBSNotBidirectional,
	})
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateNonBidiAttachFailureEvent, logger.FieldError, err)
		return
	}
	if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
		EventType:   EventTypeAttachOperationFailed,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityError,
		Title:       fmt.Sprintf(titleAttachPropagateFailedFormat, mioty.FormatEUI64(endpointEUI)),
		Description: eventDescTargetBSNotBidirectional,
		Details:     details,
		Status:      EventStatusNew,
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  mioty.FormatEUI64(endpointEUI),
		CreatedAt:   s.clock.Now(),
	}); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateNonBidiAttachFailureEvent, logger.FieldError, err)
	}
}

// recordAttachPropagated records, for the endpoint owner, that the base
// station confirmed the endpoint's propagated keys.
func (s *Server) recordAttachPropagated(ownerCtx context.Context, ownerTenantID int64, session *Session, epEUI uint64, shortAddr uint16) {
	epEUIHex := mioty.FormatEUI64(epEUI)
	bsEUIHex := mioty.FormatEUI64(session.BaseStationEUI)
	details, err := json.Marshal(map[string]interface{}{
		models.EventDetailKeyEpEui:   epEUIHex,
		models.EventDetailKeyBsEui:   bsEUIHex,
		models.EventDetailKeyShAddr:  shortAddr,
		models.EventDetailKeySuccess: true,
		models.EventDetailKeyTime:    s.clock.Now().UnixNano(),
	})
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateAttachmentEvent, logger.FieldError, err)
		return
	}
	if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    strconv.FormatInt(ownerTenantID, 10),
		EventType:   mioty.CmdAttachPropagate,
		Category:    models.EventCategoryEndpoint,
		Severity:    models.EventSeverityInfo,
		SourceType:  models.SourceTypeServiceCenter,
		SourceName:  epEUIHex,
		Title:       fmt.Sprintf(eventTitleFmtAttachPropagate, epEUIHex, bsEUIHex),
		Description: fmt.Sprintf(eventDescFmtKeysPropagated, epEUIHex, bsEUIHex, shortAddr),
		Details:     details,
	}); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateAttachmentEvent, logger.FieldError, err)
	}
}

// recordDetachPropagated records, for the endpoint owner, that the base
// station confirmed the endpoint's detachment.
func (s *Server) recordDetachPropagated(ownerCtx context.Context, ownerTenantID int64, session *Session, epEUI uint64, opID int64) {
	details, err := json.Marshal(map[string]interface{}{
		models.EventDetailKeyEpEui:       mioty.FormatEUI64(epEUI),
		models.EventDetailKeyBsEui:       mioty.FormatEUI64(session.BaseStationEUI),
		models.EventDetailKeyOperation:   "detach_propagate_complete",
		models.EventDetailKeyOperationID: fmt.Sprintf("%d", opID),
	})
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateDetachmentEvent, logger.FieldError, err)
		return
	}
	if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", ownerTenantID),
		EventType:   models.EventTypeDetachPropagateCompleted,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleEndpointDetachedFromBS, mioty.FormatEUI64(epEUI), mioty.FormatEUI64(session.BaseStationEUI)),
		Description: eventDescDetachPropagateDone,
		Details:     details,
		Status:      EventStatusNew,
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  mioty.FormatEUI64(epEUI),
		CreatedAt:   s.clock.Now(),
	}); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateDetachmentEvent, logger.FieldError, err)
	}
}
