package bssci

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// negotiateSession decides whether the connect resumes the session its
// snBsUuid names or starts a new one (BSSCI §1, §5.3.1).
func (s *Server) negotiateSession(ctx context.Context, session *Session, data map[string]interface{}) *CatalogError {
	previous, rejection := s.findResumableSession(ctx, session, data)
	if rejection != nil {
		return rejection
	}
	if previous == nil {
		s.startNewSession(session)
		return nil
	}
	return s.offerResume(ctx, session, previous)
}

// findResumableSession returns the session the connect can resume, or nil.
// Resume identity is snBsUuid scoped by tenant and base station EUI (rev1
// §5.3.1: con carries snBsUuid, snBsOpId, snScOpId - no snScUuid).
func (s *Server) findResumableSession(ctx context.Context, session *Session, data map[string]interface{}) (*Session, *CatalogError) {
	snBsUUIDData, hasBsUUID := data["snBsUuid"]
	if !hasBsUUID {
		return nil, nil
	}
	bsUUID, catErr := extractSessionUUID(snBsUUIDData)
	if catErr != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToExtractSnBsUUID, logger.FieldError, catErr.Token)
		return nil, catErr
	}
	session.BsUUID = bsUUID

	bsOpID, scOpID := resumeCounterConstraints(data)
	outcome := s.sessionSvc.HandleResume(ctx, session, bsUUID, bsOpID, scOpID, session.BaseStationEUI)
	return s.settleResumeOutcome(ctx, session, outcome)
}

// resumeCounterConstraints reads the optional snBsOpId/snScOpId resume
// constraints; an absent one is not asserted (BSSCI §5.3.1).
func resumeCounterConstraints(data map[string]interface{}) (bsOpID, scOpID *int64) {
	if bsOp, ok := getNumericField(data, "snBsOpId"); ok {
		bsOpID = &bsOp
	}
	if scOp, ok := getNumericField(data, "snScOpId"); ok {
		scOpID = &scOp
	}
	return bsOpID, scOpID
}

// settleResumeOutcome maps the typed resume outcome to the session to resume:
// infrastructure failures and inconsistent state never silently degrade into
// a fresh session while the old resumable state lingers.
func (s *Server) settleResumeOutcome(ctx context.Context, session *Session, outcome ResumeOutcome) (*Session, *CatalogError) {
	switch outcome.Disposition {
	case ResumeInfrastructureFailure:
		// The old resumable state stays intact for a later successful resume
		s.logger.ErrorContext(ctx, LogBSSCIFailedToCheckSessionResume,
			logger.FieldError, outcome.Err,
			logger.FieldEui, session.BaseStationEUI)
		return nil, NewCatalogError(errSessionResumeUnavailable, POSIX_EAGAIN)
	case ResumeInconsistent:
		return nil, s.retireInconsistentOutcome(ctx, session, outcome)
	case ResumeCompatible:
		return outcome.Previous, nil
	default:
		return nil, nil
	}
}

// retireInconsistentOutcome terminates the session whose counters or version
// contradict the connect, so the connect starts a new session.
func (s *Server) retireInconsistentOutcome(ctx context.Context, session *Session, outcome ResumeOutcome) *CatalogError {
	s.logger.WarnContext(ctx, LogBSSCIFailedToCheckSessionResume,
		logger.FieldError, outcome.Err,
		logger.FieldEui, session.BaseStationEUI)
	if outcome.Previous == nil {
		return nil
	}
	if err := s.terminateInconsistentResume(ctx, outcome.Previous); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToTerminateSession,
			logger.FieldError, err,
			logger.FieldEui, session.BaseStationEUI)
		return NewCatalogError(errSessionResumeUnavailable, POSIX_EAGAIN)
	}
	return nil
}

// startNewSession gives the connect a new session identity and counters.
func (s *Server) startNewSession(session *Session) {
	sessionUUID := uuid.New()
	session.SessionUUID = sessionUUID[:]
	session.LastBsOpId = 0
	session.LastScOpId = 0

	s.logger.InfoContext(s.safeCtx(), LogBSSCIStartingNewSession,
		logger.FieldEui, session.BaseStationEUI)
}

// offerResume restores the authoritative persisted state of the previous
// session, not the constraint values the connect reported. Until conCmp
// claims the prior row it is read through the offer and never bound to this
// connection (DbSessionID stays 0).
func (s *Server) offerResume(ctx context.Context, session *Session, previous *Session) *CatalogError {
	session.IsResumed = true
	session.DisconnectedAt = previous.DisconnectedAt
	session.SessionUUID = previous.SessionUUID
	session.LastBsOpId = previous.LastBsOpId
	session.LastScOpId = previous.LastScOpId
	session.OrganizationID = previous.OrganizationID
	session.ResolvedTenantID = previous.ResolvedTenantID

	// Load failures reject the connect before conRsp instead of silently
	// losing protocol state; an operation that cannot be rebuilt is dropped
	// alone so one bad row never loops the station through failed resumes.
	pendingOps, loadErr := s.pendingOps.load(s.safeCtx(), previous)
	if loadErr != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToLoadPendingOperationsForSessionResume,
			logger.FieldError, loadErr,
			logger.FieldSessionID, previous.DbSessionID)
		return NewCatalogError(errSessionResumeUnavailable, POSIX_EAGAIN)
	}
	pendingOps = s.reconstituteResumeOperations(ctx, previous, pendingOps)
	session.resume = &resumeOffer{previous: previous, pendingOps: pendingOps}
	floorServiceCenterCounter(session, pendingOps)

	s.logger.InfoContext(s.safeCtx(), LogBSSCIResumingPreviousSession,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldBsOpID, session.LastBsOpId,
		logger.FieldScOpID, session.LastScOpId,
		logger.FieldDbSessionIDCamel, previous.DbSessionID,
		logger.FieldOrgID, session.OrganizationID.String(),
		logger.FieldTenantID, session.ResolvedTenantID)
	return nil
}

// floorServiceCenterCounter keeps the SC counter at or below every persisted
// negative operation ID, so reissued IDs are never re-allocated.
func floorServiceCenterCounter(session *Session, pendingOps []*PendingOperation) {
	for _, op := range pendingOps {
		if op.OperationID < 0 && op.OperationID < session.LastScOpId {
			session.LastScOpId = op.OperationID
		}
	}
}

// reopenResumedBaseStationOperations lets a resumed base station complete or
// reissue, with their original IDs, the operations it may still have open:
// every persisted base-station operation and its newest one, whose completion
// the connection loss may have cut off (rev1 §5.2 / classic §3.2).
func reopenResumedBaseStationOperations(session *Session, pendingOps []*PendingOperation) {
	lastBsOpID, _ := session.OperationCounters()
	openIDs := make([]int64, 0, len(pendingOps)+1)
	if lastBsOpID > 0 {
		openIDs = append(openIDs, lastBsOpID)
	}
	for _, op := range pendingOps {
		if op.OperationID > 0 {
			openIDs = append(openIDs, op.OperationID)
		}
	}
	session.restoreOpenBaseStationOperations(openIDs...)
}

// terminateInconsistentResume atomically retires a resumable session whose
// state the connecting base station does not share: it terminates the session
// (status=terminated, can_resume=false) and removes its pending operations so
// the base station starts genuinely fresh. The previous session is a
// DB-hydrated snapshot, never a live one.
func (s *Server) terminateInconsistentResume(ctx context.Context, previous *Session) error {
	if err := s.sessionSvc.TerminateSession(ctx, previous); err != nil {
		return fmt.Errorf(errFmtTerminateInconsistentResumeSession, err)
	}
	if previous.DbSessionID != 0 && s.statusSvc != nil {
		if _, err := s.statusSvc.DeletePendingOperations(ctx, previous); err != nil {
			return fmt.Errorf(errFmtRemoveInconsistentResumeOps, err)
		}
	}
	return nil
}

// retireRefusedResume retires the session a refused resuming conRsp offered:
// the station holds session state this service center cannot know, so its
// next connect must start a new session instead of the same refused resume
// (BSSCI §1).
func (s *Server) retireRefusedResume(ctx context.Context, session *Session, code int, message string) {
	offer := session.resume
	if offer == nil {
		return
	}
	session.resume = nil
	s.logger.WarnContext(ctx, LogBSSCIResumeRefusedByBaseStation,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldSessionID, offer.previous.DbSessionID,
		logger.FieldCode, code)
	if err := s.terminateInconsistentResume(ctx, offer.previous); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToTerminateSession,
			logger.FieldError, err,
			logger.FieldEui, session.BaseStationEUI)
		return
	}
	s.events.recordResumeRefused(ctx, resolvedTenant(offer.previous, s.tenantID), session, code, message)
}

// reconstituteResumeOperations semantically rebuilds every payload-bearing
// pending operation in place before the resume is offered. An operation that
// cannot be rebuilt (for example a record whose key material predates the
// current envelope format) degrades alone: it is dropped from the resume set,
// its persisted row is removed so it cannot re-fail the next resume, and a
// durable system event records the loss.
func (s *Server) reconstituteResumeOperations(ctx context.Context, session *Session, pendingOps []*PendingOperation) []*PendingOperation {
	kept := pendingOps[:0]
	for _, op := range pendingOps {
		spec, ok := s.commands.lookup(op.OperationType)
		if op.Metadata == nil || !ok || spec.Reconstitute == nil {
			kept = append(kept, op)
			continue
		}
		reconstituted, err := spec.Reconstitute(s, op.Message, op.Metadata, op)
		if err != nil {
			s.dropUnrecoverableOperation(ctx, session, op, err)
			continue
		}
		op.Message = reconstituted
		kept = append(kept, op)
	}
	return kept
}

// dropUnrecoverableOperation records and removes a pending operation that can
// no longer be rebuilt for resume.
func (s *Server) dropUnrecoverableOperation(ctx context.Context, session *Session, op *PendingOperation, cause error) {
	s.logger.ErrorContext(ctx, LogBSSCIFailedToReconstitutePendingOperation,
		logger.FieldError, cause,
		logger.FieldOpID, op.OperationID,
		logger.FieldType, op.OperationType)
	s.recordDroppedPendingOperation(ctx, session, op, cause)
	if remErr := s.statusSvc.RemovePendingOperation(ctx, session, op.OperationID); remErr != nil {
		s.logger.WarnContext(ctx, LogBSSCIFailedToClearPersistedPendingOperation,
			logger.FieldError, remErr,
			logger.FieldOpID, op.OperationID)
	}
}

// recordDroppedPendingOperation durably records a pending operation that was
// skipped during session resume because it could no longer be rebuilt.
func (s *Server) recordDroppedPendingOperation(ctx context.Context, session *Session, op *PendingOperation, cause error) {
	if s.eventStore == nil {
		return
	}
	details, err := json.Marshal(map[string]interface{}{
		models.EventDetailKeyOpID:      op.OperationID,
		models.EventDetailKeyOperation: op.OperationType,
		models.EventDetailKeyBsEui:     mioty.FormatEUI64(session.BaseStationEUI),
		models.EventDetailKeyError:     cause.Error(),
	})
	if err != nil {
		s.logger.WarnContext(ctx, LogBSSCIFailedToRecordDroppedPendingOperation, logger.FieldError, err)
		return
	}
	if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
		EventType:   models.EventTypeSessionPendingOpDropped,
		Category:    models.EventCategoryBSSCI,
		Severity:    SeverityWarning,
		Title:       fmt.Sprintf(models.EventTitlePendingOperationDropped, op.OperationType, op.OperationID),
		Description: eventDescPendingOpRebuildSkipped,
		Details:     details,
		Status:      EventStatusNew,
		CreatedAt:   s.clock.Now(),
	}); err != nil {
		s.logger.WarnContext(ctx, LogBSSCIFailedToRecordDroppedPendingOperation, logger.FieldError, err)
	}
}
