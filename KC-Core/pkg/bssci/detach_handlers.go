package bssci

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// handleDetach handles detach operations per BSSCI 3.7
func (s *Server) handleDetach(session *Session, msg *Message, data map[string]interface{}) error {
	// Extract and validate mandatory epEui field (full-range unsigned EUI-64)
	epEUI, hasEpEUI := getUint64Field(data, "epEui")
	if !hasEpEUI {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingEpEui))
	}

	// Extract and validate ALL mandatory fields per BSSCI 3.7.1
	rxTime, hasRxTime := getNumericField(data, "rxTime")          // Unix UTC center of last subpacket (ns)
	packetCnt, hasPacketCnt := getNumericField(data, "packetCnt") // EP packet counter

	// Extract and validate mandatory SNR/RSSI fields per BSSCI 3.7.1
	snr, hasValidSnr := getFloatFieldValidated(data, "snr")    // Signal-to-noise ratio in dB
	rssi, hasValidRssi := getFloatFieldValidated(data, "rssi") // Signal strength in dBm

	// Validate mandatory field presence per BSSCI 3.7.1 and clause 2.4
	if !hasRxTime {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingRxTime))
	}
	if !hasPacketCnt {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingPacketCnt))
	}
	if !hasValidSnr {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSnrValue))
	}
	if !hasValidRssi {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRssiValue))
	}

	// Extract optional fields with validation
	var eqSnr *float64
	if _, exists := data[wireFieldEqSnr]; exists {
		reported, valid := getFloatFieldValidated(data, wireFieldEqSnr)
		if !valid {
			// Present optional field with invalid value (BSSCI 2.4)
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidEqSnrValue))
		}
		eqSnr = &reported
	}
	rxDuration, _ := getNumericField(data, "rxDuration") // First to last subpacket center (ns)
	profile := getStringField(data, "profile", "")       // MIOTY profile (e.g., "eu1")

	// Extract and validate signature format using validateByteArray helper (mandatory per BSSCI 3.7.1)
	// Cryptographic validation happens later after endpoint lookup (see below)
	// Radio spec §3.7.1 says signature is "analogous to attach" but MIOTY spec TBD: SC signature derivation method unclear
	sign, errToken := validateByteArray(data["sign"], "sign", dbconfig.AttachSignSize)
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	subpackets, validSubpackets := optionalSubpackets(data)
	if !validSubpackets {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSubpackets))
	}

	s.logger.InfoContext(s.safeCtx(), LogBSSCIEndPointDetachRequestWithTelemetry,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldEndPoint, epEUI,
		logger.FieldPacketCnt, packetCnt,
		logger.FieldRxTime, rxTime,
		logger.FieldSnr, snr,
		logger.FieldRssi, rssi,
		logger.FieldSignLen, len(sign))

	// Build typed detachMetadata struct for crash-safe persistence (BSSCI §5.7.1)
	packetCntUint, errToken := safeUint32(packetCnt)
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	typedMetadata := &detachMetadata{
		EpEui:     epEUI,
		PacketCnt: packetCntUint,
		Signature: sign,
		RxTime:    rxTime,
		SNR:       snr,
		RSSI:      rssi,
	}
	// Add optional fields
	typedMetadata.EqSnr = eqSnr
	if profile != "" {
		typedMetadata.Profile = &profile
	}
	if rxDuration > 0 {
		typedMetadata.RxDuration = &rxDuration
	}

	// Fetch endpoint early to resolve owner tenant/org for roaming support (DET-02)
	ctx := s.sessionContext(session)
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEUI)

	// Only a genuine not-found makes the endpoint unknown; any other lookup
	// error fails the detach closed so it is never mis-routed to the serving tenant.
	owner, err := s.endpointOwners.ResolveOwner(ctx, storedEUI(epEUI))
	endpoint := owner.Endpoint
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		s.logger.ErrorContext(ctx, LogBSSCIDetachOwnerLookupFailed,
			logger.FieldEpEui, epEUI,
			logger.FieldError, err)
		return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errDetachOwnerLookupFailed))
	}

	// Resolve endpoint owner tenant/org (or fall back to session tenant)
	var ownerTenantID int64
	var ownerOrgUUID uuid.UUID
	var ownerCtx context.Context
	var validationStatus string // Detach signature validation status from internal validator

	if err == nil && endpoint != nil {
		ownerTenantID = owner.TenantID
		ownerCtx, ownerOrgUUID = s.endpointOwnerContext(ownerTenantID)

		// Validate the detach signature before any persistence or state
		// mutation. When validation is enabled an injected authoritative
		// validator (required at startup) decides; when disabled - the default,
		// since the MIOTY spec does not define the detach CMAC construction -
		// the well-formed frame is accepted and durably recorded as unverified.
		if s.config != nil && s.config.DetachSignatureValidationEnabled {
			result, validationErr := s.detachValidator.ValidateDetachSignature(ownerCtx, epEUI, sign)
			if validationErr != nil || result == nil || !result.Valid {
				s.logger.WarnContext(ownerCtx, LogBSSCIDetachSignatureValidationFailed,
					logger.FieldEpEui, mioty.FormatEUI64(epEUI),
					logger.FieldError, validationErr)
				return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
			}
			validationStatus = result.ValidationStatus
		} else {
			validationStatus = ValidationStatusUnverified
		}
	} else {
		// Unknown endpoint - validate via internal validator and use returned tenant metadata (BSSCI §5.7)
		s.logger.WarnContext(ctx, LogBSSCIDetachFromUnknownEndpoint,
			logger.FieldError, err,
			logger.FieldEpEui, epEUI)

		// When validation is enabled the same injected validator decides for
		// unknown endpoints and supplies the owner tenant metadata; when
		// disabled the frame is recorded unverified under the session tenant.
		if s.config != nil && s.config.DetachSignatureValidationEnabled {
			result, validationErr := s.detachValidator.ValidateDetachSignature(ctx, epEUI, sign)
			if validationErr != nil {
				// Check for typed sentinel: endpoint not found
				if errors.Is(validationErr, ErrDetachValidationEndpointNotFound) {
					s.logger.WarnContext(ctx, LogBSSCIUnknownEndpointNotFoundDuringDetachValidation,
						logger.FieldEpEui, epEUI)
					return s.sendError(session, msg.OpId, POSIX_ENOENT, ResolveErrorMessage(errEndpointNotFound))
				}

				// Check for signature validation failure with metadata
				if errors.Is(validationErr, ErrDetachSignatureInvalid) && result != nil {
					s.logger.WarnContext(ctx, LogBSSCIUnknownEndpointDetachSignatureInvalid,
						logger.FieldEpEui, epEUI,
						logger.FieldTenantIDSnake, result.TenantID,
						logger.FieldOwnerTenantIDSnake, result.OwnerTenantID,
						logger.FieldValidationStatus, result.ValidationStatus)
					return s.sendError(session, msg.OpId, POSIX_EACCES, ResolveErrorMessage(errDetachSignatureInvalid))
				}

				// All other errors treated as signature validation failure
				s.logger.WarnContext(ctx, LogBSSCIUnknownEndpointDetachSignatureValidationFailed,
					logger.FieldEpEui, epEUI,
					logger.FieldError, validationErr)
				return s.sendError(session, msg.OpId, POSIX_EACCES, ResolveErrorMessage(errDetachSignatureInvalid))
			}

			// Double-check result validity
			if !result.Valid {
				return s.sendError(session, msg.OpId, POSIX_EACCES, ResolveErrorMessage(errDetachSignatureInvalid))
			}

			// SUCCESS: Use validator-returned tenant metadata
			ownerTenantID = result.TenantID
			validationStatus = result.ValidationStatus
			ownerCtx, ownerOrgUUID = s.endpointOwnerContext(ownerTenantID)

			s.logger.InfoContext(ctx, LogBSSCIUnknownEndpointDetachSignatureValidatedSuccessfully,
				logger.FieldEpEui, epEUI,
				logger.FieldTenantIDCamel, ownerTenantID,
				logger.FieldOwnerTenantID, result.OwnerTenantID,
				logger.FieldValidationStatusCamel, validationStatus)
		} else {
			// Validation disabled: record the detach without a cryptographic
			// check, durably marked unverified, under the session tenant.
			ownerTenantID = resolvedTenant(session, s.tenantID)
			ownerCtx = pkgcontext.WithTenantID(s.safeCtx(), ownerTenantID)
			validationStatus = ValidationStatusUnverified
			s.logger.WarnContext(ctx, LogBSSCIDetachValidatorNotConfiguredUsingSessionTenantForUnknownEndpoint,
				logger.FieldEpEui, epEUI,
				logger.FieldSessionTenant, ownerTenantID)
		}
	}

	// Update typed metadata with owner context for crash-safe resume
	typedMetadata.TenantID = ownerTenantID
	typedMetadata.OrgUUID = ownerOrgUUID

	// Detect roaming for detach operation
	var isRoaming bool
	servingTenantID := resolvedTenant(session, s.tenantID)
	if endpoint != nil {
		// The known endpoint's owner, already in ownerTenantID, is
		// authoritative and is never reassigned from the roaming result.
		var roamErr error
		isRoaming, roamErr = s.evaluateRoaming(ctx, euiBytes, servingTenantID)
		if roamErr != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIRoamingValidationFailedDuringDetach,
				logger.FieldEpEui, epEUI,
				logger.FieldServingTenant, servingTenantID,
				logger.FieldError, roamErr)
			// The endpoint is leaving regardless; record it under the true owner.
			isRoaming = false
		}

		// If roaming, record the detach event
		if isRoaming {
			s.logger.InfoContext(s.safeCtx(), LogBSSCIRoamingEndpointDetaching,
				logger.FieldEpEui, epEUI,
				logger.FieldOwnerTenant, ownerTenantID,
				logger.FieldServingTenant, servingTenantID)

			// Record roaming detach event
			if err := s.roamingSvc.RecordDetach(ctx, euiBytes, session.BaseStationEUIBytes(), servingTenantID); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRecordRoamingDetach,
					logger.FieldError, err)
				// Non-fatal: continue with detach
			}

			// Update session to remove roaming endpoint
			if session.DbSessionID > 0 {
				if err := s.roamingSvc.UpdateSessionRoaming(ctx, session.DbSessionID, euiBytes, false, servingTenantID); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateSessionRoamingForDetach,
						logger.FieldError, err)
					// Non-fatal: continue with detach
				}
			}

			// Update metadata with actual owner tenant ID
			typedMetadata.TenantID = ownerTenantID
		}
	}

	// Record the validation provenance resolved during the pre-persistence
	// validation phase above (BSSCI §5.7).
	typedMetadata.ValidationStatus = validationStatus

	// Convert typed metadata to map for JSON persistence
	metadataMap := detachMetadataToMap(typedMetadata)
	// Add subpackets to metadata map for crash-resume (not part of typed struct)
	if subpackets != nil {
		metadataMap[wireFieldSubpackets] = data[wireFieldSubpackets]
	}

	// Store endpoint.ID if known (avoid double-fetch in handleDetachComplete)
	if endpoint != nil {
		metadataMap[metadataKeyEndpointID] = endpoint.ID
		typedMetadata.EndpointID = endpoint.ID
	}

	// Persist pending operation via StatusService (handles both DB + map with SessionOpKey).
	// This row is a crash-resume aid for a BS-initiated operation (positive
	// opId): the abort-before-send rule protects SC-initiated recovery only,
	// and aborting here would drop a live detach on a transient DB failure, so
	// persistence stays best-effort (BSSCI §5.7).
	if session.DbSessionID != 0 {
		if err := s.pendingOps.persist(s.safeCtx(), session, int64(msg.OpId), mioty.CmdDetach, data, euiBytes, metadataMap); err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToPersistPendingOperationMigrationNeeded,
				logger.FieldError, err,
				logger.FieldSessionID, session.DbSessionID,
				logger.FieldOpID, msg.OpId)
		}
	}

	// Persist detach message to mioty_messages table under endpoint owner (BSSCI §5.7.1 Finding 6)
	bsEuiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bsEuiBytes, session.BaseStationEUI)
	detachMsg := &mioty.DetachMessage{
		CommandType:      mioty.CmdDetach,
		OpId:             int64(msg.OpId),
		EpEui:            euiBytes,
		BasestationEui:   bsEuiBytes,
		RxTime:           rxTime,
		PacketCnt:        packetCntUint,
		SNR:              snr,
		RSSI:             rssi,
		Signature:        sign,
		MessageType:      mioty.MessageTypeDetach,
		Direction:        mioty.DirectionUplink,
		InterfaceType:    mioty.InterfaceBSSCI,
		TenantID:         ownerTenantID,
		ValidationStatus: validationStatus,
	}
	// Set OrgUUID only when valid
	if ownerOrgUUID != uuid.Nil {
		ownerOrgUUIDStr := ownerOrgUUID.String()
		detachMsg.OrgUUID = &ownerOrgUUIDStr
	}
	// Add optional fields
	if typedMetadata.EqSnr != nil {
		detachMsg.EqSnr = typedMetadata.EqSnr
	}
	if typedMetadata.Profile != nil {
		detachMsg.Profile = typedMetadata.Profile
	}
	if typedMetadata.RxDuration != nil {
		detachMsg.RxDuration = typedMetadata.RxDuration
	}
	detachMsg.Subpackets = subpackets

	// Normalize payload for audit completeness (DET-03: convert numeric arrays to concrete types)
	normalizedPayload := normalizeDetachPayload(data)

	if err := s.protocolMessages.CreateDetachMessage(ownerCtx, detachMsg, normalizedPayload); err != nil {
		s.logger.WarnContext(ownerCtx, LogBSSCIFailedToPersistDetachMessage,
			logger.FieldError, err,
			logger.FieldEpEui, epEUI,
			logger.FieldBsEui, session.BaseStationEUI,
			logger.FieldOpID, msg.OpId)
	}

	// Update endpoint telemetry if the endpoint was found. The detach signature
	// was already validated before any persistence above.
	if endpoint != nil {
		// Update endpoints table with detach telemetry (BSSCI §5.7.1 Finding 2).
		// The detach-state update persists the last_detach_* columns that
		// Update() does not touch.
		rxTimeCopy := rxTime
		propagateStatus := PropagateStatusDetachReceived
		updates := models.EndpointDetachStateParams{
			LastAttachedBsEui:   models.OptionalBytes{Set: true, Value: session.BaseStationEUIBytes()},
			LastPropagateTime:   &rxTimeCopy,
			LastDetachTime:      &rxTimeCopy,
			LastDetachSign:      sign,
			LastDetachPacketCnt: &packetCntUint,
			PropagateStatus:     &propagateStatus,
		}
		if err := s.endpointRepo.EndpointDetachStateUpdate(ownerCtx, ownerTenantID, endpoint.ID, updates); err != nil {
			s.logger.WarnContext(ownerCtx, LogBSSCIFailedToUpdateEndpointDetachTelemetry,
				logger.FieldError, err,
				logger.FieldEpEui, epEUI)
		}
	}

	// Build detach response per BSSCI 3.7.2
	// Convert signature to Numeric[4] array format (sign is guaranteed to be 4 bytes here)
	numericSign := make([]interface{}, dbconfig.AttachSignSize)
	for i := 0; i < dbconfig.AttachSignSize; i++ {
		numericSign[i] = int(sign[i])
	}

	response := map[string]interface{}{
		"command": mioty.CmdDetachResponse,
		"opId":    msg.OpId,
		"sign":    numericSign, // Numeric[4] format per BSSCI 3.7.2
	}

	return s.sendMessage(session, response)
}

// handleDetachComplete handles detach complete operations per BSSCI 3.7.3
func (s *Server) handleDetachComplete(session *Session, msg *Message, data map[string]interface{}) error {
	// Retrieve the pending detach operation
	// StatusService is the single path for pending operation persistence
	pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId))

	if err != nil || pendingOp == nil || pendingOp.OperationType != mioty.CmdDetach {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIReceivedDetCmpWithoutPendingDetach,
			logger.FieldBaseStation, session.BaseStationEUI,
			logger.FieldOpID, int64(msg.OpId))
		return fmt.Errorf(errFmtTokenForOpID, ResolveErrorMessage(errNoPendingDetachOperation), msg.OpId)
	}

	completion := s.readDetachCompletion(session, pendingOp, data)

	// Log successful completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIDetachOperationCompletedSuccessfully,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldEndPoint, completion.epEUI,
		logger.FieldOpID, int64(msg.OpId),
		logger.FieldDuration, s.clock.Now().Sub(pendingOp.CreatedAt))

	// A det heard by several stations completes at each; only the completion
	// that detached the endpoint announces and propagates it.
	if s.decideOverTheAirDetach(completion) {
		s.propagateOverTheAirDetach(completion.ownerCtx, session, completion.epEUI)
	}
	s.recordDetachEvent(completion)

	// Clear the pending operation
	// BSSCI §§5.11-5.12.3 Gap 1: Use removePendingOperation helper (has dual-path logic)
	if err := s.pendingOps.remove(s.sessionContext(session), session, int64(msg.OpId)); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
			logger.FieldError, err,
			logger.FieldOpID, msg.OpId)
	}

	// No response needed for complete messages per BSSCI spec
	return nil
}

// detachCompletion is a completed over-the-air detach, scoped to the
// endpoint owner recorded when the det was accepted.
type detachCompletion struct {
	session       *Session
	ownerTenantID int64
	ownerCtx      context.Context
	epEUI         uint64
	endpointID    int64
	typedMeta     *detachMetadata
	metadata      map[string]interface{}
}

// readDetachCompletion reads a completed detach from its pending operation.
// Under roaming the serving session belongs to a different tenant, so every
// endpoint-state write, metric update and event of this leg targets the owner
// from the pending-operation metadata, not the session tenant.
func (s *Server) readDetachCompletion(session *Session, pendingOp *PendingOperation, data map[string]interface{}) detachCompletion {
	c := detachCompletion{
		session:       session,
		ownerTenantID: resolvedTenant(session, s.tenantID),
		ownerCtx:      s.sessionContext(session),
		metadata:      pendingOp.Metadata,
	}
	if pendingOp.Metadata != nil {
		c.typedMeta = mapToDetachMetadata(pendingOp.Metadata)
	}
	if c.typedMeta != nil {
		c.epEUI, c.endpointID = c.typedMeta.EpEui, c.typedMeta.EndpointID
		if c.typedMeta.TenantID > 0 {
			c.ownerTenantID = c.typedMeta.TenantID
			c.ownerCtx = pkgcontext.WithTenantID(c.ownerCtx, c.ownerTenantID)
			if c.typedMeta.OrgUUID != uuid.Nil {
				c.ownerCtx = pkgcontext.WithOrganizationID(c.ownerCtx, c.typedMeta.OrgUUID)
			}
		}
	} else if pendingOp.Metadata != nil {
		// Legacy or malformed records keep their EUI in the raw map.
		c.epEUI, _ = parseMetadataEUI(pendingOp.Metadata[models.EventDetailKeyEpEui])
	}
	if c.epEUI == 0 {
		c.epEUI, _ = getUint64Field(data, "epEui")
	}
	return c
}

// decideOverTheAirDetach has the attachment decider detach the endpoint the
// completion names, with the det's telemetry, records the det's radio
// metrics, and reports whether the endpoint was detached by this completion.
func (s *Server) decideOverTheAirDetach(c detachCompletion) bool {
	if c.epEUI == 0 || c.endpointID <= 0 {
		return false
	}
	reception := readDetachReception(c.typedMeta, c.metadata)
	detached, err := s.attachmentDecider.Decide(c.ownerCtx, AttachmentDecision{
		TenantID:   c.ownerTenantID,
		EndpointID: c.endpointID,
		EpEUI:      c.epEUI,
		Status:     endpoint.EndpointStatusDetached,
		OverTheAir: &OverTheAirReport{
			BaseStationEUI: c.session.BaseStationEUI,
			Status:         overTheAirDetachStatus(c.epEUI, c.typedMeta, c.metadata),
			Telemetry:      reception.telemetry(),
		},
	})
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointDetachState, logger.FieldError, err)
	}

	update := models.RadioMetricsUpdate{
		SNR:        reception.snr,
		RSSI:       reception.rssi,
		EqSNR:      reception.eqSnr,
		RxTime:     reception.rxTime,
		RxDuration: reception.rxDuration,
		Profile:    reception.profile,
	}
	if err := s.endpointRepo.UpdateRadioMetricsSelective(c.ownerCtx, c.ownerTenantID, storedEUI(c.epEUI), update); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateRadioMetrics, logger.FieldError, err)
	}
	return detached
}

// detachReception is what the det recorded of its reception.
type detachReception struct {
	rxTime     int64
	packetCnt  uint32
	snr, rssi  float64
	sign       []byte
	eqSnr      *float64
	rxDuration *int64
	profile    *string
}

// readDetachReception reads the det's reception from the typed record, or
// from the raw map of a legacy or malformed record.
func readDetachReception(typedMeta *detachMetadata, metadata map[string]interface{}) detachReception {
	if typedMeta != nil {
		return detachReception{
			rxTime: typedMeta.RxTime, packetCnt: typedMeta.PacketCnt,
			snr: typedMeta.SNR, rssi: typedMeta.RSSI, sign: typedMeta.Signature,
			eqSnr: typedMeta.EqSnr, rxDuration: typedMeta.RxDuration, profile: typedMeta.Profile,
		}
	}
	var r detachReception
	r.rxTime, _ = metadata["rxTime"].(int64)
	if pc, ok := metadata["packetCnt"].(float64); ok {
		r.packetCnt = uint32(pc)
	} else if pc, ok := metadata["packetCnt"].(int64); ok {
		r.packetCnt = uint32(pc) //nolint:gosec // G115: Packet count from metadata, range validated by protocol
	}
	r.snr, _ = metadata["snr"].(float64)
	r.rssi, _ = metadata["rssi"].(float64)
	r.sign, _ = recordBytes(metadata[wireFieldSign])
	if reported, ok := getFloatFieldValidated(metadata, wireFieldEqSnr); ok {
		r.eqSnr = &reported
	}
	if rxDur, ok := metadata["rxDuration"].(int64); ok {
		r.rxDuration = &rxDur
	}
	if prof, ok := metadata["profile"].(string); ok {
		r.profile = &prof
	}
	return r
}

// telemetry is the det's packet counter and, when it carried one, its signature.
func (r detachReception) telemetry() *endpoint.DetachTelemetry {
	packetCnt := r.packetCnt
	telemetry := &endpoint.DetachTelemetry{PacketCnt: &packetCnt}
	if len(r.sign) == 4 {
		telemetry.Sign = r.sign
	}
	return telemetry
}

// recordDetachEvent records the completed detach as a system event of the
// endpoint owner.
func (s *Server) recordDetachEvent(c detachCompletion) {
	if c.epEUI == 0 {
		return
	}
	details := map[string]interface{}{
		models.EventDetailKeyEpEui:     mioty.FormatEUI64(c.epEUI), // Use hex string to avoid JSON precision loss
		models.EventDetailKeyBsEui:     mioty.FormatEUI64(c.session.BaseStationEUI),
		models.EventDetailKeyOperation: "detach_complete",
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateDetachEvent, logger.FieldError, err)
		return
	}

	if err := s.eventStore.CreateEvent(c.ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", c.ownerTenantID),
		EventType:   models.EventTypeEndpointDetached,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleEndpointDetachedViaBS, mioty.FormatEUI64(c.epEUI), mioty.FormatEUI64(c.session.BaseStationEUI)),
		Description: eventDescEndpointDetached,
		Details:     detailsJSON,
		Status:      EventStatusNew,
		CreatedAt:   s.clock.Now(),
	}); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateDetachEvent, logger.FieldError, err)
	}
}

// overTheAirDetachStatus builds a detached epStat with the SCACI §3.13.1
// over-the-air fields the det carried.
func overTheAirDetachStatus(epEUI uint64, typedMeta *detachMetadata, metadata map[string]interface{}) *EPStatusData {
	status := &EPStatusData{EpEui: epEUI, EpStatus: pkgmioty.EPStatusDetached}
	if typedMeta != nil {
		snr, rssi := typedMeta.SNR, typedMeta.RSSI
		status.Snr = &snr
		status.Rssi = &rssi
		status.Sign = metadataNumeric4(typedMeta.Signature)
		status.EqSnr = typedMeta.EqSnr
	}
	if status.Sign == nil {
		status.Sign = metadataNumeric4(metadata[wireFieldSign])
	}
	if status.EqSnr == nil {
		if eqSnr, ok := getFloatFieldValidated(metadata, wireFieldEqSnr); ok {
			status.EqSnr = &eqSnr
		}
	}
	status.Subpackets = metadataSubpackets(metadata)
	return status
}

// propagateOverTheAirDetach tells every other station to drop an endpoint
// that detached over the air at the reporting one (radio spec §3.7.1).
func (s *Server) propagateOverTheAirDetach(ownerCtx context.Context, reporting *Session, epEUI uint64) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for _, err := range s.sendDetachPropagateTo(s.handshakeCompleteSessionIDs(reporting.ID), epEUI) {
			s.logger.WarnContext(ownerCtx, LogBSSCIDetachPropagationFailedAfterOTADetach,
				logger.FieldEpEui, mioty.FormatEUI64(epEUI),
				logger.FieldError, err)
		}
	}()
}

// normalizeDetachPayload converts raw detach message payload to concrete types for JSONB storage.
// Ensures epEui/bsEui/packetCnt/rxTime are uint64/uint32/int64 (not float64), and sign is []byte (not []interface{}).
// Required for DET-03: spec-compliant audit payload preservation (BSSCI §5.7.1).
func normalizeDetachPayload(data map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{})

	for k, v := range data {
		switch k {
		case wireFieldSign:
			// Convert []interface{} to []byte for JSONB storage
			if arr, ok := v.([]interface{}); ok {
				bytes := make([]byte, len(arr))
				for i, val := range arr {
					if num, ok := val.(float64); ok {
						bytes[i] = byte(num)
					}
				}
				normalized[k] = bytes
			} else if bytes, ok := v.([]byte); ok {
				normalized[k] = bytes
			} else {
				normalized[k] = v
			}

		case "epEui", "bsEui":
			// Ensure uint64 not float64
			if num, ok := v.(float64); ok {
				normalized[k] = uint64(num)
			} else {
				normalized[k] = v
			}

		case "packetCnt":
			// Ensure uint32 not float64
			if num, ok := v.(float64); ok {
				normalized[k] = uint32(num)
			} else {
				normalized[k] = v
			}

		case "rxTime":
			// Ensure int64 not float64
			if num, ok := v.(float64); ok {
				normalized[k] = int64(num)
			} else {
				normalized[k] = v
			}

		case wireFieldRssi, wireFieldSnr, wireFieldEqSnr:
			// Keep as float64 but ensure concrete type
			if num, ok := v.(float64); ok {
				normalized[k] = num
			} else {
				normalized[k] = v
			}

		default:
			normalized[k] = v
		}
	}

	return normalized
}
