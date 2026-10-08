package bssci

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// handlePing handles ping operations
func (s *Server) handlePing(session *Session, msg *Message, _ map[string]interface{}) error {
	// Send ping response
	response := map[string]interface{}{
		"command": mioty.CmdPingResponse,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, response)
}

// handlePingResponse completes a service-center ping (BSSCI §5.4.2) and puts
// the station's answer in its activity.
func (s *Server) handlePingResponse(session *Session, msg *Message, data map[string]interface{}) error {
	answeredAt := s.clock.Now()
	ctx := s.sessionContext(session)
	answered := map[string]interface{}{models.EventDetailKeyOpID: msg.OpId}
	result, reported := getNumericField(data, fieldPingResult)
	if reported {
		answered[models.EventDetailKeyResult] = result
	}

	s.logger.DebugContext(ctx, LogBSSCIReceivedPingResponse,
		logger.FieldSessionID, session.ID,
		logger.FieldOpID, msg.OpId,
		logger.FieldResult, result)
	s.recordStationEvent(ctx, session.BaseStationEUI, models.EventTypeBaseStationPingAnswered, answeredAt, answered)

	// Send ping complete to finish three-way handshake
	response := map[string]interface{}{
		"command": mioty.CmdPingComplete,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, response)
}

// handlePingComplete handles ping complete operations
func (s *Server) handlePingComplete(session *Session, _ *Message, _ map[string]interface{}) error {
	// Ping completed successfully - persist timestamp for monitoring (BSSCI §5.4)
	ctx := s.sessionContext(session)
	if err := s.sessionSvc.UpdatePingTimestamp(ctx, session); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToUpdateDatabaseSession,
			logger.FieldError, err,
			logger.FieldSessionID, session.DbSessionID,
			logger.FieldBaseStationEui, session.BaseStationEUI)
		// Non-fatal: log error but don't fail the handshake
	}

	return nil
}

// InitiatePing sends a ping request to a base station by EUI (BSSCI §5.4)
// Implements PingCommander interface for API layer access
func (s *Server) InitiatePing(ctx context.Context, baseStationEUI uint64, tenantID int64) (int64, error) {
	session, err := s.pingableSession(baseStationEUI, tenantID)
	if err != nil {
		return 0, err
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID and
	// persist the counter before the frame is written; never roll back. Ping
	// is idempotent liveness traffic, so no pending record is persisted.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return 0, err
	}

	msg := map[string]interface{}{
		"command": mioty.CmdPing,
		"opId":    opId,
	}

	s.logger.InfoContext(ctx, LogBSSCISendingPingToBaseStation,
		logger.FieldSessionID, session.ID,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, opId)

	sentAt := s.clock.Now()
	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			s.closeTransportAfterWriteFailure(session, opId, err)
		}
		return 0, err
	}

	s.recordStationEvent(pkgcontext.WithTenantID(ctx, tenantID), session.BaseStationEUI,
		models.EventTypeBaseStationPingSent, sentAt, map[string]interface{}{models.EventDetailKeyOpID: opId})
	return opId, nil
}

// pingableSession is the live session of the base station, once its connect
// handshake completed (BSSCI §5.4) and only for the tenant that owns it
// (defense-in-depth).
func (s *Server) pingableSession(baseStationEUI uint64, tenantID int64) (*Session, error) {
	sessionInterface := s.GetSessionByEUI(baseStationEUI)
	if sessionInterface == nil {
		return nil, NewCatalogError(errSessionNotFound, POSIX_ENOENT)
	}
	session, ok := sessionInterface.(*Session)
	if !ok {
		return nil, fmt.Errorf(errFmtInvalidSessionTypeForEUI, mioty.FormatEUI64(baseStationEUI))
	}
	if !session.HandshakeComplete {
		return nil, NewCatalogError(errCannotSendPing, POSIX_EPROTO)
	}
	if sessionTenantID := resolvedTenant(session, s.tenantID); sessionTenantID != tenantID {
		return nil, fmt.Errorf(errFmtBaseStationBelongsToDifferentTenant,
			mioty.FormatEUI64(baseStationEUI), sessionTenantID, tenantID)
	}
	return session, nil
}

// recordStationEvent puts a protocol exchange that happened at occurredAt in
// the station's activity; a failed record is logged and never returned.
func (s *Server) recordStationEvent(ctx context.Context, bsEui uint64, eventType string, occurredAt time.Time, data map[string]interface{}) {
	if err := s.stationEvents.RecordEvent(ctx, mioty.EUI64(bsEui).ToBytes(), eventType, occurredAt, data); err != nil {
		s.logger.WarnContext(ctx, LogBSSCIStationEventNotRecorded,
			logger.FieldBsEui, bsEui,
			logger.FieldEventType, eventType,
			logger.FieldError, err)
	}
}
