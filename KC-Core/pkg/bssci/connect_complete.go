package bssci

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handleConnectComplete handles the connect complete operation. The connect
// operation is already complete at the protocol level when conCmp arrives, so
// failures past this point never send a BSSCI error - the connection closes,
// the teardown hands a row it owns back resumable, and the base station
// reconnects.
func (s *Server) handleConnectComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	// conCmp is only valid after conRsp was sent
	if session.ConnectState != ConnectStateAwaitingConnectComplete {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIRejectingCommandBeforeHandshake,
			logger.FieldCommand, msg.Command,
			logger.FieldConnectState, int(session.ConnectState),
			logger.FieldOpID, msg.OpId)
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidHandshakeState)
	}

	// Validate that connect complete has opId == 0 (BSSCI-3.3)
	if msg.OpId != 0 {
		return s.rejectConnect(session, msg.OpId, POSIX_EPROTO, errInvalidConnectCompleteOpId)
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCIBaseStationConnectionCompletedSuccessfully,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldName, session.Name,
		logger.FieldSessionID, session.ID)

	ctx := s.sessionContext(session)

	// Registration and tenant authorization already ran before conRsp
	baseStation := session.pendingBaseStation
	if baseStation == nil {
		session.ConnectState = ConnectStateTerminal
		return fmt.Errorf(errFmtTokenWithEUI, ResolveErrorMessage(errBaseStationNotRegistered), mioty.FormatEUI64(session.BaseStationEUI))
	}

	pendingOps := session.resume.operations()
	session.resume = nil
	if err := s.activateSession(ctx, session, baseStation, pendingOps); err != nil {
		session.ConnectState = ConnectStateTerminal
		return err
	}
	s.recordStationCapabilities(ctx, session, baseStation)
	s.completeHandshake(ctx, session)

	if session.IsResumed {
		reopenResumedBaseStationOperations(session, pendingOps)
	}
	if err := s.reissueResumedOperations(ctx, session, pendingOps); err != nil {
		return err
	}
	s.startStationServices(ctx, session)
	return nil
}

// startStationServices starts status polling only after the reissue sequence
// completed, so the first status request cannot interleave with it, then
// settles the station's downlinks without blocking the complete message.
func (s *Server) startStationServices(ctx context.Context, session *Session) {
	s.startStatusMechanism(session)
	s.logger.InfoContext(ctx, LogBSSCIStartedStatusMechanismForBaseStation,
		logger.FieldBsEui, session.BaseStationEUI)

	served := s.servedDownlinks(ctx, session)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.settleConnectedStation(ctx, session, served)
	}()
}

// activateSession persists the session and registers the live connection and
// base-station online status BEFORE publishing to the in-memory registries. A
// failure publishes nothing; the base station has completed the connect, so
// the row this connection owns is handed back resumable by the teardown.
func (s *Server) activateSession(ctx context.Context, session *Session, baseStation *basestation.BaseStation, pendingOps []*PendingOperation) error {
	// Persisting terminates a stale active session first, aborting on failure
	// rather than leaving two active sessions
	if err := s.sessionSvc.PersistSession(ctx, session, baseStation, session.IsResumed, session.ConnectInfo); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToPersistSession,
			logger.FieldError, err,
			logger.FieldEui, session.BaseStationEUI)
		return fmt.Errorf(errFmtSessionPersistenceFailedAfterConCmp, err)
	}

	// The prior session's state is settled: reservations only the reissued
	// dlDataQue operations can confirm stay reserved, before this session is
	// published and can reserve new ones.
	s.reclaimStationReservations(ctx, session, pendingOps)

	if err := s.connectionRegistry.RegisterConnection(ctx, session, baseStation); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateConnectionStatus,
			logger.FieldEui, session.BaseStationEUI,
			logger.FieldError, err)
		return fmt.Errorf(errFmtConnectionRegistrationFailedAfterConCmp, err)
	}

	s.publishLiveSession(ctx, session)
	s.sessionSvc.StoreSessionByUUID(session)
	return nil
}

// recordStationCapabilities stores the station's bidi capability and GPS
// location (non-critical enrichment; a failure is logged, not fatal).
func (s *Server) recordStationCapabilities(ctx context.Context, session *Session, baseStation *basestation.BaseStation) {
	if s.basestationRepo == nil {
		return
	}
	updates := map[string]interface{}{
		"bidi": session.Bidirectional,
	}
	// Persist GPS coordinates from connect handshake (BSSCI §3.3.1)
	if len(session.GeoLocation) == 3 {
		updates["latitude"] = session.GeoLocation[0]
		updates["longitude"] = session.GeoLocation[1]
		updates["altitude"] = session.GeoLocation[2]
		updates["location_source"] = models.LocationSourceGPS
		updates["location_updated_at"] = s.clock.Now()
	}
	if err := s.basestationRepo.Update(ctx, resolvedTenant(session, s.tenantID), baseStation.ID, updates); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateBaseStationBidiCapability,
			logger.FieldEui, session.BaseStationEUI,
			logger.FieldBidi, session.Bidirectional,
			logger.FieldError, err)
		return
	}
	s.logger.InfoContext(ctx, LogBSSCIUpdatedBaseStationBidiCapability,
		logger.FieldEui, session.BaseStationEUI,
		logger.FieldBidi, session.Bidirectional)
}

// completeHandshake marks the connect operation complete (BSSCI-3.3-03).
func (s *Server) completeHandshake(ctx context.Context, session *Session) {
	s.sessionSvc.MarkHandshakeComplete(session)
	session.ConnectState = ConnectStateComplete

	// Active sessions read without a deadline; liveness is the ping
	// operation's job (BSSCI §5.4)
	if session.Conn == nil {
		return
	}
	if err := session.Conn.SetReadDeadline(time.Time{}); err != nil {
		s.logger.WarnContext(ctx, LogBSSCIFailedToSetReadDeadline, logger.FieldError, err)
	}
}

// reissueResumedOperations restores the validated snapshot loaded before
// conRsp (never re-read or re-written here - the DB rows are already
// authoritative) and reissues the eligible operations in deterministic order
// with their original IDs.
func (s *Server) reissueResumedOperations(ctx context.Context, session *Session, pendingOps []*PendingOperation) error {
	if len(pendingOps) == 0 {
		if session.DbSessionID > 0 {
			s.logger.DebugContext(ctx, LogBSSCINoPendingOperationsToResume,
				logger.FieldBsEui, session.BaseStationEUI,
				logger.FieldSessionID, session.DbSessionID)
		}
		return nil
	}
	s.logger.InfoContext(ctx, LogBSSCIResumingSessionWithPendingOps,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldSessionID, session.DbSessionID,
		logger.FieldPendingCount, len(pendingOps))
	for _, op := range pendingOps {
		if err := s.reissueResumedOperation(ctx, session, op); err != nil {
			return err
		}
	}
	return nil
}

// reissueResumedOperation hydrates one operation into the cache and resends
// it when the service center initiated it; base-station (positive) operations
// are hydrated for response correlation but never transmitted.
func (s *Server) reissueResumedOperation(ctx context.Context, session *Session, op *PendingOperation) error {
	s.logger.DebugContext(ctx, LogBSSCIProcessingPendingOperationForResume,
		logger.FieldOpID, op.OperationID,
		logger.FieldType, op.OperationType)
	if s.statusSvc != nil {
		s.statusSvc.RestorePendingOperation(session, op.OperationID, op)
	}
	if !s.commands.resumable(op.OperationID, op.OperationType) {
		return nil
	}

	// Normalize message types to fix JSON float64 conversion before reissuing
	err := s.sendMessage(session, s.commands.normalize(op.Message, op.OperationType))
	if err == nil {
		return nil
	}
	// A broken connection aborts activation so status polling never starts on
	// a dead transport; the rows stay persisted for the next resume
	s.logger.ErrorContext(ctx, LogBSSCIFailedToReissuePendingOperation,
		logger.FieldError, err,
		logger.FieldOpID, op.OperationID,
		logger.FieldType, op.OperationType)
	if errors.Is(err, ErrAmbiguousWrite) {
		s.closeTransportAfterWriteFailure(session, op.OperationID, err)
	}
	return fmt.Errorf(errFmtReissuePendingOperationAfterResume, op.OperationID, err)
}
