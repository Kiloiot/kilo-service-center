package bssci

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	dbconfig "github.com/Kiloiot/kilo-service-center/KC-DB/common/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// resolveEndpointTenantID looks up the endpoint's owning tenant ID, which
// differs from the session tenant when the endpoint roams.
func (s *Server) resolveEndpointTenantID(ctx context.Context, session *Session, epEui uint64) (int64, error) {
	sessionTenant := resolvedTenant(session, s.tenantID)
	owner, err := s.endpointOwners.ResolveOwner(ctx, storedEUI(epEui))
	if errors.Is(err, storage.ErrNotFound) {
		s.logger.WarnContext(ctx, LogBSSCIEndpointNotFound,
			logger.FieldEpEuiSnake, mioty.FormatEUI64(epEui),
			logger.FieldSessionTenant, sessionTenant)
		return 0, storage.ErrNotFound
	}
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToResolveEndpointTenant,
			logger.FieldEpEuiSnake, mioty.FormatEUI64(epEui),
			logger.FieldError, err)
		return 0, fmt.Errorf("%s: %w", ResolveErrorMessage(errFailedToResolveEndpointTenant), err)
	}
	if owner.TenantID != sessionTenant {
		s.logger.InfoContext(ctx, LogBSSCIResolvedRoamingEndpointTenant,
			logger.FieldEpEuiSnake, mioty.FormatEUI64(epEui),
			logger.FieldOwnerTenantSnake, owner.TenantID,
			logger.FieldSessionTenant, sessionTenant)
	}
	return owner.TenantID, nil
}

// errCipherRequired reports a missing key-material cipher at a point that must
// encrypt a network key for at-rest storage; there is no plaintext fallback.
var errCipherRequired = errors.New("key-material cipher not configured")

// wireFieldNwkSnKey is the MIOTY wire field carrying the network session key.
const wireFieldNwkSnKey = "nwkSnKey"

// encryptNwkKeyEnvelope encrypts a network session key for at-rest storage as a
// keycrypto envelope. It never returns cleartext: a missing cipher or an
// encryption failure is an error so no plaintext key is ever persisted.
func (s *Server) encryptNwkKeyEnvelope(nwkSnKey []byte) ([]byte, error) {
	if s.cipher == nil {
		return nil, errCipherRequired
	}
	envelope, err := s.cipher.Encrypt(nwkSnKey)
	if err != nil {
		return nil, fmt.Errorf(errFmtEncryptNetworkSessionKey, err)
	}
	return envelope, nil
}

// handleAttach handles attach operations per BSSCI 3.6.1 and 3.6.2
func (s *Server) handleAttach(session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// Extract and validate mandatory epEui field (full-range unsigned EUI-64)
	epEUI, hasEpEUI := getUint64Field(data, "epEui")
	if !hasEpEUI {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingEpEui))
	}

	// Extract and validate ALL mandatory fields per BSSCI 3.6.1
	rxTime, hasRxTime := getNumericField(data, "rxTime")          // Reception time (Unix UTC ns)
	attachCnt, hasAttachCnt := getNumericField(data, "attachCnt") // Attach counter

	// Validate mandatory field presence
	if !hasRxTime {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingRxTime))
	}
	if !hasAttachCnt {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingAttachCnt))
	}
	// Validate attachCnt is 24-bit unsigned (0 to 0xFFFFFF per BSSCI §5.6.1)
	if attachCnt < 0 || attachCnt > dbconfig.AttachCounterMax {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidAttachCntRange))
	}

	// Extract and validate mandatory SNR/RSSI fields per BSSCI 3.6.1
	snr, hasValidSnr := getFloatFieldValidated(data, "snr")    // Signal-to-noise ratio in dB
	rssi, hasValidRssi := getFloatFieldValidated(data, "rssi") // Signal strength in dBm

	if !hasValidSnr {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSnrValue))
	}
	if !hasValidRssi {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRssiValue))
	}

	// Extract and validate nonce (mandatory 4-byte array per BSSCI §5.6.1)
	nonceData, ok := data["nonce"]
	if !ok {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingNonce))
	}
	nonce, errToken := validateByteArray(nonceData, "nonce", dbconfig.AttachNonceSize)
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	// Extract and validate signature (mandatory 4-byte array per BSSCI §5.6.1)
	signData, ok := data["sign"]
	if !ok {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingSign))
	}
	sign, errToken := validateByteArray(signData, "sign", dbconfig.AttachSignSize)
	if errToken != "" {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	// Extract optional radio capability flags per BSSCI 3.6.1
	var dualChan bool
	_, hasDualChan := data["dualChan"]
	if hasDualChan {
		dualChan = getBoolField(data, "dualChan", false)
	}

	var repetition bool
	_, hasRepetition := data["repetition"]
	if hasRepetition {
		repetition = getBoolField(data, "repetition", false)
	}

	var wideCarrOff bool
	_, hasWideCarrOff := data["wideCarrOff"]
	if hasWideCarrOff {
		wideCarrOff = getBoolField(data, "wideCarrOff", false)
	}

	var longBlkDist bool
	_, hasLongBlkDist := data["longBlkDist"]
	if hasLongBlkDist {
		longBlkDist = getBoolField(data, "longBlkDist", false)
	}

	var eqSnr *float64
	if rawEqSnr, exists := data[wireFieldEqSnr]; exists && rawEqSnr != nil {
		reported, valid := getFloatFieldValidated(data, wireFieldEqSnr)
		if !valid {
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidEqSnrValue))
		}
		eqSnr = &reported
	}
	rxDuration, hasRxDuration := getNumericField(data, "rxDuration")

	var profile string
	var hasProfile bool
	if profileData, exists := data["profile"]; exists {
		if profileStr, ok := profileData.(string); ok {
			profile = profileStr
			hasProfile = true
		}
	}

	subpackets, validSubpackets := optionalSubpackets(data)
	if !validSubpackets {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSubpackets))
	}

	// Optional short address if assigned by Base Station; it must fit 16 bits
	rawShAddr, hasShAddr := getNumericField(data, "shAddr")
	var bsShAddr uint16
	if hasShAddr {
		var errToken string
		if bsShAddr, errToken = safeUint16(rawShAddr); errToken != "" {
			return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errToken))
		}
	}
	var scAssignedShAddr bool

	s.logger.InfoContext(s.safeCtx(), LogBSSCIEndPointAttachRequestWithFullTelemetry,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldEndPoint, epEUI,
		logger.FieldAttachCnt, attachCnt,
		logger.FieldRxTime, rxTime,
		logger.FieldNonceLen, len(nonce),
		logger.FieldSignLen, len(sign),
		logger.FieldDualChan, dualChan,
		logger.FieldRepetition, repetition,
		logger.FieldWideCarrOff, wideCarrOff,
		logger.FieldLongBlkDist, longBlkDist)

	// Store pending operation for completion tracking
	// Make a copy of the data to avoid mutations
	pendingData := map[string]interface{}{
		"epEui":       mioty.FormatEUI64(epEUI),
		"attachCnt":   attachCnt,
		"rxTime":      rxTime,
		"nonce":       nonce,
		"sign":        sign,
		"dualChan":    dualChan,
		"repetition":  repetition,
		"wideCarrOff": wideCarrOff,
		"longBlkDist": longBlkDist,
		"rssi":        rssi,
		"snr":         snr,
	}

	// Optional fields: only store when present
	if eqSnr != nil {
		pendingData[wireFieldEqSnr] = *eqSnr
	}
	if subpackets != nil {
		pendingData[wireFieldSubpackets] = data[wireFieldSubpackets]
	}
	if hasRxDuration {
		pendingData["rxDuration"] = rxDuration
	}
	// BSSCI-ATTACH-023: Empty profile preserves existing DB value (consistent with detach at line 2600)
	// Rationale: "absent or empty" should not overwrite; only non-empty strings update the profile field
	if hasProfile && profile != "" {
		pendingData["profile"] = profile
	}

	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)

	var nwkSnKey []byte
	var responseShAddr uint16

	servingTenantID := resolvedTenant(session, s.tenantID)
	owner, err := s.endpointOwners.ResolveOwner(ctx, storedEUI(epEUI))
	if errors.Is(err, storage.ErrNotFound) {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIEndpointNotProvisionedForAttach,
			logger.FieldEpEui, epEUI,
			logger.FieldError, err)
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errEndpointNotProvisioned))
	}
	if err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIAttachOwnerLookupFailed,
			logger.FieldEpEui, epEUI,
			logger.FieldError, err)
		return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errAttachOwnerLookupFailed))
	}
	endpoint, ownerTenantID := owner.Endpoint, owner.TenantID
	ownerCtx, _ := s.endpointOwnerContext(ownerTenantID)

	// The recovery record names the owner so a completion after a resume is recorded for it.
	pendingData[metadataKeyEndpointID] = endpoint.ID
	pendingData[metadataKeyEndpointTenantID] = ownerTenantID
	pendingOp := &PendingOperation{
		OperationID:   int64(msg.OpId),
		OperationType: mioty.CmdAttach,
		Message:       data,
		Metadata:      pendingData,
		CreatedAt:     s.clock.Now(),
	}
	if err := s.statusSvc.RecordPendingOperation(ctx, session, int64(msg.OpId), pendingOp, session.DbSessionID); err != nil {
		s.logger.ErrorContext(ctx, LogBSSCIFailedToRecordPendingAttachOperation, logger.FieldOpID, msg.OpId, logger.FieldError, err)
		return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errOperationFailed))
	}

	isRoaming, err := s.evaluateRoaming(ctx, epEUIBytes, servingTenantID)
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIRoamingValidationFailed,
			logger.FieldEpEui, epEUI,
			logger.FieldServingTenant, servingTenantID,
			logger.FieldError, err)
		if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPERM, ResolveErrorMessage(errRoamingNotAllowed))
	}
	if isRoaming {
		s.logger.InfoContext(s.safeCtx(), LogBSSCIRoamingEndpointAttaching,
			logger.FieldEpEui, epEUI,
			logger.FieldOwnerTenant, ownerTenantID,
			logger.FieldServingTenant, servingTenantID)

		// Roaming bookkeeping is best effort: the attach itself proceeds.
		if err := s.roamingSvc.RecordAttach(ctx, epEUIBytes, session.BaseStationEUIBytes(), servingTenantID); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRecordRoamingAttach,
				logger.FieldError, err)
		}
		if session.DbSessionID > 0 {
			if err := s.roamingSvc.UpdateSessionRoaming(ctx, session.DbSessionID, epEUIBytes, true, servingTenantID); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateSessionRoaming,
					logger.FieldError, err)
			}
		}
	}

	// Get network session key from endpoint
	nwkSnKey = endpoint.NwkSnKey

	if len(nwkSnKey) != 16 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidNetworkKeyLengthForAttach,
			logger.FieldEpEui, epEUI,
			logger.FieldKeyLength, len(nwkSnKey))
		if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errNwkSnKeyInvalidLength))
	}

	// Replay protection: enforce monotonic attach counter (BSSCI §5.6.1)
	if bound := endpoint.OverTheAirAttachCounter(); !AttachCounterAdvances(bound, attachCnt) {
		return s.refuseStaleAttachCounter(session, msg,
			logger.FieldTenantIDCamel, ownerTenantID,
			logger.FieldEpEui, epEUI,
			logger.FieldStoredAttachCnt, *bound,
			logger.FieldIncomingAttachCnt, attachCnt)
	}

	//nolint:gosec // G115: attachCnt validated to be within 0..AttachCounterMax above
	if err := ValidateAttachSignature(epEUI, uint32(attachCnt), sign, nwkSnKey); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIAttachSignatureValidationFailed,
			logger.FieldEpEui, epEUI,
			logger.FieldError, err)
		if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
	}

	sessionKey, err := DeriveSessionKey(epEUI, nonce, sign, nwkSnKey)
	if err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToDeriveSessionKey,
			logger.FieldEpEui, epEUI,
			logger.FieldError, err)
		if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
	}

	// The derived session key is persisted in cleartext form here; the
	// persistence layer encrypts it at rest through its cipher (BSSCI
	// §5.6.2), so no key protection happens in the protocol handler.

	// BSSCI §3.6.1-§3.6.2: a station-assigned short address is kept and not echoed in attRsp.
	switch {
	case hasShAddr:
		responseShAddr = bsShAddr
	case endpoint.ShAddr != nil:
		responseShAddr = *endpoint.ShAddr
		scAssignedShAddr = true
	default:
		responseShAddr = uint16(epEUI & 0xFFFF) //nolint:gosec // G115: bitwise AND guarantees value fits uint16
		scAssignedShAddr = true
	}

	attachCntCopy := attachCnt
	rxTimeCopy := rxTime
	attachUpdates := models.EndpointAttachmentStateParams{
		AttachCnt:        &attachCntCopy,
		LastAttachRxTime: &rxTimeCopy,
		Nonce:            nonce,
		Sign:             sign,
	}

	if hasRxDuration {
		rxDurationCopy := rxDuration
		attachUpdates.LastAttachRxDuration = &rxDurationCopy
	}
	if hasDualChan {
		dualChanCopy := dualChan
		attachUpdates.DualChan = &dualChanCopy
	}
	if hasRepetition {
		repetitionCopy := repetition
		attachUpdates.Repetition = &repetitionCopy
	}
	if hasWideCarrOff {
		wideCarrOffCopy := wideCarrOff
		attachUpdates.WideCarrOff = &wideCarrOffCopy
	}
	if hasLongBlkDist {
		longBlkDistCopy := longBlkDist
		attachUpdates.LongBlkDist = &longBlkDistCopy
	}
	shAddrCopy := responseShAddr
	attachUpdates.ShAddr = &shAddrCopy

	// Persist subpackets if present (BSSCI §5.6.1 optional field)
	if subpackets != nil {
		if encoded, err := json.Marshal(subpackets); err == nil {
			encodedStr := string(encoded)
			attachUpdates.LastAttachSubpackets = &encodedStr
		}
	}

	// The attachment persister owns the transactional endpoint + session
	// upsert; any persistence failure keeps the exact response behavior:
	// remove the pending operation and answer with the database error.
	rec := AttachSessionRecord{
		TenantID:         ownerTenantID,
		BSLookupTenantID: servingTenantID,
		EndpointID:       endpoint.ID,
		EndpointUpdates:  attachUpdates,
		EncryptedKey:     sessionKey,
		//nolint:gosec // G115: attachCnt validated to be within 0..AttachCounterMax above
		AttachCnt:      uint32(attachCnt),
		ShAddr:         responseShAddr,
		BaseStationEUI: session.BaseStationEUIBytes(),
	}
	if err := s.attachPersistence.PersistAttachSession(ownerCtx, rec); err != nil {
		if errors.Is(err, ErrAttachCounterStale) {
			return s.refuseStaleAttachCounter(session, msg,
				logger.FieldTenantIDCamel, ownerTenantID,
				logger.FieldEpEui, epEUI,
				logger.FieldIncomingAttachCnt, attachCnt,
				logger.FieldError, err)
		}
		if rmErr := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); rmErr != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, rmErr)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
	}

	// Update radio metrics using selective update (preserves optional fields)
	var eui models.EUI
	if len(epEUIBytes) == 8 {
		copy(eui[:], epEUIBytes)

		update := models.RadioMetricsUpdate{
			SNR:        snr,
			RSSI:       rssi,
			EqSNR:      eqSnr,
			RxTime:     rxTime,
			RxDuration: nil, // preserve if not present
			Profile:    nil, // preserve if not present
		}

		if hasRxDuration {
			update.RxDuration = &rxDuration
		}

		// BSSCI-ATTACH-023: Only update profile when non-empty to preserve existing DB value
		if hasProfile && profile != "" {
			update.Profile = &profile
		}

		if err := s.endpointRepo.UpdateRadioMetricsSelective(ownerCtx, ownerTenantID, eui, update); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateRadioMetrics, logger.FieldError, err)
		}
	}

	// Build attach response per BSSCI 3.6.2
	response := map[string]interface{}{
		"command": mioty.CmdAttachResponse,
		"opId":    msg.OpId,
	}

	numericKey := make([]interface{}, dbconfig.SessionKeySize)
	for i := 0; i < dbconfig.SessionKeySize; i++ {
		if i < len(sessionKey) {
			numericKey[i] = int(sessionKey[i])
		} else {
			numericKey[i] = 0
		}
	}

	response["nwkSnKey"] = numericKey
	if scAssignedShAddr {
		response["shAddr"] = responseShAddr
	}

	return s.sendMessage(session, response)
}

// handleAttachComplete handles attach complete operations per BSSCI 3.6.3
func (s *Server) handleAttachComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// Retrieve the pending attach operation
	// StatusService is the single path for pending operation persistence
	pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId))

	if err != nil || pendingOp == nil || pendingOp.OperationType != mioty.CmdAttach {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIReceivedAttCmpWithoutPendingAttach,
			logger.FieldBaseStation, session.BaseStationEUI,
			logger.FieldOpID, int64(msg.OpId))
		return fmt.Errorf(errFmtTokenForOpID, ResolveErrorMessage(errNoPendingAttachOperation), msg.OpId)
	}

	// Extract the endpoint EUI from pending operation
	epEUI, _ := parseMetadataEUI(pendingOp.Metadata[models.EventDetailKeyEpEui])

	// Log successful completion
	s.logger.InfoContext(s.safeCtx(), LogBSSCIAttachOperationCompletedSuccessfully,
		logger.FieldBaseStation, session.BaseStationEUI,
		logger.FieldEndPoint, epEUI,
		logger.FieldOpID, int64(msg.OpId),
		logger.FieldDuration, s.clock.Now().Sub(pendingOp.CreatedAt))

	// The completion belongs to the endpoint's owner, not the tenant of the
	// base station that heard it.
	ownerTenantID, ownerKnown := getNumericField(pendingOp.Metadata, metadataKeyEndpointTenantID)
	if !ownerKnown {
		ownerTenantID = resolvedTenant(session, s.tenantID)
		s.logger.WarnContext(ctx, LogBSSCIMissingTenantInMetadata, logger.FieldOpID, msg.OpId)
	}
	ownerCtx, _ := s.endpointOwnerContext(ownerTenantID)

	s.recordAttachEvent(ownerCtx, session, ownerTenantID, epEUI, pendingOp.Metadata)

	// Clear the pending operation
	// BSSCI §§5.11-5.12.3 Gap 1: Use removePendingOperation helper (has dual-path logic)
	if err := s.pendingOps.remove(s.sessionContext(session), session, int64(msg.OpId)); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToClearPersistedPendingOperation,
			logger.FieldError, err,
			logger.FieldOpID, msg.OpId)
	}

	if endpointID, ok := getNumericField(pendingOp.Metadata, metadataKeyEndpointID); ok && endpointID > 0 && epEUI != 0 {
		s.decideOverTheAirAttach(ownerCtx, session, AttachmentDecision{
			TenantID:   ownerTenantID,
			EndpointID: endpointID,
			EpEUI:      epEUI,
			Status:     EndpointStatusAttached,
		}, pendingOp.Metadata)
	}

	// No response needed for complete messages per BSSCI spec
	return nil
}

// recordAttachEvent records the completed attach as a system event of the
// endpoint owner.
func (s *Server) recordAttachEvent(ownerCtx context.Context, heardBy *Session, ownerTenantID int64, epEUI uint64, metadata map[string]interface{}) {
	attachCnt, _ := metadata["attachCnt"].(int64)
	rxTime, _ := metadata["rxTime"].(int64)
	rssi, _ := metadata["rssi"].(float64)
	snr, _ := metadata["snr"].(float64)
	details := map[string]interface{}{
		models.EventDetailKeyEpEui:     mioty.FormatEUI64(epEUI),
		models.EventDetailKeyBsEui:     mioty.FormatEUI64(heardBy.BaseStationEUI),
		models.EventDetailKeyAttachCnt: attachCnt,
		models.EventDetailKeyRxTime:    rxTime,
		models.EventDetailKeyRssi:      rssi,
		models.EventDetailKeySnr:       snr,
		models.EventDetailKeyOperation: "attach_complete",
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateAttachEvent, logger.FieldError, err)
		return
	}

	if err := s.eventStore.CreateEvent(ownerCtx, &models.SystemEvent{
		TenantID:    fmt.Sprintf("%d", ownerTenantID),
		EventType:   models.EventTypeEndpointAttached,
		Category:    mioty.CategoryEndpoint,
		Severity:    SeverityInfo,
		Title:       fmt.Sprintf(models.EventTitleEndpointAttachedViaBS, mioty.FormatEUI64(epEUI), mioty.FormatEUI64(heardBy.BaseStationEUI)),
		Description: eventDescEndpointAttached,
		Details:     detailsJSON,
		Status:      EventStatusNew,
		CreatedAt:   s.clock.Now(),
	}); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIFailedToCreateAttachEvent, logger.FieldError, err)
	}
}

// decideOverTheAirAttach has the attachment decider attach the endpoint a
// completed attach names (BSSCI §3.6.3), which tells its owner's application
// centers and MQTT subscribers (SCACI §3.13), and sends the attachment to the
// other connected base stations (BSSCI §5.8.2).
func (s *Server) decideOverTheAirAttach(ownerCtx context.Context, heardBy *Session, decision AttachmentDecision, metadata map[string]interface{}) {
	status := &EPStatusData{EpEui: decision.EpEUI, EpStatus: pkgmioty.EPStatusAttached}
	applyOverTheAirAttachFields(status, metadata)
	decision.OverTheAir = &OverTheAirReport{BaseStationEUI: heardBy.BaseStationEUI, Status: status}
	if _, err := s.attachmentDecider.Decide(ownerCtx, decision); err != nil {
		s.logger.ErrorContext(ownerCtx, LogBSSCIFailedToDecideOverTheAirAttach,
			logger.FieldEpEui, decision.EpEUI, logger.FieldError, err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.propagationSvc.TriggerEndpointPropagate(ownerCtx, decision.EndpointID, s.ConnectedSessionsSnapshot()); err != nil {
			s.logger.ErrorContext(ownerCtx, LogBSSCIAutomaticPropagationFailedAfterOTAAttach,
				logger.FieldEndpointID, decision.EndpointID, logger.FieldError, err)
		}
	}()
}

// AttachCounterAdvances reports whether an attach counter moves past the stored
// one, allowing the 24-bit rollover; an endpoint with no stored counter accepts any.
func AttachCounterAdvances(stored *uint32, incoming int64) bool {
	if stored == nil {
		return true
	}
	storedCnt := int64(*stored)
	rollover := storedCnt > dbconfig.AttachCounterRolloverHigh && incoming < dbconfig.AttachCounterRolloverLow
	return rollover || incoming > storedCnt
}

// refuseStaleAttachCounter answers an att whose attach counter does not advance the stored one.
func (s *Server) refuseStaleAttachCounter(session *Session, msg *Message, fields ...interface{}) error {
	s.logger.WarnContext(s.safeCtx(), LogBSSCIAttachCounterReplay, fields...)
	if err := s.pendingOps.remove(s.sessionContext(session), session, msg.OpId); err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, logger.FieldError, err)
	}
	return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errAttachCounterNotMonotonic))
}

// applyOverTheAirAttachFields fills the SCACI §3.13.1 over-the-air fields of an
// attached epStat from the attach's recovery record; the optional ones only
// when the att carried them.
func applyOverTheAirAttachFields(status *EPStatusData, metadata map[string]interface{}) {
	if attachCnt, ok := getNumericField(metadata, "attachCnt"); ok {
		if v, errToken := safeUint32(attachCnt); errToken == "" {
			status.AttachCnt = &v
		}
	}
	if snr, ok := getFloatFieldValidated(metadata, wireFieldSnr); ok {
		status.Snr = &snr
	}
	if rssi, ok := getFloatFieldValidated(metadata, wireFieldRssi); ok {
		status.Rssi = &rssi
	}
	if eqSnr, ok := getFloatFieldValidated(metadata, wireFieldEqSnr); ok {
		status.EqSnr = &eqSnr
	}
	status.Nonce = metadataNumeric4(metadata[fieldNameNonce])
	status.Sign = metadataNumeric4(metadata[wireFieldSign])
	status.Subpackets = metadataSubpackets(metadata)
}

// metadataNumeric4 reads a 4-byte recovery-record field as a Numeric[4].
func metadataNumeric4(value interface{}) *mioty.Numeric4 {
	var n mioty.Numeric4
	b, ok := recordBytes(value)
	if !ok || len(b) != len(n) {
		return nil
	}
	copy(n[:], b)
	return &n
}

// optionalSubpackets reads the optional subpackets object of an att or det,
// which BSSCI §3.6.1 and §3.7.1 define as the §3.10.1 object: absent is
// valid, a malformed one is a protocol error (§2.4).
func optionalSubpackets(data map[string]interface{}) (*mioty.Subpackets, bool) {
	raw := data[wireFieldSubpackets]
	if raw == nil {
		return nil, true
	}
	return parseSubpackets(raw)
}

// metadataSubpackets reads the subpackets object a recovery record kept as
// received; nil when the station reported none.
func metadataSubpackets(metadata map[string]interface{}) *mioty.Subpackets {
	raw, ok := metadata[wireFieldSubpackets].(map[string]interface{})
	if !ok {
		return nil
	}
	subpackets, err := NormalizeSubpackets(raw)
	if err != nil {
		return nil
	}
	return subpackets
}
