// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"context"
	"net"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// handlePing processes Ping messages per SCACI §3.4
//
// Ping Flow (Three-way handshake):
//  1. AC → SC: Ping (positive opId)
//  2. SC → AC: PingResponse (same opId)
//  3. AC → SC: PingComplete (same opId)
//
// Purpose: Keepalive mechanism to detect connection failures
func (s *Server) handlePing(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIProcessingPing, logger.FieldOpID, opId)

	session.UpdateLastSeen(s.clock.Now())

	s.persistHeartbeat(session)

	// Record AC-initiated Ping for audit trail (§3.4) if configured
	if s.config.LogPingOperations && session.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		requestData := map[string]interface{}{
			"opId":      opId,
			"initiator": initiatorAC,
		}
		if err := s.operationRecorder.Record(recCtx, session, opId, CmdPing, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingOpFailed, logger.FieldError, err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	// Send PingResponse
	resp := PingResponse{
		BaseMessage: BaseMessage{
			Command: CmdPingResponse,
			OpId:    opId,
		},
	}

	if err := s.SendPingResponse(conn, session, &resp); err != nil {
		return err
	}

	// Update PingResponse state for audit trail (§3.4) if configured
	// State transition: pending → acknowledged (row created above)
	if s.config.LogPingOperations && session.ID > 0 && s.operationRepo != nil {
		rspCtx, rspCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer rspCancel()

		responseData := map[string]interface{}{
			"opId":      opId,
			"initiator": initiatorAC,
		}
		if err := s.operationRepo.UpdateOperationState(rspCtx, session.ID, opId, models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingRspOpFailed, logger.FieldError, err)
		}
	}

	return nil
}

// handlePingResponse processes PingResponse from AC when SC initiated ping per SCACI §3.4.2
//
// Ping Flow (SC-initiated, three-way handshake):
//  1. SC → AC: Ping (negative opId)
//  2. AC → SC: PingResponse (same opId) ← handled here
//  3. SC → AC: PingComplete (same opId)
//
// Purpose: Completes SC-initiated keepalive handshake
func (s *Server) handlePingResponse(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIProcessingPingResponse, logger.FieldOpID, opId)

	session.UpdateLastSeen(s.clock.Now())

	s.persistHeartbeat(session)

	// Update SC-initiated PingResponse state for audit trail (§3.4) if configured
	// State transition: pending → acknowledged (row created in initiatePing)
	if s.config.LogPingOperations && session.ID > 0 && s.operationRepo != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		responseData := map[string]interface{}{
			"opId":      opId,
			"initiator": initiatorSC,
		}
		if err := s.operationRepo.UpdateOperationState(recCtx, session.ID, opId, models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingRspOpFailed, logger.FieldError, err)
		}
	}

	// Send PingComplete (mirror BSSCI pattern)
	session.WriteMu.Lock()
	cmp := PingComplete{
		BaseMessage: BaseMessage{
			Command: CmdPingComplete,
			OpId:    opId,
		},
	}
	err := s.sendResponse(conn, &cmp)
	session.WriteMu.Unlock()

	if err != nil {
		return err
	}

	// Update PingComplete state for audit trail (§3.4) if configured
	// State transition: acknowledged → completed (completes SC-initiated three-way handshake)
	if s.config.LogPingOperations && session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completeData := map[string]interface{}{
			"opId":      opId,
			"initiator": initiatorSC,
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId, models.OperationStateCompleted, completeData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingCmpOpFailed, logger.FieldError, err)
		}
	}

	return nil
}

// handlePingComplete processes PingComplete messages per SCACI §3.4.3
//
// This completes the three-way handshake for Ping operation.
// The AC acknowledges receipt of PingResponse.
//
// No response is sent per spec - handshake is complete.
func (s *Server) handlePingComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
		return nil
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIPingHandshakeComplete, logger.FieldOpID, opId)

	session.UpdateLastSeen(s.clock.Now())

	s.persistHeartbeat(session)

	// Update AC-initiated PingComplete state for audit trail (§3.4) if configured
	// State transition: acknowledged → completed (completes AC-initiated three-way handshake)
	if s.config.LogPingOperations && session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completeData := map[string]interface{}{
			"opId":      opId,
			"initiator": initiatorAC,
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId, models.OperationStateCompleted, completeData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingCmpOpFailed, logger.FieldError, err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	return nil
}

// initiatePing sends SC-initiated ping to detect connection failures per SCACI §3.4
//
// This method is called by the keepalive monitor when a session has been idle
// for longer than the ping interval. It initiates the three-way handshake:
//  1. SC → AC: Ping (negative opId)
//  2. AC → SC: PingResponse (same opId)
//  3. SC → AC: PingComplete (same opId)
//
// Parameters:
//   - conn: TLS connection to Application Center
//   - session: Active SCACI session
//
// Returns:
//   - error: Send error, or nil on success
func (s *Server) initiatePing(conn net.Conn, session *Session) error {
	err := s.initiateSCOperation(conn, session, scOperation{
		command: CmdPing,
		record:  s.auditPing,
		send: func(conn net.Conn, _ *Session, opId int64) error {
			return s.sendResponse(conn, &Ping{BaseMessage: BaseMessage{Command: CmdPing, OpId: opId}})
		},
	})
	if err != nil {
		return err
	}

	s.persistHeartbeat(session)
	return nil
}

// persistHeartbeat records keepalive activity (§3.4) as a detached write.
func (s *Server) persistHeartbeat(session *Session) {
	if s.sessionPersistence == nil || session.ID <= 0 {
		return
	}
	s.persistDetached(session, ConnectPersistTimeout, func(ctx context.Context) {
		if err := s.sessionPersistence.PersistHeartbeat(ctx, session); err != nil {
			s.logger.WarnContext(ctx, LogSCACIPersistHeartbeatFailed,
				logger.FieldSessionID, session.ID,
				logger.FieldTenantID, session.TenantID,
				logger.FieldError, err)
		}
	})
}

// auditPing records an SC-initiated ping for the audit trail when configured
// (§3.4). The audit is optional: a ping that cannot be audited is still sent,
// and a ping is never reissued (§1), so its record is not one to resume.
func (s *Server) auditPing(session *Session, opId int64) (Recording, error) {
	if !s.config.LogPingOperations || session.ID <= 0 {
		return NotRecorded, nil
	}
	recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer recCancel()

	requestData := map[string]interface{}{
		"opId":      opId,
		"initiator": initiatorSC,
	}
	if err := s.operationRecorder.Record(recCtx, session, opId, CmdPing, models.OperationDirectionOutbound, requestData); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingOpFailed, logger.FieldError, err)
	}
	return NotRecorded, nil
}
