package bssci

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// attachFields holds all validated fields extracted from an attach request message.
type attachFields struct {
	epEUI      int64
	rxTime     int64
	attachCnt  int64
	snr        float64
	rssi       float64
	eqSnr      float64
	nonce      []byte
	sign       []byte
	dualChan   bool
	repetition bool
	wideCarrOff bool
	longBlkDist bool
	rxDuration int64
	profile    string
	shAddr     int64
	subpackets []interface{}

	// Presence flags for optional fields
	hasDualChan   bool
	hasRepetition bool
	hasWideCarrOff bool
	hasLongBlkDist bool
	hasRxDuration bool
	hasProfile    bool
	hasShAddr     bool
}

// extractAttachFields parses and validates all mandatory and optional fields from the
// attach request message per BSSCI §5.6.1.
func (s *Server) extractAttachFields(session *Session, msg *Message, data map[string]interface{}) (*attachFields, error) {
	f := &attachFields{}

	// Mandatory: epEui
	var hasEpEUI bool
	f.epEUI, hasEpEUI = getNumericField(data, "epEui")
	if !hasEpEUI {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingEpEui))
	}

	// Mandatory: rxTime, attachCnt
	var hasRxTime, hasAttachCnt bool
	f.rxTime, hasRxTime = getNumericField(data, "rxTime")
	f.attachCnt, hasAttachCnt = getNumericField(data, "attachCnt")

	if !hasRxTime {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingRxTime))
	}
	if !hasAttachCnt {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingAttachCnt))
	}
	// Validate attachCnt is 24-bit unsigned (0 to 0xFFFFFF per BSSCI §5.6.1)
	if f.attachCnt < 0 || f.attachCnt > 0xFFFFFF {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidAttachCntRange))
	}

	// Mandatory: snr, rssi
	var hasValidSnr, hasValidRssi bool
	f.snr, hasValidSnr = getFloatFieldValidated(data, "snr")
	f.rssi, hasValidRssi = getFloatFieldValidated(data, "rssi")

	if !hasValidSnr {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSnrValue))
	}
	if !hasValidRssi {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRssiValue))
	}

	// Mandatory: nonce (4-byte array)
	nonceData, ok := data["nonce"]
	if !ok {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingNonce))
	}
	var errToken string
	f.nonce, errToken = validateByteArray(nonceData, "nonce", 4)
	if errToken != "" {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	// Mandatory: sign (4-byte array)
	signData, ok := data["sign"]
	if !ok {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingSign))
	}
	f.sign, errToken = validateByteArray(signData, "sign", 4)
	if errToken != "" {
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errToken))
	}

	// Optional radio capability flags
	if _, ok := data["dualChan"]; ok {
		f.hasDualChan = true
		f.dualChan = getBoolField(data, "dualChan", false)
	}
	if _, ok := data["repetition"]; ok {
		f.hasRepetition = true
		f.repetition = getBoolField(data, "repetition", false)
	}
	if _, ok := data["wideCarrOff"]; ok {
		f.hasWideCarrOff = true
		f.wideCarrOff = getBoolField(data, "wideCarrOff", false)
	}
	if _, ok := data["longBlkDist"]; ok {
		f.hasLongBlkDist = true
		f.longBlkDist = getBoolField(data, "longBlkDist", false)
	}

	// Optional: eqSnr (defaults to snr)
	f.eqSnr = f.snr
	if rawEqSnr, exists := data["eqSnr"]; exists && rawEqSnr != nil {
		if eqSnrFloat, valid := getFloatFieldValidated(data, "eqSnr"); valid {
			f.eqSnr = eqSnrFloat
		} else {
			return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidEqSnrValue))
		}
	}

	// Optional: rxDuration
	f.rxDuration, f.hasRxDuration = getNumericField(data, "rxDuration")

	// Optional: profile
	if profileData, exists := data["profile"]; exists {
		if profileStr, ok := profileData.(string); ok {
			f.profile = profileStr
			f.hasProfile = true
		}
	}

	// Optional: subpackets
	if sp, ok := data["subpackets"]; ok {
		if spArray, ok := sp.([]interface{}); ok {
			f.subpackets = spArray
		}
	}

	// Optional: shAddr (BS-assigned short address)
	f.shAddr, f.hasShAddr = getNumericField(data, "shAddr")

	return f, nil
}

// buildAttachPendingData constructs the metadata map stored with the pending attach operation.
func (f *attachFields) buildAttachPendingData() map[string]interface{} {
	pendingData := map[string]interface{}{
		"epEui":       pkgmioty.FormatEUI64(uint64(f.epEUI)),
		"attachCnt":   f.attachCnt,
		"rxTime":      f.rxTime,
		"nonce":       f.nonce,
		"sign":        f.sign,
		"dualChan":    f.dualChan,
		"repetition":  f.repetition,
		"wideCarrOff": f.wideCarrOff,
		"longBlkDist": f.longBlkDist,
		"rssi":        f.rssi,
		"snr":         f.snr,
		"eqSnr":       f.eqSnr,
		"subpackets":  f.subpackets,
	}

	if f.hasRxDuration {
		pendingData["rxDuration"] = f.rxDuration
	}
	// BSSCI-ATTACH-023: Empty profile preserves existing DB value (consistent with detach)
	if f.hasProfile && f.profile != "" {
		pendingData["profile"] = f.profile
	}
	return pendingData
}

// attachEndpointResult holds the resolved endpoint and related state needed after
// endpoint lookup, roaming check, replay protection, and crypto validation.
type attachEndpointResult struct {
	endpoint          *models.EndPoint
	tenantID          int64
	epEUIBytes        []byte
	nwkSnKey          []byte
	sessionKey        []byte
	encryptedKey      []byte
	responseShAddr    uint16
	scAssignedShAddr  bool
	ownerTenantID     int64
	isRoaming         bool
}

// resolveAttachEndpoint performs endpoint lookup, roaming validation, replay protection,
// signature verification, session key derivation, and short address resolution.
func (s *Server) resolveAttachEndpoint(
	ctx context.Context,
	session *Session,
	msg *Message,
	f *attachFields,
) (*attachEndpointResult, error) {
	epEUIBytes := make([]byte, 8)
	if f.epEUI < 0 {
		return nil, s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errInvalidFieldValue))
	}
	binary.BigEndian.PutUint64(epEUIBytes, uint64(f.epEUI))

	tenantID := resolvedTenant(session, s.tenantID)
	endpoint, err := s.endpointRepo.GetByEUI(ctx, tenantID, epEUIBytes)
	if err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIEndpointNotProvisionedForAttach,
			"epEui", f.epEUI, "error", err)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errEndpointNotProvisioned))
	}

	// Store endpoint metadata for attach propagation (BSSCI §5.8.2)
	if s.statusSvc != nil {
		if pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId)); err == nil {
			pendingOp.Metadata["endpointID"] = endpoint.ID
			pendingOp.Metadata["endpointTenantID"] = endpoint.TenantID
		}
	}

	result := &attachEndpointResult{
		endpoint:   endpoint,
		tenantID:   tenantID,
		epEUIBytes: epEUIBytes,
	}

	// Roaming detection and validation
	if s.roamingSvc != nil {
		isRoaming, ownerTenantID, err := s.roamingSvc.DetectAndValidateRoaming(ctx, epEUIBytes, tenantID)
		if err != nil {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIRoamingValidationFailed,
				"epEui", f.epEUI, "servingTenant", tenantID, "error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return nil, s.sendError(session, msg.OpId, POSIX_EPERM, ResolveErrorMessage(errRoamingNotAllowed))
		}

		if isRoaming {
			s.logger.InfoContext(s.safeCtx(), LogBSSCIRoamingEndpointAttaching,
				"epEui", f.epEUI, "ownerTenant", ownerTenantID, "servingTenant", tenantID)

			if err := s.roamingSvc.RecordAttach(ctx, epEUIBytes, session.BaseStationEUIBytes(), tenantID); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRecordRoamingAttach, "error", err)
			}

			if session.DbSessionID > 0 {
				if err := s.roamingSvc.UpdateSessionRoaming(ctx, session.DbSessionID, epEUIBytes, true, tenantID); err != nil {
					s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateSessionRoaming, "error", err)
				}
			}

			// Store roaming metadata in pending operation
			if s.statusSvc != nil {
				if pendingOp, err := s.statusSvc.GetPendingOperation(session, int64(msg.OpId)); err == nil {
					pendingOp.Metadata["ownerTenantID"] = ownerTenantID
					pendingOp.Metadata["isRoaming"] = true
				}
			}

			result.isRoaming = true
			result.ownerTenantID = ownerTenantID
		}
	}

	// Network session key validation
	result.nwkSnKey = endpoint.NwkSnKey
	if len(result.nwkSnKey) != 16 {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIInvalidNetworkKeyLengthForAttach,
			"epEui", f.epEUI, "keyLength", len(result.nwkSnKey))
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errNwkSnKeyInvalidLength))
	}

	// Replay protection: enforce monotonic attach counter (BSSCI §5.6.1)
	if endpoint.AttachCnt != nil {
		storedCnt := int64(*endpoint.AttachCnt)
		isRollover := storedCnt > 0xFFFF00 && f.attachCnt < 0x100
		if !isRollover && f.attachCnt <= storedCnt {
			s.logger.WarnContext(s.safeCtx(), LogBSSCIAttachCounterReplay,
				"tenantId", tenantID, "epEui", f.epEUI,
				"storedAttachCnt", storedCnt, "incomingAttachCnt", f.attachCnt)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errAttachCounterNotMonotonic))
		}
	}

	// Signature verification
	//nolint:gosec // G115: attachCnt validated to be <= 0xFFFFFF, safe for uint32
	if err := ValidateAttachSignature(uint64(f.epEUI), uint32(f.attachCnt), f.sign, result.nwkSnKey); err != nil {
		s.logger.WarnContext(s.safeCtx(), LogBSSCIAttachSignatureValidationFailed,
			"epEui", f.epEUI, "error", err)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
	}

	// Session key derivation
	sessionKey, err := DeriveSessionKey(uint64(f.epEUI), f.nonce, f.sign, result.nwkSnKey)
	if err != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToDeriveSessionKey,
			"epEui", f.epEUI, "error", err)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSignature))
	}
	result.sessionKey = sessionKey

	// Session key encryption for storage
	result.encryptedKey = sessionKey
	if s.keyEncryptor != nil {
		encrypted, err := s.keyEncryptor.EncryptKeyRaw(sessionKey)
		if err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToEncryptSessionKey, "error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return nil, s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errFailedToEncryptKey))
		}
		result.encryptedKey = encrypted
	}

	// Short address resolution
	if f.hasShAddr {
		if f.shAddr < 0 || f.shAddr > 65535 {
			return nil, s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errInvalidFieldValue))
		}
		result.responseShAddr = uint16(f.shAddr)
		if endpoint.ShAddr == nil || *endpoint.ShAddr != result.responseShAddr {
			result.scAssignedShAddr = false
		}
	} else if endpoint.ShAddr != nil {
		result.responseShAddr = *endpoint.ShAddr
		result.scAssignedShAddr = true
	} else {
		result.responseShAddr = uint16(f.epEUI & 0xFFFF) //nolint:gosec // G115: bitwise AND guarantees value fits uint16
		result.scAssignedShAddr = true
	}

	return result, nil
}

// buildAttachFieldUpdates constructs the database update map for the endpoint attach metadata.
func buildAttachFieldUpdates(f *attachFields, r *attachEndpointResult, data map[string]interface{}) map[string]interface{} {
	updates := map[string]interface{}{
		"attach_cnt":          f.attachCnt,
		"last_attach_rx_time": f.rxTime,
		"nonce":               f.nonce,
		"sign":                f.sign,
	}

	if f.hasRxDuration {
		updates["last_attach_rx_duration"] = f.rxDuration
	}
	if f.hasDualChan {
		updates["dual_chan"] = f.dualChan
	}
	if f.hasRepetition {
		updates["repetition"] = f.repetition
	}
	if f.hasWideCarrOff {
		updates["wide_carr_off"] = f.wideCarrOff
	}
	if f.hasLongBlkDist {
		updates["long_blk_dist"] = f.longBlkDist
	}
	if f.hasShAddr || r.scAssignedShAddr {
		updates["sh_addr"] = r.responseShAddr
	}

	// Persist subpackets if present (BSSCI §5.6.1 optional field)
	if raw, ok := data["subpackets"].(map[string]interface{}); ok {
		if sp, err := NormalizeSubpackets(raw); err == nil {
			if encoded, err := json.Marshal(sp); err == nil {
				updates["last_attach_subpackets"] = string(encoded)
			}
		}
	}

	return updates
}

// buildRadioMetricsUpdate constructs the selective radio metrics update from attach fields.
func (f *attachFields) buildRadioMetricsUpdate() interfaces.RadioMetricsUpdate {
	update := interfaces.RadioMetricsUpdate{
		SNR:    f.snr,
		RSSI:   f.rssi,
		EqSNR:  f.eqSnr,
		RxTime: f.rxTime,
	}

	if f.hasRxDuration {
		update.RxDuration = &f.rxDuration
	}

	// BSSCI-ATTACH-023: Only update profile when non-empty to preserve existing DB value
	if f.hasProfile && f.profile != "" {
		update.Profile = &f.profile
	}

	return update
}

// persistAttachSession runs the transactional endpoint update and session create/update
// within a single database transaction.
func (s *Server) persistAttachSession(
	ctx context.Context,
	session *Session,
	msg *Message,
	f *attachFields,
	r *attachEndpointResult,
	attachUpdates map[string]interface{},
) error {
	tx, txErr := s.storage.BeginTx(ctx)
	if txErr != nil {
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToBeginTransaction, "error", txErr)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
	}

	var commitErr error
	defer func() {
		if commitErr != nil {
			_ = tx.Rollback()
		}
	}()

	if err := tx.EndPoints().UpdateFields(ctx, r.tenantID, r.endpoint.ID, attachUpdates); err != nil {
		commitErr = err
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointAttachMetadata, "error", err)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
	}

	endpointIDStr := fmt.Sprintf("%d", r.endpoint.ID)
	activeSession, getErr := tx.EndPointSessions().GetActive(ctx, endpointIDStr)
	if getErr != nil {
		commitErr = getErr
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToLoadEndpointSession, "error", getErr)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
	}

	now := time.Now().UTC()

	// Lookup base station ID for session enrichment (tenant-scoped)
	var primaryBsID *int64
	if s.basestationRepo != nil {
		bs, bsErr := s.basestationRepo.GetByEUI(ctx, resolvedTenant(session, s.tenantID), session.BaseStationEUIBytes())
		if bsErr == nil && bs != nil {
			primaryBsID = &bs.ID
		}
	}

	shAddrInt32 := int32(r.responseShAddr)

	if activeSession != nil {
		activeSession.SessionKey = r.encryptedKey
		//nolint:gosec // G115: attachCnt validated to be <= 0xFFFFFF, safe for int32
		activeSession.AttachCnt = int32(f.attachCnt)
		activeSession.LastActivityAt = now
		activeSession.ShAddr = &shAddrInt32
		activeSession.PrimaryBaseStationID = primaryBsID
		if err := tx.EndPointSessions().Update(ctx, activeSession); err != nil {
			commitErr = err
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToUpdateEndpointSession, "error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}
	} else {
		newSession := &models.EndPointSession{
			TenantID:   r.tenantID,
			EndPointID: r.endpoint.ID,
			SessionID:  uuid.New().String(),
			SessionKey: r.encryptedKey,
			//nolint:gosec // G115: attachCnt validated to be <= 0xFFFFFF, safe for int32
			AttachCnt:            int32(f.attachCnt),
			Status:               "active",
			StartedAt:            now,
			LastActivityAt:       now,
			ShAddr:               &shAddrInt32,
			PrimaryBaseStationID: primaryBsID,
		}
		if err := tx.EndPointSessions().Create(ctx, newSession); err != nil {
			commitErr = err
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToCreateEndpointSession, "error", err)
			if err := s.removePendingOperation(session, msg.OpId); err != nil {
				s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
			}
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
		}
	}

	if err := tx.Commit(); err != nil {
		commitErr = err
		s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToCommitAttachTransaction, "error", err)
		if err := s.removePendingOperation(session, msg.OpId); err != nil {
			s.logger.ErrorContext(s.safeCtx(), LogBSSCIFailedToRemovePendingOperation, "error", err)
		}
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errDatabaseError))
	}
	commitErr = nil

	return nil
}

// sendAttachResponse builds and sends the attachRsp message per BSSCI §5.6.2.
func (s *Server) sendAttachResponse(session *Session, msg *Message, r *attachEndpointResult) error {
	response := map[string]interface{}{
		"command": mioty.CmdAttachResponse,
		"opId":    msg.OpId,
	}

	numericKey := make([]interface{}, 16)
	for i := 0; i < 16; i++ {
		if i < len(r.sessionKey) {
			numericKey[i] = int(r.sessionKey[i])
		} else {
			numericKey[i] = 0
		}
	}

	response["nwkSnKey"] = numericKey
	if r.scAssignedShAddr {
		response["shAddr"] = r.responseShAddr
	}

	return s.sendMessage(session, response)
}
