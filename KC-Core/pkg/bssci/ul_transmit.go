package bssci

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler" // Import neutral scheduler contracts
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// ============================================================================
// Sentinel Errors for Base Station Selection and Scheduling
// ============================================================================
//
// These exported errors enable proper error classification across SCACI and gRPC layers,
// ensuring correct SCACI §3.9 error semantics. Use errors.Is() to detect these conditions
// and map them to appropriate POSIX codes or gRPC status codes.

var (
	// ErrNoBidirectionalBaseStations indicates no bidirectional base stations are available
	// for downlink transmission. This is a temporary condition - the AC should retry later.
	// SCACI mapping: POSIX_EAGAIN (11)
	// gRPC mapping: codes.Unavailable or codes.FailedPrecondition
	ErrNoBidirectionalBaseStations = errors.New("no bidirectional base stations available")

	// ErrBaseStationUnavailable indicates a specific requested base station is not available
	// (disconnected, not bidirectional, or handshake incomplete). This is resource-missing condition.
	// SCACI mapping: POSIX_ENOENT (2)
	// gRPC mapping: codes.NotFound or codes.FailedPrecondition
	ErrBaseStationUnavailable = errors.New("base station not available")

	// ErrQueueNotFound indicates the requested downlink queue entry does not exist or was already removed.
	// This can occur when attempting to revoke a queue entry that was never created, already completed,
	// or already revoked. This is a permanent error condition.
	// SCACI mapping: POSIX_ENOENT (2)
	// gRPC mapping: codes.NotFound
	ErrQueueNotFound = errors.New("queue entry not found")

	// ErrBaseStationTenantMismatch indicates the requested base station belongs to a different tenant.
	// This is a security/tenant-isolation boundary violation. The operation should not be retried
	// with the same base station EUI.
	// gRPC mapping: codes.PermissionDenied
	ErrBaseStationTenantMismatch = errors.New("base station belongs to different tenant")

	// ErrSessionNotFound indicates the requested base station session does not exist.
	// This occurs when attempting DL operations to a base station that is not connected.
	// gRPC mapping: codes.NotFound
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionNotReady indicates the session handshake is not complete (BSSCI §3.3).
	// This occurs when attempting DL operations before the base station has completed the connect handshake.
	// gRPC mapping: codes.FailedPrecondition
	ErrSessionNotReady = errors.New("session not ready")

	// ErrSessionNotBidirectional indicates the session does not support bidirectional operations.
	// This occurs when attempting DL operations on a base station that only supports uplink.
	// gRPC mapping: codes.FailedPrecondition
	ErrSessionNotBidirectional = errors.New("session not bidirectional")
)

// ============================================================================
// BSSCI 3.11 UL Data Transmit Operations (SC → BS)
// ============================================================================

// SendULDataTransmit initiates a Service Center to Base Station uplink data transmission
// per MIOTY BSSCI v1.0.0 Section 3.11
// Returns the operation ID and error
func (s *Server) SendULDataTransmit(sessionID string, epEui uint64, nwkSnKey []byte,
	shAddr uint16, packetCnt uint32, userData []byte, profile string, format uint8) (int64, error) {

	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// Check if connect handshake is complete (BSSCI-3.3-03)
	if !session.HandshakeComplete {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIConnectHandshakeNotComplete,
			"sessionID", sessionID,
			"bsEui", session.BaseStationEUI)
		return 0, fmt.Errorf("%s for session %s", ResolveErrorMessage(errHandshakeNotComplete), sessionID)
	}

	// Check if base station supports bidirectional communication
	if !session.Bidirectional {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIBaseStationDoesNotSupportBidi,
			"sessionID", sessionID,
			"bsEui", session.BaseStationEUI)
		return 0, fmt.Errorf("%s %016X", ResolveErrorMessage(errBaseStationNotBidirectional), session.BaseStationEUI)
	}

	// Validate key length
	if len(nwkSnKey) != 16 {
		return 0, fmt.Errorf("%s, got %d", ResolveErrorMessage(errNwkSnKeyInvalidLength), len(nwkSnKey))
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.beginScOperation(session)
	if err != nil {
		return 0, err
	}

	// Encrypt key for storage ONLY (not for transmission)
	var encryptedKeyForStorage string
	isEncrypted := false // Track whether the key was successfully encrypted
	if s.keyEncryptor != nil {
		encrypted, err := s.keyEncryptor.EncryptKey(nwkSnKey)
		if err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToEncryptNetworkSessionKey,
				"error", err,
				"sessionID", sessionID)
			// Production: fail operation if encryption fails (no plaintext fallback)
			return 0, fmt.Errorf("failed to encrypt network session key: %w", err)
		}
		encryptedKeyForStorage = encrypted
		isEncrypted = true
	} else {
		// No encryptor available - only allowed in dev mode with KILOCENTER_ALLOW_PLAINTEXT_KEYS=true
		// (NewServer returns nil if encryptor init fails in production)
		encryptedKeyForStorage = base64.StdEncoding.EncodeToString(nwkSnKey)
		isEncrypted = false
	}

	// Convert CLEAR key to Numeric[16] array for wire protocol per MIOTY spec
	nwkSnKeyArray := make([]interface{}, 16)
	for i := 0; i < 16; i++ {
		nwkSnKeyArray[i] = uint8(nwkSnKey[i])
	}

	// Convert userData to Numeric[n] array for wire protocol (BSSCI-3.11.1)
	userDataArray := make([]interface{}, len(userData))
	for i := 0; i < len(userData); i++ {
		userDataArray[i] = uint8(userData[i])
	}

	// Build ulDataTx message per MIOTY BSSCI spec Section 3.11.1
	msg := map[string]interface{}{
		"command":   mioty.CmdULDataTransmit,
		"opId":      opId,
		"epEui":     epEui,         // Numeric[8] per spec
		"nwkSnKey":  nwkSnKeyArray, // CLEAR key as Numeric[16] array
		"shAddr":    shAddr,        // uint16 per spec
		"packetCnt": packetCnt,     // uint32 per spec
		"userData":  userDataArray, // Numeric[n] array per spec
	}

	// Add optional fields
	if format > 0 {
		msg["format"] = format
	}
	if profile != "" {
		msg["profile"] = profile
	}

	// Create EUI bytes for storage
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	// Store in pendingOps with encrypted key and userData in metadata
	userDataB64 := base64.StdEncoding.EncodeToString(userData)
	metadata := map[string]interface{}{
		"packetCnt":    packetCnt,
		"shAddr":       shAddr,
		"format":       format,
		"profile":      profile,
		"encryptedKey": encryptedKeyForStorage, // Already base64 encoded
		"isEncrypted":  isEncrypted,            // Track encryption status
		"userData":     userDataB64,
		"data":         userDataB64, // Also store as "data" for PendingOperation.Data population
	}

	// Create sanitized message for persistence (no raw keys)
	sanitizedMsg := map[string]interface{}{
		"command":   mioty.CmdULDataTransmit,
		"opId":      opId,
		"epEui":     epEui,
		"shAddr":    shAddr,
		"packetCnt": packetCnt,
		"format":    format,
		// NO nwkSnKey here - it's in encrypted metadata
	}
	if profile != "" {
		sanitizedMsg["profile"] = profile
	}

	// The recovery record must be durable before the frame is written; a
	// persistence failure aborts the send, leaving only a consumed-ID gap.
	if err := s.persistPendingOperation(session, opId, mioty.CmdULDataTransmit,
		sanitizedMsg, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistULDataTxOperation,
			"sessionID", sessionID,
			"opId", opId,
			"error", err)
		return 0, err
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCISendingULDataTransmit,
		"sessionID", sessionID,
		"opId", opId,
		"epEui", epEui,
		"packetCnt", packetCnt,
		"shAddr", shAddr,
		"userDataLen", len(userData))

	// Send message with CLEAR key to base station
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
		return 0, err
	}

	return opId, nil
}

// handleULDataTxResponse handles the base station's response to ulDataTx
// per MIOTY BSSCI v1.0.0 Section 3.11.2
func (s *Server) handleULDataTxResponse(srv *Server, session *Session, msg *Message, data map[string]interface{}) error {
	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedULDataTransmitResponse,
		"sessionID", session.ID,
		"opId", msg.OpId)

	// Extract result code if present
	var resultCode int
	if result, ok := data["result"].(float64); ok {
		resultCode = int(result)
		s.logger.InfoContext(s.sessionContext(session), LogBSSCIULDataTransmitResponseResult,
			"sessionID", session.ID,
			"opId", msg.OpId,
			"result", resultCode)
	}

	// Keep pending operation for completion step
	// BSSCI §§5.11-5.12.3 Gap 1: Dual-path for test compatibility
	var pendingOp *PendingOperation
	// StatusService is the single path for pending operation persistence
	pendingOp, err := s.statusSvc.GetPendingOperation(session, msg.OpId)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
			"opId", msg.OpId, "error", err)
	}

	if pendingOp != nil {
		_ = pendingOp // Operation acknowledged; no additional processing needed
	}

	// BSSCI three-way handshake: Service Center must send ulDataTxCmp after ulDataTxRsp
	completionMsg := map[string]interface{}{
		"command": mioty.CmdULDataTransmitComplete,
		"opId":    msg.OpId,
	}

	s.logger.DebugContext(s.sessionContext(session), LogBSSCISendingULDataTransmitComplete,
		"sessionID", session.ID,
		"opId", msg.OpId)

	// Send the completion message and bail out early on error
	if err := s.sendMessage(session, completionMsg); err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendULDataTransmitComplete), err)
	}

	// Now perform the cleanup that would happen in handleULDataTxComplete
	return s.handleULDataTxComplete(srv, session, msg, data)
}

// handleULDataTxComplete handles the completion of the ulDataTx three-way handshake
// per MIOTY BSSCI v1.0.0 Section 3.11.3
func (s *Server) handleULDataTxComplete(_ *Server, session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.InfoContext(s.sessionContext(session), LogBSSCIReceivedULDataTransmitCompletion,
		"sessionID", session.ID,
		"opId", msg.OpId)

	// Remove from pending operations
	// BSSCI §§5.11-5.12.3 Gap 1: Dual-path for test compatibility
	var pendingOp *PendingOperation
	// StatusService is the single path for pending operation persistence
	pendingOp, err := s.statusSvc.GetPendingOperation(session, msg.OpId)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCIFailedToGetPendingOperation,
			"opId", msg.OpId, "error", err)
	}

	if pendingOp == nil {
		s.logger.WarnContext(s.sessionContext(session), LogBSSCINoPendingOperationForULTransmitComplete,
			"sessionID", session.ID,
			"opId", msg.OpId)
		return nil
	}

	// Remove from database pending operations using canonical helper
	// BSSCI §§5.11-5.12.3 Gap 1: StatusService handles both DB and memory cleanup
	if err := s.removePendingOperation(session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToRemovePendingULTransmitOperation,
			"sessionID", session.ID,
			"opId", msg.OpId,
			"error", err)
	}

	// Extract endpoint EUI for logging
	var epEui uint64
	if len(pendingOp.Endpoint) == 8 {
		epEui = binary.BigEndian.Uint64(pendingOp.Endpoint)
	}

	s.logger.InfoContext(s.sessionContext(session), LogBSSCIULDataTransmitThreeWayHandshakeCompleted,
		"sessionID", session.ID,
		"opId", msg.OpId,
		"epEui", epEui)

	return nil
}

// reconstitueULDataTxMessage reconstitutes a ulDataTx message from sanitized storage
// This is used during session resume to rebuild the message with the decrypted key
func (s *Server) reconstitueULDataTxMessage(sanitizedMsg map[string]interface{}, metadata map[string]interface{}, pendingOp *PendingOperation) (map[string]interface{}, error) {
	// Create enriched context for logging
	ctx := pkgcontext.WithTenantID(context.Background(), s.tenantID)

	// Start with the sanitized message
	msg := make(map[string]interface{})
	for k, v := range sanitizedMsg {
		msg[k] = v
	}

	// Extract and decrypt the key from metadata
	// Pending operations store keys in JSON metadata as strings (base64)
	if encKeyStr, ok := metadata["encryptedKey"].(string); ok {
		isEncryptedVal, _ := metadata["isEncrypted"].(bool)

		var clearKey []byte
		var err error

		if isEncryptedVal && s.keyEncryptor != nil {
			// Key is encrypted — decrypt with raw GCM
			encKeyBytes := []byte(encKeyStr)
			clearKey, err = s.keyEncryptor.DecryptKeyRaw(encKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecryptKey), err)
			}
		} else {
			// Key is plain base64
			clearKey, err = base64.StdEncoding.DecodeString(encKeyStr)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecode), err)
			}
		}

		// Convert to Numeric[16] array for wire protocol
		nwkSnKeyArray := make([]interface{}, 16)
		for i := 0; i < 16 && i < len(clearKey); i++ {
			nwkSnKeyArray[i] = uint8(clearKey[i])
		}
		msg["nwkSnKey"] = nwkSnKeyArray
	} else {
		return nil, fmt.Errorf("%s", ResolveErrorMessage(errMissingEncryptedKey))
	}

	// Restore userData from metadata, with fallback to PendingOperation.Data for legacy ops
	if userDataStr, ok := metadata["userData"].(string); ok {
		userData, err := base64.StdEncoding.DecodeString(userDataStr)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecodeUserData), err)
		}
		// Convert userData to Numeric[n] array for wire protocol (BSSCI-3.11.1)
		userDataArray := make([]interface{}, len(userData))
		for i := 0; i < len(userData); i++ {
			userDataArray[i] = uint8(userData[i])
		}
		msg["userData"] = userDataArray
	} else {
		// For legacy operations, userData might not be in metadata
		// Try to use PendingOperation.Data if available
		if pendingOp != nil && len(pendingOp.Data) > 0 {
			s.logger.InfoContext(ctx, LogBSSCIUserDataMissingFromMetadata,
				"dataLen", len(pendingOp.Data),
				"opId", pendingOp.OperationID)
			// Convert to Numeric[n] array for wire protocol (BSSCI-3.11.1)
			userDataArray := make([]interface{}, len(pendingOp.Data))
			for i := 0; i < len(pendingOp.Data); i++ {
				userDataArray[i] = uint8(pendingOp.Data[i])
			}
			msg["userData"] = userDataArray
		} else {
			// Canonical operation ID coercion covers legacy float64 values and
			// the strict json.Number resume decode
			opId, _ := parseOpID(sanitizedMsg["opId"])

			s.logger.WarnContext(ctx, LogBSSCIUserDataMissingFromBothSources,
				"opId", opId)
			// Empty array is valid for control telegrams (BSSCI-3.11.1)
			msg["userData"] = []interface{}{}
		}
	}

	return msg, nil
}

// SelectBidirectionalSession finds a suitable bidirectional base station session for UL transmit
//
// This helper is used by both SCACI (via ScheduleULDataTransmit interface) and gRPC
// to select an appropriate base station for uplink transmission.
//
// Tenant isolation: Only selects base stations that belong to the requesting tenant,
// preventing cross-tenant UL transmit operations (BSSCI §5.11 multi-tenant isolation).
//
// Parameters:
//   - tenantID: Tenant context for session selection (enforced via ResolvedTenantID comparison)
//   - targetBsEui: Optional specific base station EUI (nil = SC chooses automatically)
//
// Returns:
//   - sessionID: Internal session identifier
//   - actualBsEui: EUI of selected base station
//   - error: ErrBaseStationTenantMismatch if requested BS belongs to different tenant
func (s *Server) SelectBidirectionalSession(tenantID int64, targetBsEui *uint64) (sessionID string, actualBsEui uint64, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If specific BS requested, find it and validate tenant ownership
	if targetBsEui != nil {
		for sid, session := range s.sessions {
			if session.BaseStationEUI == *targetBsEui &&
				session.HandshakeComplete &&
				session.Bidirectional {
				// Enforce tenant isolation: check resolved tenant (supports roaming/resumed sessions)
				if session.ResolvedTenantID != tenantID {
					// Log tenant mismatch attempt for security auditing
					ctx := s.sessionContext(session)
					s.logger.WarnContext(ctx, LogBSSCIBaseStationTenantMismatch,
						"sessionID", sid,
						"tenantID", tenantID,
						"resolvedTenantID", session.ResolvedTenantID,
						"requestedBsEui", fmt.Sprintf("%016X", *targetBsEui),
					)
					return "", 0, ErrBaseStationTenantMismatch
				}
				return sid, session.BaseStationEUI, nil
			}
		}
		// Specific BS not found or not suitable - wrap sentinel for errors.Is() detection
		return "", 0, fmt.Errorf("%s %016X: %w", ResolveErrorMessage(errBaseStationNotRegistered), *targetBsEui, ErrBaseStationUnavailable)
	}

	// Otherwise, find all suitable sessions within the requesting tenant and select deterministically
	// SCACI §3.9.1 Production Readiness: Go map iteration is non-deterministic.
	// Collect candidates and select by lowest EUI for reproducible behavior.
	var candidateSid string
	var candidateBsEui uint64
	for sid, session := range s.sessions {
		if session.HandshakeComplete &&
			session.Bidirectional &&
			session.ResolvedTenantID == tenantID {
			// First candidate or lower EUI wins (deterministic fallback selection)
			if candidateSid == "" || session.BaseStationEUI < candidateBsEui {
				candidateSid = sid
				candidateBsEui = session.BaseStationEUI
			}
		}
	}

	if candidateSid != "" {
		return candidateSid, candidateBsEui, nil
	}

	// No suitable sessions found for this tenant - return sentinel directly for errors.Is() detection
	return "", 0, ErrNoBidirectionalBaseStations
}

// FindSessionForEndpointAttachment implements SessionDirectory.FindSessionForEndpointAttachment
//
// This method supports roaming scenarios by locating a base station session without
// enforcing tenant ownership. The caller MUST validate endpoint ownership before calling.
//
// SECURITY: This method assumes the caller (e.g., QueryDLRXStatus via GetEndpointBaseStation)
// has already validated that the requesting tenant owns the endpoint. It finds the base station
// session regardless of which tenant owns the base station, enabling roaming support:
// "All base stations on a server form a shared RF mesh. Any station accepts traffic from any endpoint."
//
// Thread-safety: Uses RLock for concurrent read access to session map.
//
// Validates:
//   - HandshakeComplete (BSSCI §3.3 requirement)
//   - Bidirectional capability (DL operation requirement)
//   - Session exists
//
// Returns:
//   - sessionID: Internal session identifier
//   - error: ErrSessionNotFound, ErrSessionNotReady, ErrSessionNotBidirectional
func (s *Server) FindSessionForEndpointAttachment(bsEui uint64) (sessionID string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Search for session with matching BS EUI
	for sid, session := range s.sessions {
		if session.BaseStationEUI == bsEui {
			// Validate session handshake is complete per BSSCI §3.3
			if !session.HandshakeComplete {
				ctx := s.sessionContext(session)
				s.logger.WarnContext(ctx, LogBSSCISessionNotReady,
					"sessionID", sid,
					"bsEui", fmt.Sprintf("%016X", bsEui),
					"reason", "handshake not complete")
				return "", fmt.Errorf("%s %016X: %w",
					ResolveErrorMessage(errSessionNotReady), bsEui, ErrSessionNotReady)
			}

			// Validate session supports bidirectional operations
			if !session.Bidirectional {
				ctx := s.sessionContext(session)
				s.logger.WarnContext(ctx, LogBSSCISessionNotBidirectional,
					"sessionID", sid,
					"bsEui", fmt.Sprintf("%016X", bsEui),
					"bidirectional", session.Bidirectional)
				return "", fmt.Errorf("%s %016X: %w",
					ResolveErrorMessage(errSessionNotBidirectional), bsEui, ErrSessionNotBidirectional)
			}

			// Session is suitable for DL operations
			return sid, nil
		}
	}

	// Base station not found in connected sessions
	s.logger.DebugContext(s.ctx, LogBSSCISessionNotFound,
		"bsEui", fmt.Sprintf("%016X", bsEui),
		"reason", "not in connected sessions")
	return "", fmt.Errorf("%s %016X: %w",
		ResolveErrorMessage(errSessionNotFound), bsEui, ErrSessionNotFound)
}

// ScheduleULDataTransmit schedules an uplink transmission via SCACI interface
//
// This method satisfies the ULTransmitScheduler interface defined in KC-Core/pkg/scaci/server.go
// and is called by SCACI when an Application Center initiates an uplink transmission (SCACI §3.9).
//
// Parameters:
//   - tenantID: Tenant context for session selection
//   - req: UL data transmit request from SCACI containing endpoint details and payload
//   - targetBsEui: Optional specific base station EUI (nil = SC chooses automatically)
//
// Returns:
//   - opId: BSSCI operation ID for tracking
//   - actualBsEui: EUI of base station that will transmit
//   - error: scheduler.ErrSchedulerNoResources if temporary failure, scheduler.ErrSchedulerResourceMissing for permanent failures
func (s *Server) ScheduleULDataTransmit(tenantID int64, req *mioty.ULDataTransmit, targetBsEui *uint64) (opId int64, actualBsEui uint64, err error) {
	// Select appropriate base station session
	sessionID, bsEui, err := s.SelectBidirectionalSession(tenantID, targetBsEui)
	if err != nil {
		// Map BSSCI sentinels to scheduler contract sentinels (Dependency Inversion Principle)
		if errors.Is(err, ErrNoBidirectionalBaseStations) {
			// Temporary condition - AC should retry
			return 0, 0, scheduler.ErrSchedulerNoResources
		}
		if errors.Is(err, ErrBaseStationUnavailable) {
			// Permanent condition for current parameters
			return 0, 0, scheduler.ErrSchedulerResourceMissing
		}
		// Other errors pass through
		return 0, 0, err
	}

	// Convert nwkSnKey from [16]byte to []byte
	nwkSnKeyBytes := req.NwkSnKey[:]

	// Handle optional format (default to 0)
	format := uint8(0)
	if req.Format != nil {
		format = *req.Format
	}

	// Handle optional profile (default to empty string)
	profile := ""
	if req.Profile != nil {
		profile = *req.Profile
	}

	// Fan into existing SendULDataTransmit method
	opId, err = s.SendULDataTransmit(
		sessionID,
		req.EpEui,
		nwkSnKeyBytes,
		req.ShAddr,
		req.PacketCnt,
		req.UserData,
		profile,
		format,
	)

	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToSendULTransmit), err)
	}

	return opId, bsEui, nil
}
