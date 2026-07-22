package bssci

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler" // Import neutral scheduler contracts
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// validatePayloadSizes checks each payload entry against MaxDLUserDataBytes (MIOTY radio protocol §4.3.2).
func validatePayloadSizes(payloads [][]byte) error {
	for i, p := range payloads {
		if len(p) > mioty.MaxDLUserDataBytes {
			return fmt.Errorf("%s: entry %d is %d bytes (max %d)",
				ResolveErrorMessage(errDLPayloadTooLarge), i, len(p), mioty.MaxDLUserDataBytes)
		}
	}
	return nil
}

// ============================================================================
// DL Data Queue Response Handlers (MIOTY BSSCI v1.0.0 Section 5.12)
// ============================================================================

// handleDLDataQueueResponse handles dlDataQueRsp from base station
func (s *Server) handleDLDataQueueResponse(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// According to MIOTY BSSCI v1.0.0 spec, dlDataQueRsp only contains:
	// - command (String "dlDataQueRsp")
	// - opId (Numeric ID of the operation)

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedDLDataQueRspFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Look up the endpoint EUI, queue ID, and tenant ID from pending operations
	// StatusService is the only path for pending operation storage
	// Extract tenantID for proper roaming tenant isolation (BSSCI §5.12)
	endpointEUI, queueID, tenantIDStr := s.statusSvc.ExtractQueueMetadata(session, msg.OpId)

	// Update downlink base station ownership in database
	if s.downlinkQueueStore != nil && queueID > 0 {
		ctx := s.sessionContext(session)
		// Use tenant ID from queue metadata per BSSCI §5.12 (roaming)
		// Fallback to server default only if metadata extraction failed
		if tenantIDStr == "" {
			tenantIDStr = s.formatTenantID()
		}
		//nolint:gosec // G115: queueID > 0 guard ensures safe int64->uint64 conversion
		if err := s.downlinkQueueStore.UpdateDownlinkBaseStation(ctx, uint64(queueID), tenantIDStr, session.BaseStationEUI); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToUpdateDownlinkBaseStationOwnership,
				//nolint:gosec // G115: queueID > 0 guard ensures safe int64->uint64 conversion
				"queId", uint64(queueID),
				"bsEui", session.BaseStationEUI,
				"error", err)
		}

		// Repeat the idempotent reserved→queued confirmation: the base station
		// acknowledged the queue, so if the dispatcher's confirmation was lost
		// (send succeeded but the status write failed or the process crashed),
		// this repairs the row. An already-queued row is a no-op.
		if tenantID, parseErr := strconv.ParseInt(tenantIDStr, 10, 64); parseErr == nil {
			//nolint:gosec // G115: queueID > 0 guard ensures safe int64->uint64 conversion
			if err := s.downlinkQueueStore.MarkReservedAsQueued(ctx, uint64(queueID), tenantID,
				session.BaseStationEUI, time.Now().UnixNano(), nil, nil); err != nil {
				s.logger.WarnContext(ctx, LogBSSCIFailedToConfirmDownlinkQueued,
					//nolint:gosec // G115: queueID > 0 guard ensures safe int64->uint64 conversion
					"queId", uint64(queueID),
					"bsEui", session.BaseStationEUI,
					"error", err)
			}
		}
	}

	// Record success event via audit logger
	if endpointEUI != 0 {
		ctx := s.sessionContext(session)
		// Use tenant ID from queue metadata for accurate roaming audit trails (BSSCI §5.12)
		auditTenantID := tenantIDStr
		if auditTenantID == "" {
			auditTenantID = s.formatTenantID()
		}
		if err := s.auditLogger.RecordQueueAck(ctx, auditTenantID, session, endpointEUI, queueID, msg.OpId); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordDLDataQueueAcknowledgedEvent, "error", err)
		}
	}

	// The service center completes its own SC-initiated dlDataQue operation
	// (BSSCI §3.12): it sends dlDataQueCmp and finalizes the pending operation.
	// A spec-compliant base station never returns dlDataQueCmp, so the pending
	// row is removed here or it leaks.
	complete := s.queueSerializer.BuildDLDataQueueComplete(msg.OpId)
	if err := s.sendMessage(session, complete); err != nil {
		return err
	}
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			"error", err, "opId", msg.OpId)
	}
	return nil
}

// ============================================================================
// DL Data Result Operation Handlers (MIOTY BSSCI v1.0.0 Section 5.14)
// ============================================================================

// handleDLDataResult handles dlDataRes from base station when downlink has been sent or discarded
func (s *Server) handleDLDataResult(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Payload is normalized by handleMessage before dispatch
	// All mandatory fields guaranteed present and correctly typed
	// Conditional validation enforced: result="sent" requires txTime/packetCnt
	// Enum validation enforced: result must be "sent", "expired", or "invalid"

	// Build canonical MIOTY type from normalized payload
	dlResult := mioty.DLDataResult{
		BaseMessage: mioty.BaseMessage{
			CommandType: mioty.CmdDLDataResult,
			OpId:        msg.OpId,
		},
		EpEui:  data["epEui"].(uint64),  // Normalizer guarantees uint64
		QueId:  data["queId"].(uint64),  // Normalizer guarantees uint64
		Result: data["result"].(string), // Normalizer validates enum
	}

	// Extract optional conditional fields (nil when absent or forbidden)
	if txTime := data["txTime"]; txTime != nil {
		txTimeVal := txTime.(int64)
		dlResult.TxTime = &txTimeVal
	}
	if packetCnt := data["packetCnt"]; packetCnt != nil {
		packetCntVal := packetCnt.(uint32)
		dlResult.PacketCnt = &packetCntVal
	}
	if bsEui := data["bsEui"]; bsEui != nil {
		bsEuiVal := bsEui.(uint64)
		dlResult.BsEui = &bsEuiVal
	}

	ctx := s.sessionContext(session)

	// Pre-resolve queue tenant/org for MQTT publishing (before ProcessDLDataResult)
	var mqttOrgStr string
	if s.mqttPublisher != nil && s.tenantResolver != nil && dlResult.QueId > 0 && dlResult.QueId <= uint64(math.MaxInt64) {
		if tenantStr, resolveErr := s.tenantResolver.ResolveTenant(ctx, int64(dlResult.QueId)); resolveErr == nil {
			if tid, parseErr := strconv.ParseInt(tenantStr, 10, 64); parseErr == nil && s.orgResolver != nil {
				// A Nil org UUID means the organization is unresolved; its
				// string form is non-empty, so guard on the value not the
				// string, or the publish would fire for an unresolved org.
				if orgUUID, orgErr := s.orgResolver.GetDefaultOrgForTenant(s.safeCtx(), tid); orgErr == nil && orgUUID != uuid.Nil {
					mqttOrgStr = orgUUID.String()
				}
			}
		}
	}

	// Delegate orchestration to DownlinkService (BSSCI §5.14)
	// Service handles: tenant resolution, DB update, SCACI broadcast, audit logging, cleanup
	responseMsg, err := s.downlinkSvc.ProcessDLDataResult(ctx, session, &dlResult)
	if err != nil {
		// Service returned error - send appropriate error frame to base station
		// Determine error code based on error message
		var posixCode int
		var errToken string

		if strings.Contains(err.Error(), "queue ID out of range") {
			posixCode = POSIX_ERANGE
			errToken = errQueueIDOutOfRange
		} else if strings.Contains(err.Error(), "cannot resolve tenant") {
			posixCode = POSIX_EPROTO
			errToken = errCannotResolveTenantForQueue
		} else if strings.Contains(err.Error(), "invalid tenant ID format") {
			posixCode = POSIX_EINVAL
			errToken = errInvalidTenantIDFormat
		} else if strings.Contains(err.Error(), "queue ID not found") {
			posixCode = POSIX_EPROTO
			errToken = errQueueIDNotFound
		} else {
			// Database or other internal error
			posixCode = POSIX_EIO
			errToken = errDatabaseUpdateFailed
		}

		if sendErr := s.sendError(session, msg.OpId, posixCode, ResolveErrorMessage(errToken)); sendErr != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorFrame, "error", sendErr)
			return sendErr // Socket is broken, let session tear down
		}
		// Error frame sent successfully, keep session alive for errorAck
		return nil
	}

	// Publish DL result event to MQTT after successful processing
	if s.mqttPublisher != nil && mqttOrgStr != "" {
		go func() {
			if pubErr := s.mqttPublisher.PublishDownlinkResult(ctx, mqttOrgStr,
				dlResult.EpEui, dlResult.QueId, dlResult.Result); pubErr != nil {
				s.logger.WarnContext(ctx, LogBSSCIFailedToPublishDLResultToMQTT,
					"error", pubErr, "queId", dlResult.QueId)
			}
		}()
	} else if s.mqttPublisher != nil && mqttOrgStr == "" {
		s.logger.WarnContext(ctx, LogBSSCIMQTTPublishSkippedOrgUnresolved,
			"queId", dlResult.QueId, "event", MQTTEventKeyDownlinkResult)
	}

	// Runtime guard: Verify dlDataResRsp contains only canonical fields per BSSCI §5.14.2
	if len(responseMsg) != 2 {
		s.logger.WarnContext(ctx, LogBSSCIUnexpectedFieldsInDLDataResRsp,
			"fieldCount", len(responseMsg),
			"opId", msg.OpId,
			"expectedFields", "command,opId")
	}
	// Verify specific keys
	if _, hasCommand := responseMsg["command"]; !hasCommand {
		s.logger.ErrorContext(ctx, LogBSSCIMissingCommandFieldInResponse, "opId", msg.OpId)
	}
	if _, hasOpId := responseMsg["opId"]; !hasOpId {
		s.logger.ErrorContext(ctx, LogBSSCIMissingOpIDFieldInResponse, "opId", msg.OpId)
	}

	// Send response message returned by service
	return s.sendMessage(session, responseMsg)
}

// handleDLDataResultResponse handles dlDataResRsp from base station
func (s *Server) handleDLDataResultResponse(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIReceivedDLDataResRspFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Send dlDataResCmp to complete the three-way handshake via queue serializer
	complete := s.queueSerializer.BuildDLDataResultComplete(msg.OpId)
	return s.sendMessage(session, complete)
}

// handleDLDataResultComplete handles dlDataResCmp from base station
func (s *Server) handleDLDataResultComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Clean up pending operation from database
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both cache and DB removal
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDB,
			"sessionID", session.DbSessionID,
			"opId", msg.OpId,
			"error", err)
	}
	// Note: No manual fallback - trust StatusService single-writer pattern

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIDLDataResultOperationCompleted,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	return nil
}

// ============================================================================
// DL Data Queue Operation (MIOTY BSSCI v1.0.0 Section 5.12)
// ============================================================================

// SendDLDataQueue sends a downlink data queue operation to queue downlink data for an endpoint
// Supports both single payload and counter-dependent payloads per BSSCI §3.12.
// When dlRxStatQry is true (the SCACI §3.10.1 hint), a BSSCI dlRxStatQry
// operation (rev1 §5.16 / classic §3.16) is paired with the queue: both
// operations are durably persisted together before either frame is written,
// and the query frame precedes the queue frame so the DL RX status query is
// scheduled for the queued downlink's transmission.
func (s *Server) SendDLDataQueue(sessionID string, epEui uint64, payloads [][]byte, queId int64,
	prio float32, cntDepend bool, packetCnt []int64, format uint8,
	responseExp bool, responsePrio bool, dlWindReq bool, expOnly bool, tenantID int64,
	dlRxStatQry bool) error {

	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Check if connect handshake is complete (BSSCI §3.3)
	if !session.HandshakeComplete {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIConnectHandshakeNotCompleteDL,
			"sessionID", sessionID,
			"bsEui", session.BaseStationEUI)
		return fmt.Errorf("%s for session %s", ResolveErrorMessage(errHandshakeNotComplete), sessionID)
	}

	// Validate payload consistency based on counter-dependency mode (BSSCI §3.12.1)
	if cntDepend && len(packetCnt) != len(payloads) {
		return fmt.Errorf("%s, got %d != %d",
			ResolveErrorMessage(errCounterDependentPayloadMismatch), len(packetCnt), len(payloads))
	}
	// BSSCI §5.12: Allow empty userData for pure acknowledgement downlink in non-counter-dependent mode
	if !cntDepend && len(payloads) != 0 && len(payloads) != 1 {
		return fmt.Errorf("%s, got %d", ResolveErrorMessage(errNonCounterDependentMultiPayload), len(payloads))
	}

	// Validate individual payload sizes (MIOTY radio protocol §4.3.2)
	if err := validatePayloadSizes(payloads); err != nil {
		return err
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the IDs, persist
	// the counter once for the whole pair, persist the pending records, then
	// write the frames. IDs are never rolled back. The query ID is allocated
	// first so its pending row precedes the queue row in reissue order.
	var qryOpID int64
	if dlRxStatQry {
		qryOpID = session.NextScOpID()
	}
	opId := session.NextScOpID()
	if err := s.sessionSvc.UpdateSessionCounters(s.sessionContext(session), session); err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistSessionCounters), err)
	}

	// Build userData based on counter-dependency mode
	var userDataField interface{}
	if cntDepend {
		// Counter-dependent: userData is Numeric[m][n] where m = len(payloads)
		userDataArray := make([]interface{}, len(payloads))
		for i, payload := range payloads {
			// Convert each payload to Numeric[n]
			numericPayload := make([]interface{}, len(payload))
			for j, b := range payload {
				numericPayload[j] = uint8(b)
			}
			userDataArray[i] = numericPayload
		}
		userDataField = userDataArray
	} else {
		// Non-counter-dependent: userData is Numeric[m][n] with m=1 per BSSCI §3.12.1
		// ("single user data entry if cntDepend is false"). For an ACK-only downlink
		// ("If user data is empty, a pure acknowledgement downlink is queued") the
		// single entry is zero bytes long — outer length must still be 1 or the
		// Fraunhofer BS rejects the message with code=22 "DL data queue message
		// malformed".
		var singlePayload []interface{}
		if len(payloads) > 0 {
			singlePayload = make([]interface{}, len(payloads[0]))
			for i, b := range payloads[0] {
				singlePayload[i] = uint8(b)
			}
		} else {
			singlePayload = []interface{}{}
		}
		userDataField = []interface{}{singlePayload}
	}

	// Create dlDataQue message per BSSCI §5.12.1
	msg := map[string]interface{}{
		"command":   mioty.CmdDLDataQueue,
		"opId":      opId,
		"epEui":     epEui,
		"queId":     queId,
		"userData":  userDataField,
		"prio":      prio,
		"cntDepend": cntDepend, // BSSCI §5.12.1: Required field (always present, true or false)
	}

	// Add packetCnt array when counter-dependent mode is enabled (BSSCI §5.12: conditional on cntDepend=true)
	if cntDepend {
		// Convert packetCnt to Numeric[m] array
		packetCntArray := make([]interface{}, len(packetCnt))
		for i, cnt := range packetCnt {
			packetCntArray[i] = cnt
		}
		msg["packetCnt"] = packetCntArray
	}

	// Add optional MIOTY fields
	if format > 0 {
		msg["format"] = format
	}
	if responseExp {
		msg["responseExp"] = true
	}
	if responsePrio {
		msg["responsePrio"] = true
	}
	if dlWindReq {
		msg["dlWindReq"] = true
	}
	if expOnly {
		msg["expOnly"] = true
	}

	// Create EUI bytes for storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Encode payloads for storage (base64 encode each)
	encodedPayloads := make([]string, len(payloads))
	for i, payload := range payloads {
		encodedPayloads[i] = base64.StdEncoding.EncodeToString(payload)
	}

	// Build comprehensive metadata for session resume
	metadata := map[string]interface{}{
		"bsEui":        session.BaseStationEUI, // Track which BS has this queued message
		"epEui":        epEui,                  // Add for symmetry with dlDataRev
		"queId":        queId,
		"prio":         prio,
		"payloads":     encodedPayloads,
		"cntDepend":    cntDepend,
		"packetCnt":    packetCnt,
		"format":       format,
		"responseExp":  responseExp,
		"responsePrio": responsePrio,
		"dlWindReq":    dlWindReq,
		"expOnly":      expOnly,
		"tenantID":     strconv.FormatInt(tenantID, 10), // Store as string to avoid JSON float64 issues
	}

	// Persist the recovery records before any frame is written. For the
	// dlRxStatQry pair, the correlation row and both pending operations are
	// durably recorded together (all-or-nothing) so resume can continue the
	// pair with its original IDs; a persistence failure emits neither frame.
	var qryMsg map[string]interface{}
	if dlRxStatQry {
		qryMsg = map[string]interface{}{
			"command": mioty.CmdDLRxStatusQuery,
			"opId":    qryOpID,
			"epEui":   epEui,
		}
		if err := s.persistDLRXQueryCorrelation(session, qryOpID, epEui, euiBytes); err != nil {
			return err
		}
		pair := []*PendingOperation{
			s.buildPendingOperation(session, qryOpID, mioty.CmdDLRxStatusQuery, qryMsg, euiBytes, nil),
			s.buildPendingOperation(session, opId, mioty.CmdDLDataQueue, msg, euiBytes, metadata),
		}
		if err := s.persistPendingOperationBatch(session, pair); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLDataQueOperation,
				"sessionID", sessionID,
				"opId", opId,
				"qryOpID", qryOpID,
				"error", err)
			return err
		}
	} else if err := s.persistPendingOperation(session, opId, mioty.CmdDLDataQueue, msg, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLDataQueOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
		return err
	}

	// The query frame precedes the queue frame (BSSCI rev1 §5.16: the query is
	// scheduled for the next downlink transmission). A query failure aborts
	// the pair - the queue frame is never written to a possibly corrupt
	// connection.
	if dlRxStatQry {
		if err := s.sendMessage(session, qryMsg); err != nil {
			if errors.Is(err, ErrAmbiguousWrite) {
				// Both recovery rows are preserved; resume reissues the pair
				// with its original IDs.
				s.closeTransportAfterWriteFailure(session, qryOpID, err)
			} else {
				// Nothing reached the wire - remove both recovery rows.
				for _, cleanupOpID := range []int64{qryOpID, opId} {
					if cleanupErr := s.removePendingOperation(session, cleanupOpID); cleanupErr != nil {
						s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToClearPersistedPendingOperation,
							"sessionID", session.DbSessionID,
							"opId", cleanupOpID,
							"error", cleanupErr)
					}
				}
			}
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlRxStatQry), err)
		}
	}

	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending rows for
			// resume reissue with the original IDs and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			// Nothing reached the wire for the queue frame; its recovery row is
			// removed (an already-sent query keeps its row).
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToClearPersistedPendingOperation,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}
		return err
	}

	// Register queue-to-tenant mapping for fast result processing (BSSCI §5.14)
	// Production deployments MUST provide a tenantResolver; tests inject test doubles
	if tenantIDStr, ok := metadata["tenantID"].(string); ok {
		s.tenantResolver.RegisterQueueTenant(queId, tenantIDStr)
	}

	return nil
}

// ============================================================================
// DL Data Revoke Operation (MIOTY BSSCI v1.0.0 Section 5.13)
// ============================================================================

// SendDLDataRevoke sends a downlink data revoke operation to cancel queued downlink
func (s *Server) SendDLDataRevoke(sessionID string, epEui uint64, queId uint64) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

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
	opId, err := s.beginScOperation(session)
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

	// Convert to map for wire format (normalizeMessageTypes will ensure proper types)
	msg := map[string]interface{}{
		"command": dlDataRev.CommandType,
		"opId":    dlDataRev.OpId,
		"epEui":   dlDataRev.EpEui,
		"queId":   dlDataRev.QueId,
	}

	// Create EUI bytes for storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Store pending operation with complete metadata for routing and resume
	// Use per-session tenant ID (from cert/org resolution) for multi-tenant isolation
	tenantIDStr := s.formatTenantID() // fallback to server default
	if session.ResolvedTenantID != 0 {
		tenantIDStr = fmt.Sprintf("%d", session.ResolvedTenantID)
	}
	metadata := map[string]interface{}{
		"bsEui":    session.BaseStationEUI, // Track which BS to target
		"epEui":    epEui,                  // Mandatory field for resume
		"queId":    queId,                  // Store as uint64 for consistency
		"tenantID": tenantIDStr,            // Store per-session tenant for response handler
	}

	// The recovery record must be durable before the frame is written; a
	// persistence failure aborts the send, leaving only a consumed-ID gap.
	if err := s.persistPendingOperation(session, opId, mioty.CmdDLDataRevoke, msg, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLDataRevOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
		return err
	}

	// Record event
	if s.eventStore != nil {
		eventData := map[string]interface{}{
			"bsEui":            fmt.Sprintf("%016X", session.BaseStationEUI),
			"basestation_name": session.UserProvidedName,
			"epEui":            fmt.Sprintf("%016X", epEui),
			"queId":            queId,
			"opId":             opId,
			"operation":        OperationDLDataRevoke,
			"timestamp":        time.Now().Format(time.RFC3339),
		}

		ctx := s.sessionContext(session)
		if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
			TenantID:  fmt.Sprintf("%d", resolvedTenant(session, s.tenantID)),
			EventType: EventDLDataRevokeInitiated,
			Category:  models.EventCategoryMessage,
			Severity:  SeverityInfo,
			Title:     fmt.Sprintf(models.EventTitleDLDataRevoke, session.UserProvidedName),
			Description: fmt.Sprintf("Revoking queued downlink for endpoint %016X at base station %s (Queue ID: %d)",
				epEui, session.UserProvidedName, queId),
			SourceType: mioty.SourceTypeEndpoint,
			SourceName: fmt.Sprintf("%016X", epEui),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Details:    func() []byte { b, _ := json.Marshal(eventData); return b }(),
		}); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordDLDataRevokeInitiatedEvent, "error", err)
		}
	}

	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationAfterSendFailure,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}

		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlDataRev), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentDLDataRevToBaseStation,
		"sessionID", sessionID,
		"epEui", epEui,
		"queId", queId,
		"opId", opId)

	return nil
}

// handleDLDataRevokeResponse handles dlDataRevRsp from base station
func (s *Server) handleDLDataRevokeResponse(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedDLDataRevRspFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Look up the endpoint EUI and queue ID from pending operations
	// BSSCI §§5.11-5.12.3 Gap 1: Dual-path for test compatibility
	var endpointEUI uint64
	var queueID int64
	var tenantIDStr string
	// StatusService is the only path for pending operation storage
	pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId))
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
			"opId", msg.OpId, "error", err)
	}
	if pendingOp != nil {
		// Convert []byte to uint64
		if len(pendingOp.Endpoint) == 8 {
			endpointEUI = binary.BigEndian.Uint64(pendingOp.Endpoint)
		}
		if pendingOp.Metadata != nil {
			// Extract queue ID with type guards for int64/float64/uint64
			if qid, ok := pendingOp.Metadata["queId"].(int64); ok {
				queueID = qid
			} else if qid, ok := pendingOp.Metadata["queId"].(float64); ok {
				queueID = int64(qid)
			} else if qid, ok := pendingOp.Metadata["queId"].(uint64); ok {
				// Handle uint64 with overflow guard
				if qid > math.MaxInt64 {
					s.logger.ErrorContext(s.sessionContext(session), LogBSSCIQueueIDOutOfRange,
						"opId", msg.OpId,
						"queId", qid,
						"reason", "uint64 overflow (> math.MaxInt64)")
				} else {
					queueID = int64(qid)
				}
			}

			// Extract tenant ID with three-tier fallback (BSSCI §5.13.3 resume safety)
			// 1. Prefer stored metadata from original request
			// 2. Fall back to per-session tenant (cross-tenant roaming scenarios)
			// 3. Final fallback to server default tenant
			if tid, ok := pendingOp.Metadata["tenantID"].(string); ok {
				tenantIDStr = tid
			} else if session.ResolvedTenantID != 0 {
				tenantIDStr = fmt.Sprintf("%d", session.ResolvedTenantID)
			} else {
				tenantIDStr = s.formatTenantID()
			}
		}
	}

	// Fail fast if queue ID is invalid (BSSCI §5.13 requires valid queId).
	// The error replaces this SC-initiated operation's normal completion, so
	// the base station's errorAck finalizes the pending dlDataRev row
	// (BSSCI rev1 §5.17 / classic §3.17).
	if queueID == 0 {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIInvalidQueueIDInRevokeResponse,
			"opId", msg.OpId)
		if sendErr := s.sendErrorReplacingOperation(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidQueueID)); sendErr != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorFrame, "error", sendErr)
			return sendErr
		}
		return nil
	}

	// Delegate orchestration to DownlinkService (BSSCI §5.13)
	// Service handles: tenant resolution, DB update, cleanup
	ctx := s.sessionContext(session)
	responseMsg, err := s.downlinkSvc.ProcessRevokeResponse(ctx, session, msg.OpId, queueID, endpointEUI)
	if err != nil {
		// Service returned error - send appropriate error frame to base station
		// Use typed error detection instead of string matching
		var catalogErr *CatalogError
		if errors.As(err, &catalogErr) {
			// Service returned a catalog error with token and POSIX code
			if sendErr := s.sendErrorReplacingOperation(session, msg.OpId, catalogErr.Posix, ResolveErrorMessage(catalogErr.Token)); sendErr != nil {
				s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorFrame, "error", sendErr)
				return sendErr
			}
			return nil
		}

		// Fallback for non-catalog errors (shouldn't happen after service refactor)
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIUnexpectedRevokeResponseError,
			"error", err,
			"opId", msg.OpId)
		if sendErr := s.sendErrorReplacingOperation(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errDatabaseUpdateFailed)); sendErr != nil {
			s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToSendErrorFrame, "error", sendErr)
			return sendErr
		}
		return nil
	}

	// Record success event (kept in handler since it uses eventStore directly)
	// TODO: Move to AuditLogger.RecordDLRevokeResponse method
	if s.eventStore != nil && endpointEUI != 0 && queueID != 0 {
		// Use tenant ID extracted from metadata (no redundant DB lookup)

		eventData := map[string]interface{}{
			"bsEui":            fmt.Sprintf("%016X", session.BaseStationEUI),
			"basestation_name": session.UserProvidedName,
			"epEui":            fmt.Sprintf("%016X", endpointEUI),
			"queId":            queueID,
			"opId":             msg.OpId,
			"operation":        OperationDLDataRevoke,
			"status":           "success",
			"timestamp":        time.Now().Format(time.RFC3339),
		}

		if err := s.eventStore.CreateEvent(ctx, &models.SystemEvent{
			TenantID:  tenantIDStr,
			EventType: EventDLDataRevoked,
			Category:  models.EventCategoryMessage,
			Severity:  SeverityInfo,
			Title:     fmt.Sprintf(models.EventTitleDLDataRevoked, session.UserProvidedName),
			Description: fmt.Sprintf("Successfully revoked queued downlink for endpoint %016X at base station %s (Queue ID: %d)",
				endpointEUI, session.UserProvidedName, queueID),
			SourceType: mioty.SourceTypeEndpoint,
			SourceName: fmt.Sprintf("%016X", endpointEUI),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Details:    func() []byte { b, _ := json.Marshal(eventData); return b }(),
		}); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRecordDLDataRevokedEvent, "error", err)
		}
	}

	// The service center completes its own SC-initiated dlDataRev operation
	// (BSSCI §3.13): it sends dlDataRevCmp and finalizes the pending operation.
	// A spec-compliant base station never returns dlDataRevCmp, so the pending
	// row is removed here or it leaks.
	if err := s.sendMessage(session, responseMsg); err != nil {
		return err
	}
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			"error", err, "opId", msg.OpId)
	}
	return nil
}

// reconstitueDLDataQueMessage reconstitutes a dlDataQue message from sanitized storage
// This is used during session resume to rebuild the message with the full payload structure
func (s *Server) reconstitueDLDataQueMessage(sanitizedMsg map[string]interface{}, metadata map[string]interface{}, pendingOp *PendingOperation) (map[string]interface{}, error) {
	// Start with the sanitized message
	msg := make(map[string]interface{})
	for k, v := range sanitizedMsg {
		msg[k] = v
	}

	// Extract payloads from metadata
	var payloads [][]byte
	if encodedPayloads, ok := metadata["payloads"].([]interface{}); ok {
		payloads = make([][]byte, len(encodedPayloads))
		for i, encoded := range encodedPayloads {
			if encodedStr, ok := encoded.(string); ok {
				decoded, err := base64.StdEncoding.DecodeString(encodedStr)
				if err != nil {
					return nil, fmt.Errorf("%s %d: %w", ResolveErrorMessage(errFailedToDecodePayload), i, err)
				}
				payloads[i] = decoded
			}
		}
	} else if encodedPayloads, ok := metadata["payloads"].([]string); ok {
		// Handle direct string slice
		payloads = make([][]byte, len(encodedPayloads))
		for i, encodedStr := range encodedPayloads {
			decoded, err := base64.StdEncoding.DecodeString(encodedStr)
			if err != nil {
				return nil, fmt.Errorf("%s %d: %w", ResolveErrorMessage(errFailedToDecodePayload), i, err)
			}
			payloads[i] = decoded
		}
	} else {
		// Fallback to single payload from PendingOperation.Data
		if pendingOp != nil && len(pendingOp.Data) > 0 {
			payloads = [][]byte{pendingOp.Data}
		} else {
			payloads = [][]byte{{}} // Empty payload
		}
	}

	// Validate reconstructed payload sizes (MIOTY radio protocol §4.3.2)
	if err := validatePayloadSizes(payloads); err != nil {
		return nil, err
	}

	// Extract counter-dependency flag
	cntDepend, _ := metadata["cntDepend"].(bool)

	// Build userData based on counter-dependency mode
	var userDataField interface{}
	if cntDepend {
		// Counter-dependent: userData is Numeric[m][n] where m = len(payloads)
		userDataArray := make([]interface{}, len(payloads))
		for i, payload := range payloads {
			// Convert each payload to Numeric[n]
			numericPayload := make([]interface{}, len(payload))
			for j, b := range payload {
				numericPayload[j] = uint8(b)
			}
			userDataArray[i] = numericPayload
		}
		userDataField = userDataArray
		msg["cntDepend"] = true

		// Restore packetCnt array - handle both []int64 and []interface{}
		switch packetCnt := metadata["packetCnt"].(type) {
		case []int64:
			// Native []int64 from session storage
			packetCntArray := make([]interface{}, len(packetCnt))
			for i, cnt := range packetCnt {
				packetCntArray[i] = cnt
			}
			msg["packetCnt"] = packetCntArray
		case []interface{}:
			// JSON-decoded []interface{}
			packetCntArray := make([]interface{}, len(packetCnt))
			for i, cnt := range packetCnt {
				switch v := cnt.(type) {
				case float64:
					packetCntArray[i] = int64(v)
				case int64:
					packetCntArray[i] = v
				case int:
					packetCntArray[i] = int64(v)
				}
			}
			msg["packetCnt"] = packetCntArray
		}
	} else {
		// Non-counter-dependent: userData is single Numeric[n] or empty for ACK-only
		// BSSCI §5.12: "If user data is empty, a pure acknowledgement downlink is queued"
		if len(payloads) == 0 || (len(payloads) == 1 && len(payloads[0]) == 0) {
			// ACK-only downlink: empty userData
			userDataField = []interface{}{} // Empty Numeric[n]
		} else if len(payloads) == 1 {
			// Normal single payload
			singlePayload := make([]interface{}, len(payloads[0]))
			for i, b := range payloads[0] {
				singlePayload[i] = uint8(b)
			}
			userDataField = singlePayload // Direct Numeric[n] per BSSCI §3.12.1
		} else {
			// Multiple payloads in non-counter-dependent mode is invalid
			return nil, fmt.Errorf("%s, got %d", ResolveErrorMessage(errNonCounterDependentMultiPayload), len(payloads))
		}
	}
	msg["userData"] = userDataField

	// Restore optional MIOTY fields (canonical numeric coercion covers
	// float64, native integers, and the strict json.Number resume decode)
	if format, err := coerceInt64(metadata["format"]); err == nil && format > 0 {
		formatU8, errToken := safeUint8(format, "format")
		if errToken != "" {
			return nil, fmt.Errorf("%s: format=%d", ResolveErrorMessage(errToken), format)
		}
		msg["format"] = formatU8
	}
	if responseExp, ok := metadata["responseExp"].(bool); ok && responseExp {
		msg["responseExp"] = true
	}
	if responsePrio, ok := metadata["responsePrio"].(bool); ok && responsePrio {
		msg["responsePrio"] = true
	}
	if dlWindReq, ok := metadata["dlWindReq"].(bool); ok && dlWindReq {
		msg["dlWindReq"] = true
	}
	if expOnly, ok := metadata["expOnly"].(bool); ok && expOnly {
		msg["expOnly"] = true
	}

	return msg, nil
}

// reconstitueDLDataRevMessage reconstitutes a dlDataRev message from stored metadata
func (s *Server) reconstitueDLDataRevMessage(msg, metadata map[string]interface{}) (map[string]interface{}, error) {
	// Enforce command field
	msg["command"] = mioty.CmdDLDataRevoke

	// Restore epEui from metadata (mandatory field per BSSCI §3.13.1).
	// Canonical numeric coercion preserves the full uint64 EUI range under the
	// strict json.Number resume decode.
	if _, present := metadata["epEui"]; !present {
		return nil, fmt.Errorf("%s", ResolveErrorMessage(errMissingEpEuiInMetadata))
	}
	epEui, err := coerceUint64(metadata["epEui"])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errMissingEpEuiInMetadata), err)
	}
	msg["epEui"] = epEui

	// Restore queId as uint64 per canonical MIOTY type
	if _, present := metadata["queId"]; !present {
		return nil, fmt.Errorf("%s", ResolveErrorMessage(errMissingQueIDInMetadata))
	}
	queId, err := coerceUint64(metadata["queId"])
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

// ============================================================================
// DL RX Status Operations (MIOTY BSSCI v1.0.0 Sections 5.15-5.16)
// ============================================================================

// handleDLRXStatus handles dlRxStat from base station when endpoint reports DL reception quality
func (s *Server) handleDLRXStatus(_ *Server, session *Session, msg *Message, data map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	// Unmarshal directly into canonical MIOTY type per BSSCI §3.15.1
	var dlRxStatus mioty.DLRxStatus

	// Convert map to msgpack bytes for proper unmarshalling
	msgpackData, err := msgpack.Marshal(data)
	if err != nil {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errFailedToMarshal)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Unmarshal using canonical struct with proper field tags
	if err := msgpack.Unmarshal(msgpackData, &dlRxStatus); err != nil {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errFailedToDecode)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Set command type and operation ID
	dlRxStatus.CommandType = mioty.CmdDLRxStatus
	dlRxStatus.OpId = msg.OpId

	// Validate mandatory fields are present
	if dlRxStatus.EpEui == 0 {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxEpEui)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	if dlRxStatus.RxTime == 0 {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxTime)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Validate and extract mandatory packetCnt field per BSSCI §3.15.1
	packetCnt, hasPacketCnt := getNumericField(data, "packetCnt")
	if !hasPacketCnt {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxPacketCnt)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	// Validate packetCnt is within uint32 range (BSSCI §3.15.1 defines as unsigned)
	if packetCnt < 0 || packetCnt > math.MaxUint32 {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidDlRxPacketCnt)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	dlRxStatus.PacketCnt = uint32(packetCnt)

	// Validate and extract mandatory dlRxSnr field per BSSCI §3.15.1
	dlRxSnr, hasSnr := getFloatFieldValidated(data, "dlRxSnr")
	if !hasSnr {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxSnr)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	// Validate SNR is finite and within plausible range per BSSCI §5.15.1
	if errToken := validateFiniteFloat(dlRxSnr, mioty.DLRxSnrMinDB, mioty.DLRxSnrMaxDB); errToken != "" {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIDLRXStatusSNRValidationFailed,
			"epEui", dlRxStatus.EpEui,
			"dlRxSnr", dlRxSnr,
			"error", errToken)
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	dlRxStatus.DlRxSnr = dlRxSnr

	// Validate and extract mandatory dlRxRssi field per BSSCI §3.15.1
	dlRxRssi, hasRssi := getFloatFieldValidated(data, "dlRxRssi")
	if !hasRssi {
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingDlRxRssi)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	// Validate RSSI is finite and within plausible range per BSSCI §5.15.1
	if errToken := validateFiniteFloat(dlRxRssi, mioty.DLRxRssiMinDBm, mioty.DLRxRssiMaxDBm); errToken != "" {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIDLRXStatusRSSIValidationFailed,
			"epEui", dlRxStatus.EpEui,
			"dlRxRssi", dlRxRssi,
			"error", errToken)
		if err := s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}
	dlRxStatus.DlRxRssi = dlRxRssi

	// Create session context for tenant resolution and persistence
	ctx := s.sessionContext(session)

	s.logger.InfoContext(ctx, LogBSSCIReceivedDLRxStatFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"epEui", dlRxStatus.EpEui,
		"rxTime", dlRxStatus.RxTime,
		"packetCnt", dlRxStatus.PacketCnt,
		"dlRxSnr", dlRxStatus.DlRxSnr,
		"dlRxRssi", dlRxStatus.DlRxRssi)

	// Resolve endpoint owner tenant for roaming scenarios
	tenantID, err := s.resolveEndpointTenantID(ctx, session, dlRxStatus.EpEui)
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToResolveEndpointTenantForDLRXStatus,
			"error", err,
			"epEui", dlRxStatus.EpEui)
		if err := s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistDLRXStatus)); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendError), err)
		}
		return nil
	}

	// Resolve organization for owner tenant
	var ownerOrgUUID uuid.UUID
	if s.orgResolver != nil {
		ownerOrgUUID, err = s.orgResolver.GetDefaultOrgForTenant(ctx, tenantID)
		if err != nil {
			s.logger.WarnContext(ctx, LogBSSCIOrgLookupFailed, "tenantID", tenantID, "error", err)
			// Continue without org - ownerOrgUUID remains Nil
		}
	}

	// Build owner-scoped context with correct tenant AND organization
	ownerCtx := pkgcontext.WithTenantID(ctx, tenantID)
	if ownerOrgUUID != uuid.Nil {
		ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
	}

	// Convert endpoint EUI to bytes (used multiple times below)
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, dlRxStatus.EpEui)

	// Convert BS EUI to bytes for repository call
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

	// Check if this dlRxStat corresponds to a pending query (BSSCI §5.15 correlation)
	if s.dlrxStore != nil {
		// Correlate by tenant+endpoint only (oldest pending) per BSSCI §5.15
		// BS opId stored for audit but NOT used in WHERE clause (different namespace)
		found, err := s.dlrxStore.MarkDLRXStatusReceived(
			ownerCtx,
			tenantID,
			epEuiBytes, // Match by tenant + endpoint only
			bsEuiBytes, // Audit: which BS actually reported
			msg.OpId,   // Audit: BS opId (different namespace)
		)
		if err != nil {
			s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToCorrelateDLRXQuery,
				"epEui", dlRxStatus.EpEui,
				"opId", msg.OpId,
				"error", err)
			// Non-fatal: Continue processing even if correlation check fails
		}

		if !found {
			s.logger.WarnContext(ownerCtx, LogBSSCIUnsolicitedDLRXStatus,
				"epEui", dlRxStatus.EpEui,
				"tenant", tenantID,
				"bsEui", session.BaseStationEUI)
			// Log-only mode: continue processing unsolicited reports
		}
	}

	// Resolve endpoint owner's organization UUID (BSSCI §5.15)
	epOwnerOrgUUID, err := s.resolveOwnerOrgUUID(ownerCtx, tenantID, epEuiBytes)
	if err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToResolveEndpointOwnerOrgForDLRXStatus,
			"error", err,
			"tenant", tenantID,
			"epEui", dlRxStatus.EpEui)
		// Continue with nil org - backward compatible, but logged
	}

	// Store DL RX status in database under endpoint owner tenant + org
	if err := s.persistDLRXStatus(ownerCtx, tenantID, epOwnerOrgUUID, session.BaseStationEUI, dlRxStatus.EpEui, dlRxStatus.RxTime, dlRxStatus.PacketCnt, dlRxStatus.DlRxSnr, dlRxStatus.DlRxRssi); err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToPersistDLRxStatus,
			"error", err,
			"epEui", dlRxStatus.EpEui,
			"rxTime", dlRxStatus.RxTime,
			"tenantID", tenantID)
		// Continue processing even if persistence fails - don't break the protocol
	}

	// Record event
	if s.eventStore != nil {
		eventData := map[string]interface{}{
			"bsEui":            fmt.Sprintf("%016X", session.BaseStationEUI),
			"basestation_name": session.UserProvidedName,
			"epEui":            fmt.Sprintf("%016X", dlRxStatus.EpEui),
			"rxTime":           dlRxStatus.RxTime,
			"packetCnt":        dlRxStatus.PacketCnt,
			"dlRxSnr":          dlRxStatus.DlRxSnr,
			"dlRxRssi":         dlRxStatus.DlRxRssi,
			"opId":             msg.OpId,
			"timestamp":        time.Now().Format(time.RFC3339),
		}

		// Use owner-scoped context for event creation (roaming tenant isolation)
		if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
			TenantID:  fmt.Sprintf("%d", tenantID),
			EventType: EventTypeDLRxStatusReceived,
			Category:  models.EventCategoryMessage,
			Severity:  SeverityInfo,
			Title:     fmt.Sprintf(models.EventTitleDLRxStatus, session.UserProvidedName),
			Description: fmt.Sprintf("Endpoint %016X reported DL reception: SNR=%.1f dB, RSSI=%.1f dBm",
				dlRxStatus.EpEui, dlRxStatus.DlRxSnr, dlRxStatus.DlRxRssi),
			SourceType: mioty.SourceTypeEndpoint,
			SourceName: fmt.Sprintf("%016X", dlRxStatus.EpEui),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Details:    func() []byte { b, _ := json.Marshal(eventData); return b }(),
		}); err != nil {
			s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToRecordDLRxStatusEvent, "error", err)
		}
	}

	// Send dlRxStatRsp to acknowledge
	response := map[string]interface{}{
		"command": mioty.CmdDLRxStatusResponse,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, response)
}

// handleDLRXStatusResponse handles dlRxStatRsp from base station
func (s *Server) handleDLRXStatusResponse(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIReceivedDLRxStatRspFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	// Send dlRxStatCmp to complete the three-way handshake
	complete := map[string]interface{}{
		"command": mioty.CmdDLRxStatusComplete,
		"opId":    msg.OpId,
	}

	return s.sendMessage(session, complete)
}

// handleDLRXStatusComplete handles dlRxStatCmp from base station
func (s *Server) handleDLRXStatusComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.DebugContext(s.sessionContext(session), LogBSSCIDLRxStatusOperationCompleted,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	return nil
}

// SendDLRXStatusQuery sends a DL RX status query to request endpoint DL reception quality
func (s *Server) SendDLRXStatusQuery(sessionID string, epEui uint64) error {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the correlation and recovery records, then write
	// the frame. The counter is never rolled back.
	opId, err := s.beginScOperation(session)
	if err != nil {
		return err
	}

	// Create dlRxStatQry message per MIOTY spec Section 5.16.1
	msg := map[string]interface{}{
		"command": mioty.CmdDLRxStatusQuery,
		"opId":    opId,
		"epEui":   epEui,
	}

	// Create EUI bytes for storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// The correlation row and the recovery record must both be durable before
	// the frame is written; either persistence failure aborts the send.
	if err := s.persistDLRXQueryCorrelation(session, opId, epEui, euiBytes); err != nil {
		return err
	}
	if err := s.persistPendingOperation(session, opId, mioty.CmdDLRxStatusQuery, msg, euiBytes, nil); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLRxStatQryOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
		return err
	}

	if err := s.sendMessage(session, msg); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.removePendingOperation(session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationAfterSendFailure,
				"sessionID", session.DbSessionID,
				"opId", opId,
				"error", cleanupErr)
		}
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlRxStatQry), err)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISentDLRxStatQryToBaseStation,
		"sessionID", sessionID,
		"epEui", epEui,
		"opId", opId)

	return nil
}

// persistDLRXQueryCorrelation durably records the dl_rx_status_queries
// correlation row for an outgoing dlRxStatQry (BSSCI rev1 §5.16 / classic
// §3.16) under the endpoint owner's tenant. Correlation persistence failure is
// a pre-write failure: the caller must not put the query on the wire, because
// the eventual dlRxStat report could not be attributed. Owner-organization
// resolution failure alone stays non-fatal (nil org, backward compatible).
func (s *Server) persistDLRXQueryCorrelation(session *Session, opId int64, epEui uint64, euiBytes []byte) error {
	if s.dlrxStore == nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLRxStatQueryTracking,
			"sessionID", session.DbSessionID,
			"opId", opId,
			"epEui", epEui)
		return NewCatalogError(errFailedToPersistDLRxCorrelation, POSIX_EPROTO)
	}

	tenantID := resolvedTenant(session, s.tenantID)

	// Resolve endpoint owner's organization UUID (BSSCI §5.15)
	epOwnerOrgUUID, err := s.resolveOwnerOrgUUID(s.sessionContext(session), tenantID, euiBytes)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToResolveOwnerOrgForDLRxQuery,
			"error", err,
			"tenant", tenantID,
			"epEui", epEui)
		// Continue with nil org - backward compatible, but logged
	}

	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

	if err := s.dlrxStore.CreateDLRXStatusQuery(s.sessionContext(session), tenantID, epOwnerOrgUUID, euiBytes, bsEuiBytes, opId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistDLRxStatQueryTracking,
			"sessionID", session.DbSessionID,
			"opId", opId,
			"epEui", epEui,
			"error", err)
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistDLRxCorrelation), err)
	}

	return nil
}

// handleDLRXStatusQueryResponse handles dlRxStatQryRsp from base station
func (s *Server) handleDLRXStatusQueryResponse(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	if session == nil {
		return fmt.Errorf("%s", ResolveErrorMessage(errSessionNil))
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedDLRxStatQryRspFromBaseStation,
		"bsEui", session.BaseStationEUI,
		"opId", msg.OpId)

	// The service center completes its own SC-initiated dlRxStatQry operation
	// (BSSCI §3.16): it sends dlRxStatQryCmp and finalizes the pending
	// operation. A spec-compliant base station never returns dlRxStatQryCmp,
	// so the pending row is removed here or it leaks.
	complete := map[string]interface{}{
		"command": mioty.CmdDLRxStatusQueryComplete,
		"opId":    msg.OpId,
	}
	if err := s.sendMessage(session, complete); err != nil {
		return err
	}
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingOperationFromDatabase,
			"error", err, "opId", msg.OpId)
	}
	return nil
}

// resolveOwnerOrgUUID fetches the endpoint owner's organization UUID
// Returns nil UUID (backward compatible) if endpoint not found or orgResolver unavailable
func (s *Server) resolveOwnerOrgUUID(ctx context.Context, tenantID int64, epEuiBytes []byte) (*uuid.UUID, error) {
	// Community edition or no org resolver - return nil (backward compatible)
	if s.orgResolver == nil {
		return nil, nil
	}

	// Fetch endpoint to verify it exists and belongs to this tenant
	if s.endpointRepo == nil {
		return nil, fmt.Errorf("endpoint repository not available")
	}

	endpoint, err := s.endpointRepo.GetByEUI(ctx, tenantID, epEuiBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch endpoint: %w", err)
	}
	if endpoint == nil {
		return nil, fmt.Errorf("endpoint not found")
	}

	// Resolve org UUID for the endpoint's tenant (which may differ from BS tenant in roaming)
	orgUUID, err := s.orgResolver.GetDefaultOrgForTenant(ctx, endpoint.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve org for endpoint owner tenant %d: %w", endpoint.TenantID, err)
	}

	// Return pointer to UUID (nil if uuid.Nil for community mode)
	if orgUUID == uuid.Nil {
		return nil, nil
	}
	return &orgUUID, nil
}

// persistDLRXStatus stores a DL RX status report in the database under the endpoint owner tenant
func (s *Server) persistDLRXStatus(ctx context.Context, tenantID int64, ownerOrgUUID *uuid.UUID, bsEui uint64, epEui uint64, rxTime int64, packetCnt uint32, dlRxSnr float64, dlRxRssi float64) error {
	// Convert EUI to bytes
	epEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEuiBytes, epEui)

	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, bsEui)

	// Build canonical DLRXStatus using resolved endpoint owner tenant + org (BSSCI §5.15)
	status := &mioty.DLRXStatus{
		TenantID:       tenantID,
		OrganizationID: ownerOrgUUID, // Endpoint owner's org, not BS owner's
		EpEui:          epEuiBytes,
		BsEui:          bsEuiBytes,
		RxTime:         rxTime,
		PacketCnt:      packetCnt,
		DlRxSnr:        dlRxSnr,
		DlRxRssi:       dlRxRssi,
	}

	// Persist through the DL RX status store with session context
	if s.dlrxStore != nil {
		if err := s.dlrxStore.CreateDLRXStatus(ctx, status); err != nil {
			return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToPersistDLRXStatus), err)
		}

		s.logger.DebugContext(ctx, LogBSSCIPersistedDLRxStatus,
			"epEui", epEui,
			"rxTime", rxTime,
			"dlRxSnr", dlRxSnr,
			"dlRxRssi", dlRxRssi,
			"tenantID", tenantID)
	} else {
		s.logger.WarnContext(ctx, LogBSSCIMessageStoreNotAvailableForDLRxStatus,
			"epEui", epEui,
			"rxTime", rxTime)
	}

	return nil
}

// QueueDownlink dispatches a SCACI-queued downlink message (SCACI §3.10).
//
// This method satisfies the DownlinkScheduler interface defined in
// KC-Core/pkg/scheduler and is called by SCACI after the queue row has been
// persisted with status 'pending'. Delivery goes through the downlink
// dispatcher's DispatchQueue so both the immediate (SCACI) and deferred
// (dlOpen auto-dispatch) paths share one pending→reserved→queued lifecycle
// and both honor the dlRxStatQry pairing (SCACI §3.10.1, BSSCI rev1 §5.16 /
// classic §3.16).
//
// Returns:
//   - queuedQueId: Actual queue ID assigned (same as requested if no collision)
//   - bsEui: EUI of base station that will deliver the message
//   - error: scheduler.ErrSchedulerNoResources if temporary, scheduler.ErrSchedulerResourceMissing for permanent failures
func (s *Server) QueueDownlink(ctx context.Context, req *mioty.DLDataQueue, tenantID int64) (queuedQueId uint64, bsEui uint64, err error) {
	// Select appropriate bidirectional base station session
	sessionID, selectedBsEui, err := s.SelectBidirectionalSession(tenantID, nil)
	if err != nil {
		// Map BSSCI sentinels to scheduler contract sentinels (Dependency Inversion Principle)
		if errors.Is(err, ErrNoBidirectionalBaseStations) {
			return 0, 0, scheduler.ErrSchedulerNoResources
		}
		if errors.Is(err, ErrBaseStationUnavailable) {
			return 0, 0, scheduler.ErrSchedulerResourceMissing
		}
		return 0, 0, err
	}

	// Validate individual payload sizes (MIOTY radio protocol §4.3.2)
	if err := validatePayloadSizes(req.UserData); err != nil {
		return 0, 0, err
	}

	// Range guard for uint64 -> int64 conversion
	if req.QueId > math.MaxInt64 {
		return 0, 0, fmt.Errorf("%s: queId=%d", ResolveErrorMessage(errQueueIDOutOfRange), req.QueId)
	}

	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()
	if !exists {
		return 0, 0, scheduler.ErrSchedulerResourceMissing
	}

	if s.downlinkDispatcher == nil {
		return 0, 0, scheduler.ErrSchedulerNoResources
	}

	// Dispatch the exact persisted queue row; the dispatcher reserves it,
	// sends (with dlRxStatQry pairing when the row requests it), and confirms
	// reserved→queued.
	dispatched, err := s.downlinkDispatcher.DispatchQueue(ctx, tenantID, session.OrganizationID, session, req.QueId, req.EpEui)
	if err != nil {
		return 0, 0, err
	}
	if !dispatched {
		// No matching pending row: it was never persisted, already dispatched,
		// or revoked in the meantime.
		return 0, 0, scheduler.ErrSchedulerQueueNotFound
	}

	// Return the queue ID and selected base station EUI
	return req.QueId, selectedBsEui, nil
}

// RevokeDownlink implements DownlinkScheduler.RevokeDownlink to cancel a queued downlink
// message before delivery (SCACI §3.11 DL Data Revoke).
//
// This method:
//  1. Looks up the queue entry in the database to find which base station has it
//  2. Finds the base station session
//  3. Sends dlDataRev to the base station via BSSCI
//
// Returns:
//   - bsEui: EUI of base station that had the queue entry
//   - error: scheduler.ErrSchedulerQueueNotFound if entry doesn't exist, scheduler.ErrSchedulerResourceMissing if BS disconnected
func (s *Server) RevokeDownlink(tenantID int64, queId uint64) (bsEui uint64, err error) {
	// Look up the queue entry to find which base station and endpoint it's for
	ctx, cancel := context.WithTimeout(pkgcontext.WithTenantID(context.Background(), tenantID), dbconfig.DefaultQueryTimeout)
	defer cancel()
	tenantIDStr := fmt.Sprintf("%d", tenantID)

	var epEui uint64
	var sessionID string
	var foundSession *Session

	// Query database for queue entry
	if s.downlinkQueueStore != nil {
		// Use GetDownlinkByQueueID to find the queue entry
		downlink, err := s.downlinkQueueStore.GetDownlinkByQueueID(ctx, queId, tenantIDStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, storage.ErrNotFound) {
				// Map to scheduler contract sentinel
				return 0, scheduler.ErrSchedulerQueueNotFound
			}
			return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errDatabaseError), err)
		}

		// Parse EPEUI from hex string
		epEuiBytes, err := hex.DecodeString(downlink.EPEUI)
		if err != nil || len(epEuiBytes) != 8 {
			return 0, fmt.Errorf("%s", ResolveErrorMessage(errInvalidEndpointEUIFormat))
		}
		epEui = binary.BigEndian.Uint64(epEuiBytes)
		bsEui = downlink.BsEui

		// Find the base station session by EUI
		s.mu.RLock()
		for sid, sess := range s.sessions {
			if sess.BaseStationEUI == bsEui {
				sessionID = sid
				foundSession = sess
				break
			}
		}
		s.mu.RUnlock()

		if foundSession == nil {
			// Map to scheduler contract sentinel
			return bsEui, scheduler.ErrSchedulerResourceMissing
		}
	} else {
		// Map to scheduler contract sentinel
		return 0, scheduler.ErrSchedulerQueueNotFound
	}

	// Call existing SendDLDataRevoke to handle BSSCI protocol
	if err := s.SendDLDataRevoke(sessionID, epEui, queId); err != nil {
		return bsEui, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendDlDataRev), err)
	}

	return bsEui, nil
}
