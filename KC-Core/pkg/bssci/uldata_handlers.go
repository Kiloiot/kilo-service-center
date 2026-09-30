package bssci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger" // Shared MIOTY helpers (FormatEUI64, EPStatus)
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// handleULData handles uplink data operations
func (s *Server) handleULData(session *Session, msg *Message, data map[string]interface{}) error {
	ctx := s.sessionContext(session)

	// epEui is a full-range unsigned EUI-64 (BSSCI §5.10.1)
	epEUI, ok := getUint64Field(data, "epEui")
	if !ok {
		return fmt.Errorf("%s", ResolveErrorMessage(errMissingEpEui))
	}

	// Route by endpoint ownership before committing to ingest
	if s.dispositionResolver != nil {
		disposition, dispErr := s.dispositionResolver.Resolve(ctx, epEUI)
		if dispErr != nil {
			s.logger.ErrorContext(ctx, LogBSSCIDispositionResolutionFailedRejectingUplink,
				logger.FieldEpEuiSnake, epEUI, logger.FieldError, dispErr)
			return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistULData))
		}
		switch disposition {
		case DispositionDrop:
			// Endpoint unknown, no relay configured; accept silently to prevent BS retry storms
			s.logger.DebugContext(ctx, LogBSSCIEndpointNotFound,
				logger.FieldEpEuiSnake, epEUI)
			return s.sendMessage(session, map[string]interface{}{
				"command": mioty.CmdULDataResponse,
				"opId":    msg.OpId,
			})
		case DispositionRelay:
			if s.relayOutbox != nil {
				// Enqueue the raw BSSCI frame for CE→ECE relay.
				// Send ulDataRsp only if the insert succeeds; on failure send a BSSCI error
				// so the base station retries rather than silently losing the packet.
				if enqErr := s.enqueueRelayUplink(ctx, session, epEUI, data); enqErr != nil {
					s.logger.ErrorContext(ctx, LogBSSCIFailedToEnqueueRelayUplink,
						logger.FieldEpEuiSnake, epEUI, logger.FieldError, enqErr)
					return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistULData))
				}
			} else {
				// No relay configured yet; treat as drop
				s.logger.DebugContext(ctx, LogBSSCIEndpointNotFound, logger.FieldEpEuiSnake, epEUI)
			}
			return s.sendMessage(session, map[string]interface{}{
				"command": mioty.CmdULDataResponse,
				"opId":    msg.OpId,
			})
		case DispositionLocal:
			// Fall through to full ingest below
		}
	}

	// Get packet counter
	packetCnt, ok := getNumericField(data, "packetCnt")
	if !ok {
		return fmt.Errorf("%s", ResolveErrorMessage(errMissingPacketCnt))
	}

	// Handle userData using the normalized helper
	var userData []byte
	if userDataRaw, ok := data["userData"]; ok {
		userData = s.normalizeUserDataField(userDataRaw)
	} else {
		userData = []byte{} // Empty payload is valid per spec
	}
	if len(userData) > mioty.MaxULUserDataBytes {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errULUserDataTooLong))
	}

	// Validate mandatory signal quality metrics per BSSCI 3.10.1
	snr, snrValid := getFloatFieldValidated(data, "snr")
	if !snrValid {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSnrValue))
	}

	rssi, rssiValid := getFloatFieldValidated(data, "rssi")
	if !rssiValid {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRssiValue))
	}

	// Optional eqSnr (BSSCI §3.10.1); a pointer keeps "absent" apart from 0 dB
	var eqSnrPtr *float64
	if eqSnr, hasEqSnr := getFloatFieldValidated(data, "eqSnr"); hasEqSnr {
		eqSnrPtr = &eqSnr
	}

	// Validate rxTime as mandatory field before dedup (BSSCI 3.10.1)
	// Must validate before CheckAndRecord to prevent invalid packets from poisoning dedup cache
	rxTimeValue, hasRxTime := data["rxTime"]
	if !hasRxTime {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errMissingRxTime))
	}

	var rxTime int64
	switch v := rxTimeValue.(type) {
	case int64:
		rxTime = v
	case uint64:
		if v > math.MaxInt64 {
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errRxTimeExceedsMaximum))
		}
		rxTime = int64(v)
	case float64:
		rxTime = int64(v)
	case int:
		rxTime = int64(v)
	case uint:
		rxTime = int64(v) //nolint:gosec // G115: BSSCI rxTime from protocol fits int64
	default:
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRxTimeFormat))
	}

	if rxTime <= 0 {
		return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidRxTimeValue))
	}

	// Optional radio fields (BSSCI §3.8.1)
	var rxDurationPtr *int64
	if rxDuration, ok := getNumericField(data, "rxDuration"); ok {
		rxDurationPtr = &rxDuration
	}
	var profilePtr *string
	if profile, ok := data["profile"].(string); ok {
		profilePtr = &profile
	}
	var modePtr *string
	if mode, ok := data["mode"].(string); ok {
		modePtr = &mode
	}
	var subpacketsPtr *mioty.Subpackets
	if raw := data[wireFieldSubpackets]; raw != nil {
		sp, valid := parseSubpackets(raw)
		if !valid {
			return s.sendError(session, msg.OpId, POSIX_EPROTO, ResolveErrorMessage(errInvalidSubpackets))
		}
		subpacketsPtr = sp
	}

	// Extract additional fields for processing
	dlOpen := getBoolField(data, "dlOpen", false)
	responseExp := getBoolField(data, "responseExp", false)
	dlAck := getBoolField(data, "dlAck", false)

	// Delegate full pipeline to ingest service (dedup → persist → SCACI → MQTT).
	// Downlink dispatch is handled here since it requires the active BSSCI session object.
	if s.uplinkIngestSvc == nil {
		return errUplinkIngestSvcRequired
	}
	{
		epEuiVal := epEUI
		packetCntVal, errToken := safeUint32(packetCnt)
		if errToken != "" {
			return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(errToken))
		}
		var formatPtr *uint8
		if fmtRaw, ok := getNumericField(data, "format"); ok {
			fmtVal, fmtErrToken := safeUint8(fmtRaw)
			if fmtErrToken != "" {
				return s.sendError(session, msg.OpId, POSIX_EINVAL, ResolveErrorMessage(fmtErrToken))
			}
			formatPtr = &fmtVal
		}
		payload := &UplinkPayload{
			OpID:        msg.OpId,
			EpEUI:       epEuiVal,
			BsEUI:       session.BaseStationEUI,
			PacketCnt:   packetCntVal,
			UserData:    userData,
			SNR:         snr,
			RSSI:        rssi,
			EqSNR:       eqSnrPtr,
			RxTime:      rxTime,
			RxDuration:  rxDurationPtr,
			Profile:     profilePtr,
			Mode:        modePtr,
			Format:      formatPtr,
			Subpackets:  subpacketsPtr,
			DLOpen:      dlOpen,
			ResponseExp: responseExp,
			DlAck:       dlAck,
		}
		result, ingestErr := s.uplinkIngestSvc.Ingest(ctx, payload, UplinkIngestOptions{
			Source:          UplinkSourceBSSCI,
			ServingTenantID: resolvedTenant(session, s.tenantID),
		})
		if ingestErr != nil {
			s.logger.ErrorContext(ctx, LogBSSCIUplinkIngestFailed,
				logger.FieldEpEuiSnake, epEuiVal, logger.FieldPacketCntSnake, packetCntVal, logger.FieldError, ingestErr)
			var refusal *CatalogError
			if errors.As(ingestErr, &refusal) {
				return s.sendError(session, msg.OpId, refusal.Posix, ResolveErrorMessage(refusal.Token))
			}
			return s.sendError(session, msg.OpId, POSIX_EIO, ResolveErrorMessage(errFailedToPersistULData))
		}
		// Every reception offers the telegram's window; the dispatcher's claim fills it once.
		if dlOpen && s.downlinkDispatcher != nil {
			ownerCtx := pkgcontext.WithTenantID(ctx, result.OwnerTenantID)
			if result.OwnerOrgUUID != uuid.Nil {
				ownerCtx = pkgcontext.WithOrganizationID(ownerCtx, result.OwnerOrgUUID)
			}
			dispatched, dispErr := s.downlinkDispatcher.DispatchIfAvailable(
				ownerCtx, result.OwnerTenantID,
				session, epEuiVal, result.MessageID, responseExp,
			)
			if dispErr != nil {
				s.logger.WarnContext(ownerCtx, LogBSSCIDownlinkDispatchError,
					logger.FieldEpEui, epEuiVal, logger.FieldError, dispErr)
			} else if dispatched {
				s.logger.InfoContext(ownerCtx, LogBSSCIDownlinkDispatched,
					logger.FieldEpEui, epEuiVal, logger.FieldSessionID, session.ID, logger.FieldOwnerTenantID, result.OwnerTenantID)
			} else {
				s.logger.DebugContext(ownerCtx, LogBSSCIDownlinkWindowOpen,
					logger.FieldEpEui, epEuiVal, logger.FieldResponseExp, responseExp, logger.FieldDlAck, dlAck)
			}
		}
		return s.sendMessage(session, map[string]interface{}{
			"command": mioty.CmdULDataResponse,
			"opId":    msg.OpId,
		})
	}
}

// handleULDataComplete handles UL data completion (BSSCI-3.10.3)
func (s *Server) handleULDataComplete(session *Session, msg *Message, _ map[string]interface{}) error {
	s.logger.DebugContext(s.safeCtx(), LogBSSCIReceivedULDataCompletion,
		logger.FieldSessionID, session.ID,
		logger.FieldOpID, msg.OpId)

	// No response required for completion message (BSSCI spec)
	return nil
}

// normalizeUserDataField converts various MessagePack Numeric[n] formats to []byte
func (s *Server) normalizeUserDataField(userDataRaw interface{}) []byte {
	switch v := userDataRaw.(type) {
	case []byte:
		return v
	case []interface{}:
		// Convert Numeric[n] to []byte
		userData := make([]byte, len(v))
		for i, elem := range v {
			// Canonical numeric coercion enforces integer 0-255 range
			b, err := numericToByte(elem)
			if err != nil {
				s.logger.WarnContext(s.safeCtx(), LogBSSCIUnsupportedUserDataElementType,
					logger.FieldIndex, i,
					logger.FieldType, fmt.Sprintf("%T", elem))
				userData[i] = 0
				continue
			}
			userData[i] = b
		}
		return userData
	case string:
		// Strings might be base64 encoded - log and skip
		s.logger.WarnContext(s.safeCtx(), LogBSSCIReceivedStringUserData,
			logger.FieldValue, v)
		return []byte{}
	default:
		// Log unknown type
		s.logger.WarnContext(s.safeCtx(), LogBSSCIUnknownUserDataType,
			logger.FieldType, fmt.Sprintf("%T", v))
		return []byte{}
	}
}

// enqueueRelayUplink stores the uplink frame in the relay outbox.
func (s *Server) enqueueRelayUplink(ctx context.Context, session *Session, epEUI uint64, data map[string]interface{}) error {
	rawFrame, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf(errFmtEncodeRelayFrame, err)
	}
	_, err = s.relayOutbox.Enqueue(ctx, epEUI, session.BaseStationEUI, rawFrame, s.clock.Now().UnixNano())
	return err
}
