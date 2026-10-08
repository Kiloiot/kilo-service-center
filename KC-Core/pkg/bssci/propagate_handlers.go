package bssci

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
)

// reconstitueAttachPropagateMessage rebuilds the attachPropagate wire message
// from its sanitized recovery record: the cleartext network key was stripped
// before persistence and lives only as an encrypted envelope in metadata, so
// resume decrypts it and restores the Numeric[16] key array. A record
// persisted before key sanitization carries no envelope and still holds the
// key in the message itself, so it is returned as-is.
func (s *Server) reconstitueAttachPropagateMessage(sanitizedMsg, metadata map[string]interface{}) (map[string]interface{}, error) {
	msg := make(map[string]interface{}, len(sanitizedMsg)+1)
	for k, v := range sanitizedMsg {
		msg[k] = v
	}

	encKeyStr, ok := metadata["encryptedKey"].(string)
	if !ok {
		if _, hasKey := msg[wireFieldNwkSnKey]; hasKey {
			return msg, nil
		}
		return nil, fmt.Errorf("%s", ResolveErrorMessage(errMissingEncryptedKey))
	}
	if s.cipher == nil {
		return nil, errCipherRequired
	}
	// Current records carry the text-envelope form shared with ulDataTx;
	// records written before the unification carry base64 of the binary
	// envelope.
	var clearKey []byte
	var err error
	if keycrypto.IsTextEnvelope(encKeyStr) {
		clearKey, err = s.cipher.DecryptString(encKeyStr)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecryptKey), err)
		}
	} else {
		envelope, decodeErr := base64.StdEncoding.DecodeString(encKeyStr)
		if decodeErr != nil {
			return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecode), decodeErr)
		}
		clearKey, err = s.cipher.Decrypt(envelope)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToDecryptKey), err)
		}
	}

	nwkSnKeyArray := make([]interface{}, dbconfig.SessionKeySize)
	for i := 0; i < dbconfig.SessionKeySize && i < len(clearKey); i++ {
		nwkSnKeyArray[i] = uint8(clearKey[i])
	}
	msg[wireFieldNwkSnKey] = nwkSnKeyArray
	return msg, nil
}

// propagateRecordOwner reads the endpoint owner a propagate recovery record
// was sent for; found is false when the record names no tenant. An absent or
// unparsable organization yields uuid.Nil, the community default.
func propagateRecordOwner(metadata map[string]interface{}) (tenantID int64, found bool, org uuid.UUID) {
	tenantID, found = getNumericField(metadata, metadataKeyTenantID)
	if orgText, ok := metadata[metadataKeyOrganizationID].(string); ok {
		if parsed, err := uuid.Parse(orgText); err == nil {
			org = parsed
		}
	}
	return tenantID, found, org
}

// attachPropagateFields are the attPrp field values a recovery record holds.
type attachPropagateFields struct {
	shortAddr     uint16
	bidirectional bool
	lastPacketCnt uint32
	dualChannel   bool
	repetition    bool
	wideCarrOff   bool
	longBlkDist   bool
}

// decodeAttachPropagateFields reads the sent attPrp fields back through the
// canonical numeric coercion: the live cache holds the writer's Go types, a
// record reloaded from the database holds JSON numbers.
func (s *Server) decodeAttachPropagateFields(ctx context.Context, opID int64, metadata map[string]interface{}) attachPropagateFields {
	fields := attachPropagateFields{
		bidirectional: getBoolField(metadata, metadataKeyBidirectional, false),
		dualChannel:   getBoolField(metadata, metadataKeyDualChannel, false),
		repetition:    getBoolField(metadata, metadataKeyRepetition, false),
		wideCarrOff:   getBoolField(metadata, metadataKeyWideCarrOff, false),
		longBlkDist:   getBoolField(metadata, metadataKeyLongBlkDist, false),
	}
	var errToken string
	if fields.shortAddr, errToken = recordUnsigned(metadata, metadataKeyShortAddr, safeUint16); errToken != "" {
		s.logInvalidRecordField(ctx, opID, metadataKeyShortAddr, errToken)
	}
	if fields.lastPacketCnt, errToken = recordUnsigned(metadata, metadataKeyLastPacketCnt, safeUint32); errToken != "" {
		s.logInvalidRecordField(ctx, opID, metadataKeyLastPacketCnt, errToken)
	}
	return fields
}

// recordUnsigned reads a mandatory unsigned recovery-record field through the
// canonical numeric coercion and the bounded conversion for its width.
func recordUnsigned[T uint16 | uint32](metadata map[string]interface{}, key string, convert func(int64) (T, string)) (T, string) {
	raw, ok := getNumericField(metadata, key)
	if !ok {
		return 0, errMandatoryFieldMissing
	}
	return convert(raw)
}

func (s *Server) logInvalidRecordField(ctx context.Context, opID int64, field, errToken string) {
	s.logger.WarnContext(ctx, LogBSSCIInvalidAttachPropagateRecordField,
		logger.FieldOpID, opID,
		logger.FieldField, field,
		logger.FieldErrorTokenSnake, errToken)
}

// SendAttachPropagateToSession sends one session an endpoint's attachment,
// normalizing nullable endpoint fields before delegating to SendAttachPropagate
//
// BSSCI §5.8-5.8.3: Automatic endpoint propagation
func (s *Server) SendAttachPropagateToSession(
	ctx context.Context,
	session *Session,
	endpoint *models.EndPoint,
) error {
	// Normalize nullable short address (default 0 when nil)
	var shAddr uint16
	if endpoint.ShAddr != nil {
		shAddr = *endpoint.ShAddr
	}

	// LastPacketCnt is uint32 (not nullable) - direct access
	lastPkt := endpoint.LastPacketCnt

	nwkSnKey, err := s.sessionKeys.NetworkSessionKey(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("%s: %w", ResolveErrorMessage(errNetworkSessionKeyUnavailable), err)
	}
	if len(nwkSnKey) != dbconfig.SessionKeySize {
		return fmt.Errorf(errFmtNwkSnKeyWrongLength, endpoint.ID, len(nwkSnKey))
	}

	// Convert boolean to uint8 (inline pattern from existing code)
	repetition := uint8(0)
	if endpoint.Repetition {
		repetition = uint8(1)
	}

	// Delegate to existing SendAttachPropagate with all validated fields
	return s.SendAttachPropagate(
		session.ID,
		endpoint.EUI.ToUint64(),
		nwkSnKey,
		shAddr,
		endpoint.Bidi,
		lastPkt,
		endpoint.DualChan,
		repetition,
		endpoint.WideCarrOff,
		endpoint.LongBlkDist,
	)
}

// SendAttachPropagateBySessionID implements interface method for propagation service
// Looks up session by ID and delegates to SendAttachPropagateToSession
//
// BSSCI §5.8-5.8.3: Automatic endpoint propagation
func (s *Server) SendAttachPropagateBySessionID(
	ctx context.Context,
	sessionID string,
	endpoint *models.EndPoint,
) error {
	// Session-specific propagation (BSSCI §5.8.3)
	session, exists := s.sessions.get(sessionID)

	if !exists {
		return fmt.Errorf(errFmtSessionNotFound, sessionID)
	}

	// Delegate to SendAttachPropagateToSession with context as first parameter
	return s.SendAttachPropagateToSession(ctx, session, endpoint)
}

// SendAttachPropagate sends an attach propagate command to a specific session
func (s *Server) SendAttachPropagate(sessionID string, endpointEUI uint64, nwkSnKey []byte,
	shortAddr uint16, bidirectional bool, lastPacketCnt uint32, dualChannel bool,
	repetition uint8, wideCarrOff bool, longBlkDist bool,
) error {
	// BSSCI-3.8.1-01: Validate nwkSnKey is exactly 16 bytes
	if len(nwkSnKey) != 16 {
		return fmt.Errorf(errFmtTokenGotBytes, ResolveErrorMessage(errInvalidNwkSnKeyLength), len(nwkSnKey))
	}

	session, exists := s.sessions.get(sessionID)

	ctx := s.sessionContext(session)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// BSSCI-3.3-03: Don't send operations to sessions that haven't completed handshake
	if !session.HandshakeComplete {
		return fmt.Errorf(errFmtTokenForSession, ResolveErrorMessage(errHandshakeNotComplete), sessionID)
	}

	// Check if we should propagate to this base station
	// For downlink to work, BOTH the base station AND endpoint must support bidirectional
	// If the endpoint doesn't support bidirectional, no point propagating to any base station
	// If the base station doesn't support bidirectional, surface the issue to operators
	if bidirectional && !session.Bidirectional {
		s.logger.WarnContext(s.safeCtx(), LogBSSCICannotAttachPropagateToNonBidiBaseStation,
			logger.FieldSessionID, sessionID,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldEpEui, endpointEUI)

		s.recordNonBidiAttachFailure(ctx, session, endpointEUI)

		return fmt.Errorf("%s: %s", ResolveErrorMessage(errBaseStationNotBidirectional), mioty.FormatEUI64(session.BaseStationEUI))
	}

	// Generate per-session SC operation ID with atomic decrement (BSSCI §5.2)
	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	// Convert repetition uint8 to boolean (non-zero means repetition enabled)
	repetitionBool := repetition > 0

	// Convert nwkSnKey from []byte to []interface{} array per MIOTY spec (Numeric[16])
	// MIOTY spec requires this as Numeric[16], not Binary
	// Use interface{} slice to ensure proper MessagePack encoding as array
	nwkSnKeyArray := make([]interface{}, 16)
	for i := 0; i < dbconfig.SessionKeySize && i < len(nwkSnKey); i++ {
		nwkSnKeyArray[i] = uint8(nwkSnKey[i])
	}

	// Build attach propagate message per BSSCI v1.0.0 spec
	message := map[string]interface{}{
		"command":       mioty.CmdAttachPropagate,
		"opId":          opId,
		"epEui":         endpointEUI, // MUST be Numeric[8] per BSSCI spec, NOT string!
		"bidi":          bidirectional,
		"nwkSnKey":      nwkSnKeyArray, // Numeric[16] array per MIOTY spec
		"shAddr":        shortAddr,     // Keep as uint16 per MIOTY spec
		"lastPacketCnt": lastPacketCnt,
		"dualChan":      dualChannel,
		"repetition":    repetitionBool,
		"wideCarrOff":   wideCarrOff,
		"longBlkDist":   longBlkDist,
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingAttachPropagate,
		logger.FieldSessionID, sessionID,
		logger.FieldEndpointEuiCamel, endpointEUI,
		logger.FieldShortAddr, shortAddr,
		logger.FieldBidirectional, bidirectional)

	// Encrypt the network session key for at-rest storage. The recovery record
	// and audit message never carry the cleartext key: it lives only as an
	// envelope in the pending-operation metadata, from which resume
	// reconstitutes the wire message.
	encryptedKeyEnvelope, err := s.encryptNwkKeyEnvelope(nwkSnKey)
	if err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToEncryptNetworkSessionKey,
			logger.FieldError, err, logger.FieldEpEui, endpointEUI)
		if rmErr := s.pendingOps.remove(s.sessionContext(session), session, opId); rmErr != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, rmErr)
		}
		return err
	}
	// The recovery metadata carries the same key in the text-envelope form
	// shared with the ulDataTx recovery record, so both resume readers use
	// one format.
	encryptedKeyText, err := s.cipher.EncryptString(nwkSnKey)
	if err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToEncryptNetworkSessionKey,
			logger.FieldError, err, logger.FieldEpEui, endpointEUI)
		if rmErr := s.pendingOps.remove(s.sessionContext(session), session, opId); rmErr != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, rmErr)
		}
		return err
	}

	// Resolve owner tenant + ctx before metadata (BSSCI §5.8.3 multi-tenant roaming).
	owner, err := s.resolvePropagationOwner(ctx, session, endpointEUI)
	if err != nil {
		return err
	}
	euiBytes, endpointTenantID, ownerOrgUUID, ownerCtx := owner.euiBytes, owner.tenantID, owner.orgID, owner.ctx

	// Create metadata with owner tenant/org info (BSSCI §5.8.3). The encrypted
	// key rides here so the sanitized recovery record can be reconstituted on
	// resume without ever persisting the cleartext key.
	metadata := map[string]interface{}{
		"epEui":                  mioty.FormatEUI64(endpointEUI),
		metadataKeyTenantID:      endpointTenantID,
		metadataKeyShortAddr:     shortAddr,
		metadataKeyBidirectional: bidirectional,
		metadataKeyLastPacketCnt: lastPacketCnt,
		metadataKeyDualChannel:   dualChannel,
		metadataKeyRepetition:    repetitionBool, // Store as bool per BSSCI §5.8.1
		metadataKeyWideCarrOff:   wideCarrOff,
		metadataKeyLongBlkDist:   longBlkDist,
		"encryptedKey":           encryptedKeyText,
	}
	if ownerOrgUUID != uuid.Nil {
		metadata[metadataKeyOrganizationID] = ownerOrgUUID.String()
	}

	// The persisted recovery record strips the cleartext key; the wire message
	// keeps it. Resume rebuilds the key from the encrypted metadata.
	sanitizedMessage := make(map[string]interface{}, len(message))
	for k, v := range message {
		if k == wireFieldNwkSnKey {
			continue
		}
		sanitizedMessage[k] = v
	}

	// The recovery record must be durable before the frame is written; a
	// persistence failure aborts the send, leaving only a consumed-ID gap.
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdAttachPropagate, sanitizedMessage, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperation, logger.FieldError, err)
		return err
	}

	// Update endpoint in database with attach propagate information
	if endpoint := owner.endpoint; endpoint != nil {
		updates := models.EndpointAttachSessionParams{
			PropagatedAt:  s.clock.Now(),
			ShAddr:        shortAddr,
			Bidi:          bidirectional,
			LastPacketCnt: lastPacketCnt,
			DualChan:      dualChannel,
			Repetition:    repetitionBool, // Use bool per BSSCI §5.8.1
			WideCarrOff:   wideCarrOff,
			LongBlkDist:   longBlkDist,
		}

		// The attachment persister owns the owner-tenant transaction;
		// error strings propagate to the caller unchanged. The session key
		// is handed over in cleartext: the persistence layer encrypts it at
		// rest through its cipher (BSSCI §5.6.2).
		if err := s.attachPersistence.PersistAttachPropagateSession(ownerCtx, AttachPropagateSessionRecord{
			TenantID:        endpointTenantID,
			EndpointID:      endpoint.ID,
			EndpointUpdates: updates,
			EncryptedKey:    nwkSnKey,
			ShAddr:          shortAddr,
			BaseStationEUI:  session.BaseStationEUIBytes(),
		}); err != nil {
			return err
		}

		s.logger.DebugContext(s.safeCtx(), LogBSSCIUpdatedEndpointWithAttachPropagateInfo,
			logger.FieldEpEui, endpointEUI,
			logger.FieldShortAddr, shortAddr)
	} else {
		s.logger.DebugContext(s.safeCtx(), LogBSSCIEndpointNotFoundInDatabaseForAttachPropagate,
			logger.FieldEpEui, endpointEUI)
	}

	// NOTE: Event creation moved to RecordAttachPropagate (called on successful message persistence)
	// which creates a single attPrp event. Removed duplicate attach_propagate_initiated events
	// that were creating 2 extra records per operation.

	if err := s.sendMessage(session, message); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}

		return err
	}

	// BSSCI §5.8.3: Persist attach propagate message to mioty_messages for audit trail
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

	var orgUUIDStr *string
	if ownerOrgUUID != uuid.Nil {
		orgStr := ownerOrgUUID.String()
		orgUUIDStr = &orgStr
	}

	// The audit message stores the same encrypted envelope built above for
	// the recovery record; the cleartext network key is never persisted
	// (production security requirement - BSSCI §5.6.2).
	propagateMsg := &mioty.AttachPropagateMessage{
		CommandType:   mioty.CmdAttachPropagate,
		OpId:          opId,
		EpEui:         endpointEUI,
		Bidi:          bidirectional,
		NwkSnKey:      encryptedKeyEnvelope, // Encrypted envelope for storage
		ShAddr:        shortAddr,
		LastPacketCnt: lastPacketCnt,
		DualChan:      dualChannel,
		Repetition:    repetitionBool,
		WideCarrOff:   wideCarrOff,
		LongBlkDist:   longBlkDist,
		// Metadata fields
		BasestationEui: bsEuiBytes,
		TenantID:       endpointTenantID, // Endpoint owner tenant (BSSCI §5.8.3 roaming)
		OrgUUID:        orgUUIDStr,       // Endpoint owner org (roaming)
		MessageType:    mioty.MessageTypeAttachPropagate,
		Direction:      mioty.DirectionDownlink,
		InterfaceType:  mioty.InterfaceBSSCI,
	}

	if s.protocolMessages != nil {
		if err := s.protocolMessages.CreateAttachPropagateMessage(ownerCtx, propagateMsg); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogBSSCIFailedToPersistAttachPropagateMessage,
				logger.FieldError, err,
				logger.FieldEpEui, endpointEUI,
				logger.FieldBsEui, session.BaseStationEUI,
				logger.FieldOpID, opId)
			// Continue - persistence failure shouldn't block operation
		}
	}

	return nil
}

// handleAttachPropagateResponse handles attach propagate response from base station
func (s *Server) handleAttachPropagateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// BSSCI 3.8.2: attPrpRsp only carries command and opId. No result field per spec.
	// Default to 0 (success) if result field is missing - spec-compliant behavior.
	result := getNumericFieldInt(data, "result", 0)

	s.logger.DebugContext(s.safeCtx(), LogBSSCIAttachPropagateResponseReceived,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId,
		logger.FieldResult, result)

	// Result codes: 0 = success, non-zero = error
	if result != 0 {
		// Attach propagate failed - DO NOT send completion or update endpoint
		return s.handlePropagateResponseFailure(ctx, session, msg, result, propagateResponseConfig{
			rejectedLog:     LogBSSCIAttachPropagateRejectedByBaseStation,
			failureErrToken: errAttachPropagateFailed,
			operationType:   EventTypeAttachPropagateFailed,
			eventType:       EventTypeEndpointAttachFailed,
			titleFormat:     TitleAttachPropagateFailedForEndpointOnBS,
		})
	}

	// Success case - proceed with three-way handshake completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIAttachPropagateAcceptedByBaseStation,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// BSSCI three-way handshake: Service Center must send attPrpCmp after successful attPrpRsp
	completionMsg := map[string]interface{}{
		"command": mioty.CmdAttachPropagateComplete,
		"opId":    msg.OpId, // Use same operation ID
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingAttachPropagateComplete,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Send the completion message
	if err := s.sendMessage(session, completionMsg); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendAttachPropagateComplete,
			logger.FieldBaseStation, session.BaseStationEUI,
			logger.FieldOpID, msg.OpId,
			logger.FieldError, err)
		// Keep op in pending for retry
		return err
	}

	// Now perform the cleanup that would happen in handleAttachPropagateComplete
	// Since WE send the Cmp (not receive it), we need to do the cleanup here
	return s.handleAttachPropagateComplete(session, msg, data)
}

// attachPropagateCompletion describes a completed attach propagate for the
// station's tenant, naming the endpoint only when that tenant owns it.
func (s *Server) attachPropagateCompletion(session *Session, opID int64, pendingOp *PendingOperation) string {
	station := mioty.FormatEUI64(session.BaseStationEUI)
	if pendingOp == nil {
		return fmt.Sprintf(eventDescFmtAttachPropagateDone, station, opID)
	}
	epEUI, hasEUI := parseMetadataEUI(pendingOp.Metadata[models.EventDetailKeyEpEui])
	owner, hasOwner, _ := propagateRecordOwner(pendingOp.Metadata)
	if !hasEUI || !hasOwner || owner != resolvedTenant(session, s.tenantID) {
		return fmt.Sprintf(eventDescFmtAttachPropagateDone, station, opID)
	}
	return fmt.Sprintf(eventDescFmtEndpointPropagateDone, mioty.FormatEUI64(epEUI), station, opID)
}

// handleAttachPropagateComplete performs cleanup after attach propagate completion
// This is called after we send attPrpCmp (Service Center sends it, not base station)
func (s *Server) handleAttachPropagateComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	ctx := s.sessionContext(session)

	s.logger.InfoContext(s.safeCtx(), LogBSSCIAttachPropagateCompleted,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// BSSCI-3.8.3-01: Get pending operation BEFORE removing it to access metadata
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
	pendingOp := s.pendingOperationOrWarn(session, msg.OpId)

	// Remove completed operation from pending operations for MIOTY session resume
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation,
			logger.FieldError, err,
			logger.FieldOpID, msg.OpId,
			logger.FieldSessionID, session.DbSessionID)
	}

	// BSSCI §3.8.3: Record completion event UNCONDITIONALLY for audit trail
	// This is distinct from RecordAttachPropagate (EventTypeAttachPropagateInitiated)
	// which is called in the pendingOp block for endpoint-specific tracking
	// NOTE: Endpoint EUI extracted from pendingOp below; this block may lack epEUI
	completionEvent := &models.SystemEvent{
		TenantID:    strconv.FormatInt(resolvedTenant(session, s.tenantID), 10),
		EventType:   EventTypeAttachPropagateCompleted,
		Title:       TitleAttachPropagateCompleted,
		Description: s.attachPropagateCompletion(session, msg.OpId, pendingOp),
		Severity:    SeverityInfo,
		Category:    mioty.CategoryEndpoint, // Use endpoint category for UI visibility
		SourceType:  mioty.SourceTypeEndpoint,
		SourceName:  mioty.FormatEUI64(session.BaseStationEUI),
		Status:      EventStatusNew,
		CreatedAt:   s.clock.Now(),
	}

	if err := s.eventStore.CreateEvent(ctx, completionEvent); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateCompletionEvent,
			logger.FieldError, err,
			logger.FieldOpID, msg.OpId)
	}

	// Update endpoint to track which base station accepted the attachment
	if pendingOp != nil && pendingOp.Metadata != nil {
		// Extract owner tenant/org from metadata for roaming support
		ownerTenantID, hasOwnerTenant, ownerOrg := propagateRecordOwner(pendingOp.Metadata)
		if !hasOwnerTenant {
			ownerTenantID = resolvedTenant(session, s.tenantID)
			s.logger.WarnContext(ctx, LogBSSCIMissingTenantInMetadata, logger.FieldOpID, msg.OpId)
		}
		sent := s.decodeAttachPropagateFields(ctx, msg.OpId, pendingOp.Metadata)

		ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)
		if ownerOrg != uuid.Nil {
			ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
		}

		// Extract endpoint EUI from pending operation metadata
		// Handle both uint64 (direct) and float64 (from JSON deserialization)
		var epEUI uint64
		var hasEUI bool

		epEUI, hasEUI = parseMetadataEUI(pendingOp.Metadata[models.EventDetailKeyEpEui])

		if hasEUI {
			// Convert endpoint EUI to bytes
			epEUIBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(epEUIBytes[:], epEUI)

			// Get endpoint to update attachment state
			owner, err := s.endpointOwners.ResolveOwner(ownerCtx, storedEUI(epEUI))
			if err != nil {
				s.logger.WarnContext(ownerCtx, LogBSSCIEndpointNotFoundForPropagate, logger.FieldEpEui, epEUI)
				return nil
			}
			endpoint := owner.Endpoint
			// Re-assign to actual endpoint tenant (metadata may be stale)
			ownerTenantID = owner.TenantID

			// Re-resolve organization for the actual tenant (metadata org may be wrong);
			// a failed lookup keeps the metadata org.
			newOrg, orgErr := s.orgResolver.GetDefaultOrgForTenant(ctx, ownerTenantID)
			if orgErr != nil {
				s.logger.WarnContext(ctx, LogBSSCIOrgLookupFailed,
					logger.FieldTenantID, ownerTenantID,
					logger.FieldError, orgErr,
					logger.FieldContext, contextAttachPropagateFallback,
					logger.FieldFallbackOrg, ownerOrg.String())
			} else {
				ownerOrg = newOrg
			}

			// Rebuild context with actual tenant and re-resolved org
			// Root in the server context (carries no session tenant) to avoid session pollution
			ownerCtx = pkgcontext.WithTenantID(s.safeCtx(), ownerTenantID)
			if ownerOrg != uuid.Nil {
				ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
			}

			s.reclaimEndpointQueue(ownerCtx, session, ownerTenantID, epEUI, pendingOp.CreatedAt)

			// Update endpoint with attachment to this base station
			propagateTime := s.clock.Now().UnixNano()
			propagateStatus := PropagateStatusAttached
			propagated := true
			propagatedAt := s.clock.Now()
			attachUpdates := models.EndpointAttachmentStateParams{
				LastAttachedBsEui: session.BaseStationEUIBytes(),
				LastPropagateTime: &propagateTime,
				PropagateStatus:   &propagateStatus,
				Propagated:        &propagated,
				PropagatedAt:      models.OptionalNullTime{Set: true, Time: &propagatedAt},
			}

			// The station confirms the attachment the service center decided; it neither decides nor announces it.
			if err := s.endpointRepo.EndpointAttachmentStateUpdate(ownerCtx, ownerTenantID, endpoint.ID, attachUpdates); err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointAttachmentState,
					logger.FieldError, err,
					logger.FieldEpEui, epEUI,
					logger.FieldBsEui, session.BaseStationEUI)
			} else {
				s.logger.InfoContext(s.safeCtx(), LogBSSCIEndpointAttachedToBaseStation,
					logger.FieldEpEui, epEUI,
					logger.FieldBsEui, session.BaseStationEUI)
			}

			s.recordAttachPropagated(ownerCtx, ownerTenantID, session, epEUI, sent.shortAddr)

			// BSSCI §3.8.3: Persist attPrpCmp to messages table
			// Gated by hasEUI to ensure usable audit rows with valid ep_eui
			// NOTE: This is separate from the attPrp row persisted at send time
			// NwkSnKey intentionally omitted: already in attPrp row, avoid key duplication
			// Build bsEUIBytes at this scope for message persistence
			var msgBsEUIBytes [8]byte
			binary.BigEndian.PutUint64(msgBsEUIBytes[:], session.BaseStationEUI)

			completionMsg := &mioty.AttachPropagateMessage{
				CommandType:    mioty.CmdAttachPropagateComplete,
				OpId:           int64(msg.OpId),
				TenantID:       ownerTenantID,
				BasestationEui: msgBsEUIBytes[:],
				EpEui:          epEUI,
				ShAddr:         sent.shortAddr,
				Bidi:           sent.bidirectional,
				LastPacketCnt:  sent.lastPacketCnt,
				DualChan:       sent.dualChannel,
				Repetition:     sent.repetition,
				WideCarrOff:    sent.wideCarrOff,
				LongBlkDist:    sent.longBlkDist,
				// NwkSnKey omitted: security concern, already in attPrp row
				MessageType:   mioty.MessageTypeAttachPropagate,
				Direction:     mioty.DirectionDownlink,
				InterfaceType: mioty.InterfaceBSSCI,
			}

			if err := s.protocolMessages.CreateAttachPropagateMessage(ownerCtx, completionMsg); err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToPersistAttachPropagateComplete,
					logger.FieldError, err,
					logger.FieldOpID, msg.OpId,
					logger.FieldEpEui, epEUI,
					logger.FieldBsEui, session.BaseStationEUI)
			}
		}
	}

	// No response needed for complete messages
	return nil
}

// SendDetachPropagate sends a detach propagate command to a specific session
func (s *Server) SendDetachPropagate(sessionID string, endpointEUI uint64) error {
	session, exists := s.sessions.get(sessionID)

	ctx := s.sessionContext(session)

	if !exists {
		return fmt.Errorf("%s: %s", ResolveErrorMessage(errSessionNotFound), sessionID)
	}

	// BSSCI-3.3-03: Don't send operations to sessions that haven't completed handshake
	if !session.HandshakeComplete {
		return fmt.Errorf(errFmtTokenForSession, ResolveErrorMessage(errHandshakeNotComplete), sessionID)
	}

	// Durable order (BSSCI rev1 §5.2 / classic §3.2): allocate the ID, persist
	// the counter, persist the pending record, then write the frame. The
	// counter is never rolled back.
	opId, err := s.pendingOps.begin(s.sessionContext(session), session)
	if err != nil {
		return err
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCISendingDetachPropagate,
		logger.FieldSessionID, sessionID,
		logger.FieldEndpointEuiCamel, endpointEUI)

	// Resolve owner tenant + ctx before metadata (BSSCI §3.9 multi-tenant roaming).
	owner, err := s.resolvePropagationOwner(ctx, session, endpointEUI)
	if err != nil {
		return err
	}
	euiBytes, endpointTenantID, ownerOrgUUID, ownerCtx := owner.euiBytes, owner.tenantID, owner.orgID, owner.ctx

	// Build detach propagate message per BSSCI v1.0.0 spec
	// Per BSSCI §3.9.1: detPrp requires only command, opId, epEui (shAddr is NOT in spec)
	message := map[string]interface{}{
		"command": mioty.CmdDetachPropagate,
		"opId":    opId,
		"epEui":   endpointEUI, // MUST be Numeric[8] per BSSCI spec, NOT string!
	}

	// Create metadata with owner tenant/org info (BSSCI §5.9)
	metadata := map[string]interface{}{
		"endpointEUI":       mioty.FormatEUI64(endpointEUI), // Backward compatibility
		"epEui":             endpointEUI,                    // MUST be Numeric[8] per BSSCI spec, NOT string!
		metadataKeyTenantID: endpointTenantID,               // Owner tenant for roaming support
	}
	if ownerOrgUUID != uuid.Nil {
		metadata[metadataKeyOrganizationID] = ownerOrgUUID.String()
	}

	// The recovery record must be durable before the frame is written; a
	// persistence failure aborts the send, leaving only a consumed-ID gap.
	if err := s.pendingOps.persist(s.safeCtx(), session, opId, mioty.CmdDetachPropagate, message, euiBytes, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperation, logger.FieldError, err)
		return err
	}

	// Update endpoint in database to mark detach propagate initiated
	if endpoint := owner.endpoint; endpoint != nil {
		propagateStatus := PropagateStatusDetaching // In-flight status
		propagated := false
		updates := models.EndpointDetachStateParams{
			PropagateStatus:   &propagateStatus,
			Propagated:        &propagated,
			LastAttachedBsEui: models.OptionalBytes{Set: true, Value: session.BaseStationEUIBytes()},
		}

		err := s.endpointRepo.EndpointDetachStateUpdate(ownerCtx, endpointTenantID, endpoint.ID, updates)
		switch {
		case errors.Is(err, storage.ErrNotFound):
			// A delete detaches first, so the row can be gone before the propagate records on it.
			s.logger.DebugContext(s.safeCtx(), LogBSSCIEndpointNotFoundInDatabaseForDetachPropagate,
				logger.FieldEpEui, endpointEUI)
		case err != nil:
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointWithDetachInfo,
				logger.FieldEpEui, endpointEUI,
				logger.FieldEndpointIDCamel, endpoint.ID,
				logger.FieldError, err)
		default:
			s.logger.DebugContext(s.safeCtx(), LogBSSCIUpdatedEndpointWithDetachInfo,
				logger.FieldEpEui, endpointEUI)
		}
	} else {
		// Endpoint not found in database - this is okay for detach
		s.logger.DebugContext(s.safeCtx(), LogBSSCIEndpointNotFoundInDatabaseForDetachPropagate,
			logger.FieldEpEui, endpointEUI)
	}

	// NOTE: Duplicate event creation removed. The message persistence below creates
	// the detPrp record in the messages table, and completion creates the detPrpCmp record.
	// This avoids creating 2 extra events per operation.

	// Persist detach propagate message to messages audit trail (BSSCI §5.9)
	// Encode BS EUI to bytes for storage
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)

	// Convert org UUID to *string for nullable DB column
	var orgUUIDStr *string
	if ownerOrgUUID != uuid.Nil {
		orgStr := ownerOrgUUID.String()
		orgUUIDStr = &orgStr
	}

	// Build DetachPropagateMessage (mirrors attach propagate at 4846-4865)
	propagateMsg := &mioty.DetachPropagateMessage{
		// Protocol fields from BSSCI §5.9.1
		CommandType: mioty.CmdDetachPropagate,
		OpId:        opId,
		EpEui:       endpointEUI,

		// Metadata for messages table storage
		BasestationEui: bsEuiBytes,
		TenantID:       endpointTenantID,
		OrgUUID:        orgUUIDStr,
		MessageType:    mioty.MessageTypeDetachPropagate,
		Direction:      mioty.DirectionDownlink,
		InterfaceType:  mioty.InterfaceBSSCI,
		ReceivedAt:     s.clock.Now(),
		CreatedAt:      s.clock.Now(),
		UpdatedAt:      s.clock.Now(),
	}

	// Persist to messages (non-blocking - log errors only)
	if err := s.protocolMessages.CreateDetachPropagateMessage(ownerCtx, propagateMsg); err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToPersistDetachPropagateMessage,
			logger.FieldEpEuiSnake, endpointEUI,
			logger.FieldBsEuiSnake, session.BaseStationEUI,
			logger.FieldTenantIDSnake, endpointTenantID,
			logger.FieldError, err)
		// Don't return error - persistence failure shouldn't abort protocol handshake
	}

	if err := s.sendMessage(session, message); err != nil {
		if errors.Is(err, ErrAmbiguousWrite) {
			// The frame may be partially on the wire: keep the pending row for
			// resume reissue with the original ID and close the transport.
			s.closeTransportAfterWriteFailure(session, opId, err)
		} else if cleanupErr := s.pendingOps.remove(s.sessionContext(session), session, opId); cleanupErr != nil {
			// Nothing reached the wire; the recovery row is removed.
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, opId,
				logger.FieldError, cleanupErr)
		}

		return err
	}

	return nil
}

// handleDetachPropagateResponse handles detach propagate response from base station
func (s *Server) handleDetachPropagateResponse(session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// BSSCI 3.9.2: detPrpRsp only carries command and opId. No result field per spec.
	// Default to 0 (success) if result field is missing - spec-compliant behavior.
	result := getNumericFieldInt(data, "result", 0)

	s.logger.DebugContext(s.safeCtx(), LogBSSCIDetachPropagateResponseReceived,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId,
		logger.FieldResult, result)

	// Result codes: 0 = success, non-zero = error
	if result != 0 {
		// Detach propagate failed - DO NOT send completion or update endpoint
		return s.handlePropagateResponseFailure(ctx, session, msg, result, propagateResponseConfig{
			rejectedLog:     LogBSSCIDetachPropagateRejectedByBaseStation,
			failureErrToken: errDetachPropagateFailed,
			operationType:   EventTypeDetachPropagateFailed,
			eventType:       EventTypeEndpointDetachFailed,
			titleFormat:     TitleDetachPropagateFailedForEndpointOnBS,
		})
	}

	// Success case - proceed with three-way handshake completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIDetachPropagateAcceptedByBaseStation,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// BSSCI three-way handshake: Service Center must send detPrpCmp after successful detPrpRsp
	completionMsg := map[string]interface{}{
		"command": mioty.CmdDetachPropagateComplete,
		"opId":    msg.OpId, // Use same operation ID
	}

	s.logger.DebugContext(s.safeCtx(), LogBSSCISendingDetachPropagateComplete,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// Send the completion message
	if err := s.sendMessage(session, completionMsg); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToSendDetachPropagateComplete,
			logger.FieldBaseStation, session.BaseStationEUI,
			logger.FieldOpID, msg.OpId,
			logger.FieldError, err)
		return err
	}

	// Now perform the cleanup that would happen in handleDetachPropagateComplete
	// Since WE send the Cmp (not receive it), we need to do the cleanup here
	return s.handleDetachPropagateComplete(session, msg, data)
}

// handleDetachPropagateComplete performs cleanup after detach propagate completion
// This is called after we send detPrpCmp (Service Center sends it, not base station)
func (s *Server) handleDetachPropagateComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.InfoContext(s.safeCtx(), LogBSSCIDetachPropagateCompleted,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId)

	// BSSCI-3.9.3-01: Get pending operation BEFORE removing it to access metadata
	// BSSCI §§5.11-5.12.3 Gap 1: Use StatusService for pending operation access
	pendingOp := s.pendingOperationOrWarn(session, msg.OpId)

	// Remove completed operation from pending operations for MIOTY session resume
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation,
			logger.FieldError, err,
			logger.FieldOpID, msg.OpId,
			logger.FieldSessionID, session.DbSessionID)
	}

	// BSSCI-3.9.3-01: Update endpoint to mark as detached from this base station
	if pendingOp != nil && pendingOp.Metadata != nil {
		// Extract endpoint EUI from pending operation metadata
		var epEUI uint64
		var hasEUI bool

		if v, ok := pendingOp.Metadata["endpointEUI"]; ok {
			epEUI, hasEUI = parseMetadataEUI(v)
		}
		if !hasEUI {
			epEUI, hasEUI = parseMetadataEUI(pendingOp.Metadata[models.EventDetailKeyEpEui])
		}

		if hasEUI {
			// Convert endpoint EUI to bytes
			epEUIBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(epEUIBytes[:], epEUI)

			// Extract owner tenant/org from metadata (BSSCI §5.9 multi-tenant roaming)
			var endpointTenantID int64
			var ownerOrgUUID uuid.UUID
			var ownerCtx context.Context

			endpointTenantID, _, ownerOrgUUID = propagateRecordOwner(pendingOp.Metadata)

			// Build owner context
			if endpointTenantID > 0 {
				// Root in the server context (carries no session tenant) to avoid session pollution
				ownerCtx = pkgcontext.WithTenantID(s.safeCtx(), endpointTenantID)
				if ownerOrgUUID != uuid.Nil {
					ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrgUUID)
				}
			} else {
				// Fallback to session tenant if metadata missing
				endpointTenantID = resolvedTenant(session, s.tenantID)
				// Root in the server context (carries no session tenant) to avoid session pollution
				ownerCtx = pkgcontext.WithTenantID(s.safeCtx(), endpointTenantID)
			}

			// The station confirms the detachment the service center decided; it neither decides nor announces it.
			epModel, err := s.endpointRepo.GetByEUI(ownerCtx, endpointTenantID, epEUIBytes)
			if err == nil && epModel != nil {
				s.logger.InfoContext(s.safeCtx(), LogBSSCIEndpointDetachedFromBaseStation,
					logger.FieldEpEui, epEUI,
					logger.FieldBsEui, session.BaseStationEUI)

				// Create single event for successful detachment (consolidated from 2 redundant events)
				// Extract operation ID safely (avoid nil dereference)
				opID := msg.OpId
				if pendingOp != nil {
					opID = pendingOp.OperationID
				}

				s.recordDetachPropagated(ownerCtx, endpointTenantID, session, epEUI, opID)

				// BSSCI §5.9.3: Persist detPrpCmp to messages table
				// Gated by hasEUI to ensure usable audit rows with valid ep_eui
				var msgBsEUIBytes [8]byte
				binary.BigEndian.PutUint64(msgBsEUIBytes[:], session.BaseStationEUI)

				completionMsg := &mioty.DetachPropagateMessage{
					CommandType:    mioty.CmdDetachPropagateComplete,
					OpId:           msg.OpId, // Already int64
					TenantID:       endpointTenantID,
					BasestationEui: msgBsEUIBytes[:],
					EpEui:          epEUI,
					MessageType:    mioty.MessageTypeDetachPropagate,
					Direction:      mioty.DirectionDownlink,
					InterfaceType:  mioty.InterfaceBSSCI,
				}
				if ownerOrgUUID != uuid.Nil {
					orgStr := ownerOrgUUID.String()
					completionMsg.OrgUUID = &orgStr
				}

				if err := s.protocolMessages.CreateDetachPropagateMessage(ownerCtx, completionMsg); err != nil {
					s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToPersistDetachPropagateComplete,
						logger.FieldError, err,
						logger.FieldOpID, msg.OpId,
						logger.FieldEpEui, epEUI,
						logger.FieldBsEui, session.BaseStationEUI)
				}
			} else {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIEndpointNotFoundForDetachCompletion,
					logger.FieldEpEui, epEUI,
					logger.FieldError, err)
			}
		}
	}

	// No response needed for complete messages
	return nil
}

// propagationOwner is the endpoint a propagation is sent for, scoped to the
// tenant that owns it; endpoint is nil when no tenant owns the EUI.
type propagationOwner struct {
	euiBytes []byte
	tenantID int64
	orgID    uuid.UUID
	ctx      context.Context
	endpoint *models.EndPoint
}

// resolvePropagationOwner determines the owning tenant + organization for an
// endpoint by EUI (BSSCI §5.8.3 / §3.9 multi-tenant roaming) and builds an
// owner-scoped context. An endpoint no tenant owns is propagated for the
// session tenant; a failed lookup is an error so a propagate is never
// recorded for the wrong tenant.
func (s *Server) resolvePropagationOwner(ctx context.Context, session *Session, endpointEUI uint64) (propagationOwner, error) {
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, endpointEUI)

	owner, err := s.endpointOwners.ResolveOwner(ctx, storedEUI(endpointEUI))
	if errors.Is(err, storage.ErrNotFound) {
		sessionTenantID := resolvedTenant(session, s.tenantID)
		return propagationOwner{euiBytes: euiBytes, tenantID: sessionTenantID, ctx: pkgcontext.WithTenantID(ctx, sessionTenantID)}, nil
	}
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIPropagateOwnerLookupFailed,
			logger.FieldEpEui, endpointEUI,
			logger.FieldError, err)
		return propagationOwner{}, fmt.Errorf("%s: %w", ResolveErrorMessage(errPropagateOwnerLookupFailed), err)
	}

	ownerCtx, ownerOrgUUID := s.endpointOwnerContext(owner.TenantID)
	return propagationOwner{euiBytes: euiBytes, tenantID: owner.TenantID, orgID: ownerOrgUUID, ctx: ownerCtx, endpoint: owner.Endpoint}, nil
}

// propagateResponseConfig carries the per-operation catalog references the
// shared propagate-response failure handler needs.
type propagateResponseConfig struct {
	// rejectedLog is the outer "rejected by base station" log token emitted
	// before the failure-handling block runs.
	rejectedLog string
	// failureErrToken is the error-catalog token used both for the error
	// response message (via ResolveErrorMessage) and as a stable identifier
	// in failure metadata.
	failureErrToken string
	// operationType is the snake_case token written to details["operation"]
	// (matches SQL filters in operation_status_queries.go). MUST be one of
	// EventTypeAttachPropagateFailed or EventTypeDetachPropagateFailed.
	operationType string
	// eventType is the snake_case token written to SystemEvent.EventType.
	// MUST be one of EventTypeEndpointAttachFailed or EventTypeEndpointDetachFailed.
	eventType string
	// titleFormat is a `fmt.Sprintf` format string accepting (epEUI, bsEUI)
	// from pkg/bssci/constants.go — TitleAttachPropagateFailedForEndpointOnBS
	// or TitleDetachPropagateFailedForEndpointOnBS.
	titleFormat string
}

// markPendingOpFailed marks the pending operation as failed, merges failure
// metadata into pendingOp.Metadata, and persists the result. Returns the
// resolved owner tenant/org context (roaming-safe), the merged metadata, the
// canonical operation ID, and ok=true iff the session carries a persisted
// operation to update. Callers should gate event creation on ok.
func (s *Server) markPendingOpFailed(
	ctx context.Context,
	session *Session,
	msg *Message,
	result int,
) (ownerTenantID int64, ownerOrg uuid.UUID, metadata map[string]interface{}, opID int64, ok bool) {
	if session.DbSessionID <= 0 {
		return 0, uuid.Nil, nil, 0, false
	}

	pendingOp := s.pendingOperationOrWarn(session, msg.OpId)

	metadata = make(map[string]interface{})
	if pendingOp != nil && pendingOp.Metadata != nil {
		for k, v := range pendingOp.Metadata {
			metadata[k] = v
		}
	}
	metadata[metadataKeyFailed] = true
	metadata["failureReason"] = fmt.Sprintf(PropagateFailureReasonFormat, result)
	metadata[metadataKeyFailedAt] = s.clock.Now().UnixNano()
	metadata["failureCode"] = result

	if err := s.pendingOps.updateMetadata(s.safeCtx(), session, msg.OpId, metadata); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToPersistFailureMetadata,
			logger.FieldError, err,
			logger.FieldOpID, msg.OpId)
	}

	ownerTenantID, hasOwnerTenant, ownerOrg := propagateRecordOwner(metadata)
	if !hasOwnerTenant {
		ownerTenantID = resolvedTenant(session, s.tenantID)
		s.logger.WarnContext(ctx, LogBSSCIMissingTenantInMetadata, logger.FieldOpID, msg.OpId)
	}

	opID = int64(msg.OpId)
	if pendingOp != nil {
		opID = pendingOp.OperationID
	}

	return ownerTenantID, ownerOrg, metadata, opID, true
}

// handlePropagateResponseFailure runs the shared failure path for attach- and
// detach-propagate response handlers when the base station reports a non-zero
// result. It logs the rejection, marks the pending operation as failed with
// metadata, creates symmetric endpoint + base-station system events with
// owner-tenant context (BSSCI §5.8.3 / §3.9 roaming), and sends a
// catalog-derived error response to the base station per BSSCI-4-01. Returns
// nil to keep the connection open — an individual operation failure should
// not close the session.
func (s *Server) handlePropagateResponseFailure(
	ctx context.Context,
	session *Session,
	msg *Message,
	result int,
	cfg propagateResponseConfig,
) error {
	s.logger.ErrorContext(s.safeCtx(), cfg.rejectedLog,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldOpID, msg.OpId,
		logger.FieldResult, result)

	ownerTenantID, ownerOrg, metadata, opID, ok := s.markPendingOpFailed(ctx, session, msg, result)
	if ok {
		ownerCtx := pkgcontext.WithTenantID(ctx, ownerTenantID)
		if ownerOrg != uuid.Nil {
			ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, ownerOrg)
		}

		epEUI := extractEndpointEUIFromMetadata(metadata)
		s.events.recordPropagateFailure(ownerCtx, ownerTenantID, cfg, propagateFailure{
			epEUI: epEUI, bsEUI: session.BaseStationEUI, opID: opID, result: result,
		})
	}

	// Send error response to base station per BSSCI-4-01. POSIX error code is
	// mandatory; the response message is sourced from the catalog so attach and
	// detach paths produce identical wire format.
	errorMsg := map[string]interface{}{
		"command": mioty.CmdError,
		"opId":    msg.OpId,
		"code":    POSIX_EPROTO,
		"message": ResolveErrorMessage(cfg.failureErrToken),
	}

	if err := s.sendMessage(session, errorMsg); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToSendErrorMessageToBaseStation,
			logger.FieldError, err)
	}

	// Keep connection open - individual operation failure shouldn't close session
	return nil
}
