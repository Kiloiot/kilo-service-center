// Package scaci implements the MIOTY Service Center Application Center Interface (SCACI) v1.0.0
package scaci

import "math"

// ============================================================================
// Assembly Validation (SCACI §§2.5, 3.12.1, 3.13.1)
// ============================================================================
//
// These validators enforce spec-required fields at assembly time before frames
// are emitted on the wire. All SC-originated messages (DLDataResult, EPStatus)
// MUST be validated before sendResponse.
//
// Design principles:
// - Pure functions returning error token or empty string
// - Reuse existing tokens (errMissingEpEui, errEpEuiZero) where applicable
// - Minimal validation per spec - only enforce what §§2.5, 3.12.1, 3.13.1 require
// - Zero values allowed unless spec explicitly forbids them

// ValidateDLDataResult validates a DLDataResult message per SCACI §3.12.1
//
// Required fields (always):
//   - epEui: must be non-zero (reuses errEpEuiZero)
//   - queId: must be non-zero (reuses errQueIDZero)
//   - result: must be valid enum value
//   - opId: must be negative (SC-originated per §3.2)
//
// Conditional fields (when result == ResultSent):
//   - bsEui: must be non-nil AND non-zero (valid EUI64)
//   - txTime: must be non-nil (zero IS valid per spec - presence required)
//   - packetCnt: must be non-nil (zero IS valid per spec - presence required)
//
// Returns error token on failure, empty string on success.
func ValidateDLDataResult(msg *DLDataResult) string {
	if msg == nil {
		return errInvalidMessageFormat // Caller should not pass nil
	}

	// Mandatory: epEui non-zero (reuse existing tokens)
	if msg.EpEui == 0 {
		return errEpEuiZero
	}

	// Mandatory: queId non-zero
	if msg.QueID == 0 {
		return errQueIDZero
	}

	// Mandatory: result must be present and valid enum
	if msg.Result == "" {
		return errDLDataResultMissingResult
	}
	if !ValidDLDataResults[msg.Result] {
		return errDLDataResultInvalidResultEnum
	}

	// Mandatory: opId must be negative (SC-originated per §3.2)
	if msg.OpId >= 0 {
		return errDLDataResultOpIDNotNegative
	}

	// Conditional: when result == ResultSent, additional fields required
	if msg.Result == ResultSent {
		// bsEui must be non-nil AND non-zero (valid EUI64)
		if msg.BsEui == nil {
			return errDLDataResultSentMissingBsEui
		}
		if *msg.BsEui == 0 {
			return errDLDataResultSentInvalidBsEui
		}

		// txTime must be present (zero IS valid per spec)
		if msg.TxTime == nil {
			return errDLDataResultSentMissingTxTime
		}

		// packetCnt must be present (zero IS valid per spec)
		if msg.PacketCnt == nil {
			return errDLDataResultSentMissingPktCnt
		}
	}

	return ""
}

// ValidateEPStatus validates an EPStatus message per SCACI §§3.2, 3.13.1
//
// Required fields (always):
//   - opId: must be negative (SC-originated per §3.2)
//   - epEui: must be non-zero (reuses errEpEuiZero)
//   - epStatus: must be valid enum value (EPStatusAttached/EPStatusDetached)
//
// OTA field requirements per §3.13.1:
//   - attached: attachCnt, nonce, sign required
//   - detached: sign required
//   - eqSnr, subpackets: optional (not validated)
//
// Returns error token on failure, empty string on success.
func ValidateEPStatus(msg *EPStatus, opId int64) string {
	if msg == nil {
		return errInvalidMessageFormat // Caller should not pass nil
	}

	// Mandatory: opId must be negative (SC-originated per §3.2)
	if opId >= 0 {
		return errOpIDSignMismatch
	}

	// Mandatory: epEui non-zero (reuse existing tokens)
	if msg.EpEui == 0 {
		return errEpEuiZero
	}

	// Mandatory: epStatus must be present and valid enum
	if msg.EpStatus == "" {
		return errEPStatusMissingStatus
	}
	if !ValidEPStatuses[msg.EpStatus] {
		return errEPStatusInvalidStatusEnum
	}

	// OTA field requirements per §3.13.1
	switch msg.EpStatus {
	case EPStatusAttached:
		// attached: attachCnt, nonce, sign required
		if msg.AttachCnt == nil {
			return errEPStatusMissingAttachCnt
		}
		if msg.Nonce == nil {
			return errEPStatusMissingNonce
		}
		if msg.Sign == nil {
			return errEPStatusMissingSign
		}
	case EPStatusDetached:
		// detached: sign required
		if msg.Sign == nil {
			return errEPStatusMissingSign
		}
	}

	return ""
}

// ValidateConnectResponse validates a ConnectResponse message per SCACI §3.3.2
//
// Required fields (always):
//   - scEui: must be non-zero
//   - snScUuid: must be non-zero (all-zeros UUID is invalid)
//
// Optional fields (no validation needed):
//   - version: SCACI §3.3.2 specifies optional with default to requested version
//     (caller applies default before validation if needed)
//   - vendor, model, name, swVersion, info
//   - snResume: bool, false is valid
//
// Returns error token on failure, empty string on success.
func ValidateConnectResponse(msg *ConnectResponse) string {
	if msg == nil {
		return errInvalidMessageFormat // Caller should not pass nil
	}

	// Note: version is OPTIONAL per SCACI §3.3.2 - defaults to requested version
	// Caller (SendConnectResponse) applies default before wire emission

	// Mandatory: scEui non-zero
	if msg.ScEui == 0 {
		return errConnectResponseMissingScEui
	}

	// Mandatory: snScUuid non-zero (all-zeros is invalid session UUID)
	var zeroUUID UUID16
	if msg.SnScUUID == zeroUUID {
		return errConnectResponseMissingSnScUUID
	}

	return ""
}

// ValidateStatusResponse validates a StatusResponse message per SCACI §3.5.2
//
// Required fields (always):
//   - code: int, 0 IS valid (means "ok")
//   - message: must be non-empty
//   - time: must be non-zero (Unix timestamp)
//
// Optional fields (no validation needed):
//   - uptime, baseStations[]
//
// Returns error token on failure, empty string on success.
func ValidateStatusResponse(msg *StatusResponse) string {
	if msg == nil {
		return errInvalidMessageFormat // Caller should not pass nil
	}

	// Note: code is an int, 0 IS valid per spec (means "ok")
	// No validation needed for code field

	// Mandatory: message must be non-empty
	if msg.Message == "" {
		return errStatusResponseMissingMessage
	}

	// Mandatory: time must be non-zero (Unix UTC timestamp)
	if msg.Time == 0 {
		return errStatusResponseMissingTime
	}

	return ""
}

// ValidateULData validates a ULData message per SCACI §3.8.1
//
// Required fields (always):
//   - epEui: must be non-zero (reuses errEpEuiZero)
//   - baseStations: must be non-empty slice with valid entries
//   - userData: must not be nil (empty slice IS valid)
//   - packetCnt: uint32 value type - zero IS valid, no validation needed
//   - dlOpen, responseExp, dlAck: bool value types - false IS valid
//
// Per-baseStation validation (SCACI §3.8.1):
//   - bsEui: must be non-zero (valid EUI64)
//   - rxTime: must be positive (Unix UTC ns timestamp)
//   - snr, rssi: must not be NaN or Inf
//   - eqSnr (if present): must not be NaN or Inf
//   - rxDuration (if present): must be non-negative
//   - subpackets (if present): snr/rssi arrays must have equal length
//
// Optional fields (no validation needed):
//   - format, duplicate
//
// Returns error token on failure, empty string on success.
func ValidateULData(msg *ULData) string {
	if msg == nil {
		return errInvalidMessageFormat // Caller should not pass nil
	}

	// Mandatory: epEui non-zero (reuse existing token)
	if msg.EpEui == 0 {
		return errEpEuiZero
	}

	// Mandatory: baseStations must be non-empty
	if len(msg.BaseStations) == 0 {
		return errULDataMissingBaseStations
	}

	// Per-baseStation validation per SCACI §3.8.1
	for _, bs := range msg.BaseStations {
		// bsEui must be non-zero (valid EUI64)
		if bs.BsEui == 0 {
			return errULDataInvalidBsEui
		}

		// rxTime must be positive (Unix UTC ns)
		if bs.RxTime <= 0 {
			return errULDataInvalidRxTime
		}

		// snr must not be NaN or Inf
		if math.IsNaN(bs.Snr) || math.IsInf(bs.Snr, 0) {
			return errULDataInvalidSnr
		}

		// rssi must not be NaN or Inf
		if math.IsNaN(bs.Rssi) || math.IsInf(bs.Rssi, 0) {
			return errULDataInvalidRssi
		}

		// eqSnr (if present) must not be NaN or Inf - 0.0 IS valid
		if bs.EqSnr != nil {
			if math.IsNaN(*bs.EqSnr) || math.IsInf(*bs.EqSnr, 0) {
				return errULDataInvalidEqSnr
			}
		}

		// rxDuration (if present) must be non-negative
		if bs.RxDuration != nil && *bs.RxDuration < 0 {
			return errULDataInvalidRxDuration
		}

		// subpackets (if present): snr and rssi arrays must have equal length
		if bs.Subpackets != nil {
			if len(bs.Subpackets.SNR) != len(bs.Subpackets.RSSI) {
				return errULDataSubpacketLengthMismatch
			}
		}
	}

	// Mandatory: userData must not be nil (empty slice IS valid per spec)
	if msg.UserData == nil {
		return errUserDataEmpty
	}

	// Note: packetCnt, dlOpen, responseExp, dlAck are value types
	// Go zero values (0, false) ARE valid per SCACI spec - no validation needed

	return ""
}

// ValidateULDataTransmit validates a ULDataTransmit message per SCACI §3.9.1
//
// Required fields (always):
//   - epEui: must be non-zero (reuses errEpEuiZero)
//   - nwkSnKey: must be exactly 16 bytes (enforced at compile time via [16]byte)
//   - shAddr: value type, zero IS valid per spec (presence mandated, not value)
//   - packetCnt: value type, zero IS valid per spec (presence mandated, not value)
//   - userData: must not be nil (empty slice IS valid per spec)
//
// Optional fields (no validation needed):
//   - bsEui: optional target base station
//   - profile: optional profile identifier
//   - format: optional, defaults to 0 per §2.4 (handled at handler level)
//
// Returns error token on failure, empty string on success.
func ValidateULDataTransmit(msg *ULDataTransmit) string {
	if msg == nil {
		return errInvalidMessageFormat // Caller should not pass nil
	}

	// Mandatory: epEui non-zero (reuse existing token)
	if msg.EpEui == 0 {
		return errEpEuiZero
	}

	// Mandatory: userData must not be nil (empty slice IS valid per spec)
	if msg.UserData == nil {
		return errUserDataEmpty
	}

	// Note: nwkSnKey is [16]byte - type system enforces length at compile time
	// Note: shAddr and packetCnt are value types; spec only mandates presence,
	// not non-zero values. Zero is valid per current spec.

	return ""
}

// ValidateError validates an Error message per SCACI §3.14.1
//
// Required fields (always):
//   - code: must be non-zero (POSIX_OK=0 is semantically invalid for Error)
//     Per SCACI §3.14.1, Error messages indicate failures; code=0 (success)
//     would be contradictory. Reject to prevent silent misuse.
//   - message: must be non-empty
//
// Optional fields (no validation needed):
//   - errorToken
//
// Returns error token on failure, empty string on success.
func ValidateError(msg *Error) string {
	if msg == nil {
		return errInvalidMessageFormat // Caller should not pass nil
	}

	// Mandatory: code must be non-zero (POSIX_OK=0 invalid for Error messages)
	if msg.Code == 0 {
		return errErrorMissingCode
	}

	// Mandatory: message must be non-empty
	if msg.Message == "" {
		return errErrorMissingMessage
	}

	return ""
}

// ValidateOpIDSign validates opId sign matches command initiator per SCACI §3.2
//
// Per SCACI §3.2:
//   - AC-initiated commands require positive opId (> 0)
//   - SC-initiated commands require negative opId (< 0)
//   - Connect handshake uses opId=0 (validated separately in routeMessage)
//   - Ping/Error can be initiated by either party (any sign valid)
//
// On failure, caller MUST send POSIX_EINVAL with returned token and skip opId persistence.
//
// Returns error token if validation fails, empty string on success.
func ValidateOpIDSign(command string, opId int64) string {
	initiator, exists := CommandInitiatorMap[command]
	if !exists {
		return "" // Unknown command handled via unsupported command error
	}

	switch initiator {
	case InitiatorConnect:
		// opId=0 validation done separately in routeMessage
		return ""
	case InitiatorEither:
		// Ping/Error can use either sign
		return ""
	case InitiatorAC:
		if opId <= 0 {
			return errOpIDSignMismatch
		}
	case InitiatorSC:
		if opId >= 0 {
			return errOpIDSignMismatch
		}
	}
	return ""
}
