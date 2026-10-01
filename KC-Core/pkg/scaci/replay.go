package scaci

import (
	"cmp"
	"context"
	"net"
	"slices"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	// Shared MIOTY helpers (FormatEUI64, EPStatus)
	// Organization resolver for propagation context
	// BSSCI §5.8-5.8.3 attach propagation contracts
	// Import neutral scheduler contracts

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/encoding"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// replayPendingOperations retrieves and resends incomplete SC-initiated operations per SCACI §1
// "In the case of a connection loss and reestablishment... only operations, which had not been
// completed before the connection loss are reissued."
//
// Uses session context to preserve tenant/org isolation during replay.
func (s *Server) replayPendingOperations(conn net.Conn, session *Session) {
	if s.operationRepo == nil || session.ID <= 0 {
		return
	}

	// Use session context for tenant isolation
	ctx, cancel := context.WithTimeout(s.sessionContext(session), ReplayOperationTimeout)
	defer cancel()

	pendingOps, err := s.operationRepo.GetPendingOperations(ctx, session.ID)
	if err != nil {
		s.logger.WarnContext(s.sessionContext(session), LogSCACIGetPendingOpsFailed, logger.FieldError, err)
		return
	}
	// Reissued in the order they were issued: SC opIds strictly decrement (SCACI §3.2).
	slices.SortStableFunc(pendingOps, func(a, b *models.SCACIOperation) int { return cmp.Compare(b.OpId, a.OpId) })

	for _, op := range pendingOps {
		if !serviceCenterIssued(op) {
			continue
		}

		// Verify tenant matches session (cross-tenant protection)
		if op.TenantID != session.TenantID {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACICrossTenantReplayRejected,
				logger.FieldOpTenantID, op.TenantID, logger.FieldSessionTenantID, session.TenantID)
			continue
		}

		s.logger.InfoContext(s.sessionContext(session), LogSCACIReplayingPendingOp,
			logger.FieldOpID, op.OpId, logger.FieldCommand, op.Command)
		if err := s.replaySingleOperation(conn, session, op); err != nil {
			s.logger.ErrorContext(s.sessionContext(session), LogSCACIReplayOpFailed,
				logger.FieldOpID, op.OpId, logger.FieldCommand, op.Command, logger.FieldError, err)
		}
	}
}

// serviceCenterIssued reports an operation the service center initiated
// (SCACI §3.2: negative opIds), the only kind a resume reissues (§1).
func serviceCenterIssued(op *models.SCACIOperation) bool {
	return op.OpId < 0 && op.Direction == string(models.OperationDirectionOutbound)
}

// replaySingleOperation dispatches replay to command-specific handlers
func (s *Server) replaySingleOperation(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	spec, known := s.commands.lookup(op.Command)
	if !known || !spec.Replayable {
		if known && spec.NonReplayReason != "" {
			s.logger.DebugContext(s.sessionContext(session), LogSCACISkipNonReplayable,
				logger.FieldCommand, op.Command, logger.FieldReason, spec.NonReplayReason)
		} else {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIUnknownReplayCommand,
				logger.FieldCommand, op.Command)
		}
		return nil
	}
	return s.sendSCOperation(session, op.OpId, op.Command, func() error {
		return spec.Replay(s, conn, session, op)
	})
}

// replayDLDataResult reconstructs and resends dlDataRes from stored RequestData
// Storage format: map[string]interface{} with epEui/queId/result fields
func (s *Server) replayDLDataResult(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	data := op.RequestData

	epEui, ok := extractEUIFromJSON(data, MetadataKeyEpEui)
	if !ok {
		return errMissingStoredEpEui
	}

	queId, _ := extractUint64FromJSON(data, "queId")
	result, _ := data["result"].(string)

	msg := DLDataResult{
		BaseMessage: BaseMessage{
			Command: CmdDLDataResult,
			OpId:    op.OpId, // Preserve original opId for replay
		},
		EpEui:  epEui,
		QueID:  queId,
		Result: result,
	}

	// Extract optional fields
	if txTime, ok := extractInt64FromJSON(data, "txTime"); ok {
		msg.TxTime = &txTime
	}
	if packetCnt, ok := extractInt64FromJSON(data, "packetCnt"); ok {
		packetCntU32 := uint32(packetCnt) // #nosec G115 - SCACI §3.12 defines PacketCnt as 32-bit
		msg.PacketCnt = &packetCntU32
	}
	if bsEui, ok := extractEUIFromJSON(data, metadataKeyBsEui); ok {
		msg.BsEui = &bsEui
	}

	// Validate and send via helper (SCACI §3.12.1)
	return s.SendDLDataResult(conn, session, &msg)
}

// replayEPStatus reconstructs and resends epStatus from stored RequestData
// Handles base64-encoded nonce/sign and validates before send via SendEPStatus
func (s *Server) replayEPStatus(conn net.Conn, session *Session, op *models.SCACIOperation) error {
	data := op.RequestData

	epEui, ok := extractEUIFromJSON(data, MetadataKeyEpEui)
	if !ok {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIReplayEPStatusInvalidData, logger.FieldOpID, op.OpId, logger.FieldField, MetadataKeyEpEui)
		return errMissingStoredEpEui
	}

	// epStatus stored as string enum
	epStatus, ok := data[metadataKeyEpStatus].(string)
	if !ok || epStatus == "" {
		s.logger.ErrorContext(s.sessionContext(session), LogSCACIReplayEPStatusInvalidData, logger.FieldOpID, op.OpId, logger.FieldField, metadataKeyEpStatus)
		return errMissingStoredEpStatus
	}

	// Reconstruct EPStatus with preserved opId AND command
	msg := &EPStatus{
		BaseMessage: BaseMessage{
			Command: CmdEPStatus, // MUST set command explicitly
			OpId:    op.OpId,
		},
		EpEui:    epEui,
		EpStatus: epStatus,
	}

	// Restore optional OTA fields
	// attachCnt stored as float64 (JSON number)
	if attachCntF, ok := data["attachCnt"].(float64); ok {
		v := uint32(attachCntF)
		msg.AttachCnt = &v
	}

	// nonce/sign stored as base64 strings (via encoding.EncodeUserData)
	if nonceStr, ok := data[metadataKeyNonce].(string); ok && nonceStr != "" {
		nonceBytes, err := encoding.DecodeUserData(nonceStr)
		if err == nil && len(nonceBytes) == 4 {
			var nonce mioty.Numeric4
			copy(nonce[:], nonceBytes)
			msg.Nonce = &nonce
		} else {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIReplayEPStatusFieldDecodeErr, logger.FieldOpID, op.OpId, logger.FieldField, metadataKeyNonce)
		}
	}
	if signStr, ok := data[metadataKeySign].(string); ok && signStr != "" {
		signBytes, err := encoding.DecodeUserData(signStr)
		if err == nil && len(signBytes) == 4 {
			var sign mioty.Numeric4
			copy(sign[:], signBytes)
			msg.Sign = &sign
		} else {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIReplayEPStatusFieldDecodeErr, logger.FieldOpID, op.OpId, logger.FieldField, metadataKeySign)
		}
	}

	// snr/rssi/eqSnr stored as float64
	if snr, ok := data["snr"].(float64); ok {
		msg.Snr = &snr
	}
	if rssi, ok := data["rssi"].(float64); ok {
		msg.Rssi = &rssi
	}
	if eqSnr, ok := data["eqSnr"].(float64); ok {
		msg.EqSnr = &eqSnr
	}

	// Reconstruct subpackets if present (JSON array → mioty.Subpackets)
	if subpacketsRaw, ok := data[metadataKeySubpackets]; ok && subpacketsRaw != nil {
		subpackets, err := reconstructSubpackets(subpacketsRaw)
		if err != nil {
			s.logger.WarnContext(s.sessionContext(session), LogSCACIReplayEPStatusFieldDecodeErr, logger.FieldOpID, op.OpId, logger.FieldField, metadataKeySubpackets, logger.FieldError, err)
			// Continue without subpackets - non-fatal
		} else {
			msg.Subpackets = subpackets
		}
	}

	s.logger.InfoContext(s.sessionContext(session), LogSCACIReplayingEPStatus,
		logger.FieldOpID, op.OpId, logger.FieldEpEui, mioty.FormatEUI64(epEui), logger.FieldEpStatus, epStatus)

	// Use SendEPStatus for validation and logging (NOT sendResponse)
	return s.SendEPStatus(conn, session, msg)
}

// reconstructSubpackets converts JSON-decoded subpackets back to mioty.Subpackets
// mioty.Subpackets is a struct with SNR, RSSI, Frequency, Phase slices
func reconstructSubpackets(raw interface{}) (*mioty.Subpackets, error) {
	// Handle JSON-decoded map[string]interface{} → mioty.Subpackets
	rawMap, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errSubpacketsNotAMap
	}

	result := &mioty.Subpackets{}

	// Extract SNR slice
	if snrRaw, ok := rawMap["snr"].([]interface{}); ok {
		for _, v := range snrRaw {
			if f, ok := v.(float64); ok {
				result.SNR = append(result.SNR, f)
			}
		}
	}

	// Extract RSSI slice
	if rssiRaw, ok := rawMap["rssi"].([]interface{}); ok {
		for _, v := range rssiRaw {
			if f, ok := v.(float64); ok {
				result.RSSI = append(result.RSSI, f)
			}
		}
	}

	// Extract Frequency slice
	if freqRaw, ok := rawMap["frequency"].([]interface{}); ok {
		for _, v := range freqRaw {
			if f, ok := v.(float64); ok {
				result.Frequency = append(result.Frequency, int64(f))
			}
		}
	}

	// Extract Phase slice (optional)
	if phaseRaw, ok := rawMap["phase"].([]interface{}); ok {
		for _, v := range phaseRaw {
			if f, ok := v.(float64); ok {
				result.Phase = append(result.Phase, f)
			}
		}
	}

	return result, nil
}
