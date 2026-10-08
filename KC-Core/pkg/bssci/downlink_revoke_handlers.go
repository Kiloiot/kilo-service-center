package bssci

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Import neutral scheduler contracts
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// SendDLDataRevoke sends a downlink data revoke operation to cancel queued
// downlink. ownerTenantID is the tenant that owns the downlink, which a
// roaming base station's session tenant is not: the revoke is recorded under
// the endpoint owner.
func (s *Server) SendDLDataRevoke(sessionID string, epEui uint64, queId uint64, ownerTenantID int64) error {
	session, exists := s.sessions.get(sessionID)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Check if handshake is complete (BSSCI-3.3-03)
	if !session.HandshakeComplete {
		return fmt.Errorf("%s", ResolveErrorMessage(errCannotSendDlDataRev))
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	// Create dlDataRev using canonical MIOTY type per spec Section 5.13.1
	dlDataRev := &mioty.DLDataRevoke{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdDLDataRevoke,
			OpId:        opId,
		},
		EpEui: epEui,
		QueId: queId,
	}

	// Convert to map for wire format (the command normalizer fixes the numeric types)
	msg := map[string]interface{}{
		"command": dlDataRev.CommandType,
		"opId":    dlDataRev.OpId,
		"epEui":   dlDataRev.EpEui,
		"queId":   dlDataRev.QueId,
	}

	// Create EUI bytes for storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Store pending operation with complete metadata for routing and resume;
	// the owner tenant files the revoke response under the endpoint owner.
	ownerTenantStr := strconv.FormatInt(ownerTenantID, 10)
	metadata := map[string]interface{}{
		models.EventDetailKeyBsEui:    session.BaseStationEUI, // Track which BS to target
		models.EventDetailKeyEpEui:    epEui,                  // Mandatory field for resume
		models.EventDetailKeyQueID:    queId,                  // Store as uint64 for consistency
		models.EventDetailKeyTenantID: ownerTenantStr,
	}

	// The recovery record must be durable before the frame is written; a
	// persistence failure aborts the send, leaving only a consumed-ID gap.
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdDLDataRevoke, msg, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLDataRevOperation,
			logger.FieldSessionID, sessionID,
			logger.FieldOpID, opId,
			logger.FieldError, err)
		return err
	}

	s.recordRevokeInitiated(session, ownerTenantID, epEui, queId, opId)

	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationAfterSendFailure,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}

		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlDataRev), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentDLDataRevToBaseStation,
		logger.FieldSessionID, sessionID,
		logger.FieldEpEui, epEui,
		logger.FieldQueID, queId,
		logger.FieldOpID, opId)

	return nil
}

// handleDLDataRevokeResponse handles dlDataRevRsp from base station
func (s *Server) handleDLDataRevokeResponse(session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedDLDataRevRspFromBaseStation,
		logger.FieldBsEui, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	target := s.revokeOperationOf(session, msg.OpId)
	endpointEUI, queueID := target.endpointEUI, target.queueID

	// Fail fast if queue ID is invalid (BSSCI §5.13 requires valid queId).
	// The error replaces this SC-initiated operation's normal completion, so
	// the base station's errorAck finalizes the pending dlDataRev row
	// (BSSCI rev1 §5.17 / classic §3.17).
	if queueID == 0 {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIInvalidQueueIDInRevokeResponse,
			logger.FieldOpID, msg.OpId)
		if sendErr := s.sendErrorReplacingOperation(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidQueueID)); sendErr != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorFrame, logger.FieldError, sendErr)
			return sendErr
		}
		return nil
	}

	// Delegate orchestration to DownlinkService (BSSCI §5.13)
	// Service handles: tenant resolution, DB update, cleanup
	ctx := s.sessionContext(session)
	responseMsg, revoked, err := s.downlinkSvc.ProcessRevokeResponse(ctx, session, msg.OpId, queueID, endpointEUI)
	if err != nil {
		catalogErr := catalogErrorOf(err)
		if sendErr := s.sendErrorReplacingOperation(session, msg.OpId, catalogErr.Posix, ResolveErrorMessage(catalogErr.Token)); sendErr != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorFrame, logger.FieldError, sendErr)
			return sendErr
		}
		return nil
	}

	s.recordRevocation(ctx, session, target, msg.OpId, revoked)

	// The service center completes its own SC-initiated dlDataRev operation
	// (BSSCI §3.13): it sends dlDataRevCmp and finalizes the pending operation.
	// A spec-compliant base station never returns dlDataRevCmp, so the pending
	// row is removed here or it leaks.
	if err := s.sendMessage(session, responseMsg); err != nil {
		return err
	}
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			logger.FieldError, err, logger.FieldOpID, msg.OpId)
	}
	return nil
}

// reconstitueDLDataRevMessage reconstitutes a dlDataRev message from stored metadata
func (s *Server) reconstitueDLDataRevMessage(msg, metadata map[string]interface{}) (map[string]interface{}, error) {
	// Enforce command field
	msg["command"] = mioty.CmdDLDataRevoke

	// Restore epEui from metadata (mandatory field per BSSCI §3.13.1).
	// Canonical numeric coercion preserves the full uint64 EUI range under the
	// strict json.Number resume decode.
	if _, present := metadata[models.EventDetailKeyEpEui]; !present {
		return nil, fmt.Errorf("%s", ResolveErrorMessage(errMissingEpEuiInMetadata))
	}
	epEui, err := coerceUint64(metadata[models.EventDetailKeyEpEui])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errMissingEpEuiInMetadata), err)
	}
	msg["epEui"] = epEui

	// Restore queId as uint64 per canonical MIOTY type
	if _, present := metadata[models.EventDetailKeyQueID]; !present {
		return nil, fmt.Errorf("%s", ResolveErrorMessage(errMissingQueIDInMetadata))
	}
	queId, err := coerceUint64(metadata[models.EventDetailKeyQueID])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errNegativeQueueIDInMetadata), err)
	}
	msg["queId"] = queId

	// Fix opId type if needed
	if opId, ok := parseOpID(msg["opId"]); ok {
		msg["opId"] = opId
	}

	return msg, nil
}

// recordRevokeInitiated files the revoke sent to the station holding the
// downlink under the endpoint owner, naming the station.
func (s *Server) recordRevokeInitiated(session *Session, ownerTenantID int64, epEui, queId uint64, opId int64) {
	if s.eventStore == nil {
		return
	}
	ctx := s.sessionContext(session)
	station := IdentifyStation(ctx, s.basestationRepo, ownerTenantID, session.BaseStationEUI)
	eventData := map[string]interface{}{
		models.EventDetailKeyBsEui:     station.EUI,
		models.EventDetailKeyEpEui:     mioty.FormatEUI64(epEui),
		models.EventDetailKeyQueID:     queId,
		models.EventDetailKeyOpID:      opId,
		models.EventDetailKeyOperation: OperationDLDataRevoke,
		models.EventDetailKeyTimestamp: s.clock.Now().Format(time.RFC3339),
	}
	if station.Name != "" {
		eventData[models.EventDetailKeyBaseStationName] = station.Name
	}
	details, err := json.Marshal(eventData)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToRecordDLDataRevokeInitiatedEvent, logger.FieldError, err)
		return
	}
	if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
		TenantID:      strconv.FormatInt(ownerTenantID, 10),
		EventType:     models.EventTypeDLDataRevokeInitiated,
		Category:      models.EventCategoryMessage,
		Severity:      models.EventSeverityInfo,
		Title:         models.EventTitleDLDataRevoke,
		Description:   fmt.Sprintf(eventDescFmtRevokingQueuedDownlink, queId, mioty.FormatEUI64(epEui), station.Label()),
		SourceType:    mioty.SourceTypeEndpoint,
		SourceName:    mioty.FormatEUI64(epEui),
		BasestationID: station.ID,
		CreatedAt:     s.clock.Now(),
		UpdatedAt:     s.clock.Now(),
		Details:       details,
	}); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToRecordDLDataRevokeInitiatedEvent, logger.FieldError, err)
	}
}
