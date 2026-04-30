// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import (
	"context"
	"net"
	"time"

	dbconfig "github.com/kilocenter/KC-DB/common/config"
	"github.com/kilocenter/KC-DB/storage/models"
)

// handleStatus processes Status messages per SCACI §3.5
//
// Status Flow (Three-way handshake):
//  1. AC → SC: Status (positive opId)
//  2. SC → AC: StatusResponse (same opId, with SC status fields + base station array)
//  3. AC → SC: StatusComplete (same opId)
//
// Purpose: Query Service Center operational status and base station telemetry
func (s *Server) handleStatus(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIProcessingStatus, "opId", opId)

	session.UpdateLastSeen()

	// Record AC-initiated Status for audit trail (§3.5) if configured
	if s.config.LogStatusOperations && session.ID > 0 && s.operationRecorder != nil {
		recCtx, recCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer recCancel()

		requestData := map[string]interface{}{
			"opId":      opId,
			"initiator": "ac",
		}
		if err := s.operationRecorder.Record(recCtx, session, opId, CmdStatus, models.OperationDirectionInbound, requestData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordStatusOpFailed, "error", err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	// Calculate SC time and uptime (Unix UTC nanoseconds)
	now := time.Now().UTC()
	scTime := now.UnixNano()

	// Calculate Service Center uptime in seconds (delegated to StatusService)
	uptimeSec := s.statusSvc.GetUptime()

	// Query base stations for this tenant (delegated to StatusService)
	ctx, cancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
	defer cancel()

	// Track dependency failure for proper POSIX code
	var dependencyFailed bool

	baseStations, err := s.statusSvc.GetBaseStations(ctx, session.TenantID)
	if err != nil {
		// Log error and set degraded status (not silent failure)
		s.logger.WarnContext(s.sessionContext(session), LogSCACIStatusDependencyFailed,
			"tenantID", session.TenantID,
			"error", err)
		baseStations = nil
		dependencyFailed = true
	}

	// Transform base station records to SCACI protocol format
	var bsStatusArray []BaseStationStatus
	for _, bs := range baseStations {
		bsStatus := BuildBaseStationStatus(bs)
		bsStatusArray = append(bsStatusArray, bsStatus)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIStatusResponsePrepared,
		"baseStationCount", len(bsStatusArray),
		"tenantID", session.TenantID,
		"degraded", dependencyFailed)

	// Determine status code and message based on dependency health
	statusCode := POSIX_OK
	statusMessage := StatusMessageOperational
	if dependencyFailed {
		statusCode = POSIX_EIO // I/O error - database unavailable
		statusMessage = StatusMessageDegraded
	}

	// Send StatusResponse with base station telemetry
	resp := StatusResponse{
		BaseMessage: BaseMessage{
			Command: CmdStatusResponse,
			OpId:    opId,
		},
		Code:         statusCode,
		Message:      statusMessage,
		Time:         scTime,
		Uptime:       &uptimeSec,
		BaseStations: bsStatusArray,
	}

	if err := s.SendStatusResponse(conn, session, &resp); err != nil {
		return err
	}

	// Update StatusResponse state for audit trail (§3.5) if configured
	// State transition: pending → acknowledged (row created above)
	if s.config.LogStatusOperations && session.ID > 0 && s.operationRepo != nil {
		rspCtx, rspCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer rspCancel()

		responseData := map[string]interface{}{
			"opId":             opId,
			"initiator":        "ac",
			"baseStationCount": len(bsStatusArray),
			"degraded":         dependencyFailed,
		}
		if err := s.operationRepo.UpdateOperationState(rspCtx, session.ID, opId, models.OperationStateAcknowledged, responseData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordStatusRspOpFailed, "error", err)
		}
	}

	return nil
}

// handleStatusComplete processes StatusComplete messages per SCACI §3.5.3
//
// This completes the three-way handshake for Status operation.
// The AC acknowledges receipt of StatusResponse.
//
// No response is sent per spec - handshake is complete.
func (s *Server) handleStatusComplete(conn net.Conn, session *Session, opId int64) error {
	if session == nil {
		return s.sendErrorWithCatalog(conn, nil, opId, POSIX_EINVAL, errNoActiveSession)
	}

	s.logger.DebugContext(s.sessionContext(session), LogSCACIStatusHandshakeComplete, "opId", opId)

	session.UpdateLastSeen()

	// Update AC-initiated StatusComplete state for audit trail (§3.5) if configured
	// State transition: acknowledged → completed (completes AC-initiated three-way handshake)
	if s.config.LogStatusOperations && session.ID > 0 && s.operationRepo != nil {
		cmpCtx, cmpCancel := context.WithTimeout(s.sessionContext(session), dbconfig.DefaultQueryTimeout)
		defer cmpCancel()

		completeData := map[string]interface{}{
			"opId":      opId,
			"initiator": "ac",
		}
		if err := s.operationRepo.UpdateOperationState(cmpCtx, session.ID, opId, models.OperationStateCompleted, completeData); err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIRecordStatusCmpOpFailed, "error", err)
			// Continue - operation tracking is for audit, not critical path
		}
	}

	return nil
}
