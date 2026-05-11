// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"context"
	"net"

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
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIProcessingPing, "opId", opId)

	session.UpdateLastSeen()

	// Persist heartbeat to database (async, best-effort per §3.4)
	if s.sessionPersistence != nil && session != nil && session.ID > 0 {
		s.sessionPersistence.PersistHeartbeatAsync(s.sessionContext(session), session)
	}

	// Record AC-initiated Ping for audit trail (§3.4) if configured
	if s.config.LogPingOperations && session.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		requestData := map[string]interface{}{
			"opId":      opId,
			"initiator": "ac",
		}
		if err := s.operationRecorder.Record(recCtx, session, opId, CmdPing, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingOpFailed, "error", err)
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
			"initiator": "ac",
		}
		if err := s.operationRepo.UpdateOperationState(rspCtx, session.ID, opId, models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingRspOpFailed, "error", err)
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
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIProcessingPingResponse, "opId", opId)

	session.UpdateLastSeen()

	// Persist heartbeat to database (async, best-effort per §3.4)
	if s.sessionPersistence != nil && session != nil && session.ID > 0 {
		s.sessionPersistence.PersistHeartbeatAsync(s.sessionContext(session), session)
	}

	// Update SC-initiated PingResponse state for audit trail (§3.4) if configured
	// State transition: pending → acknowledged (row created in initiatePing)
	if s.config.LogPingOperations && session.ID > 0 && s.operationRepo != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		responseData := map[string]interface{}{
			"opId":      opId,
			"initiator": "sc",
		}
		if err := s.operationRepo.UpdateOperationState(recCtx, session.ID, opId, models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingRspOpFailed, "error", err)
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
			"initiator": "sc",
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId, models.OperationStateCompleted, completeData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingCmpOpFailed, "error", err)
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
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIPingHandshakeComplete, "opId", opId)

	session.UpdateLastSeen()

	// Persist heartbeat to database (async, best-effort per §3.4)
	if s.sessionPersistence != nil && session != nil && session.ID > 0 {
		s.sessionPersistence.PersistHeartbeatAsync(s.sessionContext(session), session)
	}

	// Update AC-initiated PingComplete state for audit trail (§3.4) if configured
	// State transition: acknowledged → completed (completes AC-initiated three-way handshake)
	if s.config.LogPingOperations && session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completeData := map[string]interface{}{
			"opId":      opId,
			"initiator": "ac",
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId, models.OperationStateCompleted, completeData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordPingCmpOpFailed, "error", err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	return nil
}
